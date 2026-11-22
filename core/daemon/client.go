package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/NetWorkClub/NetPortal/core/api"
)

// Client sends JSON-line RPC requests to netportal-daemon.
type Client struct {
	endpoint string
	seq      atomic.Uint64
}

// NewClient creates an IPC client targeting endpoint (or DefaultEndpoint if empty).
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint()
	}
	return &Client{endpoint: endpoint}
}

// Call executes a single RPC method and decodes the result into out if non-nil.
func (c *Client) Call(ctx context.Context, method string, params any, out any) error {
	conn, err := DialIPC(ctx, c.endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	req := api.RPCRequest{
		ID:     fmt.Sprintf("req-%d", c.seq.Add(1)),
		Method: method,
		Params: params,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if _, err := conn.Write(payload); err != nil {
		return err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("daemon: empty response for %s", method)
	}

	var resp api.RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("daemon %s (%s): %s", method, resp.Error.Code, resp.Error.Message)
	}
	if out != nil && resp.Result != nil {
		b, err := json.Marshal(resp.Result)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, out)
	}
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	var pong string
	return c.Call(ctx, "ping", nil, &pong)
}

func (c *Client) Status(ctx context.Context) (api.StatusSnapshot, error) {
	var st api.StatusSnapshot
	err := c.Call(ctx, "status", nil, &st)
	return st, err
}

func (c *Client) Connect(ctx context.Context, req ConnectRequest) (api.StatusSnapshot, error) {
	var st api.StatusSnapshot
	err := c.Call(ctx, "connect", req, &st)
	return st, err
}

func (c *Client) Disconnect(ctx context.Context) (api.StatusSnapshot, error) {
	var st api.StatusSnapshot
	err := c.Call(ctx, "disconnect", nil, &st)
	return st, err
}

func (c *Client) Probe(ctx context.Context, node api.NodeItem) (api.ProbeOutcome, error) {
	var out api.ProbeOutcome
	err := c.Call(ctx, "probe", node, &out)
	return out, err
}

func (c *Client) Shutdown(ctx context.Context) error {
	return c.Call(ctx, "shutdown", nil, nil)
}
