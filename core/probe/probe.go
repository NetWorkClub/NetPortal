// Package probe implements pre-connect two-tier node probing:
// Tier 1 (fast TCP/TLS handshake) and Tier 2 (ephemeral no-TUN URL test + anti-bot detection).
package probe

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
)

const DefaultTestURL = "https://www.gstatic.com/generate_204"

// ProbeTCP measures TCP handshake latency (and optional TLS handshake if sni != "")
// without touching system TUN or proxy settings.
func ProbeTCP(ctx context.Context, node api.NodeItem, timeout time.Duration) api.ProbeOutcome {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	endpoint := strings.TrimSpace(node.Endpoint)
	if endpoint == "" {
		return api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   "empty endpoint",
		}
	}

	start := time.Now()
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   err.Error(),
		}
	}

	if sni := node.Params["sni"]; sni != "" && (node.Params["security"] == "tls" || node.Params["security"] == "reality") {
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         sni,
			InsecureSkipVerify: true, // Handshake timing only; certificate trust is verified inside the engine.
		})
		_ = tlsConn.SetDeadline(time.Now().Add(timeout))
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = tlsConn.Close()
			return api.ProbeOutcome{
				NodeID:  node.ID,
				TcpMs:   -1,
				AntiBot: api.AntiBotError,
				Error:   err.Error(),
			}
		}
		_ = tlsConn.Close()
	} else {
		_ = conn.Close()
	}

	ms := int(time.Since(start).Milliseconds())
	if ms < 1 {
		ms = 1
	}
	return api.ProbeOutcome{
		NodeID:  node.ID,
		TcpMs:   ms,
		AntiBot: api.AntiBotClean,
	}
}

// ProbeURL runs an ephemeral HTTP GET through the provided in-memory Dialer
// (no TUN, no system proxy) and evaluates latency plus anti-bot/captcha signals.
func ProbeURL(ctx context.Context, node api.NodeItem, dialer engine.Dialer, targetURL string, timeout time.Duration) api.ProbeOutcome {
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	if targetURL == "" {
		targetURL = DefaultTestURL
	}
	if dialer == nil {
		return api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   "nil dialer",
		}
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:       dialer.DialContext,
			DisableKeepAlives: true,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   err.Error(),
		}
	}
	req.Header.Set("User-Agent", "NetPortal/1.0")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   err.Error(),
		}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	ms := int(time.Since(start).Milliseconds())
	if ms < 1 {
		ms = 1
	}

	antiBot := DetectAntiBot(resp.StatusCode, resp.Header, string(bodyBytes))
	return api.ProbeOutcome{
		NodeID:  node.ID,
		TcpMs:   ms,
		UrlMs:   &ms,
		AntiBot: antiBot,
	}
}

// DetectAntiBot inspects HTTP status, headers, and HTML body for Cloudflare Turnstile,
// Google Sorry, or generic captcha challenges.
func DetectAntiBot(statusCode int, headers http.Header, body string) api.AntiBotState {
	if headers != nil && strings.EqualFold(headers.Get("cf-mitigated"), "challenge") {
		return api.AntiBotChallenge
	}
	lower := strings.ToLower(body)
	for _, marker := range []string{
		"cf-turnstile",
		"challenges.cloudflare.com",
		"just a moment...",
		"/sorry/index",
		"g-recaptcha",
		"unusual traffic from your computer",
	} {
		if strings.Contains(lower, marker) {
			return api.AntiBotChallenge
		}
	}
	if statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable {
		return api.AntiBotChallenge
	}
	if statusCode >= 200 && statusCode < 400 {
		return api.AntiBotClean
	}
	return api.AntiBotError
}

// ProbeBatch runs probeFn across nodes using a bounded worker pool.
func ProbeBatch(ctx context.Context, nodes []api.NodeItem, workers int, probeFn func(context.Context, api.NodeItem) api.ProbeOutcome) []api.ProbeOutcome {
	if workers <= 0 {
		workers = 8
	}
	results := make([]api.ProbeOutcome, len(nodes))
	jobs := make(chan int, len(nodes))
	for i := range nodes {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < workers && w < len(nodes); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if ctx.Err() != nil {
					results[idx] = api.ProbeOutcome{
						NodeID:  nodes[idx].ID,
						TcpMs:   -1,
						AntiBot: api.AntiBotError,
						Error:   ctx.Err().Error(),
					}
					continue
				}
				results[idx] = probeFn(ctx, nodes[idx])
			}
		}()
	}
	wg.Wait()
	return results
}
