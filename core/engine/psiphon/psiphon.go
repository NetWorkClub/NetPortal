// Package psiphon wraps psiphon-tunnel-core for censorship-resistant egress
// with selectable exit country and sticky-fallback chain integration.
package psiphon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
)

const EngineID = "psiphon"

// Config defines Psiphon tunnel parameters.
type Config struct {
	BinaryPath   string `json:"binaryPath"`
	EgressRegion string `json:"egressRegion"` // ISO 3166-1 alpha-2 e.g. "DE", "NL", "US"
	SocksPort    int    `json:"socksPort"`
}

// Engine implements engine.Engine for Psiphon.
type Engine struct {
	mu           sync.RWMutex
	running      bool
	cmd          *exec.Cmd
	activeNodeID string
	socksPort    int
	lastErr      string
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
		SupportsTestDialer: false,
		RequiresElevation:  false,
	}
}

func (e *Engine) Validate(cfg any) error {
	c, ok := cfg.(Config)
	if !ok {
		return fmt.Errorf("psiphon: expected Config, got %T", cfg)
	}
	if len(strings.TrimSpace(c.EgressRegion)) > 2 {
		return errors.New("psiphon: egressRegion must be a 2-letter ISO code")
	}
	return nil
}

func (e *Engine) Start(ctx context.Context, runtime engine.EngineRuntime) error {
	binPath := runtime.Node.Params["binaryPath"]
	if binPath == "" {
		return errors.New("psiphon: binaryPath parameter is required")
	}
	_ = os.Chmod(binPath, 0o755)

	port := runtime.SocksPort
	if port <= 0 {
		port = 10809
	}

	args := []string{"-listen", fmt.Sprintf("127.0.0.1:%d", port)}
	if region := strings.TrimSpace(runtime.Node.Params["region"]); region != "" {
		args = append(args, "-region", strings.ToUpper(region))
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return engine.ErrAlreadyRunning
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Start(); err != nil {
		e.lastErr = err.Error()
		return fmt.Errorf("psiphon: start: %w", err)
	}

	e.cmd = cmd
	e.activeNodeID = runtime.Node.ID
	e.socksPort = port
	e.running = true
	e.lastErr = ""

	go func() {
		err := cmd.Wait()
		e.mu.Lock()
		if e.cmd == cmd {
			e.running = false
			e.cmd = nil
			e.activeNodeID = ""
			if err != nil {
				e.lastErr = err.Error()
			}
		}
		e.mu.Unlock()
	}()
	return nil
}

func (e *Engine) Stop(_ context.Context) error {
	e.mu.Lock()
	cmd := e.cmd
	e.cmd = nil
	e.activeNodeID = ""
	e.running = false
	e.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}

func (e *Engine) Health(_ context.Context) engine.EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	st := api.StatusDisconnected
	iface := ""
	if e.running {
		st = api.StatusConnected
		iface = fmt.Sprintf("127.0.0.1:%d", e.socksPort)
	}
	return engine.EngineStatus{
		ID:            EngineID,
		State:         st,
		ActiveNodeID:  e.activeNodeID,
		InterfaceName: iface,
		LastError:     e.lastErr,
	}
}

func (e *Engine) TestDialer(_ api.NodeItem) (engine.Dialer, error) {
	return nil, errors.New("psiphon: TestDialer not supported")
}
