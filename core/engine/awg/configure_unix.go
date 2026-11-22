//go:build !windows

package awg

import (
	"fmt"
	"net/netip"
	"os/exec"
	"strconv"

	"github.com/NetWorkClub/NetPortal/core/parser"
)

func configureDesktopTUN(ifName string, cfg *parser.AWGConfig) error {
	for _, raw := range cfg.Interface.Addresses {
		if _, err := netip.ParsePrefix(raw); err == nil {
			if err := exec.Command("ip", "address", "add", raw, "dev", ifName).Run(); err != nil {
				return fmt.Errorf("ip address add %s: %w", raw, err)
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
	return nil
}

func cleanupDesktopTUN(ifName string, _ *parser.AWGConfig) {
	if ifName == "" {
		return
	}
	_ = exec.Command("ip", "route", "del", "0.0.0.0/1", "dev", ifName).Run()
	_ = exec.Command("ip", "route", "del", "128.0.0.0/1", "dev", ifName).Run()
	_ = exec.Command("ip", "link", "set", "dev", ifName, "down").Run()
}
