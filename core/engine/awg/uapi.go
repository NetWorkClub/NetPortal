package awg

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/NetWorkClub/NetPortal/core/parser"
)

// ToUAPI converts a validated AWGConfig into WireGuard/AmneziaWG UAPI format
// accepted by amneziawg-go/v3 Device.IpcSet.
func ToUAPI(cfg *parser.AWGConfig) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("awg: nil config")
	}
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	var b strings.Builder
	privHex, err := b64KeyToHex(cfg.Interface.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("private_key: %w", err)
	}
	writeUAPI(&b, "private_key", privHex)

	if cfg.Interface.ListenPort > 0 {
		writeUAPI(&b, "listen_port", strconv.Itoa(cfg.Interface.ListenPort))
	}
	if cfg.Interface.FwMark != "" {
		writeUAPI(&b, "fwmark", cfg.Interface.FwMark)
	}

	writeUAPIIntPtr(&b, "jc", cfg.Interface.Jc)
	writeUAPIIntPtr(&b, "jmin", cfg.Interface.Jmin)
	writeUAPIIntPtr(&b, "jmax", cfg.Interface.Jmax)
	writeUAPIIntPtr(&b, "s1", cfg.Interface.S1)
	writeUAPIIntPtr(&b, "s2", cfg.Interface.S2)
	writeUAPIIntPtr(&b, "s3", cfg.Interface.S3)
	writeUAPIIntPtr(&b, "s4", cfg.Interface.S4)

	if cfg.Interface.H1.Set {
		writeUAPI(&b, "h1", cfg.Interface.H1.String())
	}
	if cfg.Interface.H2.Set {
		writeUAPI(&b, "h2", cfg.Interface.H2.String())
	}
	if cfg.Interface.H3.Set {
		writeUAPI(&b, "h3", cfg.Interface.H3.String())
	}
	if cfg.Interface.H4.Set {
		writeUAPI(&b, "h4", cfg.Interface.H4.String())
	}
	if cfg.Interface.I1 != "" {
		writeUAPI(&b, "i1", cfg.Interface.I1)
	}
	if cfg.Interface.I2 != "" {
		writeUAPI(&b, "i2", cfg.Interface.I2)
	}
	if cfg.Interface.I3 != "" {
		writeUAPI(&b, "i3", cfg.Interface.I3)
	}
	if cfg.Interface.I4 != "" {
		writeUAPI(&b, "i4", cfg.Interface.I4)
	}
	if cfg.Interface.I5 != "" {
		writeUAPI(&b, "i5", cfg.Interface.I5)
	}
	if cfg.Interface.HeaderProtectionKey != "" {
		hpkHex, err := b64KeyToHex(cfg.Interface.HeaderProtectionKey)
		if err != nil {
			return "", fmt.Errorf("header_protection_key: %w", err)
		}
		writeUAPI(&b, "header_protection_key", hpkHex)
	}
	writeUAPIIntPtr(&b, "content_padding_addition", cfg.Interface.ContentPaddingAddition)
	writeUAPIIntPtr(&b, "rekey_after_time", cfg.Interface.RekeyAfterTime)
	writeUAPIIntPtr(&b, "rekey_timeout", cfg.Interface.RekeyTimeout)
	writeUAPIIntPtr(&b, "reject_after_time", cfg.Interface.RejectAfterTime)
	writeUAPIIntPtr(&b, "keepalive_timeout", cfg.Interface.KeepaliveTimeout)
	writeUAPIIntPtr(&b, "max_handshake_attempts", cfg.Interface.MaxHandshakeAttempts)

	if cfg.Interface.RandomTrailers != nil {
		writeUAPI(&b, "random_trailers", strconv.FormatBool(*cfg.Interface.RandomTrailers))
	}
	if cfg.Interface.DisableCookies != nil {
		writeUAPI(&b, "disable_cookies", strconv.FormatBool(*cfg.Interface.DisableCookies))
	}

	for _, peer := range cfg.Peers {
		pubHex, err := b64KeyToHex(peer.PublicKey)
		if err != nil {
			return "", fmt.Errorf("public_key: %w", err)
		}
		writeUAPI(&b, "public_key", pubHex)

		if peer.PresharedKey != "" {
			pskHex, err := b64KeyToHex(peer.PresharedKey)
			if err != nil {
				return "", fmt.Errorf("preshared_key: %w", err)
			}
			writeUAPI(&b, "preshared_key", pskHex)
		}
		if peer.Endpoint != "" {
			writeUAPI(&b, "endpoint", peer.Endpoint)
		}
		if peer.PersistentKeepalive.Set {
			writeUAPI(&b, "persistent_keepalive_interval", peer.PersistentKeepalive.String())
		}
		for _, allowed := range peer.AllowedIPs {
			writeUAPI(&b, "allowed_ip", allowed)
		}
	}

	return b.String(), nil
}

func b64KeyToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return "", err
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("expected 32-byte key, got %d", len(raw))
	}
	return hex.EncodeToString(raw), nil
}

func writeUAPI(b *strings.Builder, k, v string) {
	b.WriteString(k)
	b.WriteByte('=')
	b.WriteString(v)
	b.WriteByte('\n')
}

func writeUAPIIntPtr(b *strings.Builder, k string, v *int) {
	if v != nil {
		writeUAPI(b, k, strconv.Itoa(*v))
	}
}
