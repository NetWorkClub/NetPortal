package parser

import (
	"strings"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

const (
	testPrivKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	testPubKey  = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
	testHPK     = "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC="
)

func TestParseAndSerializeAWG20(t *testing.T) {
	conf := `[Interface]
PrivateKey = ` + testPrivKey + `
Address = 10.8.1.2/32, fd00::2/128
DNS = 1.1.1.1, 1.0.0.1
MTU = 1280
Jc = 4
Jmin = 40
Jmax = 70
S1 = 15
S2 = 25
H1 = 1001
H2 = 1002
H3 = 1003
H4 = 1004
I1 = <b 0xc700000001><r 16>

[Peer]
PublicKey = ` + testPubKey + `
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 203.0.113.10:51820
PersistentKeepalive = 25
`

	cfg, err := ParseAWGConf(conf)
	if err != nil {
		t.Fatalf("ParseAWGConf failed: %v", err)
	}
	if ver := cfg.DetectVersion(); ver != VersionAWG20 {
		t.Fatalf("expected %s, got %s", VersionAWG20, ver)
	}
	if cfg.Interface.I1 == "" || *cfg.Interface.Jc != 4 {
		t.Fatalf("unexpected AWG 2.0 fields: %+v", cfg.Interface)
	}

	ser := cfg.Serialize()
	reparsed, err := ParseAWGConf(ser)
	if err != nil {
		t.Fatalf("reparse failed: %v", err)
	}
	if reparsed.Interface.I1 != cfg.Interface.I1 || reparsed.Peers[0].Endpoint != "203.0.113.10:51820" {
		t.Fatalf("round-trip mismatch")
	}
}

func TestParseAWG30WithRanges(t *testing.T) {
	conf := `[Interface]
PrivateKey = ` + testPrivKey + `
Address = 10.10.0.2/24
S3 = 16
S4 = 20
H1 = 1000-2000
H2 = 3000-4000
H3 = 5000-6000
H4 = 7000-8000
HeaderProtectionKey = ` + testHPK + `
ContentPaddingAddition = 64
RekeyAfterTime = 120
MaxHandshakeAttempts = 5

[Peer]
PublicKey = ` + testPubKey + `
AllowedIPs = 0.0.0.0/0
Endpoint = 198.51.100.2:443
PersistentKeepalive = 15-30
`

	cfg, err := ParseAWGConf(conf)
	if err != nil {
		t.Fatalf("ParseAWGConf AWG 3.0 failed: %v", err)
	}
	if ver := cfg.DetectVersion(); ver != VersionAWG30 {
		t.Fatalf("expected %s, got %s", VersionAWG30, ver)
	}
	if cfg.Interface.H1.String() != "1000-2000" {
		t.Fatalf("expected H1=1000-2000, got %s", cfg.Interface.H1.String())
	}
	if cfg.Peers[0].PersistentKeepalive.String() != "15-30" {
		t.Fatalf("expected PersistentKeepalive=15-30, got %s", cfg.Peers[0].PersistentKeepalive.String())
	}
}

func TestParseAWG31AndNodeConversion(t *testing.T) {
	conf := `[Interface]
PrivateKey = ` + testPrivKey + `
Address = 10.10.0.5/32
RandomTrailers = on
DisableCookies = 1

[Peer]
PublicKey = ` + testPubKey + `
AllowedIPs = 0.0.0.0/0
Endpoint = 198.51.100.99:51820
`

	cfg, err := ParseAWGConf(conf)
	if err != nil {
		t.Fatalf("ParseAWGConf AWG 3.1 failed: %v", err)
	}
	if ver := cfg.DetectVersion(); ver != VersionAWG31 {
		t.Fatalf("expected %s, got %s", VersionAWG31, ver)
	}

	node := cfg.ToNodeItem("src-1", "node-1", "Berlin AWG 3.1", conf)
	if node.Proto != "AWG 3.1" || node.ProtoGroup != api.ProtoGroupAWG {
		t.Fatalf("unexpected NodeItem: %+v", node)
	}
	if node.Params["RandomTrailers"] != "on" || node.Params["DisableCookies"] != "on" {
		t.Fatalf("expected AWG 3.1 params in NodeItem, got %+v", node.Params)
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		conf string
	}{
		{
			name: "bad private key",
			conf: "[Interface]\nPrivateKey = short\nAddress = 10.0.0.1/32\n[Peer]\nPublicKey = " + testPubKey,
		},
		{
			name: "Jmin greater than Jmax",
			conf: "[Interface]\nPrivateKey = " + testPrivKey + "\nAddress = 10.0.0.1/32\nJmin = 100\nJmax = 50\n[Peer]\nPublicKey = " + testPubKey,
		},
		{
			name: "inverted H1 range",
			conf: "[Interface]\nPrivateKey = " + testPrivKey + "\nAddress = 10.0.0.1/32\nH1 = 500-100\n[Peer]\nPublicKey = " + testPubKey,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseAWGConf(tc.conf); err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestVPNURIRoundTrip(t *testing.T) {
	conf := "[Interface]\nPrivateKey = " + testPrivKey + "\nAddress = 10.0.0.2/32\nRandomTrailers = on\n\n[Peer]\nPublicKey = " + testPubKey + "\nEndpoint = 198.51.100.7:51820\nAllowedIPs = 0.0.0.0/0\n"

	uri, err := EncodeVPNURI("Amnezia Berlin", conf)
	if err != nil {
		t.Fatalf("EncodeVPNURI failed: %v", err)
	}
	if !strings.HasPrefix(uri, "vpn://") {
		t.Fatalf("expected vpn:// prefix, got %q", uri)
	}

	name, decodedConf, err := DecodeVPNURI(uri)
	if err != nil {
		t.Fatalf("DecodeVPNURI failed: %v", err)
	}
	if name != "Amnezia Berlin" {
		t.Fatalf("expected name 'Amnezia Berlin', got %q", name)
	}
	det, err := DetectImport(uri)
	if err != nil {
		t.Fatalf("DetectImport failed: %v", err)
	}
	if det.Proto != "AWG 3.1" {
		t.Fatalf("expected AWG 3.1 detection, got %s", det.Proto)
	}
	if _, err := ParseAWGConf(decodedConf); err != nil {
		t.Fatalf("decoded conf invalid: %v", err)
	}
}
