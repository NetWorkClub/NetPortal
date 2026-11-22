// Package engine defines the unified lifecycle contract, capabilities,
// runtime hooks, and conflict guard shared by all NetPortal protocol engines.
package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/NetWorkClub/NetPortal/core/api"
)

var (
	ErrEngineNotFound = errors.New("engine: not found")
	ErrAlreadyRunning = errors.New("engine: already running")
	ErrNotRunning     = errors.New("engine: not running")
)

// Dialer creates outbound network connections without mutating system TUN/proxy state.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// EngineCapabilities declares what a protocol engine supports and requires.
type EngineCapabilities struct {
	SupportsTUN        bool
	SupportsSOCKS      bool
	SupportsTestDialer bool
	RequiresElevation  bool
}

// EngineRuntime passes platform-specific handles and hooks into Engine.Start.
type EngineRuntime struct {
	Node         api.NodeItem
	Interface    string
	TunFD        int
	SocksPort    int
	DNSEndpoints []string
	ProtectFD    func(fd int) bool
	OnLog        func(entry api.LogEntry)
}

// EngineStatus reports live runtime health and byte counters.
type EngineStatus struct {
	ID            string               `json:"id"`
	State         api.ConnectionStatus `json:"state"`
	ActiveNodeID  string               `json:"activeNodeId,omitempty"`
	InterfaceName string               `json:"interfaceName,omitempty"`
	RxBytes       uint64               `json:"rxBytes"`
	TxBytes       uint64               `json:"txBytes"`
	LastError     string               `json:"lastError,omitempty"`
}

// Engine is the uniform contract implemented by every NetPortal protocol/bypass engine.
type Engine interface {
	ID() string
	Capabilities() EngineCapabilities
	Validate(cfg any) error
	Start(ctx context.Context, runtime EngineRuntime) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) EngineStatus
	TestDialer(node api.NodeItem) (Dialer, error)
}

// Registry stores available protocol engines keyed by Engine.ID().
type Registry struct {
	mu      sync.RWMutex
	engines map[string]Engine
}

// NewRegistry creates an empty engine registry.
func NewRegistry() *Registry {
	return &Registry{
		engines: make(map[string]Engine),
	}
}

// Register adds or replaces an engine in the registry.
func (r *Registry) Register(eng Engine) error {
	if eng == nil || eng.ID() == "" {
		return errors.New("engine: invalid engine registration")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.engines[eng.ID()] = eng
	return nil
}

// Get retrieves an engine by ID.
func (r *Registry) Get(id string) (Engine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	eng, ok := r.engines[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrEngineNotFound, id)
	}
	return eng, nil
}

// ListIDs returns all registered engine IDs.
func (r *Registry) ListIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.engines))
	for id := range r.engines {
		out = append(out, id)
	}
	return out
}

// DetectConflicts inspects active environment/layer conflicts before starting an engine.
func DetectConflicts(serviceMode api.ServiceMode, hostsActive bool) []api.ConflictReport {
	var reports []api.ConflictReport

	if serviceMode == api.ServiceModeVPN && hostsActive {
		reports = append(reports, api.ConflictReport{
			ID:     "conflict-tun-hosts",
			Title:  "TUN + Smart Hosts overlap",
			Detail: "Active Smart Hosts entries may override DNS resolution inside the VPN tunnel.",
			Actions: []api.ConflictAction{
				{Label: "Disable Smart Hosts", Action: "disable_hosts", Primary: true},
				{Label: "Keep both", Action: "ignore", Primary: false},
			},
		})
	}

	for _, envKey := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		if val := os.Getenv(envKey); val != "" && serviceMode == api.ServiceModeVPN {
			reports = append(reports, api.ConflictReport{
				ID:     "conflict-env-proxy-" + envKey,
				Title:  "External proxy environment variable detected",
				Detail: fmt.Sprintf("%s=%s may bypass or conflict with TUN routing.", envKey, val),
				Actions: []api.ConflictAction{
					{Label: "Unset for child processes", Action: "unset_env_proxy", Primary: true},
					{Label: "Ignore", Action: "ignore", Primary: false},
				},
			})
		}
	}

	return reports
}
