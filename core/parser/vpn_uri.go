package parser

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/NetWorkClub/NetPortal/core/api"
)

const vpnURIPrefix = "vpn://"

type amneziaContainerPayload struct {
	Description string           `json:"description,omitempty"`
	Containers  []map[string]any `json:"containers,omitempty"`
	DefaultCont string           `json:"defaultContainer,omitempty"`
}

// DecodeVPNURI decodes an Amnezia vpn:// URI (Qt qCompress + base64url, raw zlib, or plain text)
// and extracts the tunnel description and INI config text.
func DecodeVPNURI(uri string) (name string, confText string, err error) {
	trimmed := strings.TrimSpace(uri)
	if !strings.HasPrefix(trimmed, vpnURIPrefix) {
		return "", "", errors.New("URI must start with vpn://")
	}
	payloadB64 := strings.TrimPrefix(trimmed, vpnURIPrefix)
	payloadB64 = strings.TrimRight(payloadB64, "=")

	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		// Fallback to standard base64 if URL encoding was not used.
		raw, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed, vpnURIPrefix))
		if err != nil {
			return "", "", fmt.Errorf("decode vpn:// base64: %w", err)
		}
	}

	decompressed, err := tryDecompressVPNPayload(raw)
	if err != nil {
		return "", "", err
	}

	text := strings.TrimSpace(string(decompressed))
	if strings.HasPrefix(text, "{") {
		var doc amneziaContainerPayload
		if err := json.Unmarshal(decompressed, &doc); err != nil {
			return "", "", fmt.Errorf("decode Amnezia JSON payload: %w", err)
		}
		for _, c := range doc.Containers {
			for _, key := range []string{"awg", "wireguard"} {
				if sub, ok := c[key].(map[string]any); ok {
					if lc, ok := sub["last_config"].(string); ok && strings.TrimSpace(lc) != "" {
						return doc.Description, extractLastConfigText(lc), nil
					}
				}
			}
		}
		return "", "", errors.New("no awg/wireguard last_config found in Amnezia JSON")
	}

	if strings.Contains(text, "[Interface]") {
		return "", text, nil
	}
	return "", "", errors.New("unsupported vpn:// payload content")
}

func extractLastConfigText(lastConfig string) string {
	trimmed := strings.TrimSpace(lastConfig)
	if strings.HasPrefix(trimmed, "{") {
		var nested struct {
			Config string `json:"config"`
		}
		if err := json.Unmarshal([]byte(trimmed), &nested); err == nil && nested.Config != "" {
			return nested.Config
		}
	}
	return trimmed
}

func tryDecompressVPNPayload(raw []byte) ([]byte, error) {
	// Case 1: Qt qCompress format (4-byte big-endian uncompressed size + zlib stream).
	if len(raw) > 6 && raw[4] == 0x78 {
		if out, err := inflateZlib(raw[4:]); err == nil {
			expected := binary.BigEndian.Uint32(raw[:4])
			if expected == 0 || uint32(len(out)) == expected {
				return out, nil
			}
			return out, nil
		}
	}
	// Case 2: Raw zlib stream without 4-byte prefix.
	if len(raw) > 2 && raw[0] == 0x78 {
		if out, err := inflateZlib(raw); err == nil {
			return out, nil
		}
	}
	// Case 3: Uncompressed JSON or INI text.
	return raw, nil
}

func inflateZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	// Limit to 4 MiB to protect against decompression bombs.
	return io.ReadAll(io.LimitReader(r, 4<<20))
}

// EncodeVPNURI encodes an INI config (and optional name) into an Amnezia vpn:// URI
// using Qt qCompress framing (4-byte BE length + zlib) + unpadded base64url.
func EncodeVPNURI(name, confText string) (string, error) {
	if strings.TrimSpace(confText) == "" {
		return "", errors.New("confText must not be empty")
	}
	var plain []byte
	if name != "" {
		nested, err := json.Marshal(map[string]string{"config": confText})
		if err != nil {
			return "", err
		}
		doc := amneziaContainerPayload{
			Description: name,
			DefaultCont: "amnezia-awg",
			Containers: []map[string]any{
				{
					"container": "amnezia-awg",
					"awg": map[string]any{
						"last_config": string(nested),
					},
				},
			},
		}
		plain, err = json.Marshal(doc)
		if err != nil {
			return "", err
		}
	} else {
		plain = []byte(confText)
	}

	var compressed bytes.Buffer
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(plain)))
	compressed.Write(lenBuf[:])

	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(plain); err != nil {
		_ = zw.Close()
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}

	return vpnURIPrefix + base64.RawURLEncoding.EncodeToString(compressed.Bytes()), nil
}

// DetectImport classifies raw user input (AWG .conf or vpn:// URI).
func DetectImport(raw string) (api.ImportDetectionResult, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return api.ImportDetectionResult{}, errors.New("empty import input")
	}

	conf := trimmed
	if strings.HasPrefix(trimmed, vpnURIPrefix) {
		_, decoded, err := DecodeVPNURI(trimmed)
		if err != nil {
			return api.ImportDetectionResult{}, err
		}
		conf = decoded
	}

	if strings.Contains(conf, "[Interface]") {
		cfg, err := ParseAWGConf(conf)
		if err != nil {
			return api.ImportDetectionResult{}, err
		}
		ver := cfg.DetectVersion()
		return api.ImportDetectionResult{
			Proto:      string(ver),
			ProtoGroup: api.ProtoGroupAWG,
			Label:      string(ver),
		}, nil
	}

	if nodes, err := ParseSubscription("detect", trimmed); err == nil && len(nodes) > 0 {
		label := nodes[0].Proto
		if len(nodes) > 1 {
			label = fmt.Sprintf("%s (%d nodes)", nodes[0].Proto, len(nodes))
		}
		return api.ImportDetectionResult{
			Proto:      nodes[0].Proto,
			ProtoGroup: nodes[0].ProtoGroup,
			Label:      label,
		}, nil
	}

	return api.ImportDetectionResult{}, errors.New("unrecognized configuration format")
}
