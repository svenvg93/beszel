package agent

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/entities/monitor"
)

const networkMonitorUserAgent = "Beszel-Agent/" + beszel.Version + " (+https://beszel.dev)"

// monitorProbe performs one check. Errors are recorded as loss by the task runner.
// Implementations must honor cancellation and bound their execution time.
type monitorProbe func(context.Context, monitor.Config) (probeResult, error)

// probeResult is the outcome of one successful check.
type probeResult struct {
	responseUs int64
	// timings is set by HTTP checks.
	timings *monitor.HTTPTimings
}

// monitorHTTPClients returns the HTTP client for an IP version and TLS mode.
type monitorHTTPClients func(ipVersion uint8, skipTLSVerify bool) *http.Client

type monitorHTTPClientKey struct {
	ipVersion     uint8
	skipTLSVerify bool
}

// newMonitorHTTPClients builds one client per IP version and TLS mode. Each has its
// own transport, so an idle connection of one family is never reused by another.
func newMonitorHTTPClients() monitorHTTPClients {
	clients := make(map[monitorHTTPClientKey]*http.Client, 6)
	for _, ipVersion := range []uint8{monitor.IPVersionAuto, monitor.IPVersion4, monitor.IPVersion6} {
		for _, skipTLSVerify := range []bool{false, true} {
			clients[monitorHTTPClientKey{ipVersion, skipTLSVerify}] = newMonitorHTTPClient(ipVersion, skipTLSVerify)
		}
	}
	return func(ipVersion uint8, skipTLSVerify bool) *http.Client {
		return clients[monitorHTTPClientKey{ipVersion, skipTLSVerify}]
	}
}

func newMonitorHTTPClient(ipVersion uint8, skipTLSVerify bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Every check opens a new connection, so DNS, connect and TLS are measured
	// each time and the response time does not depend on whether the interval
	// is shorter than the idle connection timeout.
	transport.DisableKeepAlives = true
	if skipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	network := monitor.Network("tcp", ipVersion)
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, _, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			host, _, _ := net.SplitHostPort(address)
			return nil, lookupError(err, ipVersion, host)
		}
		return conn, nil
	}
	return &http.Client{Timeout: monitor.MaxProbeTimeout, Transport: transport}
}

// networkMonitorProbe returns a probe that picks the HTTP client for each
// monitor's IP version and TLS mode from clients.
func networkMonitorProbe(clients monitorHTTPClients) monitorProbe {
	return func(ctx context.Context, config monitor.Config) (probeResult, error) {
		if !monitor.ValidIPVersion(config.IPVersion) {
			return probeResult{}, fmt.Errorf("unknown monitor IP version: %d", config.IPVersion)
		}
		var responseUs int64
		var err error
		switch config.Protocol {
		case "icmp":
			responseUs, err = monitorICMP(ctx, config.Target, config.IPVersion)
		case "tcp":
			responseUs, err = monitorTCP(ctx, config.Target, config.Port, config.IPVersion)
		case "http":
			var timings monitor.HTTPTimings
			responseUs, timings, err = monitorHTTP(ctx, clients(config.IPVersion, config.SkipTLSVerify), config)
			return probeResult{responseUs: responseUs, timings: &timings}, err
		case "dns":
			responseUs, err = monitorDNS(ctx, config.Target, config.Server, config.IPVersion)
		default:
			err = fmt.Errorf("unknown monitor protocol: %s", config.Protocol)
		}
		return probeResult{responseUs: responseUs}, err
	}
}

// lookupMonitorIPs resolves host to addresses of the given IP version. It never
// returns an empty slice without an error.
func lookupMonitorIPs(ctx context.Context, resolver *net.Resolver, host string, ipVersion uint8) ([]net.IP, error) {
	ips, err := resolver.LookupIP(ctx, monitor.Network("ip", ipVersion), host)
	if err != nil {
		return nil, lookupError(err, ipVersion, host)
	}
	if len(ips) == 0 {
		return nil, noAddressError(ipVersion, host)
	}
	return ips, nil
}

// lookupError replaces Go's resolver errors, such as "lookup example.com on
// 127.0.0.53:53: no such host", with shorter messages naming the host. Go also
// reports "no such host" when only the other family's records exist.
func lookupError(err error, ipVersion uint8, host string) error {
	var dnsErr *net.DNSError
	var addrErr *net.AddrError
	switch {
	case errors.As(err, &dnsErr) && dnsErr.IsNotFound, errors.As(err, &addrErr):
		return noAddressError(ipVersion, host)
	case dnsErr != nil && dnsErr.IsTimeout:
		return fmt.Errorf("DNS lookup for %s timed out", host)
	case dnsErr != nil:
		return fmt.Errorf("DNS lookup for %s failed: %s", host, dnsErr.Err)
	default:
		return err
	}
}

func noAddressError(ipVersion uint8, host string) error {
	if ipVersion == monitor.IPVersionAuto {
		return fmt.Errorf("no address found for %s", host)
	}
	return fmt.Errorf("no IPv%d address for %s", ipVersion, host)
}

// monitorTCP measures connection establishment time, including address fallback
// but excluding DNS resolution.
// Returns -1 and an error on failure.
func monitorTCP(ctx context.Context, target string, port uint16, ipVersion uint8) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Resolve DNS first, outside the timing window but within the probe deadline.
	ips, err := lookupMonitorIPs(ctx, net.DefaultResolver, target, ipVersion)
	if err != nil {
		return -1, err
	}
	network := monitor.Network("tcp", ipVersion)
	portString := fmt.Sprintf("%d", port)
	deadline, _ := ctx.Deadline()

	// Share the remaining probe budget across addresses so an unresponsive
	// first address cannot consume all the time available for alternatives.
	start := time.Now()
	for i, ip := range ips {
		if err := ctx.Err(); err != nil {
			return -1, err
		}
		dialer := net.Dialer{Timeout: time.Until(deadline) / time.Duration(len(ips)-i)}
		var conn net.Conn
		conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), portString))
		if err != nil {
			continue
		}
		responseUs := time.Since(start).Microseconds()
		conn.Close()
		return responseUs, nil
	}
	return -1, err
}

// monitorDNS measures DNS resolution response time in microseconds. If server is
// non-empty, the lookup is sent to that DNS server (host or host:port, default
// port 53) instead of the system resolver. IPv4 and IPv6 query only the A or AAAA
// record. Returns -1 and an error on failure.
func monitorDNS(ctx context.Context, target, server string, ipVersion uint8) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resolver := net.DefaultResolver
	if server != "" {
		resolver = dnsResolverForServer(server)
	}

	start := time.Now()
	if _, err := lookupMonitorIPs(ctx, resolver, target, ipVersion); err != nil {
		return -1, err
	}
	return time.Since(start).Microseconds(), nil
}

// dnsResolverForServer builds a resolver that sends lookups to the given DNS
// server address instead of the system resolver. server may be a bare host or
// host:port; when no port is given, the standard DNS port 53 is used.
func dnsResolverForServer(server string) *net.Resolver {
	address := server
	if _, _, err := net.SplitHostPort(server); err != nil {
		address = net.JoinHostPort(server, "53")
	}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, address)
		},
	}
}

// monitorHTTP measures HTTP GET request response in microseconds, along with
// the time spent in each phase of the request.
// config.StatusCodes lists the accepted status codes; see httpStatusAccepted.
// Returns -1 and an error on failure.
func monitorHTTP(ctx context.Context, client *http.Client, config monitor.Config) (int64, monitor.HTTPTimings, error) {
	if client == nil {
		client = http.DefaultClient
	}
	var trace httpTimingTrace
	start := time.Now()
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace.clientTrace()), http.MethodGet, config.Target, nil)
	if err != nil {
		return -1, monitor.HTTPTimings{}, err
	}
	req.Header.Set("User-Agent", networkMonitorUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return -1, monitor.HTTPTimings{}, err
	}
	responseUs := time.Since(start).Microseconds()
	resp.Body.Close()
	if !httpStatusAccepted(config.StatusCodes, resp.StatusCode) {
		return -1, monitor.HTTPTimings{}, fmt.Errorf("HTTP error: %s", resp.Status)
	}
	return responseUs, trace.result(), nil
}

// httpTimingTrace records HTTP phase durations from httptrace hooks. Hooks can
// run on transport goroutines, and connection attempts to several addresses can
// overlap, so all fields are guarded by mu.
type httpTimingTrace struct {
	mu           sync.Mutex
	timings      monitor.HTTPTimings
	dnsStart     time.Time
	connectStart time.Time
	tlsStart     time.Time
	wroteRequest time.Time
}

func (t *httpTimingTrace) clientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { t.start(&t.dnsStart) },
		DNSDone: func(info httptrace.DNSDoneInfo) {
			t.done(&t.dnsStart, &t.timings.DNS, info.Err)
		},
		// Measure from the first attempt to the one that succeeds, so time lost
		// on unreachable addresses counts toward the connect phase.
		ConnectStart: func(string, string) {
			t.mu.Lock()
			if t.connectStart.IsZero() {
				t.connectStart = time.Now()
			}
			t.mu.Unlock()
		},
		ConnectDone: func(_, _ string, err error) {
			t.done(&t.connectStart, &t.timings.Connect, err)
		},
		TLSHandshakeStart: func() { t.start(&t.tlsStart) },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			t.done(&t.tlsStart, &t.timings.TLS, err)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				t.start(&t.wroteRequest)
			}
		},
		GotFirstResponseByte: func() {
			t.done(&t.wroteRequest, &t.timings.Server, nil)
		},
	}
}

func (t *httpTimingTrace) start(at *time.Time) {
	t.mu.Lock()
	*at = time.Now()
	t.mu.Unlock()
}

// done adds the time since *start to *total and clears *start for the next
// redirect hop. A failed step adds nothing; for connects, another address may
// still succeed and is then measured from the first attempt.
func (t *httpTimingTrace) done(start *time.Time, total *int64, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err != nil || start.IsZero() {
		return
	}
	*total += time.Since(*start).Microseconds()
	*start = time.Time{}
}

func (t *httpTimingTrace) result() monitor.HTTPTimings {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.timings
}

// httpStatusAccepted reports whether code counts as a successful probe. spec is
// a comma-separated list of codes and inclusive ranges such as "200-299,401".
// An empty spec accepts any status below 400. Entries that cannot be parsed
// match nothing, so a malformed spec fails the probe instead of hiding an outage.
func httpStatusAccepted(spec string, code int) bool {
	if spec == "" {
		return code < 400
	}
	for entry := range strings.SplitSeq(spec, ",") {
		low, high, isRange := strings.Cut(strings.TrimSpace(entry), "-")
		first, err := strconv.Atoi(strings.TrimSpace(low))
		if err != nil {
			continue
		}
		last := first
		if isRange {
			if last, err = strconv.Atoi(strings.TrimSpace(high)); err != nil {
				continue
			}
		}
		if code >= first && code <= last {
			return true
		}
	}
	return false
}
