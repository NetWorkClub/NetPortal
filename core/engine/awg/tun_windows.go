//go:build windows

package awg

import (
	"errors"

	"github.com/amnezia-vpn/amneziawg-go/v3/tun"
)

func openTUNFromFD(fd int, mtu int) (tun.Device, error) {
	return nil, errors.New("awg: TunFD handoff is not supported on Windows")
}
