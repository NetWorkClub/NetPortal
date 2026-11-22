// Package parser implements config and subscription parsers/serializers
// for WireGuard, AmneziaWG 2.0/3.0/3.1, and Amnezia vpn:// URIs.
package parser

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/NetWorkClub/NetPortal/core/api"
)

// UintRange represents either a single uint64 value (Min == Max) or an inclusive min-max range.
type UintRange struct {
	Min uint64
	Max uint64
	Set bool
}

// ParseUintRange parses "N" or "min-max".
func ParseUintRange(raw string) (UintRange, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return UintRange{}, errors.New("empty range")
	}
	if idx := strings.IndexByte(s, '-'); idx != -1 {
		left := strings.TrimSpace(s[:idx])
		right := strings.TrimSpace(s[idx+1:])
		if left == "" || right == "" {
			return UintRange{}, fmt.Errorf("invalid range %q", raw)
		}
		minVal, err := strconv.ParseUint(left, 10, 64)
		if err != nil {
			return UintRange{}, fmt.Errorf("invalid range min %q: %w", raw, err)
		}
		maxVal, err := strconv.ParseUint(right, 10, 64)
		if err != nil {
			return UintRange{}, fmt.Errorf("invalid range max %q: %w", raw, err)
		}
		if minVal > maxVal {
			return UintRange{}, fmt.Errorf("range min %d exceeds max %d", minVal, maxVal)
		}
		return UintRange{Min: minVal, Max: maxVal, Set: true}, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return UintRange{}, fmt.Errorf("invalid uint %q: %w", raw, err)
	}
	return UintRange{Min: v, Max: v, Set: true}, nil
}

// String formats the range back to WireGuard/AWG config syntax.
func (r UintRange) String() string {
	if !r.Set {
		return ""
	}
	if r.Min == r.Max {
		return strconv.FormatUint(r.Min, 10)
	}
	return fmt.Sprintf("%d-%d", r.Min, r.Max)
}

// IsRange reports whether Min != Max.
func (r UintRange) IsRange() bool {
	return r.Set && r.Min != r.Max
}

// AWGVersion identifies the detected WireGuard/AmneziaWG specification level.
type AWGVersion string

const (
	VersionWireGuard AWGVersion = "WireGuard"
	VersionAWG20     AWGVersion = "AWG 2.0"
	VersionAWG30     AWGVersion = "AWG 3.0"
	VersionAWG31     AWGVersion = "AWG 3.1"
)

// AWGInterface represents the [Interface] section of a WireGuard / AmneziaWG config.
type AWGInterface struct {
	PrivateKey string
	Addresses  []string
	DNS        []string
	MTU        int
	ListenPort int
	Table      string
	FwMark     string
	PreUp      []string
	PostUp     []string
	PreDown    []string
	PostDown   []string

	// AWG 2.0 obfuscation & junk packets
	Jc   *int
	Jmin *int
	Jmax *int
	S1   *int
	S2   *int
	S3   *int
	S4   *int
	H1   UintRange
	H2   UintRange
	H3   UintRange
	H4   UintRange
	I1   string
	I2   string
	I3   string
	I4   string
	I5   string

	// AWG 3.0 header protection & handshake timers
	HeaderProtectionKey    string
	ContentPaddingAddition *int
	RekeyAfterTime         *int
	RekeyTimeout           *int
	RejectAfterTime        *int
	KeepaliveTimeout       *int
	MaxHandshakeAttempts   *int

	// AWG 3.1 trailer & cookie controls
	RandomTrailers *bool
	DisableCookies *bool

	// Extra preserves unknown/vendor keys without dropping them on round-trip.
	Extra map[string]string
}

// AWGPeer represents a [Peer] section of a WireGuard / AmneziaWG config.
type AWGPeer struct {
	PublicKey           string
	PresharedKey        string
	AllowedIPs          []string
	Endpoint            string
	PersistentKeepalive UintRange
	Extra               map[string]string
}

// AWGConfig represents a complete parsed WireGuard / AmneziaWG configuration.
type AWGConfig struct {
	Interface AWGInterface
	Peers     []AWGPeer
}

// ParseAWGConf parses a WireGuard / AmneziaWG 2.0 / 3.0 / 3.1 INI configuration.
func ParseAWGConf(raw string) (*AWGConfig, error) {
	cfg := &AWGConfig{
		Interface: AWGInterface{
			Extra: make(map[string]string),
		},
	}

	const (
		secNone = iota
		secInterface
		secPeer
	)
	section := secNone
	currentPeerIdx := -1
	hasInterface := false

	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			secName := strings.ToLower(strings.TrimSpace(trimmed[1 : len(trimmed)-1]))
			switch secName {
			case "interface":
				section = secInterface
				hasInterface = true
			case "peer":
				section = secPeer
				cfg.Peers = append(cfg.Peers, AWGPeer{Extra: make(map[string]string)})
				currentPeerIdx = len(cfg.Peers) - 1
			default:
				return nil, fmt.Errorf("line %d: unknown section [%s]", lineNo+1, secName)
			}
			continue
		}

		eq := strings.IndexByte(trimmed, '=')
		if eq == -1 {
			return nil, fmt.Errorf("line %d: expected key = value", lineNo+1)
		}
		key := strings.TrimSpace(trimmed[:eq])
		val := strings.TrimSpace(trimmed[eq+1:])

		switch section {
		case secInterface:
			if err := applyInterfaceKV(&cfg.Interface, key, val); err != nil {
				return nil, fmt.Errorf("line %d (%s): %w", lineNo+1, key, err)
			}
		case secPeer:
			if err := applyPeerKV(&cfg.Peers[currentPeerIdx], key, val); err != nil {
				return nil, fmt.Errorf("line %d (%s): %w", lineNo+1, key, err)
			}
		default:
			return nil, fmt.Errorf("line %d: key %q outside of [Interface] or [Peer]", lineNo+1, key)
		}
	}

	if !hasInterface {
		return nil, errors.New("missing [Interface] section")
	}
	if len(cfg.Peers) == 0 {
		return nil, errors.New("missing [Peer] section")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyInterfaceKV(iface *AWGInterface, key, val string) error {
	switch strings.ToLower(key) {
	case "privatekey":
		iface.PrivateKey = val
	case "address":
		iface.Addresses = append(iface.Addresses, splitCSV(val)...)
	case "dns":
		iface.DNS = append(iface.DNS, splitCSV(val)...)
	case "mtu":
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		iface.MTU = v
	case "listenport":
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		iface.ListenPort = v
	case "table":
		iface.Table = val
	case "fwmark":
		iface.FwMark = val
	case "preup":
		iface.PreUp = append(iface.PreUp, val)
	case "postup":
		iface.PostUp = append(iface.PostUp, val)
	case "predown":
		iface.PreDown = append(iface.PreDown, val)
	case "postdown":
		iface.PostDown = append(iface.PostDown, val)
	case "jc":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.Jc = p
	case "jmin":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.Jmin = p
	case "jmax":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.Jmax = p
	case "s1":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.S1 = p
	case "s2":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.S2 = p
	case "s3":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.S3 = p
	case "s4":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.S4 = p
	case "h1":
		r, err := ParseUintRange(val)
		if err != nil {
			return err
		}
		iface.H1 = r
	case "h2":
		r, err := ParseUintRange(val)
		if err != nil {
			return err
		}
		iface.H2 = r
	case "h3":
		r, err := ParseUintRange(val)
		if err != nil {
			return err
		}
		iface.H3 = r
	case "h4":
		r, err := ParseUintRange(val)
		if err != nil {
			return err
		}
		iface.H4 = r
	case "i1":
		iface.I1 = val
	case "i2":
		iface.I2 = val
	case "i3":
		iface.I3 = val
	case "i4":
		iface.I4 = val
	case "i5":
		iface.I5 = val
	case "headerprotectionkey":
		iface.HeaderProtectionKey = val
	case "contentpaddingaddition":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.ContentPaddingAddition = p
	case "rekeyaftertime":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.RekeyAfterTime = p
	case "rekeytimeout":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.RekeyTimeout = p
	case "rejectaftertime":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.RejectAfterTime = p
	case "keepalivetimeout":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.KeepaliveTimeout = p
	case "maxhandshakeattempts":
		p, err := parseIntPtr(val)
		if err != nil {
			return err
		}
		iface.MaxHandshakeAttempts = p
	case "randomtrailers":
		b, err := parseFlexBool(val)
		if err != nil {
			return err
		}
		iface.RandomTrailers = &b
	case "disablecookies":
		b, err := parseFlexBool(val)
		if err != nil {
			return err
		}
		iface.DisableCookies = &b
	default:
		iface.Extra[key] = val
	}
	return nil
}

func applyPeerKV(peer *AWGPeer, key, val string) error {
	switch strings.ToLower(key) {
	case "publickey":
		peer.PublicKey = val
	case "presharedkey":
		peer.PresharedKey = val
	case "allowedips":
		peer.AllowedIPs = append(peer.AllowedIPs, splitCSV(val)...)
	case "endpoint":
		peer.Endpoint = val
	case "persistentkeepalive":
		r, err := ParseUintRange(val)
		if err != nil {
			return err
		}
		peer.PersistentKeepalive = r
	default:
		peer.Extra[key] = val
	}
	return nil
}

// Validate checks semantic correctness of keys, CIDRs, ranges, and AWG parameters.
func (c *AWGConfig) Validate() error {
	if err := validateWGKey("PrivateKey", c.Interface.PrivateKey, true); err != nil {
		return err
	}
	if err := validateWGKey("HeaderProtectionKey", c.Interface.HeaderProtectionKey, false); err != nil {
		return err
	}
	if c.Interface.HeaderProtectionKey != "" && c.Interface.S3 != nil && *c.Interface.S3 < 12 {
		return fmt.Errorf("S3 must be at least 12 to use HeaderProtectionKey, got %d", *c.Interface.S3)
	}
	if len(c.Interface.Addresses) == 0 {
		return errors.New("interface Address is required")
	}
	for _, addr := range c.Interface.Addresses {
		if err := validateAddrOrPrefix(addr); err != nil {
			return fmt.Errorf("invalid interface Address %q: %w", addr, err)
		}
	}
	if c.Interface.MTU < 0 || c.Interface.MTU > 65535 {
		return fmt.Errorf("invalid MTU %d", c.Interface.MTU)
	}
	if c.Interface.ListenPort < 0 || c.Interface.ListenPort > 65535 {
		return fmt.Errorf("invalid ListenPort %d", c.Interface.ListenPort)
	}
	if c.Interface.Jc != nil && *c.Interface.Jc < 0 {
		return errors.New("Jc must be non-negative")
	}
	if c.Interface.Jmin != nil && *c.Interface.Jmin < 0 {
		return errors.New("Jmin must be non-negative")
	}
	if c.Interface.Jmax != nil && *c.Interface.Jmax < 0 {
		return errors.New("Jmax must be non-negative")
	}
	if c.Interface.Jmin != nil && c.Interface.Jmax != nil && *c.Interface.Jmin > *c.Interface.Jmax {
		return fmt.Errorf("Jmin (%d) cannot exceed Jmax (%d)", *c.Interface.Jmin, *c.Interface.Jmax)
	}
	for _, pair := range []struct {
		name string
		r    UintRange
	}{
		{"H1", c.Interface.H1},
		{"H2", c.Interface.H2},
		{"H3", c.Interface.H3},
		{"H4", c.Interface.H4},
	} {
		if pair.r.Set && pair.r.Max > 0xFFFFFFFF {
			return fmt.Errorf("%s exceeds uint32 maximum: %d", pair.name, pair.r.Max)
		}
	}
	for i, peer := range c.Peers {
		if err := validateWGKey(fmt.Sprintf("Peer[%d].PublicKey", i), peer.PublicKey, true); err != nil {
			return err
		}
		if err := validateWGKey(fmt.Sprintf("Peer[%d].PresharedKey", i), peer.PresharedKey, false); err != nil {
			return err
		}
		for _, cidr := range peer.AllowedIPs {
			if err := validateAddrOrPrefix(cidr); err != nil {
				return fmt.Errorf("Peer[%d] invalid AllowedIPs %q: %w", i, cidr, err)
			}
		}
		if peer.PersistentKeepalive.Set && peer.PersistentKeepalive.Max > 65535 {
			return fmt.Errorf("Peer[%d] PersistentKeepalive exceeds uint16 max", i)
		}
	}
	return nil
}

// DetectVersion classifies the config as WireGuard, AWG 2.0, AWG 3.0, or AWG 3.1.
func (c *AWGConfig) DetectVersion() AWGVersion {
	if c.Interface.RandomTrailers != nil || c.Interface.DisableCookies != nil {
		return VersionAWG31
	}
	if c.Interface.HeaderProtectionKey != "" ||
		c.Interface.ContentPaddingAddition != nil ||
		c.Interface.RekeyAfterTime != nil ||
		c.Interface.RekeyTimeout != nil ||
		c.Interface.RejectAfterTime != nil ||
		c.Interface.KeepaliveTimeout != nil ||
		c.Interface.MaxHandshakeAttempts != nil ||
		c.Interface.S3 != nil ||
		c.Interface.S4 != nil ||
		c.Interface.H1.IsRange() ||
		c.Interface.H2.IsRange() ||
		c.Interface.H3.IsRange() ||
		c.Interface.H4.IsRange() {
		return VersionAWG30
	}
	for _, p := range c.Peers {
		if p.PersistentKeepalive.IsRange() {
			return VersionAWG30
		}
	}
	if c.Interface.Jc != nil ||
		c.Interface.Jmin != nil ||
		c.Interface.Jmax != nil ||
		c.Interface.S1 != nil ||
		c.Interface.S2 != nil ||
		c.Interface.H1.Set ||
		c.Interface.H2.Set ||
		c.Interface.H3.Set ||
		c.Interface.H4.Set ||
		c.Interface.I1 != "" ||
		c.Interface.I2 != "" ||
		c.Interface.I3 != "" ||
		c.Interface.I4 != "" ||
		c.Interface.I5 != "" {
		return VersionAWG20
	}
	return VersionWireGuard
}

// Serialize formats the AWGConfig back into canonical INI text.
func (c *AWGConfig) Serialize() string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	writeKV(&b, "PrivateKey", c.Interface.PrivateKey)
	if len(c.Interface.Addresses) > 0 {
		writeKV(&b, "Address", strings.Join(c.Interface.Addresses, ", "))
	}
	if len(c.Interface.DNS) > 0 {
		writeKV(&b, "DNS", strings.Join(c.Interface.DNS, ", "))
	}
	if c.Interface.MTU > 0 {
		writeKV(&b, "MTU", strconv.Itoa(c.Interface.MTU))
	}
	if c.Interface.ListenPort > 0 {
		writeKV(&b, "ListenPort", strconv.Itoa(c.Interface.ListenPort))
	}
	if c.Interface.Table != "" {
		writeKV(&b, "Table", c.Interface.Table)
	}
	if c.Interface.FwMark != "" {
		writeKV(&b, "FwMark", c.Interface.FwMark)
	}
	for _, cmd := range c.Interface.PreUp {
		writeKV(&b, "PreUp", cmd)
	}
	for _, cmd := range c.Interface.PostUp {
		writeKV(&b, "PostUp", cmd)
	}
	for _, cmd := range c.Interface.PreDown {
		writeKV(&b, "PreDown", cmd)
	}
	for _, cmd := range c.Interface.PostDown {
		writeKV(&b, "PostDown", cmd)
	}
	writeIntPtr(&b, "Jc", c.Interface.Jc)
	writeIntPtr(&b, "Jmin", c.Interface.Jmin)
	writeIntPtr(&b, "Jmax", c.Interface.Jmax)
	writeIntPtr(&b, "S1", c.Interface.S1)
	writeIntPtr(&b, "S2", c.Interface.S2)
	writeIntPtr(&b, "S3", c.Interface.S3)
	writeIntPtr(&b, "S4", c.Interface.S4)
	if c.Interface.H1.Set {
		writeKV(&b, "H1", c.Interface.H1.String())
	}
	if c.Interface.H2.Set {
		writeKV(&b, "H2", c.Interface.H2.String())
	}
	if c.Interface.H3.Set {
		writeKV(&b, "H3", c.Interface.H3.String())
	}
	if c.Interface.H4.Set {
		writeKV(&b, "H4", c.Interface.H4.String())
	}
	if c.Interface.I1 != "" {
		writeKV(&b, "I1", c.Interface.I1)
	}
	if c.Interface.I2 != "" {
		writeKV(&b, "I2", c.Interface.I2)
	}
	if c.Interface.I3 != "" {
		writeKV(&b, "I3", c.Interface.I3)
	}
	if c.Interface.I4 != "" {
		writeKV(&b, "I4", c.Interface.I4)
	}
	if c.Interface.I5 != "" {
		writeKV(&b, "I5", c.Interface.I5)
	}
	if c.Interface.HeaderProtectionKey != "" {
		writeKV(&b, "HeaderProtectionKey", c.Interface.HeaderProtectionKey)
	}
	writeIntPtr(&b, "ContentPaddingAddition", c.Interface.ContentPaddingAddition)
	writeIntPtr(&b, "RekeyAfterTime", c.Interface.RekeyAfterTime)
	writeIntPtr(&b, "RekeyTimeout", c.Interface.RekeyTimeout)
	writeIntPtr(&b, "RejectAfterTime", c.Interface.RejectAfterTime)
	writeIntPtr(&b, "KeepaliveTimeout", c.Interface.KeepaliveTimeout)
	writeIntPtr(&b, "MaxHandshakeAttempts", c.Interface.MaxHandshakeAttempts)
	if c.Interface.RandomTrailers != nil {
		writeKV(&b, "RandomTrailers", formatOnOff(*c.Interface.RandomTrailers))
	}
	if c.Interface.DisableCookies != nil {
		writeKV(&b, "DisableCookies", formatOnOff(*c.Interface.DisableCookies))
	}

	for _, peer := range c.Peers {
		b.WriteString("\n[Peer]\n")
		writeKV(&b, "PublicKey", peer.PublicKey)
		if peer.PresharedKey != "" {
			writeKV(&b, "PresharedKey", peer.PresharedKey)
		}
		if len(peer.AllowedIPs) > 0 {
			writeKV(&b, "AllowedIPs", strings.Join(peer.AllowedIPs, ", "))
		}
		if peer.Endpoint != "" {
			writeKV(&b, "Endpoint", peer.Endpoint)
		}
		if peer.PersistentKeepalive.Set {
			writeKV(&b, "PersistentKeepalive", peer.PersistentKeepalive.String())
		}
	}
	return b.String()
}

// ToNodeItem converts a parsed AWGConfig into a UI-ready NodeItem.
func (c *AWGConfig) ToNodeItem(sourceID, nodeID, name, rawConf string) api.NodeItem {
	ver := c.DetectVersion()
	endpoint := ""
	if len(c.Peers) > 0 {
		endpoint = c.Peers[0].Endpoint
	}
	mtu := c.Interface.MTU
	if mtu == 0 {
		mtu = 1280
	}
	params := make(map[string]string)
	if c.Interface.Jc != nil {
		params["Jc"] = strconv.Itoa(*c.Interface.Jc)
	}
	if c.Interface.Jmin != nil {
		params["Jmin"] = strconv.Itoa(*c.Interface.Jmin)
	}
	if c.Interface.Jmax != nil {
		params["Jmax"] = strconv.Itoa(*c.Interface.Jmax)
	}
	if c.Interface.S1 != nil {
		params["S1"] = strconv.Itoa(*c.Interface.S1)
	}
	if c.Interface.S2 != nil {
		params["S2"] = strconv.Itoa(*c.Interface.S2)
	}
	if c.Interface.S3 != nil {
		params["S3"] = strconv.Itoa(*c.Interface.S3)
	}
	if c.Interface.S4 != nil {
		params["S4"] = strconv.Itoa(*c.Interface.S4)
	}
	if c.Interface.H1.Set {
		params["H1"] = c.Interface.H1.String()
	}
	if c.Interface.H2.Set {
		params["H2"] = c.Interface.H2.String()
	}
	if c.Interface.H3.Set {
		params["H3"] = c.Interface.H3.String()
	}
	if c.Interface.H4.Set {
		params["H4"] = c.Interface.H4.String()
	}
	if c.Interface.I1 != "" {
		params["I1"] = c.Interface.I1
	}
	if c.Interface.HeaderProtectionKey != "" {
		params["HeaderProtectionKey"] = "set"
	}
	if c.Interface.RandomTrailers != nil {
		params["RandomTrailers"] = formatOnOff(*c.Interface.RandomTrailers)
	}
	if c.Interface.DisableCookies != nil {
		params["DisableCookies"] = formatOnOff(*c.Interface.DisableCookies)
	}
	if rawConf == "" {
		rawConf = c.Serialize()
	}
	if name == "" {
		name = string(ver) + " " + endpoint
	}
	return api.NodeItem{
		ID:         nodeID,
		SourceID:   sourceID,
		ISO:        "--",
		Name:       strings.TrimSpace(name),
		Endpoint:   endpoint,
		Proto:      string(ver),
		ProtoGroup: api.ProtoGroupAWG,
		AntiBot:    api.AntiBotClean,
		MTU:        mtu,
		Params:     params,
		RawConfig:  rawConf,
	}
}

func splitCSV(val string) []string {
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseIntPtr(val string) (*int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseFlexBool(val string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "on", "yes":
		return true, nil
	case "0", "false", "off", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", val)
	}
}

func formatOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func validateWGKey(field, val string, required bool) error {
	if val == "" {
		if required {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return fmt.Errorf("%s is not valid base64: %w", field, err)
	}
	if len(decoded) != 32 {
		return fmt.Errorf("%s must decode to 32 bytes, got %d", field, len(decoded))
	}
	return nil
}

func validateAddrOrPrefix(raw string) error {
	if _, err := netip.ParsePrefix(raw); err == nil {
		return nil
	}
	if _, err := netip.ParseAddr(raw); err == nil {
		return nil
	}
	return errors.New("must be a valid IP or CIDR prefix")
}

func writeKV(b *strings.Builder, k, v string) {
	b.WriteString(k)
	b.WriteString(" = ")
	b.WriteString(v)
	b.WriteByte('\n')
}

func writeIntPtr(b *strings.Builder, k string, p *int) {
	if p != nil {
		writeKV(b, k, strconv.Itoa(*p))
	}
}
