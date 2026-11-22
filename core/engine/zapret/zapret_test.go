package zapret

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestNFTRulesIdempotentAndBOMFree(t *testing.T) {
	rules := GenerateNFTRules(200, []int{80, 443}, []int{443})
	if !strings.HasPrefix(rules, "add table inet netportal\nflush table inet netportal\n") {
		t.Fatalf("expected idempotent table add/flush header, got:\n%s", rules)
	}

	nftPath := filepath.Join(t.TempDir(), "rules.nft")
	if err := WriteNFTRulesBOMFree(nftPath, rules); err != nil {
		t.Fatalf("WriteNFTRulesBOMFree failed: %v", err)
	}
	raw, err := os.ReadFile(nftPath)
	if err != nil {
		t.Fatalf("read rules.nft: %v", err)
	}
	if bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("rules.nft must not contain UTF-8 BOM")
	}
}

func TestBuildStrategyArgs(t *testing.T) {
	args := BuildStrategyArgs(api.ZapretStrategy{
		ID:      "alt1",
		Name:    "Alt QUIC",
		YT:      "--filter-tcp=443 --dpi-desync=fake",
		QUIC:    "--filter-udp=443 --dpi-desync=fake",
		Discord: "--filter-udp=50000-50100 --dpi-desync=fake",
	})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--filter-udp=443") || !strings.Contains(joined, "--new") {
		t.Fatalf("unexpected strategy args: %s", joined)
	}
}
