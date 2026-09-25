//go:build darwin

// Beszel issue #2416 test: runs the gopsutil calls the agent makes on startup,
// one at a time, so a crash shows exactly which call caused it.
//
// Usage:
//
//	go run .          # all checks, skipping cpu.Info()
//	go run . -repro   # also run cpu.Info() at the end
//
// go.mod replaces gopsutil with the fix from shirou/gopsutil#2163, so with
// -repro cpu.Info() should print "Apple M6 @ 0 MHz" instead of crashing.
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
	"golang.org/x/sys/unix"
)

func step(name string, fn func() (any, error)) {
	fmt.Printf("%-28s ... ", name)
	v, err := fn()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("OK  %v\n", v)
}

func main() {
	repro := flag.Bool("repro", false, "also call cpu.Info() (crashes on affected Macs)")
	flag.Parse()

	// The proposed fix: read the CPU model directly instead of via cpu.Info().
	step("sysctl brand_string", func() (any, error) { return unix.Sysctl("machdep.cpu.brand_string") })

	step("host.Info", func() (any, error) {
		i, err := host.Info()
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("%s %s %s", i.Platform, i.PlatformVersion, i.KernelArch), nil
	})
	step("host.HostID", func() (any, error) { return host.HostID() })
	step("host.Uptime", func() (any, error) { return host.Uptime() })
	step("cpu.Counts(false)", func() (any, error) { return cpu.Counts(false) })
	step("cpu.Counts(true)", func() (any, error) { return cpu.Counts(true) })
	step("cpu.Times", func() (any, error) {
		t, err := cpu.Times(false)
		return len(t), err
	})
	step("mem.VirtualMemory", func() (any, error) {
		m, err := mem.VirtualMemory()
		if err != nil {
			return nil, err
		}
		return m.Total, nil
	})
	step("load.Avg", func() (any, error) { return load.Avg() })
	step("disk.Partitions", func() (any, error) {
		p, err := disk.Partitions(false)
		return len(p), err
	})
	step("disk.Usage(/)", func() (any, error) {
		u, err := disk.Usage("/")
		if err != nil {
			return nil, err
		}
		return u.Total, nil
	})
	step("disk.IOCounters", func() (any, error) {
		c, err := disk.IOCounters()
		return len(c), err
	})
	step("net.IOCounters", func() (any, error) {
		c, err := psnet.IOCounters(true)
		return len(c), err
	})
	step("sensors.Temperatures", func() (any, error) {
		t, err := sensors.TemperaturesWithContext(context.Background())
		return len(t), err
	})

	if *repro {
		fmt.Println("\nRunning cpu.Info() - a crash here confirms the bug:")
		step("cpu.Info", func() (any, error) {
			i, err := cpu.Info()
			if err != nil || len(i) == 0 {
				return nil, err
			}
			return fmt.Sprintf("%s @ %.0f MHz", i[0].ModelName, i[0].Mhz), nil
		})
	}

	fmt.Println("\nDone - no crash.")
}
