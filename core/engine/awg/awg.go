// Package awg wraps github.com/amnezia-vpn/amneziawg-go/v3 for WireGuard and
// AmneziaWG 2.0 / 3.0 / 3.1 tunnels on Desktop and Android.
package awg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
	"github.com/NetWorkClub/NetPortal/core/parser"
	"github.com/amnezia-vpn/amneziawg-go/v3/conn"
	"github.com/amnezia-vpn/amneziawg-go/v3/device"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun/netstack"
)

const EngineID = "awg"

// Engine implements engine.Engine using amneziawg-go/v3.
type Engine struct {
	mu           sync.RWMutex
	running      bool
	dev          *device.Device
	tunDev       tun.Device
	activeCfg    *parser.AWGConfig
	isDesktopTUN bool
	activeNodeID string
	ifaceName    string
	lastErr      string
}

// New creates a new AmneziaWG engine instance.
func New() *Engine {
	return &Engine{}
}

func (e *Engine) ID() string {
	return EngineID
}

func (e *Engine) Capabilities() engine.EngineCapabilities {
	return engine.EngineCapabilities{
		SupportsTUN:        true,
		SupportsSOCKS:      false,
		SupportsTestDialer: true,
		RequiresElevation:  true,
	}
}

func (e *Engine) Validate(cfg any) error {
	_, err := resolveConfig(cfg)
	return err
}

func (e *Engine) Start(ctx context.Context, runtime engine.EngineRuntime) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parsed, err := resolveConfig(runtime.Node)
	if err != nil {
		return err
	}
	uapiConf, err := ToUAPI(parsed)
	if err != nil {
		return err
	}

	mtu := parsed.Interface.MTU
	if mtu <= 0 {
		mtu = 1280
	}
	ifName := runtime.Interface
	if ifName == "" {
		ifName = "netportal0"
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return engine.ErrAlreadyRunning
	}

	var tunDev tun.Device
	if runtime.TunFD > 0 {
		tunDev, err = openTUNFromFD(runtime.TunFD, mtu)
	} else {
		tunDev, err = tun.CreateTUN(ifName, mtu)
	}
	if err != nil {
		e.lastErr = err.Error()
		return fmt.Errorf("awg: create tun %q: %w", ifName, err)
	}

	logger := newDeviceLogger(runtime.OnLog)
	bind := conn.NewDefaultBind()
	dev := device.NewDevice(tunDev, bind, logger)
	if err := dev.IpcSet(uapiConf); err != nil {
		dev.Close()
		e.lastErr = err.Error()
		return fmt.Errorf("awg: apply uapi config: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		e.lastErr = err.Error()
		return fmt.Errorf("awg: bring device up: %w", err)
	}

	if runtime.ProtectFD != nil {
		protectBindSockets(bind, runtime.ProtectFD)
	}

	if realName, nameErr := tunDev.Name(); nameErr == nil && realName != "" {
		ifName = realName
	}

	isDesktopTUN := runtime.TunFD <= 0
	if isDesktopTUN {
		if err := configureDesktopTUN(ifName, parsed); err != nil {
			dev.Close()
			e.lastErr = err.Error()
			return fmt.Errorf("awg: configure tun %q: %w", ifName, err)
		}
	}

	e.dev = dev
	e.tunDev = tunDev
	e.activeCfg = parsed
	e.isDesktopTUN = isDesktopTUN
	e.activeNodeID = runtime.Node.ID
	e.ifaceName = ifName
	e.lastErr = ""
	e.running = true
	return nil
}

func protectBindSockets(bind conn.Bind, protectFD func(int) bool) {
	peek, ok := bind.(conn.PeekLookAtSocketFd)
	if !ok {
		return
	}
	if fd4, err := peek.PeekLookAtSocketFd4(); err == nil && fd4 >= 0 {
		_ = protectFD(fd4)
	}
	if fd6, err := peek.PeekLookAtSocketFd6(); err == nil && fd6 >= 0 {
		_ = protectFD(fd6)
	}
}

func (e *Engine) Stop(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return nil
	}
	if e.isDesktopTUN && e.ifaceName != "" {
		cleanupDesktopTUN(e.ifaceName, e.activeCfg)
	}
	if e.dev != nil {
		e.dev.Close()
		e.dev = nil
		e.tunDev = nil
	} else if e.tunDev != nil {
		_ = e.tunDev.Close()
		e.tunDev = nil
	}
	e.running = false
	e.activeCfg = nil
	e.isDesktopTUN = false
	e.activeNodeID = ""
	e.ifaceName = ""
	return nil
}

func (e *Engine) Health(_ context.Context) engine.EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := engine.EngineStatus{
		ID:            EngineID,
		State:         api.StatusDisconnected,
		ActiveNodeID:  e.activeNodeID,
		InterfaceName: e.ifaceName,
		LastError:     e.lastErr,
	}
	if !e.running || e.dev == nil {
		return status
	}

	status.State = api.StatusConnected
	if raw, err := e.dev.IpcGet(); err == nil {
		rx, tx := parseIpcTraffic(raw)
		status.RxBytes = rx
		status.TxBytes = tx
	}
	return status
}

// TestDialer creates an ephemeral in-memory netstack WireGuard/AWG tunnel without
// touching system TUN interfaces or requiring elevated privileges.
func (e *Engine) TestDialer(node api.NodeItem) (engine.Dialer, error) {
	parsed, err := resolveConfig(node)
	if err != nil {
		return nil, err
	}
	uapiConf, err := ToUAPI(parsed)
	if err != nil {
		return nil, err
	}

	localAddrs, err := extractLocalAddrs(parsed.Interface.Addresses)
	if err != nil {
		return nil, err
	}
	dnsAddrs := extractDNSAddrs(parsed.Interface.DNS)
	mtu := parsed.Interface.MTU
	if mtu <= 0 {
		mtu = 1280
	}

	tunDev, tnet, err := netstack.CreateNetTUN(localAddrs, dnsAddrs, mtu)
	if err != nil {
		return nil, fmt.Errorf("awg: create netstack tun: %w", err)
	}

	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, ""))
	if err := dev.IpcSet(uapiConf); err != nil {
		dev.Close()
		return nil, fmt.Errorf("awg: netstack uapi: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("awg: netstack up: %w", err)
	}

	return &netstackDialer{
		tnet: tnet,
		dev:  dev,
	}, nil
}

type netstackDialer struct {
	tnet *netstack.Net
	dev  *device.Device
	once sync.Once
}

func (d *netstackDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.tnet.DialContext(ctx, network, address)
}

// Close tears down the ephemeral netstack tunnel.
func (d *netstackDialer) Close() error {
	d.once.Do(func() {
		if d.dev != nil {
			d.dev.Close()
		}
	})
	return nil
}

var _ io.Closer = (*netstackDialer)(nil)

func resolveConfig(cfg any) (*parser.AWGConfig, error) {
	switch v := cfg.(type) {
	case *parser.AWGConfig:
		if v == nil {
			return nil, errors.New("awg: nil config")
		}
		return v, v.Validate()
	case parser.AWGConfig:
		return &v, v.Validate()
	case api.NodeItem:
		return parseRawOrURI(v.RawConfig)
	case *api.NodeItem:
		if v == nil {
			return nil, errors.New("awg: nil node")
		}
		return parseRawOrURI(v.RawConfig)
	case string:
		return parseRawOrURI(v)
	default:
		return nil, fmt.Errorf("awg: unsupported config type %T", cfg)
	}
}

func parseRawOrURI(raw string) (*parser.AWGConfig, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("awg: empty raw configuration")
	}
	if strings.HasPrefix(trimmed, "vpn://") {
		_, confText, err := parser.DecodeVPNURI(trimmed)
		if err != nil {
			return nil, err
		}
		trimmed = confText
	}
	return parser.ParseAWGConf(trimmed)
}

func extractLocalAddrs(cidrs []string) ([]netip.Addr, error) {
	out := make([]netip.Addr, 0, len(cidrs))
	for _, raw := range cidrs {
		if p, err := netip.ParsePrefix(raw); err == nil {
			out = append(out, p.Addr())
			continue
		}
		if a, err := netip.ParseAddr(raw); err == nil {
			out = append(out, a)
			continue
		}
	}
	if len(out) == 0 {
		return nil, errors.New("awg: no valid local interface addresses")
	}
	return out, nil
}

func extractDNSAddrs(dns []string) []netip.Addr {
	out := make([]netip.Addr, 0, len(dns))
	for _, raw := range dns {
		if a, err := netip.ParseAddr(raw); err == nil {
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		out = append(out, netip.MustParseAddr("1.1.1.1"))
	}
	return out
}

func parseIpcTraffic(raw string) (rx uint64, tx uint64) {
	for _, line := range strings.Split(raw, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "rx_bytes":
			if n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64); err == nil {
				rx += n
			}
		case "tx_bytes":
			if n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64); err == nil {
				tx += n
			}
		}
	}
	return rx, tx
}

func newDeviceLogger(onLog func(api.LogEntry)) *device.Logger {
	if onLog == nil {
		return device.NewLogger(device.LogLevelSilent, "")
	}
	return &device.Logger{
		Verbosef: func(format string, args ...any) {
			onLog(api.LogEntry{Lvl: api.LogInfo, Cat: api.LogCatCore, Msg: fmt.Sprintf(format, args...)})
		},
		Errorf: func(format string, args ...any) {
			onLog(api.LogEntry{Lvl: api.LogError, Cat: api.LogCatCore, Msg: fmt.Sprintf(format, args...)})
		},
	}
}
