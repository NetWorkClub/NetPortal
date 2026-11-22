package api

// EventType identifies asynchronous notifications emitted by core/daemon to UI clients.
type EventType string

const (
	EventStatusChanged EventType = "status_changed"
	EventTrafficTick   EventType = "traffic_tick"
	EventLogEntry      EventType = "log_entry"
	EventProbeResult   EventType = "probe_result"
	EventConflict      EventType = "conflict_detected"
)

// Event is the envelope pushed over Wails events, WebSocket, or Android bridge.
type Event struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
}

// StatusSnapshot describes the live connection and engine state.
type StatusSnapshot struct {
	Status     ConnectionStatus `json:"status"`
	EngineID   string           `json:"engineId,omitempty"`
	NodeID     string           `json:"nodeId,omitempty"`
	TargetName string           `json:"targetName,omitempty"`
	Interface  string           `json:"interface,omitempty"`
	LastError  string           `json:"lastError,omitempty"`
	StartedAt  int64            `json:"startedAt,omitempty"`
}

// TrafficStats reports live throughput and cumulative byte counters.
type TrafficStats struct {
	RxBytesPerSec uint64 `json:"rxBytesPerSec"`
	TxBytesPerSec uint64 `json:"txBytesPerSec"`
	RxTotalBytes  uint64 `json:"rxTotalBytes"`
	TxTotalBytes  uint64 `json:"txTotalBytes"`
}

// ImportRequest submits raw config text or a subscription URL to core.
type ImportRequest struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content"`
}

// ImportResponse returns the parsed source and nodes created from an import.
type ImportResponse struct {
	Source SourceItem `json:"source"`
	Nodes  []NodeItem `json:"nodes"`
}

// RPCRequest is a generic JSON envelope for Named Pipe / UDS / WebSocket RPC.
type RPCRequest struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// RPCError represents a structured error in an RPCResponse.
type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RPCResponse is the reply envelope matching an RPCRequest ID.
type RPCResponse struct {
	ID     string    `json:"id"`
	Result any       `json:"result,omitempty"`
	Error  *RPCError `json:"error,omitempty"`
}
