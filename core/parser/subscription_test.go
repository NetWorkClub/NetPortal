package parser

import (
	"encoding/base64"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestParseProxyURIs(t *testing.T) {
	vless := "vless://11111111-1111-1111-1111-111111111111@de.example.com:443?security=reality&sni=google.com&fp=chrome&pbk=abc&sid=01#DE%20Frankfurt"
	n1, err := ParseProxyURI("sub-1", vless)
	if err != nil {
		t.Fatalf("vless parse failed: %v", err)
	}
	if n1.Proto != "VLESS Reality" || n1.ProtoGroup != api.ProtoGroupProxy || n1.ISO != "DE" || n1.Name != "DE Frankfurt" {
		t.Fatalf("unexpected vless node: %+v", n1)
	}
	if n1.Params["sni"] != "google.com" {
		t.Fatalf("expected sni=google.com, got %q", n1.Params["sni"])
	}

	hy2 := "hy2://secret@nl.example.com:8443?sni=nl.example.com#NL%20Amsterdam"
	n2, err := ParseProxyURI("sub-1", hy2)
	if err != nil {
		t.Fatalf("hy2 parse failed: %v", err)
	}
	if n2.Proto != "Hysteria2" || n2.ISO != "NL" {
		t.Fatalf("unexpected hy2 node: %+v", n2)
	}

	masque := "masque://cf.example.com:443?sni=cloudflare.com#MASQUE%20US"
	n3, err := ParseProxyURI("sub-1", masque)
	if err != nil {
		t.Fatalf("masque parse failed: %v", err)
	}
	if n3.ProtoGroup != api.ProtoGroupMASQUE {
		t.Fatalf("expected MASQUE protoGroup, got %s", n3.ProtoGroup)
	}

	vmessJSON := `{"ps":"FI Helsinki","add":"fi.example.com","port":443,"id":"22222222-2222-2222-2222-222222222222","net":"ws","tls":"tls"}`
	vmess := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))
	n4, err := ParseProxyURI("sub-1", vmess)
	if err != nil {
		t.Fatalf("vmess parse failed: %v", err)
	}
	if n4.Proto != "VMess" || n4.ISO != "FI" || n4.Endpoint != "fi.example.com:443" {
		t.Fatalf("unexpected vmess node: %+v", n4)
	}

	ss := "ss://" + base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:pass123@se.example.com:8388")) + "#SE%20Stockholm"
	n5, err := ParseProxyURI("sub-1", ss)
	if err != nil {
		t.Fatalf("ss parse failed: %v", err)
	}
	if n5.Proto != "Shadowsocks" || n5.Params["method"] != "aes-256-gcm" {
		t.Fatalf("unexpected ss node: %+v", n5)
	}
}

func TestParseBase64SubscriptionAndSingBoxJSON(t *testing.T) {
	plainList := "vless://11111111-1111-1111-1111-111111111111@de.example.com:443?security=tls#DE-1\ntrojan://pass@nl.example.com:443#NL-1\n"
	b64 := base64.StdEncoding.EncodeToString([]byte(plainList))

	nodes, err := ParseSubscription("sub-b64", b64)
	if err != nil {
		t.Fatalf("ParseSubscription b64 failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	singbox := `{"outbounds":[{"type":"vless","tag":"DE SingBox","server":"1.2.3.4","server_port":443},{"type":"masque","tag":"MASQUE Edge","server":"5.6.7.8","server_port":443}]}`
	sbNodes, err := ParseSubscription("sub-sb", singbox)
	if err != nil {
		t.Fatalf("ParseSubscription sing-box failed: %v", err)
	}
	if len(sbNodes) != 2 || sbNodes[1].ProtoGroup != api.ProtoGroupMASQUE {
		t.Fatalf("unexpected sing-box nodes: %+v", sbNodes)
	}
}
