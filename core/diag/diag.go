// Package diag generates safe, redacted network diagnostic reports
// stripping private keys, pre-shared keys, and bearer/proxy credentials.
package diag

import (
	"encoding/json"
	"regexp"
	"runtime"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
)

var (
	iniSecretRe = regexp.MustCompile(`(?im)^(\s*(?:PrivateKey|PresharedKey|HeaderProtectionKey|private_key|preshared_key|header_protection_key)\s*=\s*)(.+)$`)
	uriCredsRe  = regexp.MustCompile(`(?i)((?:vless|trojan|hy2|hysteria2|tuic)://)([^@\s]+)@`)
	bearerRe    = regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._\-]+`)
)

// DiagnosticReport is a sanitized snapshot safe to share in bug reports.
type DiagnosticReport struct {
	GeneratedAt string               `json:"generatedAt"`
	OS          string               `json:"os"`
	Arch        string               `json:"arch"`
	GoVersion   string               `json:"goVersion"`
	Status      api.StatusSnapshot   `json:"status"`
	NodeCount   int                  `json:"nodeCount"`
	Conflicts   []api.ConflictReport `json:"conflicts"`
	Logs        []api.LogEntry       `json:"logs"`
}

// RedactSecrets masks private keys, preshared keys, proxy user/UUIDs, and bearer tokens.
func RedactSecrets(input string) string {
	out := iniSecretRe.ReplaceAllString(input, "${1}[REDACTED]")
	out = uriCredsRe.ReplaceAllString(out, "${1}[REDACTED]@")
	out = bearerRe.ReplaceAllString(out, "${1}[REDACTED]")
	return out
}

// BuildReport constructs a redacted DiagnosticReport JSON payload.
func BuildReport(status api.StatusSnapshot, nodes []api.NodeItem, conflicts []api.ConflictReport, logs []api.LogEntry) ([]byte, error) {
	redactedLogs := make([]api.LogEntry, len(logs))
	for i, l := range logs {
		l.Msg = RedactSecrets(l.Msg)
		redactedLogs[i] = l
	}
	if conflicts == nil {
		conflicts = []api.ConflictReport{}
	}

	rep := DiagnosticReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   runtime.Version(),
		Status:      status,
		NodeCount:   len(nodes),
		Conflicts:   conflicts,
		Logs:        redactedLogs,
	}
	return json.MarshalIndent(rep, "", "  ")
}
