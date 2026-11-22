//go:build windows

package awg

import (
	"fmt"
	"net/netip"
	"os/exec"
	"strings"

	"github.com/NetWorkClub/NetPortal/core/parser"
)

func configureDesktopTUN(ifName string, cfg *parser.AWGConfig) error {
	for _, raw := range cfg.Interface.Addresses {
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			continue
		}
		if p.Addr().Is4() {
			mask := prefixToMaskV4(p.Bits())
			if err := exec.Command("netsh", "interface", "ip", "set", "address",
				"name="+ifName, "static", p.Addr().String(), mask).Run(); err != nil {
				return fmt.Errorf("netsh set IPv4 address: %w", err)
			}
		}
	}
	for i, dns := range cfg.Interface.DNS {
		if a, err := netip.ParseAddr(dns); err == nil && a.Is4() {
			if i == 0 {
				_ = exec.Command("netsh", "interface", "ip", "set", "dns",
					"name="+ifName, "static", a.String()).Run()
			} else {
				_ = exec.Command("netsh", "interface", "ip", "add", "dns",
					"name="+ifName, a.String(), fmt.Sprintf("index=%d", i+1)).Run()
			}
		}
	}
	if cfg.Interface.MTU > 0 {
		_ = exec.Command("netsh", "interface", "ipv4", "set", "subinterface",
			ifName, fmt.Sprintf("mtu=%d", cfg.Interface.MTU), "store=active").Run()
	}
	for _, peer := range cfg.Peers {
		for _, allowed := range peer.AllowedIPs {
			p, err := netip.ParsePrefix(allowed)
			if err != nil || !p.Addr().Is4() {
				continue
			}
			if p.Bits() == 0 {
				_ = exec.Command("netsh", "interface", "ipv4", "add", "route",
					"0.0.0.0/1", ifName).Run()
				_ = exec.Command("netsh", "interface", "ipv4", "add", "route",
					"128.0.0.0/1", ifName).Run()
			} else {
				_ = exec.Command("netsh", "interface", "ipv4", "add", "route",
					p.String(), ifName).Run()
			}
		}
	}
	return nil
}

func cleanupDesktopTUN(ifName string, _ *parser.AWGConfig) {
	if ifName == "" {
		return
	}
	_ = exec.Command("netsh", "interface", "ipv4", "delete", "route", "0.0.0.0/1", ifName).Run()
	_ = exec.Command("netsh", "interface", "ipv4", "delete", "route", "128.0.0.0/1", ifName).Run()
}

func prefixToMaskV4(bits int) string {
	if bits < 0 {
		bits = 0
	}
	if bits > 32 {
		bits = 32
	}
	var m uint32
	if bits > 0 {
		m = ^uint32(0) << (32 - bits)
	}
	parts := []string{
		fmt.Sprintf("%d", (m>>24)&0xFF),
		fmt.Sprintf("%d", (m>>16)&0xFF),
		fmt.Sprintf("%d", (m>>8)&0xFF),
		fmt.Sprintf("%d", m&0xFF),
	}
	return strings.Join(parts, ".")
}
