//go:build !windows

package awg

import (
	"os"

	"github.com/amnezia-vpn/amneziawg-go/v3/tun"
)

func openTUNFromFD(fd int, mtu int) (tun.Device, error) {
	file := os.NewFile(uintptr(fd), "netportal-tun")
	return tun.CreateTUNFromFile(file, mtu)
}
