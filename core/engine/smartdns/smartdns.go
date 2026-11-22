package smartdns

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
)

const EngineID = "smartdns"

// Config configures the SmartDNS / Smart Hosts engine.
type Config struct {
	Mode      api.SmartDnsMode  `json:"mode"`
	HostsPath string            `json:"hostsPath,omitempty"`
	Entries   map[string]string `json:"entries,omitempty"`
	Listen    string            `json:"listen,omitempty"`
}

// Engine implements engine.Engine for Smart Hosts and local HTTP CONNECT Smart DNS proxy.
type Engine struct {
	mu       sync.RWMutex
	running  bool
	cfg      Config
	listener net.Listener
	httpSrv  *http.Server
	rxBytes  atomic.Uint64
	txBytes  atomic.Uint64
	lastErr  string
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) ID() string {
	return EngineID
}

func (e *Engine) Capabilities() engine.EngineCapabilities {
	return engine.EngineCapabilities{
		SupportsTUN:        false,
		SupportsSOCKS:      true,
		SupportsTestDialer: true,
		RequiresElevation:  false,
	}
}

func (e *Engine) Validate(cfg any) error {
	c, ok := cfg.(Config)
	if !ok {
		return fmt.Errorf("smartdns: expected Config, got %T", cfg)
	}
	if c.Mode == "" {
		return errors.New("smartdns: mode is required")
	}
	return nil
}

func (e *Engine) Start(ctx context.Context, runtime engine.EngineRuntime) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return engine.ErrAlreadyRunning
	}

	addr := "127.0.0.1:0"
	if runtime.SocksPort > 0 {
		addr = fmt.Sprintf("127.0.0.1:%d", runtime.SocksPort)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		e.lastErr = err.Error()
		return err
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				e.handleConnect(w, r)
				return
			}
			http.Error(w, "NetPortal SmartDNS HTTP CONNECT proxy", http.StatusMethodNotAllowed)
		}),
	}

	e.listener = ln
	e.httpSrv = srv
	e.running = true
	e.lastErr = ""

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.mu.Lock()
			e.lastErr = err.Error()
			e.mu.Unlock()
		}
	}()

	go func() {
		<-ctx.Done()
		_ = e.Stop(context.Background())
	}()

	return nil
}

func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	srv := e.httpSrv
	ln := e.listener
	cfg := e.cfg
	e.httpSrv = nil
	e.listener = nil
	e.running = false
	e.mu.Unlock()

	if srv != nil {
		_ = srv.Shutdown(ctx)
	}
	if ln != nil {
		_ = ln.Close()
	}
	if cfg.Mode == api.SmartDnsHosts && cfg.HostsPath != "" {
		_ = RemoveHostsEntries(cfg.HostsPath)
	}
	return nil
}

func (e *Engine) Health(_ context.Context) engine.EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	state := api.StatusDisconnected
	iface := ""
	if e.running {
		state = api.StatusConnected
		if e.listener != nil {
			iface = e.listener.Addr().String()
		}
	}
	return engine.EngineStatus{
		ID:            EngineID,
		State:         state,
		InterfaceName: iface,
		RxBytes:       e.rxBytes.Load(),
		TxBytes:       e.txBytes.Load(),
		LastError:     e.lastErr,
	}
}

func (e *Engine) TestDialer(_ api.NodeItem) (engine.Dialer, error) {
	return &net.Dialer{Timeout: 5 * time.Second}, nil
}

func (e *Engine) handleConnect(w http.ResponseWriter, r *http.Request) {
	targetConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		_ = targetConn.Close()
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		_ = targetConn.Close()
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go func() {
		defer targetConn.Close()
		defer clientConn.Close()
		n, _ := io.Copy(targetConn, clientConn)
		e.txBytes.Add(uint64(n))
	}()
	go func() {
		defer targetConn.Close()
		defer clientConn.Close()
		n, _ := io.Copy(clientConn, targetConn)
		e.rxBytes.Add(uint64(n))
	}()
}
