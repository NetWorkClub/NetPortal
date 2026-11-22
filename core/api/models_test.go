package api

import (
	"encoding/json"
	"testing"
)

func TestNodeItemJSONRoundTrip(t *testing.T) {
	tcp := 42
	urlMs := 118
	orig := NodeItem{
		ID:         "node-awg-1",
		SourceID:   "src-local",
		ISO:        "DE",
		Name:       "AWG 3.1 Frankfurt",
		Endpoint:   "203.0.113.10:51820",
		Proto:      "AWG 3.1",
		ProtoGroup: ProtoGroupAWG,
		TcpMs:      &tcp,
		UrlMs:      &urlMs,
		AntiBot:    AntiBotClean,
		Probing:    false,
		MTU:        1280,
		Params: map[string]string{
			"Jc":             "4",
			"RandomTrailers": "on",
		},
	}

	raw, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded NodeItem
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ID != orig.ID || decoded.ProtoGroup != ProtoGroupAWG {
		t.Fatalf("unexpected decoded node: %+v", decoded)
	}
	if decoded.TcpMs == nil || *decoded.TcpMs != 42 {
		t.Fatalf("expected TcpMs=42, got %v", decoded.TcpMs)
	}
	if decoded.Params["RandomTrailers"] != "on" {
		t.Fatalf("expected RandomTrailers=on, got %q", decoded.Params["RandomTrailers"])
	}
}

func TestRoutingProfileEmptyRulesDefault(t *testing.T) {
	profile := RoutingProfile{
		ID:            "prof-clean",
		NameRu:        "Чистый профиль",
		NameEn:        "Clean Profile",
		DefaultTarget: "direct",
		Rules:         []RoutingRule{},
	}

	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded RoutingProfile
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.Rules) != 0 {
		t.Fatalf("expected zero hidden default rules, got %d", len(decoded.Rules))
	}
}
