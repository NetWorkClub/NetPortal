// Package mobile provides a gomobile-compatible facade over NetPortal core
// for Android (.aar) and iOS (.xcframework) native shells.
//
// All exported symbols strictly use gomobile-safe primitive types and simple
// callback interfaces, exchanging structured DTOs as JSON strings matching core/api.
package mobile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
	"github.com/NetWorkClub/NetPortal/core/engine/awg"
	"github.com/NetWorkClub/NetPortal/core/parser"
	"github.com/NetWorkClub/NetPortal/core/storage"
)

// PlatformBridge is implemented by Android Kotlin (VpnService) or iOS Swift
// to protect outbound sockets and receive core lifecycle events.
type PlatformBridge interface {
	ProtectFD(fd int) bool
	OnEvent(eventJSON string)
}

var (
	mu     sync.Mutex
	store  *storage.Store
	bridge PlatformBridge
	awgEng *awg.Engine
)

// Init initializes local storage inside dataDir and registers the native platform bridge.
func Init(dataDir string, platformBridge PlatformBridge) error {
	if dataDir == "" {
		return errors.New("mobile: dataDir must not be empty")
	}
	st, err := storage.Open(filepath.Join(dataDir, "netportal-state.json"))
	if err != nil {
		return err
	}
	mu.Lock()
	store = st
	bridge = platformBridge
	if awgEng == nil {
		awgEng = awg.New()
	}
	mu.Unlock()
	return nil
}

// ImportConfig parses a WireGuard/AWG .conf or vpn:// URI and returns api.ImportResponse JSON.
func ImportConfig(name, content string) (string, error) {
	mu.Lock()
	st := store
	mu.Unlock()
	if st == nil {
		return "", errors.New("mobile: core not initialized")
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", errors.New("mobile: empty config content")
	}

	confText := trimmed
	if strings.HasPrefix(trimmed, "vpn://") {
		decodedName, decodedConf, err := parser.DecodeVPNURI(trimmed)
		if err != nil {
			return "", err
		}
		if name == "" {
			name = decodedName
		}
		confText = decodedConf
	}

	cfg, err := parser.ParseAWGConf(confText)
	if err != nil {
		return "", err
	}

	ts := time.Now().UnixNano()
	srcID := fmt.Sprintf("src-%d", ts)
	nodeID := fmt.Sprintf("node-%d", ts)
	if name == "" {
		name = string(cfg.DetectVersion())
	}

	src := api.SourceItem{
		ID:   srcID,
		Name: name,
		Meta: string(cfg.DetectVersion()),
	}
	node := cfg.ToNodeItem(srcID, nodeID, name, confText)

	if err := st.UpsertSource(src); err != nil {
		return "", err
	}
	if err := st.UpsertNodes([]api.NodeItem{node}); err != nil {
		return "", err
	}

	b, err := json.Marshal(api.ImportResponse{
		Source: src,
		Nodes:  []api.NodeItem{node},
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ListNodesJSON returns all stored nodes encoded as JSON.
func ListNodesJSON() string {
	mu.Lock()
	st := store
	mu.Unlock()
	if st == nil {
		return "[]"
	}
	b, err := json.Marshal(st.ListNodes())
	if err != nil {
		return "[]"
	}
	return string(b)
}

// ProbeNodeJSON runs a pre-connect TCP probe and returns api.ProbeOutcome JSON.
func ProbeNodeJSON(nodeJSON, kind string) (string, error) {
	var node api.NodeItem
	if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
		return "", err
	}
	endpoint := strings.TrimSpace(node.Endpoint)
	if endpoint == "" {
		return marshalProbe(api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   "empty endpoint",
		})
	}

	start := time.Now()
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(context.Background(), "tcp", endpoint)
	if err != nil {
		return marshalProbe(api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   err.Error(),
		})
	}
	_ = conn.Close()
	ms := int(time.Since(start).Milliseconds())
	if ms < 1 {
		ms = 1
	}
	out := api.ProbeOutcome{
		NodeID:  node.ID,
		TcpMs:   ms,
		AntiBot: api.AntiBotClean,
	}
	if kind == "url" {
		out.UrlMs = &ms
	}
	return marshalProbe(out)
}

func marshalProbe(out api.ProbeOutcome) (string, error) {
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// StartAWG starts the AmneziaWG engine bound to an Android VpnService TUN file descriptor.
func StartAWG(nodeID string, tunFD int) error {
	mu.Lock()
	st := store
	eng := awgEng
	br := bridge
	mu.Unlock()
	if st == nil || eng == nil {
		return errors.New("mobile: core not initialized")
	}
	node, ok := st.GetNode(nodeID)
	if !ok {
		return fmt.Errorf("mobile: node %q not found", nodeID)
	}

	var protectFn func(int) bool
	if br != nil {
		protectFn = br.ProtectFD
	}
	return eng.Start(context.Background(), engine.EngineRuntime{
		Node:      node,
		TunFD:     tunFD,
		ProtectFD: protectFn,
	})
}

// Stop stops the active mobile tunnel.
func Stop() error {
	mu.Lock()
	eng := awgEng
	mu.Unlock()
	if eng == nil {
		return nil
	}
	return eng.Stop(context.Background())
}

// StatusJSON returns api.StatusSnapshot encoded as JSON.
func StatusJSON() string {
	mu.Lock()
	eng := awgEng
	mu.Unlock()
	if eng == nil {
		b, _ := json.Marshal(api.StatusSnapshot{Status: api.StatusDisconnected})
		return string(b)
	}
	st := eng.Health(context.Background())
	b, _ := json.Marshal(api.StatusSnapshot{
		Status:     st.State,
		EngineID:   st.ID,
		NodeID:     st.ActiveNodeID,
		Interface:  st.InterfaceName,
		LastError:  st.LastError,
	})
	return string(b)
}
