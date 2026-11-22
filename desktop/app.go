package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/daemon"
	"github.com/NetWorkClub/NetPortal/core/diag"
	"github.com/NetWorkClub/NetPortal/core/engine"
	"github.com/NetWorkClub/NetPortal/core/engine/awg"
	"github.com/NetWorkClub/NetPortal/core/parser"
	"github.com/NetWorkClub/NetPortal/core/probe"
	"github.com/NetWorkClub/NetPortal/core/storage"
)

// App exposes NetPortal core, storage, and daemon IPC methods to Wails and --web.
type App struct {
	ctx    context.Context
	store  *storage.Store
	daemon *daemon.Client

	mu     sync.Mutex
	status api.StatusSnapshot
}

// NewApp initializes local storage and the daemon IPC client.
func NewApp(statePath, daemonEndpoint string) (*App, error) {
	if statePath == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			cfgDir = "."
		}
		statePath = filepath.Join(cfgDir, "NetPortal", "state.json")
	}
	store, err := storage.Open(statePath)
	if err != nil {
		return nil, err
	}
	return &App{
		ctx:    context.Background(),
		store:  store,
		daemon: daemon.NewClient(daemonEndpoint),
		status: api.StatusSnapshot{
			Status: api.StatusDisconnected,
		},
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	_ = a.Disconnect()
}

// EnsureDaemonRunning checks daemon reachability and triggers single-session elevation if needed.
func (a *App) EnsureDaemonRunning() error {
	if err := a.daemon.Ping(a.ctx); err == nil {
		return nil
	}
	binPath, err := FindDaemonBinary()
	if err != nil {
		return err
	}
	if err := LaunchElevatedDaemon(binPath); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := a.daemon.Ping(a.ctx); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("netportal-daemon did not respond after elevation")
}

// Connect sends a connection request to the privileged netportal-daemon.
func (a *App) Connect(payload api.ConnectPayload) error {
	if err := a.EnsureDaemonRunning(); err != nil {
		return err
	}

	if payload.Standalone {
		engineID := resolveStandaloneEngineID(payload.TargetName)
		req := daemon.ConnectRequest{
			Payload: payload,
			Node: api.NodeItem{
				ID:   engineID,
				Name: payload.TargetName,
			},
			EngineID: engineID,
		}
		snap, err := a.daemon.Connect(a.ctx, req)
		if err != nil {
			return err
		}
		a.mu.Lock()
		a.status = snap
		a.mu.Unlock()
		return nil
	}

	var node api.NodeItem
	if payload.NodeID != "" {
		if found, ok := a.store.GetNode(payload.NodeID); ok {
			node = found
		}
	}
	if node.ID == "" {
		for _, n := range a.store.ListNodes() {
			if n.Name == payload.TargetName || n.ID == payload.TargetName {
				node = n
				break
			}
		}
	}
	if node.ID == "" {
		return fmt.Errorf("node %q not found", payload.TargetName)
	}

	req := daemon.ConnectRequest{
		Payload:  payload,
		Node:     node,
		EngineID: resolveNodeEngineID(node),
	}
	snap, err := a.daemon.Connect(a.ctx, req)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.status = snap
	a.mu.Unlock()
	return nil
}

func resolveStandaloneEngineID(targetName string) string {
	switch strings.ToLower(strings.TrimSpace(targetName)) {
	case "zapret":
		return "zapret"
	case "smartdns":
		return "smartdns"
	default:
		return "byedpi"
	}
}

func resolveNodeEngineID(node api.NodeItem) string {
	switch node.ProtoGroup {
	case api.ProtoGroupAWG, api.ProtoGroupWARP:
		return "awg"
	case api.ProtoGroupProxy, api.ProtoGroupMASQUE:
		return "proxy"
	case api.ProtoGroupPsiphon:
		return "psiphon"
	default:
		return string(node.ProtoGroup)
	}
}

// Disconnect stops the active tunnel/bypass via netportal-daemon.
func (a *App) Disconnect() error {
	if err := a.daemon.Ping(a.ctx); err != nil {
		a.mu.Lock()
		a.status = api.StatusSnapshot{Status: api.StatusDisconnected}
		a.mu.Unlock()
		return nil
	}
	snap, err := a.daemon.Disconnect(a.ctx)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.status = snap
	a.mu.Unlock()
	return nil
}

// GetStatus returns live daemon status or local fallback if daemon is offline.
func (a *App) GetStatus() api.StatusSnapshot {
	if snap, err := a.daemon.Status(a.ctx); err == nil {
		a.mu.Lock()
		a.status = snap
		a.mu.Unlock()
		return snap
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

// ProbeNode runs an unprivileged pre-connect TCP or ephemeral no-TUN URL probe.
func (a *App) ProbeNode(node api.NodeItem, kind string) (api.ProbeOutcome, error) {
	if kind == "url" && node.ProtoGroup == api.ProtoGroupAWG && strings.TrimSpace(node.RawConfig) != "" {
		dialer, err := awg.New().TestDialer(node)
		if err == nil {
			if closer, ok := dialer.(interface{ Close() error }); ok {
				defer closer.Close()
			}
			return probe.ProbeURL(a.ctx, node, dialer, probe.DefaultTestURL, 6*time.Second), nil
		}
	}
	out := probe.ProbeTCP(a.ctx, node, 3*time.Second)
	if kind == "url" && out.TcpMs > 0 {
		ms := out.TcpMs
		out.UrlMs = &ms
	}
	return out, nil
}

// ImportConfig parses WireGuard/AWG .conf, vpn:// URI, or proxy subscriptions and persists them.
func (a *App) ImportConfig(req api.ImportRequest) (api.ImportResponse, error) {
	trimmed := strings.TrimSpace(req.Content)
	if trimmed == "" {
		return api.ImportResponse{}, fmt.Errorf("empty configuration content")
	}

	ts := time.Now().UnixNano()
	srcID := fmt.Sprintf("src-%d", ts)
	name := strings.TrimSpace(req.Name)

	nodes, err := parser.ParseSubscription(srcID, trimmed)
	if err != nil {
		return api.ImportResponse{}, err
	}
	if name == "" {
		name = nodes[0].Proto
	}
	if len(nodes) == 1 && strings.TrimSpace(req.Name) != "" {
		nodes[0].Name = name
	}

	src := api.SourceItem{
		ID:   srcID,
		Name: name,
		Meta: fmt.Sprintf("%d node(s)", len(nodes)),
	}

	if err := a.store.UpsertSource(src); err != nil {
		return api.ImportResponse{}, err
	}
	if err := a.store.UpsertNodes(nodes); err != nil {
		return api.ImportResponse{}, err
	}

	return api.ImportResponse{
		Source: src,
		Nodes:  nodes,
	}, nil
}

// ExportDiagnostics builds a sanitized diagnostic JSON report with all secrets redacted.
func (a *App) ExportDiagnostics() (string, error) {
	status := a.GetStatus()
	nodes := a.store.ListNodes()
	conflicts := engine.DetectConflicts(api.ServiceMode(a.store.GetSettings().ServiceMode), false)
	raw, err := diag.BuildReport(status, nodes, conflicts, nil)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (a *App) ListSources() []api.SourceItem {
	return a.store.ListSources()
}

func (a *App) ListNodes() []api.NodeItem {
	return a.store.ListNodes()
}

func (a *App) GetSettings() api.AppSettings {
	return a.store.GetSettings()
}

func (a *App) SaveSettings(settings api.AppSettings) error {
	return a.store.SaveSettings(settings)
}
