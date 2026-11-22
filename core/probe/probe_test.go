package probe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
)

type directDialer struct{}

func (directDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, network, address)
}

func TestProbeTCPAndURLWithAntiBot(t *testing.T) {
	cleanSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanSrv.Close()

	challengeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html><title>Just a moment...</title></html>"))
	}))
	defer challengeSrv.Close()

	node := api.NodeItem{
		ID:       "n-1",
		Endpoint: cleanSrv.Listener.Addr().String(),
	}
	tcpOut := ProbeTCP(context.Background(), node, 2*time.Second)
	if tcpOut.TcpMs < 1 || tcpOut.AntiBot != api.AntiBotClean {
		t.Fatalf("unexpected TCP probe: %+v", tcpOut)
	}

	urlClean := ProbeURL(context.Background(), node, directDialer{}, cleanSrv.URL, 2*time.Second)
	if urlClean.UrlMs == nil || urlClean.AntiBot != api.AntiBotClean {
		t.Fatalf("expected Clean URL probe, got %+v", urlClean)
	}

	urlChal := ProbeURL(context.Background(), node, directDialer{}, challengeSrv.URL, 2*time.Second)
	if urlChal.AntiBot != api.AntiBotChallenge {
		t.Fatalf("expected Challenge anti-bot state, got %+v", urlChal)
	}

	batch := ProbeBatch(context.Background(), []api.NodeItem{node, node}, 2, func(ctx context.Context, n api.NodeItem) api.ProbeOutcome {
		return ProbeTCP(ctx, n, time.Second)
	})
	if len(batch) != 2 || batch[0].AntiBot != api.AntiBotClean {
		t.Fatalf("unexpected batch result: %+v", batch)
	}
}
