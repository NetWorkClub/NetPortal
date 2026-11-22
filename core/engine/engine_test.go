package engine

import (
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestDetectConflicts(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8080")
	reports := DetectConflicts(api.ServiceModeVPN, true)
	if len(reports) < 2 {
		t.Fatalf("expected at least 2 conflict reports (hosts + HTTP_PROXY), got %d", len(reports))
	}
}
