// Package zapret manages winws (Windows WinDivert) and nfqws + BOM-free nftables
// (Linux NFQUEUE) DPI bypass orchestration.
package zapret

import (
	"bytes"
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

const EngineID = "zapret"

// GenerateNFTRules produces an idempotent, strictly BOM-free nftables ruleset
// for Linux nfqws packet queueing.
func GenerateNFTRules(queueNum int, tcpPorts, udpPorts []int) string {
	if queueNum <= 0 {
		queueNum = 200
	}
	if len(tcpPorts) == 0 {
		tcpPorts = []int{80, 443}
	}
	if len(udpPorts) == 0 {
		udpPorts = []int{443}
	}

	var b strings.Builder
	b.WriteString("add table inet netportal\n")
	b.WriteString("flush table inet netportal\n")
	b.WriteString("table inet netportal {\n")
	b.WriteString("  chain postrouting {\n")
	b.WriteString("    type filter hook postrouting priority mangle; policy accept;\n")
	b.WriteString(fmt.Sprintf("    tcp dport { %s } ct original packets 1-6 queue num %d bypass\n", joinInts(tcpPorts), queueNum))
	b.WriteString(fmt.Sprintf("    udp dport { %s } ct original packets 1-4 queue num %d bypass\n", joinInts(udpPorts), queueNum))
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String()
}

// WriteNFTRulesBOMFree writes the nftables ruleset in strict UTF-8 without BOM.
func WriteNFTRulesBOMFree(path string, rules string) error {
	clean := bytes.TrimPrefix([]byte(rules), []byte{0xEF, 0xBB, 0xBF})
	return os.WriteFile(path, clean, 0o600)
}

// BuildStrategyArgs builds winws / nfqws CLI arguments from a ZapretStrategy.
func BuildStrategyArgs(s api.ZapretStrategy) []string {
	var args []string
	if strings.TrimSpace(s.YT) != "" {
		args = append(args, strings.Fields(s.YT)...)
	}
	if strings.TrimSpace(s.QUIC) != "" {
		args = append(args, "--new")
		args = append(args, strings.Fields(s.QUIC)...)
	}
	if strings.TrimSpace(s.Discord) != "" {
		args = append(args, "--new")
		args = append(args, strings.Fields(s.Discord)...)
	}
	return args
}

// Engine implements engine.Engine for Zapret (winws / nfqws).
type Engine struct {
	mu      sync.RWMutex
	running bool
	cmd     *exec.Cmd
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
		SupportsSOCKS:      false,
		SupportsTestDialer: false,
		RequiresElevation:  true,
	}
}

func (e *Engine) Validate(cfg any) error {
	s, ok := cfg.(api.ZapretStrategy)
	if !ok {
		return fmt.Errorf("zapret: expected api.ZapretStrategy, got %T", cfg)
	}
	if len(BuildStrategyArgs(s)) == 0 {
		return errors.New("zapret: strategy must contain at least one rule")
	}
	return nil
}

func (e *Engine) Start(ctx context.Context, runtime engine.EngineRuntime) error {
	binPath := runtime.Node.Params["binaryPath"]
	if binPath == "" {
		return errors.New("zapret: binaryPath parameter is required")
	}
	strat := api.ZapretStrategy{
		ID:      runtime.Node.ID,
		Name:    runtime.Node.Name,
		YT:      runtime.Node.Params["yt"],
		QUIC:    runtime.Node.Params["quic"],
		Discord: runtime.Node.Params["discord"],
	}
	args := BuildStrategyArgs(strat)
	if len(args) == 0 {
		return errors.New("zapret: empty strategy arguments")
	}
	_ = os.Chmod(binPath, 0o755)

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return engine.ErrAlreadyRunning
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Start(); err != nil {
		e.lastErr = err.Error()
		return fmt.Errorf("zapret: start %s: %w", binPath, err)
	}
	e.cmd = cmd
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
	if e.running {
		st = api.StatusConnected
	}
	return engine.EngineStatus{
		ID:        EngineID,
		State:     st,
		LastError: e.lastErr,
	}
}

func (e *Engine) TestDialer(_ api.NodeItem) (engine.Dialer, error) {
	return nil, errors.New("zapret: TestDialer not supported")
}

func joinInts(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}
