package smartdns

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
)

func TestHostsPatcherBOMFreeAndIdempotent(t *testing.T) {
	hostsPath := filepath.Join(t.TempDir(), "hosts")
	// Simulate existing file with UTF-8 BOM and user entry.
	initial := append([]byte{0xEF, 0xBB, 0xBF}, []byte("127.0.0.1 localhost\n")...)
	if err := os.WriteFile(hostsPath, initial, 0o644); err != nil {
		t.Fatalf("write initial hosts: %v", err)
	}

	err := ApplyHostsEntries(hostsPath, map[string]string{
		"chat.openai.com": "104.18.32.47",
		"gemini.google.com": "142.250.74.14",
	})
	if err != nil {
		t.Fatalf("ApplyHostsEntries failed: %v", err)
	}

	data, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("read hosts: %v", err)
	}
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("hosts file must not contain UTF-8 BOM")
	}
	text := string(data)
	if !strings.Contains(text, "127.0.0.1 localhost") || !strings.Contains(text, "104.18.32.47 chat.openai.com") {
		t.Fatalf("unexpected hosts content:\n%s", text)
	}

	if err := RemoveHostsEntries(hostsPath); err != nil {
		t.Fatalf("RemoveHostsEntries failed: %v", err)
	}
	after, _ := os.ReadFile(hostsPath)
	if strings.Contains(string(after), hostsBeginMarker) || !strings.Contains(string(after), "127.0.0.1 localhost") {
		t.Fatalf("hosts cleanup failed:\n%s", string(after))
	}
}

func TestSmartDNSEngineStartStop(t *testing.T) {
	eng := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eng.Start(ctx, engine.EngineRuntime{}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	st := eng.Health(ctx)
	if st.State != api.StatusConnected || st.InterfaceName == "" {
		t.Fatalf("expected connected proxy address, got %+v", st)
	}
	if err := eng.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
