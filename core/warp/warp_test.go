package warp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestWarpRegisterBuildAndScan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "dev-123456",
			"token": "tok-abc",
			"config": map[string]any{
				"peers": []map[string]any{
					{
						"public_key": "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
						"endpoint":   map[string]string{"v4": "162.159.192.1:2408"},
					},
				},
				"interface": map[string]any{
					"addresses": map[string]string{
						"v4": "172.16.0.2",
						"v6": "2606:4700:110::2",
					},
				},
			},
		})
	}))
	defer srv.Close()

	acc, err := Register(context.Background(), srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if acc.ID != "dev-123456" || acc.PrivateKey == "" || acc.PeerPubKey == "" {
		t.Fatalf("unexpected account: %+v", acc)
	}

	awgNode, err := BuildProfile(acc, PresetAWG31Stealth, "")
	if err != nil {
		t.Fatalf("BuildProfile AWG 3.1 failed: %v", err)
	}
	if awgNode.Proto != "AWG 3.1" || awgNode.Params["RandomTrailers"] != "on" {
		t.Fatalf("unexpected AWG 3.1 node: %+v", awgNode)
	}

	masqueNode, err := BuildProfile(acc, PresetMASQUE, "")
	if err != nil {
		t.Fatalf("BuildProfile MASQUE failed: %v", err)
	}
	if masqueNode.ProtoGroup != api.ProtoGroupMASQUE {
		t.Fatalf("expected MASQUE protoGroup, got %s", masqueNode.ProtoGroup)
	}

	scanned := ScanEndpoints(context.Background(), []api.WarpEndpoint{
		{Addr: "127.0.0.1:2408", Region: "Local"},
	}, 500*time.Millisecond)
	if len(scanned) != 1 || scanned[0].RTT < 1 {
		t.Fatalf("unexpected scan result: %+v", scanned)
	}
}
