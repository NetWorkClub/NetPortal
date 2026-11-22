package awg

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

const sampleAWG31Conf = `[Interface]
PrivateKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
Address = 10.8.1.2/32
DNS = 1.1.1.1
MTU = 1280
Jc = 4
Jmin = 40
Jmax = 70
S1 = 15
S2 = 25
S3 = 16
S4 = 20
H1 = 1000-2000
H2 = 3000-4000
H3 = 5000-6000
H4 = 7000-8000
HeaderProtectionKey = CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC=
RandomTrailers = on
DisableCookies = on

[Peer]
PublicKey = BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=
AllowedIPs = 0.0.0.0/0
Endpoint = 203.0.113.10:51820
PersistentKeepalive = 15-25
`

func TestToUAPIAndValidate(t *testing.T) {
	eng := New()
	if err := eng.Validate(sampleAWG31Conf); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	cfg, err := resolveConfig(sampleAWG31Conf)
	if err != nil {
		t.Fatalf("resolveConfig failed: %v", err)
	}
	uapi, err := ToUAPI(cfg)
	if err != nil {
		t.Fatalf("ToUAPI failed: %v", err)
	}

	for _, expected := range []string{
		"private_key=0000000000000000000000000000000000000000000000000000000000000000",
		"h1=1000-2000",
		"random_trailers=true",
		"disable_cookies=true",
		"persistent_keepalive_interval=15-25",
	} {
		if !strings.Contains(uapi, expected) {
			t.Fatalf("expected %q in UAPI output:\n%s", expected, uapi)
		}
	}
}

func TestEphemeralNetstackDialer(t *testing.T) {
	eng := New()
	node := api.NodeItem{
		ID:        "test-awg-31",
		Proto:     "AWG 3.1",
		RawConfig: sampleAWG31Conf,
	}

	dialer, err := eng.TestDialer(node)
	if err != nil {
		t.Fatalf("TestDialer failed: %v", err)
	}
	if closer, ok := dialer.(io.Closer); ok {
		defer closer.Close()
	}

	st := eng.Health(context.Background())
	if st.ID != EngineID || st.State != api.StatusDisconnected {
		t.Fatalf("unexpected health on TestDialer-only engine: %+v", st)
	}
}
