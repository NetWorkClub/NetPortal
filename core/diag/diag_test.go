package diag

import (
	"strings"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestRedactSecretsAndBuildReport(t *testing.T) {
	sample := `PrivateKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
HeaderProtectionKey = CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC=
vless://11111111-2222-3333-4444-555555555555@de.example.com:443#DE
Authorization: Bearer super_secret_token_123`

	redacted := RedactSecrets(sample)
	for _, secret := range []string{
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC=",
		"11111111-2222-3333-4444-555555555555",
		"super_secret_token_123",
	} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("secret %q leaked in redacted output:\n%s", secret, redacted)
		}
	}

	reportBytes, err := BuildReport(
		api.StatusSnapshot{Status: api.StatusConnected, EngineID: "awg"},
		[]api.NodeItem{{ID: "n-1"}},
		nil,
		[]api.LogEntry{{Time: "12:00:00", Lvl: api.LogInfo, Cat: api.LogCatCore, Msg: sample}},
	)
	if err != nil {
		t.Fatalf("BuildReport failed: %v", err)
	}
	if strings.Contains(string(reportBytes), "super_secret_token_123") {
		t.Fatalf("secret leaked in DiagnosticReport")
	}
}
