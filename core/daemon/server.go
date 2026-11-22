// Package daemon implements the privileged IPC server and client for NetPortal.
package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/NetWorkClub/NetPortal/core/api"
	"github.com/NetWorkClub/NetPortal/core/engine"
)

// ConnectRequest carries the target node and runtime options for daemon Connect.
type ConnectRequest struct {
	Payload   api.ConnectPayload `json:"payload"`
	Node      api.NodeItem       `json:"node"`
	EngineID  string             `json:"engineId"`
	Interface string             `json:"interface,omitempty"`
}

// Server handles JSON-line RPC calls over a local IPC listener.
type Server struct {
	registry *engine.Registry

	mu           sync.Mutex
	activeEngine engine.Engine
	activeNodeID string
	targetName   string
	startedAt    int64
	lastErr      string

	onShutdown func()
}

// NewServer constructs a daemon RPC server backed by the given engine registry.
func NewServer(registry *engine.Registry, onShutdown func()) *Server {
	return &Server{
		registry:   registry,
		onShutdown: onShutdown,
	}
}

// Serve accepts connections on ln until ctx is cancelled or ln is closed.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handleConn(ctx, conn)
	}
}

// StopActive gracefully stops any running protocol engine.
func (s *Server) StopActive(ctx context.Context) error {
	s.mu.Lock()
	eng := s.activeEngine
	s.activeEngine = nil
	s.activeNodeID = ""
	s.targetName = ""
	s.startedAt = 0
	s.mu.Unlock()

	if eng != nil {
		return eng.Stop(ctx)
	}
	return nil
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(conn)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req api.RPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(api.RPCResponse{
				Error: &api.RPCError{Code: "bad_request", Message: err.Error()},
			})
			continue
		}
		resp := s.dispatch(ctx, req)
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(ctx context.Context, req api.RPCRequest) api.RPCResponse {
	switch req.Method {
	case "ping":
		return api.RPCResponse{ID: req.ID, Result: "pong"}
	case "status":
		return api.RPCResponse{ID: req.ID, Result: s.statusSnapshot(ctx)}
	case "connect":
		var cr ConnectRequest
		if err := decodeParams(req.Params, &cr); err != nil {
			return errResp(req.ID, "invalid_params", err)
		}
		if err := s.connect(ctx, cr); err != nil {
			return errResp(req.ID, "connect_failed", err)
		}
		return api.RPCResponse{ID: req.ID, Result: s.statusSnapshot(ctx)}
	case "disconnect":
		if err := s.StopActive(ctx); err != nil {
			return errResp(req.ID, "disconnect_failed", err)
		}
		return api.RPCResponse{ID: req.ID, Result: s.statusSnapshot(ctx)}
	case "probe":
		var node api.NodeItem
		if err := decodeParams(req.Params, &node); err != nil {
			return errResp(req.ID, "invalid_params", err)
		}
		return api.RPCResponse{ID: req.ID, Result: probeTCP(ctx, node)}
	case "shutdown":
		_ = s.StopActive(ctx)
		if s.onShutdown != nil {
			go s.onShutdown()
		}
		return api.RPCResponse{ID: req.ID, Result: "shutting_down"}
	default:
		return api.RPCResponse{
			ID:    req.ID,
			Error: &api.RPCError{Code: "method_not_found", Message: fmt.Sprintf("unknown method %q", req.Method)},
		}
	}
}

func (s *Server) connect(ctx context.Context, cr ConnectRequest) error {
	engineID := cr.EngineID
	if engineID == "" {
		engineID = string(cr.Node.ProtoGroup)
	}
	eng, err := s.registry.Get(engineID)
	if err != nil {
		return err
	}
	if err := eng.Validate(cr.Node); err != nil {
		return err
	}

	if err := s.StopActive(ctx); err != nil {
		return err
	}

	runtime := engine.EngineRuntime{
		Node:      cr.Node,
		Interface: cr.Interface,
	}
	if err := eng.Start(ctx, runtime); err != nil {
		s.mu.Lock()
		s.lastErr = err.Error()
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.activeEngine = eng
	s.activeNodeID = cr.Node.ID
	s.targetName = cr.Payload.TargetName
	s.startedAt = time.Now().Unix()
	s.lastErr = ""
	s.mu.Unlock()
	return nil
}

func (s *Server) statusSnapshot(ctx context.Context) api.StatusSnapshot {
	s.mu.Lock()
	eng := s.activeEngine
	nodeID := s.activeNodeID
	targetName := s.targetName
	startedAt := s.startedAt
	lastErr := s.lastErr
	s.mu.Unlock()

	if eng == nil {
		return api.StatusSnapshot{
			Status:    api.StatusDisconnected,
			LastError: lastErr,
		}
	}
	st := eng.Health(ctx)
	return api.StatusSnapshot{
		Status:     st.State,
		EngineID:   st.ID,
		NodeID:     nodeID,
		TargetName: targetName,
		Interface:  st.InterfaceName,
		LastError:  st.LastError,
		StartedAt:  startedAt,
	}
}

func probeTCP(ctx context.Context, node api.NodeItem) api.ProbeOutcome {
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
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return api.ProbeOutcome{
			NodeID:  node.ID,
			TcpMs:   -1,
			AntiBot: api.AntiBotError,
			Error:   err.Error(),
		}
	}
	_ = conn.Close()
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

func decodeParams(raw any, dst any) error {
	if raw == nil {
		return errors.New("missing params")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

func errResp(id, code string, err error) api.RPCResponse {
	return api.RPCResponse{
		ID:    id,
		Error: &api.RPCError{Code: code, Message: err.Error()},
	}
}
