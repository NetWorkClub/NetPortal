//go:build !windows

package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// DefaultEndpoint returns the default Unix Domain Socket path for netportal-daemon.
func DefaultEndpoint() string {
	if os.Geteuid() == 0 {
		return "/run/netportal/daemon.sock"
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "netportal-daemon.sock")
	}
	return fmt.Sprintf("/tmp/netportal-daemon-%d.sock", os.Getuid())
}

// ListenIPC creates a Unix Domain Socket with 0600 permissions.
func ListenIPC(endpoint string) (net.Listener, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint()
	}
	if err := os.MkdirAll(filepath.Dir(endpoint), 0o755); err != nil {
		return nil, err
	}
	_ = os.Remove(endpoint)
	ln, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(endpoint, 0o600); err != nil {
		_ = ln.Close()
		return nil, err
	}
	return ln, nil
}

// DialIPC connects to the netportal-daemon Unix Domain Socket.
func DialIPC(ctx context.Context, endpoint string) (net.Conn, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint()
	}
	var d net.Dialer
	return d.DialContext(ctx, "unix", endpoint)
}
