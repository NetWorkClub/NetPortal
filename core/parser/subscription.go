package parser

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/NetWorkClub/NetPortal/core/api"
)

// ParseSubscription parses a single proxy URI, multi-line URI list, base64-encoded
// subscription body, sing-box JSON outbounds, or AWG .conf / vpn:// URI.
func ParseSubscription(sourceID, raw string) ([]api.NodeItem, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("parser: empty subscription input")
	}

	// 1. JSON (sing-box outbounds or array of outbounds)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if nodes, ok := tryParseSingBoxJSON(sourceID, trimmed); ok && len(nodes) > 0 {
			return nodes, nil
		}
	}

	// 2. Single AWG .conf or vpn:// URI
	if strings.HasPrefix(trimmed, vpnURIPrefix) {
		name, confText, err := DecodeVPNURI(trimmed)
		if err != nil {
			return nil, err
		}
		cfg, err := ParseAWGConf(confText)
		if err != nil {
			return nil, err
		}
		return []api.NodeItem{cfg.ToNodeItem(sourceID, stableID(sourceID, trimmed), name, confText)}, nil
	}
	if strings.Contains(trimmed, "[Interface]") && strings.Contains(trimmed, "[Peer]") {
		cfg, err := ParseAWGConf(trimmed)
		if err != nil {
			return nil, err
		}
		return []api.NodeItem{cfg.ToNodeItem(sourceID, stableID(sourceID, trimmed), "", trimmed)}, nil
	}

	// 3. Base64-encoded subscription list
	text := trimmed
	if !strings.Contains(trimmed, "://") {
		if decoded, ok := tryDecodeBase64Subscription(trimmed); ok {
			text = decoded
		}
	}

	var nodes []api.NodeItem
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		node, err := ParseProxyURI(sourceID, l)
		if err == nil {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return nil, errors.New("parser: no supported nodes found in subscription")
	}
	return nodes, nil
}

// ParseProxyURI parses a single proxy share URI (vless, vmess, trojan, hy2, tuic, ss, masque).
func ParseProxyURI(sourceID, rawURI string) (api.NodeItem, error) {
	trimmed := strings.TrimSpace(rawURI)
	lower := strings.ToLower(trimmed)

	switch {
	case strings.HasPrefix(lower, "vmess://"):
		return parseVMessURI(sourceID, trimmed)
	case strings.HasPrefix(lower, "ss://"):
		return parseShadowsocksURI(sourceID, trimmed)
	case strings.HasPrefix(lower, "vless://"):
		return parseStandardProxyURI(sourceID, trimmed, "VLESS", api.ProtoGroupProxy)
	case strings.HasPrefix(lower, "trojan://"):
		return parseStandardProxyURI(sourceID, trimmed, "Trojan", api.ProtoGroupProxy)
	case strings.HasPrefix(lower, "hy2://"), strings.HasPrefix(lower, "hysteria2://"):
		return parseStandardProxyURI(sourceID, trimmed, "Hysteria2", api.ProtoGroupProxy)
	case strings.HasPrefix(lower, "tuic://"):
		return parseStandardProxyURI(sourceID, trimmed, "TUIC", api.ProtoGroupProxy)
	case strings.HasPrefix(lower, "masque://"):
		return parseStandardProxyURI(sourceID, trimmed, "MASQUE", api.ProtoGroupMASQUE)
	default:
		return api.NodeItem{}, fmt.Errorf("parser: unsupported URI scheme in %q", trimmed)
	}
}

func parseStandardProxyURI(sourceID, rawURI, proto string, group api.NodeProtoGroup) (api.NodeItem, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return api.NodeItem{}, fmt.Errorf("parse %s URI: %w", proto, err)
	}
	if u.Host == "" {
		return api.NodeItem{}, fmt.Errorf("%s URI missing host", proto)
	}

	name := strings.TrimSpace(u.Fragment)
	if unescaped, err := url.PathUnescape(name); err == nil && unescaped != "" {
		name = unescaped
	}
	if name == "" {
		name = proto + " " + u.Hostname()
	}

	params := make(map[string]string)
	q := u.Query()
	for k, vals := range q {
		if len(vals) > 0 && vals[0] != "" {
			params[k] = vals[0]
		}
	}
	if u.User != nil {
		if user := u.User.Username(); user != "" {
			params["user"] = user
		}
		if pass, ok := u.User.Password(); ok && pass != "" {
			params["password"] = pass
		}
	}

	// Refine label for VLESS + Reality
	displayProto := proto
	if proto == "VLESS" && strings.EqualFold(params["security"], "reality") {
		displayProto = "VLESS Reality"
	}

	return api.NodeItem{
		ID:         stableID(sourceID, rawURI),
		SourceID:   sourceID,
		ISO:        inferISOFromName(name),
		Name:       name,
		Endpoint:   u.Host,
		Proto:      displayProto,
		ProtoGroup: group,
		AntiBot:    api.AntiBotClean,
		MTU:        1500,
		Params:     params,
		RawConfig:  rawURI,
	}, nil
}

func parseVMessURI(sourceID, rawURI string) (api.NodeItem, error) {
	b64 := strings.TrimSpace(strings.TrimPrefix(rawURI, "vmess://"))
	decoded, err := decodeFlexBase64(b64)
	if err != nil {
		return api.NodeItem{}, fmt.Errorf("decode vmess base64: %w", err)
	}
	var payload struct {
		PS   string `json:"ps"`
		Add  string `json:"add"`
		Port any    `json:"port"`
		ID   string `json:"id"`
		Net  string `json:"net"`
		TLS  string `json:"tls"`
		SNI  string `json:"sni"`
		Host string `json:"host"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return api.NodeItem{}, fmt.Errorf("decode vmess json: %w", err)
	}
	if payload.Add == "" {
		return api.NodeItem{}, errors.New("vmess missing server address")
	}
	portStr := fmt.Sprint(payload.Port)
	endpoint := netJoinHostPort(payload.Add, portStr)
	name := strings.TrimSpace(payload.PS)
	if name == "" {
		name = "VMess " + payload.Add
	}
	params := map[string]string{
		"user": payload.ID,
		"type": payload.Net,
	}
	if payload.TLS != "" {
		params["security"] = payload.TLS
	}
	if payload.SNI != "" {
		params["sni"] = payload.SNI
	}
	if payload.Host != "" {
		params["host"] = payload.Host
	}
	if payload.Path != "" {
		params["path"] = payload.Path
	}

	return api.NodeItem{
		ID:         stableID(sourceID, rawURI),
		SourceID:   sourceID,
		ISO:        inferISOFromName(name),
		Name:       name,
		Endpoint:   endpoint,
		Proto:      "VMess",
		ProtoGroup: api.ProtoGroupProxy,
		AntiBot:    api.AntiBotClean,
		MTU:        1500,
		Params:     params,
		RawConfig:  rawURI,
	}, nil
}

func parseShadowsocksURI(sourceID, rawURI string) (api.NodeItem, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return api.NodeItem{}, fmt.Errorf("parse ss URI: %w", err)
	}
	name := strings.TrimSpace(u.Fragment)
	params := make(map[string]string)

	host := u.Host
	if u.User == nil && !strings.Contains(host, ":") {
		// Legacy SIP002 / base64(method:password@host:port)
		body := strings.TrimPrefix(rawURI, "ss://")
		if hash := strings.IndexByte(body, '#'); hash != -1 {
			name = body[hash+1:]
			body = body[:hash]
		}
		if unescaped, err := url.PathUnescape(name); err == nil && unescaped != "" {
			name = unescaped
		}
		decoded, err := decodeFlexBase64(body)
		if err != nil {
			return api.NodeItem{}, fmt.Errorf("decode ss base64: %w", err)
		}
		creds, server, ok := strings.Cut(string(decoded), "@")
		if !ok {
			return api.NodeItem{}, errors.New("invalid ss URI format")
		}
		method, pass, _ := strings.Cut(creds, ":")
		params["method"] = method
		params["password"] = pass
		host = server
	} else if u.User != nil {
		userInfo := u.User.Username()
		if decoded, err := decodeFlexBase64(userInfo); err == nil && strings.Contains(string(decoded), ":") {
			method, pass, _ := strings.Cut(string(decoded), ":")
			params["method"] = method
			params["password"] = pass
		} else {
			params["method"] = userInfo
			if pass, ok := u.User.Password(); ok {
				params["password"] = pass
			}
		}
	}

	if host == "" {
		return api.NodeItem{}, errors.New("ss URI missing host")
	}
	if name == "" {
		name = "Shadowsocks " + host
	}

	return api.NodeItem{
		ID:         stableID(sourceID, rawURI),
		SourceID:   sourceID,
		ISO:        inferISOFromName(name),
		Name:       name,
		Endpoint:   host,
		Proto:      "Shadowsocks",
		ProtoGroup: api.ProtoGroupProxy,
		AntiBot:    api.AntiBotClean,
		MTU:        1500,
		Params:     params,
		RawConfig:  rawURI,
	}, nil
}

func tryParseSingBoxJSON(sourceID, raw string) ([]api.NodeItem, bool) {
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil || len(doc.Outbounds) == 0 {
		return nil, false
	}
	var nodes []api.NodeItem
	for _, ob := range doc.Outbounds {
		typ, _ := ob["type"].(string)
		tag, _ := ob["tag"].(string)
		server, _ := ob["server"].(string)
		if typ == "" || server == "" {
			continue
		}
		port := 0
		if p, ok := ob["server_port"].(float64); ok {
			port = int(p)
		}
		endpoint := server
		if port > 0 {
			endpoint = netJoinHostPort(server, strconv.Itoa(port))
		}
		if tag == "" {
			tag = strings.ToUpper(typ) + " " + server
		}
		group := api.ProtoGroupProxy
		if strings.EqualFold(typ, "masque") {
			group = api.ProtoGroupMASQUE
		}
		rawOb, _ := json.Marshal(ob)
		nodes = append(nodes, api.NodeItem{
			ID:         stableID(sourceID, string(rawOb)),
			SourceID:   sourceID,
			ISO:        inferISOFromName(tag),
			Name:       tag,
			Endpoint:   endpoint,
			Proto:      typ,
			ProtoGroup: group,
			AntiBot:    api.AntiBotClean,
			MTU:        1500,
			Params:     map[string]string{"type": typ},
			RawConfig:  string(rawOb),
		})
	}
	return nodes, len(nodes) > 0
}

func tryDecodeBase64Subscription(raw string) (string, bool) {
	decoded, err := decodeFlexBase64(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	s := string(decoded)
	if strings.Contains(s, "://") {
		return s, true
	}
	return "", false
}

func decodeFlexBase64(s string) ([]byte, error) {
	clean := strings.TrimSpace(s)
	clean = strings.TrimRight(clean, "=")
	if b, err := base64.RawStdEncoding.DecodeString(clean); err == nil {
		return b, nil
	}
	return base64.RawURLEncoding.DecodeString(clean)
}

func stableID(sourceID, seed string) string {
	h := sha256.Sum256([]byte(sourceID + "|" + seed))
	return "node-" + hex.EncodeToString(h[:6])
}

func netJoinHostPort(host, port string) string {
	if port == "" {
		return host
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]:" + port
	}
	return host + ":" + port
}

func inferISOFromName(name string) string {
	upper := strings.ToUpper(name)
	for _, code := range []string{"DE", "NL", "FI", "SE", "US", "UK", "GB", "FR", "PL", "TR", "JP", "SG", "CH"} {
		if strings.Contains(upper, " "+code) || strings.HasPrefix(upper, code+" ") || strings.Contains(upper, "["+code+"]") || strings.Contains(upper, "-"+code) {
			if code == "UK" {
				return "GB"
			}
			return code
		}
	}
	return "--"
}
