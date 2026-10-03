package hub

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"path/filepath"
	"strings"
	"time"

	"github.com/henrygd/beszel/internal/hub/utils"

	"github.com/pocketbase/pocketbase/core"
	"tailscale.com/net/tsaddr"
	"tailscale.com/tsnet"
)

// tailscale runs an embedded Tailscale node so the hub is reachable on the
// tailnet (and optionally the public internet via Funnel) at
// https://<TS_HOSTNAME>.<tailnet>.ts.net, without a separate tailscaled.
type tailscale struct {
	app     core.App
	srv     *tsnet.Server
	funnel  bool
	domain  string // MagicDNS name, set once listening
	servers []*http.Server
}

// newTailscale returns a tailscale node configured from the environment,
// or nil if TS_HOSTNAME is not set.
func newTailscale(app core.App) *tailscale {
	hostname, _ := utils.GetEnv("TS_HOSTNAME")
	if hostname == "" {
		return nil
	}
	stateDir, _ := utils.GetEnv("TS_STATE_DIR")
	if stateDir == "" {
		stateDir = filepath.Join(app.DataDir(), "tsnet")
	}
	authKey, _ := utils.GetEnv("TS_AUTHKEY")
	funnel, _ := utils.GetEnv("TS_FUNNEL")
	logger := app.Logger().With("component", "tailscale")
	return &tailscale{
		app:    app,
		funnel: funnel == "true",
		srv: &tsnet.Server{
			Hostname: hostname,
			Dir:      stateDir,
			AuthKey:  authKey,
			// backend logs are very verbose, so only show them in debug
			Logf: func(format string, args ...any) {
				logger.Debug(strings.TrimSpace(fmt.Sprintf(format, args...)))
			},
			// user-facing messages, such as the login URL
			UserLogf: func(format string, args ...any) {
				logger.Info(strings.TrimSpace(fmt.Sprintf(format, args...)))
			},
		},
	}
}

// serve starts listening on the tailnet in the background and serves handler
// over HTTPS, redirecting plain HTTP to HTTPS. base supplies server timeouts.
func (ts *tailscale) serve(handler http.Handler, base *http.Server) {
	newServer := func(h http.Handler) *http.Server {
		srv := &http.Server{
			Handler:           h,
			ReadTimeout:       base.ReadTimeout,
			ReadHeaderTimeout: base.ReadHeaderTimeout,
			WriteTimeout:      base.WriteTimeout,
			BaseContext:       base.BaseContext,
		}
		ts.servers = append(ts.servers, srv)
		return srv
	}
	httpsServer := newServer(handler)
	httpServer := newServer(http.HandlerFunc(ts.redirectToHTTPS))

	go func() {
		logger := ts.app.Logger().With("component", "tailscale")
		// listening blocks until the node is logged in
		var ln net.Listener
		var err error
		if ts.funnel {
			ln, err = ts.srv.ListenFunnel("tcp", ":443")
		} else {
			ln, err = ts.srv.ListenTLS("tcp", ":443")
		}
		if err != nil {
			logger.Error("Failed to listen on tailnet", "err", err)
			return
		}
		if domains := ts.srv.CertDomains(); len(domains) > 0 {
			ts.domain = domains[0]
			url := "https://" + ts.domain
			logger.Info("Serving on tailnet", "url", url, "funnel", ts.funnel)
			if appURL, _ := utils.GetEnv("APP_URL"); appURL == "" {
				logger.Info("Set APP_URL=" + url + " to use this address in links and agent install commands")
			}
		}
		if httpLn, err := ts.srv.Listen("tcp", ":80"); err != nil {
			logger.Error("Failed to listen on tailnet", "err", err)
		} else {
			go httpServer.Serve(httpLn)
		}
		if err := httpsServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Tailnet server stopped", "err", err)
		}
	}()
}

// close shuts down the tailnet servers and the Tailscale node.
func (ts *tailscale) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, srv := range ts.servers {
		_ = srv.Shutdown(ctx)
	}
	_ = ts.srv.Close()
}

// Handles reports whether host is a Tailscale IP or MagicDNS name, so SSH
// connections to agents on the tailnet go through the embedded node.
func (ts *tailscale) Handles(host string) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		return tsaddr.IsTailscaleIP(addr)
	}
	return strings.HasSuffix(strings.TrimSuffix(host, "."), ".ts.net")
}

// DialContext dials address on the tailnet.
func (ts *tailscale) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return ts.srv.Dial(ctx, network, address)
}

// redirectToHTTPS redirects plain HTTP on the tailnet to the HTTPS address,
// using the full MagicDNS name since the certificate doesn't cover short names.
func (ts *tailscale) redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	host := ts.domain
	if host == "" {
		host = r.Host
	}
	http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusMovedPermanently)
}

// tailscaleStatus is the hub's Tailscale node status returned to the UI.
type tailscaleStatus struct {
	Enabled      bool     `json:"enabled"`
	Funnel       bool     `json:"funnel"`
	BackendState string   `json:"backendState"`
	AuthURL      string   `json:"authURL,omitempty"`
	Version      string   `json:"version"`
	Tailnet      string   `json:"tailnet,omitempty"`
	URL          string   `json:"url,omitempty"`
	IPs          []string `json:"ips"`
	Health       []string `json:"health"`
}

// status returns the node's current status.
func (ts *tailscale) status(ctx context.Context) (*tailscaleStatus, error) {
	lc, err := ts.srv.LocalClient()
	if err != nil {
		return nil, err
	}
	st, err := lc.Status(ctx)
	if err != nil {
		return nil, err
	}
	res := &tailscaleStatus{
		Enabled:      true,
		Funnel:       ts.funnel,
		BackendState: st.BackendState,
		AuthURL:      st.AuthURL,
		Version:      st.Version,
		IPs:          make([]string, 0, len(st.TailscaleIPs)),
		Health:       st.Health,
	}
	if res.Health == nil {
		res.Health = []string{}
	}
	if st.CurrentTailnet != nil {
		res.Tailnet = st.CurrentTailnet.Name
	}
	if len(st.CertDomains) > 0 {
		res.URL = "https://" + st.CertDomains[0]
	}
	for _, ip := range st.TailscaleIPs {
		res.IPs = append(res.IPs, ip.String())
	}
	return res, nil
}
