package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCallOutputModes(t *testing.T) {
	body := `{"message":"` + strings.Repeat(`中文\"memory`, 2500) + `"}`
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	for _, isError := range []bool{false, true} {
		name := "test_json"
		if isError {
			name = "test_error"
		}
		server.AddTool(&mcp.Tool{Name: name, InputSchema: map[string]any{"type": "object"}},
			func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: body}}, IsError: isError,
				}, nil
			})
	}
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil))
	defer httpServer.Close()
	previousOpts := opts
	opts.url, opts.timeout = httpServer.URL, 5*time.Second
	defer func() { opts = previousOpts }()

	for _, test := range []struct {
		name, tool, jq string
		truncated      bool
		toolError      bool
	}{
		{"default-json-text", "test_json", "", true, false},
		{"raw-json", "test_json", ".", false, false},
		{"jsonl", "test_json", ".content[], .content[]", false, false},
		{"tool-error-skips-jq", "test_error", ".", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "stdout-")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			previousStdout := os.Stdout
			os.Stdout = file
			err = runCall(context.Background(), test.tool, nil, "", test.jq, 20000, "")
			os.Stdout = previousStdout
			if test.toolError {
				if !errors.Is(err, errToolResult) {
					t.Fatalf("expected tool error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(file.Name())
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(data), "[agwctl: truncated,"); got != test.truncated {
				t.Fatalf("truncated = %v, want %v", got, test.truncated)
			}
			if !test.truncated {
				if len(data) <= 20000 {
					t.Fatal("fixture did not exceed the default cap")
				}
				for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
					if !json.Valid([]byte(line)) {
						t.Fatal("jq output is not valid JSON/JSONL")
					}
				}
			}
		})
	}
}
