package out

import (
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	if got := Truncate("short", 10); got != "short" {
		t.Fatalf("got %q", got)
	}
	got := Truncate(strings.Repeat("x", 30), 10)
	if !strings.HasPrefix(got, strings.Repeat("x", 10)) {
		t.Fatalf("missing body: %q", got)
	}
	if !strings.Contains(got, "20 chars omitted") {
		t.Fatalf("missing marker: %q", got)
	}
	if Truncate("abc", 0) != "abc" {
		t.Fatal("0 must disable")
	}
}

func TestFirstLine(t *testing.T) {
	if got := FirstLine("a\nb\n"); got != "a" {
		t.Fatalf("got %q", got)
	}
}
