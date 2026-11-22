package mobile

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

type stubBridge struct{}

func (stubBridge) ProtectFD(fd int) bool { return true }
func (stubBridge) OnEvent(string)        {}

func TestMobileFacadeImportAndProbe(t *testing.T) {
	if err := Init(t.TempDir(), stubBridge{}); err != nil {
		t.Fatalf("Init failed: %v", err)
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
	respJSON, err := ImportConfig("Android AWG", conf)
	if err != nil {
		t.Fatalf("ImportConfig failed: %v", err)
	}
	var resp api.ImportResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal import response: %v", err)
	}
	if len(resp.Nodes) != 1 || resp.Nodes[0].Proto != "AWG 3.1" {
		t.Fatalf("unexpected imported node: %+v", resp)
	}

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

	nodeBytes, _ := json.Marshal(api.NodeItem{
		ID:       resp.Nodes[0].ID,
		Endpoint: ln.Addr().String(),
	})
	probeJSON, err := ProbeNodeJSON(string(nodeBytes), "tcp")
	if err != nil {
		t.Fatalf("ProbeNodeJSON failed: %v", err)
	}
	var outcome api.ProbeOutcome
	if err := json.Unmarshal([]byte(probeJSON), &outcome); err != nil {
		t.Fatalf("unmarshal probe outcome: %v", err)
	}
	if outcome.TcpMs < 1 || outcome.AntiBot != api.AntiBotClean {
		t.Fatalf("unexpected probe outcome: %+v", outcome)
	}
}
