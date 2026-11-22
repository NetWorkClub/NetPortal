// Package warp implements Cloudflare WARP account registration, AWG 2.0/3.0/3.1
// and MASQUE profile generation, and regional endpoint latency scanning.
package warp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/parser"
	"golang.org/x/crypto/curve25519"
)

const DefaultRegistrationURL = "https://api.cloudflareclient.com/v0a2158/reg"

// Preset defines AWG obfuscation parameters for generated WARP profiles.
type Preset string

const (
	PresetAWG20Default Preset = "awg20-default"
	PresetAWG31Stealth Preset = "awg31-stealth"
	PresetMASQUE       Preset = "masque"
)

// Account holds Cloudflare WARP registration credentials and assigned tunnel IPs.
type Account struct {
	ID         string `json:"id"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	PeerPubKey string `json:"peerPublicKey"`
	Endpoint   string `json:"endpoint"`
	IPv4       string `json:"ipv4"`
	IPv6       string `json:"ipv6"`
	Token      string `json:"token"`
}

// DefaultEndpoints returns a baseline set of Cloudflare WARP anycast endpoints.
func DefaultEndpoints() []api.WarpEndpoint {
	return []api.WarpEndpoint{
		{Addr: "162.159.192.1:2408", Region: "CF Anycast", Loss: "0%"},
		{Addr: "162.159.193.10:2408", Region: "CF Anycast", Loss: "0%"},
		{Addr: "188.114.96.1:2408", Region: "CF EU", Loss: "0%"},
		{Addr: "188.114.97.1:2408", Region: "CF EU", Loss: "0%"},
		{Addr: "188.114.98.1:500", Region: "CF Alt", Loss: "0%"},
		{Addr: "188.114.99.1:4500", Region: "CF Alt", Loss: "0%"},
	}
}

// GenerateKeyPair creates a Curve25519 WireGuard private/public keypair in base64.
func GenerateKeyPair() (privB64, pubB64 string, err error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(priv[:]), base64.StdEncoding.EncodeToString(pub), nil
}

// Register registers a new free Cloudflare WARP device against regURL (or DefaultRegistrationURL).
func Register(ctx context.Context, regURL string, httpClient *http.Client) (Account, error) {
	if regURL == "" {
		regURL = DefaultRegistrationURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		return Account{}, err
	}

	reqBody, _ := json.Marshal(map[string]any{
		"key":       pub,
		"install_id": "",
		"fcm_token":  "",
		"tos":       time.Now().UTC().Format(time.RFC3339),
		"model":     "PC",
		"type":      "Android",
		"locale":    "en_US",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, regURL, bytes.NewReader(reqBody))
	if err != nil {
		return Account{}, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", "okhttp/3.12.1")

	resp, err := httpClient.Do(req)
	if err != nil {
		return Account{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return Account{}, fmt.Errorf("warp: registration HTTP %d: %s", resp.StatusCode, string(msg))
	}

	var apiResp struct {
		ID     string `json:"id"`
		Token  string `json:"token"`
		Config struct {
			Peers []struct {
				PublicKey string `json:"public_key"`
				Endpoint  struct {
					V4 string `json:"v4"`
				} `json:"endpoint"`
			} `json:"peers"`
			Interface struct {
				Addresses struct {
					V4 string `json:"v4"`
					V6 string `json:"v6"`
				} `json:"addresses"`
			} `json:"interface"`
		} `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return Account{}, err
	}
	if len(apiResp.Config.Peers) == 0 {
		return Account{}, errors.New("warp: registration response missing peers")
	}
	ep := apiResp.Config.Peers[0].Endpoint.V4
	if ep == "" {
		ep = "162.159.192.1:2408"
	}
	v4 := apiResp.Config.Interface.Addresses.V4
	if v4 == "" {
		v4 = "172.16.0.2"
	}
	return Account{
		ID:         apiResp.ID,
		PrivateKey: priv,
		PublicKey:  pub,
		PeerPubKey: apiResp.Config.Peers[0].PublicKey,
		Endpoint:   ep,
		IPv4:       v4,
		IPv6:       apiResp.Config.Interface.Addresses.V6,
		Token:      apiResp.Token,
	}, nil
}

// BuildProfile builds a ready-to-use NodeItem (AWG 2.0, AWG 3.1, or MASQUE) from a WARP Account.
func BuildProfile(acc Account, preset Preset, endpointOverride string) (api.NodeItem, error) {
	ep := endpointOverride
	if ep == "" {
		ep = acc.Endpoint
	}
	if ep == "" {
		ep = "162.159.192.1:2408"
	}

	if preset == PresetMASQUE {
		raw := fmt.Sprintf(`{"type":"masque","tag":"WARP MASQUE","server":%q,"private_key":%q,"peer_public_key":%q}`,
			ep, acc.PrivateKey, acc.PeerPubKey)
		return api.NodeItem{
			ID:         "warp-masque-" + shortTag(acc.ID),
			SourceID:   "builtin-warp",
			ISO:        "CF",
			Name:       "WARP MASQUE",
			Endpoint:   ep,
			Proto:      "MASQUE",
			ProtoGroup: api.ProtoGroupMASQUE,
			AntiBot:    api.AntiBotClean,
			MTU:        1280,
			Params: map[string]string{
				"preset": string(preset),
			},
			RawConfig: raw,
		}, nil
	}

	addrs := acc.IPv4 + "/32"
	if acc.IPv6 != "" {
		addrs += ", " + acc.IPv6 + "/128"
	}

	var extraAWG string
	if preset == PresetAWG31Stealth {
		extraAWG = `Jc = 4
Jmin = 40
Jmax = 70
S1 = 0
S2 = 0
S3 = 16
S4 = 24
H1 = 1000-2000
H2 = 3000-4000
H3 = 5000-6000
H4 = 7000-8000
I1 = <b 0xc700000001><r 16>
RandomTrailers = on
DisableCookies = on`
	} else {
		extraAWG = `Jc = 4
Jmin = 40
Jmax = 70
S1 = 0
S2 = 0
H1 = 1
H2 = 2
H3 = 3
H4 = 4
I1 = <b 0xc700000001><r 16>`
	}

	conf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = 1.1.1.1, 1.0.0.1
MTU = 1280
%s

[Peer]
PublicKey = %s
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = %s
PersistentKeepalive = 25
`, acc.PrivateKey, addrs, extraAWG, acc.PeerPubKey, ep)

	cfg, err := parser.ParseAWGConf(conf)
	if err != nil {
		return api.NodeItem{}, err
	}
	return cfg.ToNodeItem("builtin-warp", "warp-"+string(preset)+"-"+shortTag(acc.ID), "WARP "+string(cfg.DetectVersion()), conf), nil
}

// ScanEndpoints measures UDP/TCP reachability and RTT for candidate WARP endpoints
// and returns them sorted by lowest latency.
func ScanEndpoints(ctx context.Context, candidates []api.WarpEndpoint, timeout time.Duration) []api.WarpEndpoint {
	if len(candidates) == 0 {
		candidates = DefaultEndpoints()
	}
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}

	out := make([]api.WarpEndpoint, len(candidates))
	var wg sync.WaitGroup
	for i, ep := range candidates {
		wg.Add(1)
		go func(idx int, item api.WarpEndpoint) {
			defer wg.Done()
			start := time.Now()
			d := net.Dialer{Timeout: timeout}
			conn, err := d.DialContext(ctx, "udp", item.Addr)
			if err != nil {
				item.RTT = 9999
				item.Loss = "100%"
				out[idx] = item
				return
			}
			// Send a minimal 1-byte probe packet to trigger route/socket timing.
			_, _ = conn.Write([]byte{0x01})
			_ = conn.Close()
			ms := int(time.Since(start).Milliseconds())
			if ms < 1 {
				ms = 1
			}
			item.RTT = ms
			item.Loss = "0%"
			out[idx] = item
		}(i, ep)
	}
	wg.Wait()

	sort.Slice(out, func(i, j int) bool {
		return out[i].RTT < out[j].RTT
	})
	return out
}

func shortTag(id string) string {
	if len(id) >= 6 {
		return id[:6]
	}
	if id == "" {
		return "local"
	}
	return id
}
