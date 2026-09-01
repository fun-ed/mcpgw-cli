package gw

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type anyArgs struct{}

func newTestServer(t *testing.T) (clientTransport mcp.Transport) {
	t.Helper()
	srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	srv.AddTool(&mcp.Tool{
		Name:        "alpha_ping",
		Description: "pings things\nwith more lines",
		InputSchema: map[string]any{"type": "object"},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	})
	ct, st := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(context.Background(), st, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	return ct
}

func TestConnectAndListTools(t *testing.T) {
	c, err := ConnectTransport(context.Background(), newTestServer(t), 5*time.Second)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "alpha_ping" {
		t.Fatalf("got tools %v", tools)
	}
}

func TestConnectFailureIsSentinel(t *testing.T) {
	_, err := Connect(context.Background(), "http://127.0.0.1:1/mcp", time.Second)
	if !errors.Is(err, ErrConnect) {
		t.Fatalf("want ErrConnect, got %v", err)
	}
}