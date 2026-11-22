package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestAppImportProbeAndWebAuth(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	app, err := NewApp(statePath, "")
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	conf := `[Interface]
PrivateKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
Address = 10.8.1.2/32
RandomTrailers = on

[Peer]
PublicKey = BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=
AllowedIPs = 0.0.0.0/0
Endpoint = 203.0.113.10:51820
`
	imp, err := app.ImportConfig(api.ImportRequest{
		Name:    "Test AWG 3.1",
		Content: conf,
	})
	if err != nil {
		t.Fatalf("ImportConfig failed: %v", err)
	}
	if len(imp.Nodes) != 1 || imp.Nodes[0].Proto != "AWG 3.1" {
		t.Fatalf("unexpected import result: %+v", imp)
	}

	// Verify local TCP probe against an ephemeral listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	probe, err := app.ProbeNode(api.NodeItem{
		ID:       imp.Nodes[0].ID,
		Endpoint: ln.Addr().String(),
	}, "tcp")
	if err != nil {
		t.Fatalf("ProbeNode failed: %v", err)
	}
	if probe.TcpMs < 1 || probe.AntiBot != api.AntiBotClean {
		t.Fatalf("unexpected probe: %+v", probe)
	}

	// Verify --web handler enforces Bearer token.
	handler := newWebHandler(app, assets, "secret-session-token")

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	unauthRec := httptest.NewRecorder()
	handler.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", unauthRec.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	authReq.Header.Set("Authorization", "Bearer secret-session-token")
	authRec := httptest.NewRecorder()
	handler.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", authRec.Code)
	}

	var nodes []api.NodeItem
	if err := json.Unmarshal(authRec.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode /api/nodes failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node from /api/nodes, got %d", len(nodes))
	}
}
