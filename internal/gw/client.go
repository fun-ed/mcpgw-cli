package gw

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ClientName    = "agwctl"
	ClientVersion = "0.2.0"
)

// ErrConnect marks failures during transport setup or the initialize handshake.
var ErrConnect = fmt.Errorf("gateway connect failed")

type Client struct {
	session *mcp.ClientSession
	cancel  context.CancelFunc
}

// Connect opens a stateful streamable HTTP session at url. The timeout
// bounds the initialize handshake; the context also drives the session
// lifetime, so Close must be called to release it.
func Connect(ctx context.Context, url string, timeout time.Duration) (*Client, error) {
	tr := &mcp.StreamableClientTransport{
		Endpoint: url,
		// A CLI only speaks request/response; the standalone SSE GET stream
		// costs seconds against the gateway's 8-target fanout.
		DisableStandaloneSSE: true,
	}
	return ConnectTransport(ctx, tr, timeout)
}

// ConnectTransport is the testable core of Connect.
func ConnectTransport(ctx context.Context, tr mcp.Transport, timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	client := mcp.NewClient(&mcp.Implementation{Name: ClientName, Version: ClientVersion}, nil)
	session, err := client.Connect(ctx, tr, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%w: %v", ErrConnect, err)
	}
	return &Client{session: session, cancel: cancel}, nil
}

// ProtocolVersion reports the version the gateway agreed to in initialize.
func (c *Client) ProtocolVersion() string {
	if r := c.session.InitializeResult(); r != nil {
		return r.ProtocolVersion
	}
	return ""
}

// ListTools returns all tools, following pagination cursors.
func (c *Client) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	var tools []*mcp.Tool
	var cursor string
	for {
		res, err := c.session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		tools = append(tools, res.Tools...)
		if res.NextCursor == "" {
			return tools, nil
		}
		cursor = res.NextCursor
	}
}

// CallTool invokes one tool. args must marshal to a JSON object.
func (c *Client) CallTool(ctx context.Context, name string, args any) (*mcp.CallToolResult, error) {
	return c.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
}

func (c *Client) Close() error {
	err := c.session.Close()
	c.cancel()
	return err
}