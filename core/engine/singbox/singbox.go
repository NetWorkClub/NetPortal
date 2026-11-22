// Package singbox integrates sing-box as a unified multi-protocol proxy engine
// supporting VLESS (Reality/Vision/XHTTP), VMess, Trojan, Hysteria2, TUIC, SS, and MASQUE.
package singbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
	"github.com/NetWorkClub/NetPortal/core/router"
)

const EngineID = "proxy"

// Engine implements engine.Engine backed by the sing-box binary/core.
type Engine struct {
	mu           sync.RWMutex
	running      bool
	cmd          *exec.Cmd
	cfgPath      string
	activeNodeID string
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
		SupportsTUN:        true,
		SupportsSOCKS:      true,
		SupportsTestDialer: false,
		RequiresElevation:  true,
	}
}

func (e *Engine) Validate(cfg any) error {
	switch v := cfg.(type) {
	case api.NodeItem:
		if strings.TrimSpace(v.Endpoint) == "" {
			return errors.New("singbox: node endpoint is required")
		}
		return nil
	case api.RoutingProfile:
		return router.ValidateProfile(v)
	default:
		return fmt.Errorf("singbox: unsupported config type %T", cfg)
	}
}

func (e *Engine) Start(ctx context.Context, runtime engine.EngineRuntime) error {
	binPath := runtime.Node.Params["binaryPath"]
	if binPath == "" {
		binPath = "sing-box"
	} else {
		_ = os.Chmod(binPath, 0o755)
	}

	prof := router.NewEmptyProfile("runtime", "Активный", "Active", "node:"+runtime.Node.ID)
	compiled, err := router.CompileSingBoxConfig(prof, []api.NodeItem{runtime.Node}, nil)
	if err != nil {
		return err
	}

	tmpDir := os.TempDir()
	cfgPath := filepath.Join(tmpDir, fmt.Sprintf("netportal-singbox-%s.json", runtime.Node.ID))
	if err := os.WriteFile(cfgPath, compiled, 0o600); err != nil {
		return fmt.Errorf("singbox: write config: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		_ = os.Remove(cfgPath)
		return engine.ErrAlreadyRunning
	}

	cmd := exec.CommandContext(ctx, binPath, "run", "-c", cfgPath)
	if err := cmd.Start(); err != nil {
		_ = os.Remove(cfgPath)
		e.lastErr = err.Error()
		return fmt.Errorf("singbox: start: %w", err)
	}

	e.cmd = cmd
	e.cfgPath = cfgPath
	e.activeNodeID = runtime.Node.ID
	e.running = true
	e.lastErr = ""

	go func() {
		err := cmd.Wait()
		e.mu.Lock()
		if e.cmd == cmd {
			e.running = false
			e.cmd = nil
			e.activeNodeID = ""
			_ = os.Remove(e.cfgPath)
			e.cfgPath = ""
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
	cfgPath := e.cfgPath
	e.cmd = nil
	e.cfgPath = ""
	e.activeNodeID = ""
	e.running = false
	e.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if cfgPath != "" {
		_ = os.Remove(cfgPath)
	}
	return nil
}

func (e *Engine) Health(_ context.Context) engine.EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	st := api.StatusDisconnected
	if e.running {
		st = api.StatusConnected
	}
	return engine.EngineStatus{
		ID:           EngineID,
		State:        st,
		ActiveNodeID: e.activeNodeID,
		LastError:    e.lastErr,
	}
}

func (e *Engine) TestDialer(_ api.NodeItem) (engine.Dialer, error) {
	return nil, errors.New("singbox: use core/probe for pre-connect testing")
}
