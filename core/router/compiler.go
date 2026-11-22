package router

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"github.com/NetWorkClub/NetPortal/core/api"
)

// CompileSingBoxConfig compiles a RoutingProfile and selected NodeItems into a
// sing-box JSON configuration without injecting any hidden default split rules.
func CompileSingBoxConfig(prof api.RoutingProfile, nodes []api.NodeItem, fallbackChain []string) ([]byte, error) {
	if err := ValidateProfile(prof); err != nil {
		return nil, err
	}

	outbounds := []map[string]any{
		{"type": "direct", "tag": "direct"},
		{"type": "block", "tag": "block"},
	}

	for _, n := range nodes {
		if ob := nodeToSingBoxOutbound(n); ob != nil {
			outbounds = append(outbounds, ob)
		}
	}

	if len(fallbackChain) > 1 {
		outbounds = append(outbounds, map[string]any{
			"type":      "urltest",
			"tag":       "sticky-fallback",
			"outbounds": fallbackChain,
			"url":       "https://www.gstatic.com/generate_204",
			"interval":  "30s",
		})
	}

	rules := make([]map[string]any, 0, len(prof.Rules))
	for _, r := range prof.Rules {
		if !r.Enabled {
			continue
		}
		if compiled := compileRuleToSingBox(r); compiled != nil {
			rules = append(rules, compiled)
		}
	}

	doc := map[string]any{
		"inbounds": []map[string]any{
			{
				"type":                     "tun",
				"tag":                      "tun-in",
				"inet4_address":            "172.19.0.1/30",
				"auto_route":               true,
				"strict_route":             true,
				"sniff":                    true,
				"sniff_override_destination": true,
			},
		},
		"outbounds": outbounds,
		"route": map[string]any{
			"rules": rules,
			"final": normalizeTargetTag(prof.DefaultTarget),
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}

func compileRuleToSingBox(r api.RoutingRule) map[string]any {
	rule := map[string]any{
		"outbound": normalizeTargetTag(r.Target),
	}
	if r.Invert {
		rule["invert"] = true
	}
	if len(r.Apps) > 0 && r.AppMode == api.AppMatchInclude {
		rule["process_name"] = r.Apps
	}
	if len(r.AndroidApps) > 0 && r.AppMode == api.AppMatchInclude {
		rule["package_name"] = r.AndroidApps
	}
	if len(r.Domains) > 0 {
		rule["domain"] = r.Domains
	}
	if len(r.Suffixes) > 0 {
		rule["domain_suffix"] = r.Suffixes
	}
	if len(r.Keywords) > 0 {
		rule["domain_keyword"] = r.Keywords
	}
	if r.Regex != "" {
		rule["domain_regex"] = []string{r.Regex}
	}
	if len(r.CIDRs) > 0 {
		rule["ip_cidr"] = r.CIDRs
	}
	if r.Network == api.TransportTCP || r.Network == api.TransportUDP {
		rule["network"] = string(r.Network)
	}
	if len(r.Ports) > 0 {
		rule["port_range"] = r.Ports
	}
	return rule
}

func normalizeTargetTag(target string) string {
	t := strings.TrimSpace(target)
	t = strings.TrimPrefix(t, "node:")
	t = strings.TrimPrefix(t, "group:")
	if t == "" {
		return "direct"
	}
	return t
}

func nodeToSingBoxOutbound(n api.NodeItem) map[string]any {
	host, portStr, err := net.SplitHostPort(n.Endpoint)
	if err != nil {
		host = n.Endpoint
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)
	if port <= 0 {
		port = 443
	}

	switch {
	case strings.HasPrefix(strings.ToLower(n.Proto), "vless"):
		ob := map[string]any{
			"type":        "vless",
			"tag":         n.ID,
			"server":      host,
			"server_port": port,
			"uuid":        n.Params["user"],
		}
		if sni := n.Params["sni"]; sni != "" {
			ob["tls"] = map[string]any{
				"enabled":     true,
				"server_name": sni,
			}
		}
		return ob
	case strings.HasPrefix(strings.ToLower(n.Proto), "trojan"):
		return map[string]any{
			"type":        "trojan",
			"tag":         n.ID,
			"server":      host,
			"server_port": port,
			"password":    n.Params["user"],
		}
	case strings.HasPrefix(strings.ToLower(n.Proto), "hysteria2"):
		return map[string]any{
			"type":        "hysteria2",
			"tag":         n.ID,
			"server":      host,
			"server_port": port,
			"password":    n.Params["user"],
		}
	default:
		return nil
	}
}
