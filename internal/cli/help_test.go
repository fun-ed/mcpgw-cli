package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMCPHelpIsOffline(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{"root", []string{"--help"}, []string{"agwctl help mcp"}},
		{"guide", []string{"help", "mcp"}, []string{"Exit 0", "operation ID", "--json", "--jq .", "10-minute"}},
		{"guide flag", []string{"mcp", "--help"}, []string{"Verify the task result", "raw JSONL"}},
		{"call", []string{"call", "--help"}, []string{"input arguments", "--json @request.json", "--stdin", "isError"}},
		{"tools", []string{"tools", "--help"}, []string{"list and describe always query the live gateway"}},
		{"list", []string{"tools", "list", "--help"}, []string{"live tools/list"}},
		{"search", []string{"tools", "search", "--help"}, []string{"not the web", "--refresh"}},
		{"describe", []string{"tools", "describe", "--help"}, []string{"current inputSchema"}},
		{"doctor", []string{"doctor", "--help"}, []string{"does not execute tools/call", "Exit 6"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := NewRoot()
			var output bytes.Buffer
			root.cmd.SetOut(&output)
			root.cmd.SetErr(&output)
			root.cmd.SetArgs(append([]string{"--url", server.URL, "--timeout", "1ms"}, tc.args...))
			if err := root.cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("help missing %q: %s", want, output.String())
				}
			}
		})
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("help sent %d gateway requests", got)
	}
}
