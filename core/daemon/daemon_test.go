package daemon

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
	"github.com/NetWorkClub/NetPortal/core/engine/awg"
)

func TestDaemonRPCPingStatusAndProbe(t *testing.T) {
	reg := engine.NewRegistry()
	if err := reg.Register(awg.New()); err != nil {
		t.Fatalf("Register awg failed: %v", err)
	}

	endpoint := filepath.Join(t.TempDir(), "daemon.sock")
	if runtime.GOOS == "windows" {
		endpoint = fmt.Sprintf(`\\.\pipe\netportal-test-%d`, time.Now().UnixNano())
	}

	ln, err := ListenIPC(endpoint)
	if err != nil {
		t.Fatalf("ListenIPC failed: %v", err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := NewServer(reg, cancel)
	go func() {
		_ = srv.Serve(ctx, ln)
	}()

	// Start a local TCP listener to test real pre-connect TCP probe.
	tcpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("tcp listen failed: %v", err)
	}
	defer tcpLn.Close()
	go func() {
		for {
			c, err := tcpLn.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	client := NewClient(endpoint)
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	st, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if st.Status != api.StatusDisconnected {
		t.Fatalf("expected disconnected, got %s", st.Status)
	}

	probe, err := client.Probe(ctx, api.NodeItem{
		ID:       "local-probe",
		Endpoint: tcpLn.Addr().String(),
	})
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if probe.TcpMs < 1 || probe.AntiBot != api.AntiBotClean {
		t.Fatalf("unexpected probe result: %+v", probe)
	}

	if err := client.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}
