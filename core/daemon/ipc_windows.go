//go:build windows

package daemon

import (
	"context"
	"net"

	"github.com/Microsoft/go-winio"
)

// DefaultEndpoint returns the default Windows Named Pipe path for netportal-daemon.
func DefaultEndpoint() string {
	return `\\.\pipe\netportal-daemon`
}

// ListenIPC opens a Named Pipe restricted to SYSTEM, Administrators, and Creator Owner.
func ListenIPC(endpoint string) (net.Listener, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint()
	}
	cfg := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;CO)",
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}
	return winio.ListenPipe(endpoint, cfg)
}

// DialIPC connects to the netportal-daemon Named Pipe.
func DialIPC(ctx context.Context, endpoint string) (net.Conn, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint()
	}
	return winio.DialPipeContext(ctx, endpoint)
}
