package cli

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRowsMinimalShape(t *testing.T) {
	tools := []*mcp.Tool{
		{Name: "deepwiki_ask_question", Description: "Ask any question about a repo\nsecond line"},
		{Name: "weird", Description: "no target prefix"},
	}
	rows := Rows(tools)
	if rows[0].Name != "deepwiki_ask_question" || rows[0].Target != "deepwiki" {
		t.Fatalf("row0 = %+v", rows[0])
	}
	if rows[0].Description != "Ask any question about a repo" {
		t.Fatalf("description should be first line, got %q", rows[0].Description)
	}
	if rows[1].Target != "" {
		t.Fatalf("unprefixed tool must have empty target, got %q", rows[1].Target)
	}
}

func TestTruncAndFirstLine(t *testing.T) {
	if got := Trunc("0123456789", 4); got != "0123" {
		t.Fatalf("Trunc = %q", got)
	}
	if got := FirstLine("a\nb"); got != "a" {
		t.Fatalf("FirstLine = %q", got)
	}
}