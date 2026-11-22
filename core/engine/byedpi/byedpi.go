// Package byedpi manages the rootless ciadpi (ByeDPI) SOCKS5 DPI desync engine
// across Desktop and Android (libciadpi.so).
package byedpi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
)

const EngineID = "byedpi"

// Config defines ciadpi binary location and desync CLI flags.
type Config struct {
	BinaryPath string   `json:"binaryPath"`
	SocksPort  int      `json:"socksPort"`
	Flags      []string `json:"flags"`
}

// DefaultFlags returns a baseline DPI desync flag set.
func DefaultFlags() []string {
	return []string{"--split", "1", "--disorder", "1", "--tlsrec", "1+s"}
}

// BuildArgs builds the ciadpi CLI argument slice.
func BuildArgs(port int, flags []string) []string {
	if port <= 0 {
		port = 10808
	}
	if len(flags) == 0 {
		flags = DefaultFlags()
	}
	args := []string{"--ip", "127.0.0.1", "--port", strconv.Itoa(port)}
	args = append(args, flags...)
	return args
}

// Engine implements engine.Engine for ByeDPI (ciadpi).
type Engine struct {
	mu      sync.RWMutex
	running bool
	cmd     *exec.Cmd
	port    int
	lastErr string
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
		return fmt.Errorf("byedpi: expected Config, got %T", cfg)
	}
	if strings.TrimSpace(c.BinaryPath) == "" {
		return errors.New("byedpi: binaryPath is required")
	}
	return nil
}

func (e *Engine) Start(ctx context.Context, runtime engine.EngineRuntime) error {
	binPath := runtime.Node.Params["binaryPath"]
	if binPath == "" {
		return errors.New("byedpi: binaryPath parameter is required")
	}
	port := runtime.SocksPort
	if port <= 0 {
		port = 10808
	}

	var flags []string
	if raw := strings.TrimSpace(runtime.Node.Params["flags"]); raw != "" {
		flags = strings.Fields(raw)
	}

	// Ensure executable permissions programmatically (never require manual chmod).
	_ = os.Chmod(binPath, 0o755)

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return engine.ErrAlreadyRunning
	}

	cmd := exec.CommandContext(ctx, binPath, BuildArgs(port, flags)...)
	if err := cmd.Start(); err != nil {
		e.lastErr = err.Error()
		return fmt.Errorf("byedpi: start ciadpi: %w", err)
	}

	e.cmd = cmd
	e.port = port
	e.running = true
	e.lastErr = ""

	go func() {
		err := cmd.Wait()
		e.mu.Lock()
		if e.cmd == cmd {
			e.running = false
			e.cmd = nil
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
		iface = fmt.Sprintf("127.0.0.1:%d", e.port)
	}
	return engine.EngineStatus{
		ID:            EngineID,
		State:         st,
		InterfaceName: iface,
		LastError:     e.lastErr,
	}
}

func (e *Engine) TestDialer(_ api.NodeItem) (engine.Dialer, error) {
	return nil, errors.New("byedpi: TestDialer not supported directly")
}
