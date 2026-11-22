//go:build !windows

package awg

import (
	"fmt"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"

	"github.com/NetWorkClub/NetPortal/core/parser"
)

func configureDesktopTUN(ifName string, cfg *parser.AWGConfig) ([]string, error) {
	var bypassIPs []string
	gw, dev := findDefaultRouteUnix()

	if gw != "" && dev != "" {
		for _, peer := range cfg.Peers {
			if strings.TrimSpace(peer.Endpoint) == "" {
				continue
			}
			resolved, err := resolveEndpointAddrPort(peer.Endpoint)
			if err != nil {
				continue
			}
			ap, err := netip.ParseAddrPort(resolved)
			if err != nil || !ap.Addr().Is4() {
				continue
			}
			ipStr := ap.Addr().String()
			if err := exec.Command("ip", "route", "replace", ipStr+"/32", "via", gw, "dev", dev).Run(); err == nil {
				bypassIPs = append(bypassIPs, ipStr)
			}
		}
	}

	for _, raw := range cfg.Interface.Addresses {
		if _, err := netip.ParsePrefix(raw); err == nil {
			if err := exec.Command("ip", "address", "add", raw, "dev", ifName).Run(); err != nil {
				return bypassIPs, fmt.Errorf("ip address add %s: %w", raw, err)
			}
		}
	}
	if cfg.Interface.MTU > 0 {
		_ = exec.Command("ip", "link", "set", "dev", ifName, "mtu", strconv.Itoa(cfg.Interface.MTU), "up").Run()
	} else {
		_ = exec.Command("ip", "link", "set", "dev", ifName, "up").Run()
	}
	for _, peer := range cfg.Peers {
		for _, allowed := range peer.AllowedIPs {
			p, err := netip.ParsePrefix(allowed)
			if err != nil {
				continue
			}
			if p.Bits() == 0 && p.Addr().Is4() {
				_ = exec.Command("ip", "route", "replace", "0.0.0.0/1", "dev", ifName).Run()
				_ = exec.Command("ip", "route", "replace", "128.0.0.0/1", "dev", ifName).Run()
			} else {
				_ = exec.Command("ip", "route", "replace", p.String(), "dev", ifName).Run()
			}
		}
	}

	if len(cfg.Interface.DNS) > 0 {
		if _, err := exec.LookPath("resolvectl"); err == nil {
			args := []string{"dns", ifName}
			for _, rawDNS := range cfg.Interface.DNS {
				if addr, err := netip.ParseAddr(strings.TrimSpace(rawDNS)); err == nil {
					args = append(args, addr.String())
				}
			}
			if len(args) > 2 {
				_ = exec.Command("resolvectl", args...).Run()
			}
		}
	}

	return bypassIPs, nil
}

func cleanupDesktopTUN(ifName string, bypassIPs []string) {
	if ifName != "" {
		if _, err := exec.LookPath("resolvectl"); err == nil {
			_ = exec.Command("resolvectl", "revert", ifName).Run()
		}
		_ = exec.Command("ip", "route", "del", "0.0.0.0/1", "dev", ifName).Run()
		_ = exec.Command("ip", "route", "del", "128.0.0.0/1", "dev", ifName).Run()
		_ = exec.Command("ip", "link", "set", "dev", ifName, "down").Run()
	}
	for _, ip := range bypassIPs {
		_ = exec.Command("ip", "route", "del", ip+"/32").Run()
	}
}

func findDefaultRouteUnix() (gw string, dev string) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "", ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] == "via" {
				gw = fields[i+1]
			}
			if fields[i] == "dev" {
				dev = fields[i+1]
			}
		}
		if gw != "" && dev != "" {
			return gw, dev
		}
	}
	return "", ""
}
