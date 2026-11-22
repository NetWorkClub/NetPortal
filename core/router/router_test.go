package router

import (
	"encoding/json"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestZeroHiddenDefaultsAndTwoTierEvaluation(t *testing.T) {
	empty := NewEmptyProfile("p-empty", "Чистый", "Empty", "node:vless-de")
	if len(empty.Rules) != 0 {
		t.Fatalf("expected 0 hidden rules, got %d", len(empty.Rules))
	}

	compiledEmpty, err := CompileSingBoxConfig(empty, nil, nil)
	if err != nil {
		t.Fatalf("CompileSingBoxConfig empty failed: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(compiledEmpty, &doc); err != nil {
		t.Fatalf("unmarshal compiled config: %v", err)
	}
	route := doc["route"].(map[string]any)
	if len(route["rules"].([]any)) != 0 || route["final"] != "vless-de" {
		t.Fatalf("unexpected compiled empty route: %+v", route)
	}

	prof := NewEmptyProfile("p-custom", "Рабочий", "Work", "direct")
	prof.Rules = []api.RoutingRule{
		{
			ID:          "r-app",
			Name:        "Discord App",
			Tier:        api.RuleTierApp,
			Enabled:     true,
			AppMode:     api.AppMatchInclude,
			Apps:        []string{"discord.exe"},
			AndroidApps: []string{"com.discord"},
			Target:      "node:awg-31",
		},
		{
			ID:       "r-pool",
			Name:     "OpenAI Pool",
			Tier:     api.RuleTierNetwork,
			Enabled:  true,
			Suffixes: []string{".openai.com"},
			IPPools:  []string{"198.51.100.10-198.51.100.50"},
			Target:   "node:vless-de",
		},
	}

	// Desktop process match
	tgt, matched, _ := Evaluate(prof, Flow{Process: "discord.exe", Domain: "api.openai.com", DstIP: "198.51.100.20"})
	if tgt != "node:awg-31" || matched == nil || matched.ID != "r-app" {
		t.Fatalf("expected app-tier precedence node:awg-31, got %s", tgt)
	}

	// Android package match
	tgt, _, _ = Evaluate(prof, Flow{AndroidPackage: "com.discord"})
	if tgt != "node:awg-31" {
		t.Fatalf("expected android package match node:awg-31, got %s", tgt)
	}

	// Network-tier IP pool + suffix match
	tgt, matched, _ = Evaluate(prof, Flow{Process: "chrome.exe", Domain: "chat.openai.com", DstIP: "198.51.100.25"})
	if tgt != "node:vless-de" || matched == nil || matched.ID != "r-pool" {
		t.Fatalf("expected network-tier match node:vless-de, got %s", tgt)
	}

	// Fallback to explicit DefaultTarget
	tgt, matched, _ = Evaluate(prof, Flow{Process: "curl", Domain: "example.org", DstIP: "93.184.216.34"})
	if tgt != "direct" || matched != nil {
		t.Fatalf("expected default target direct, got %s", tgt)
	}
}

func TestStickyChainCrossProtocolFailover(t *testing.T) {
	chain, err := NewStickyChain([]string{"vless-de", "vless-nl", "psiphon-de", "warp-awg31"}, 2, true)
	if err != nil {
		t.Fatalf("NewStickyChain failed: %v", err)
	}
	if chain.Active() != "vless-de" {
		t.Fatalf("expected primary vless-de, got %s", chain.Active())
	}

	// Single failure must NOT hop IP (threshold = 2)
	if active := chain.ReportHealth(false, false); active != "vless-de" {
		t.Fatalf("expected sticky hold on 1st failure, got %s", active)
	}

	// Second consecutive failure triggers failover to vless-nl
	if active := chain.ReportHealth(false, false); active != "vless-nl" {
		t.Fatalf("expected failover to vless-nl, got %s", active)
	}

	// Primary recovers -> stickBackOnRecovery switches back cleanly
	if active := chain.ReportHealth(true, true); active != "vless-de" {
		t.Fatalf("expected stick-back to vless-de, got %s", active)
	}
}
