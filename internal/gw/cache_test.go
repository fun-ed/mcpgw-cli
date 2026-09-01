package gw

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestToolCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := &ToolCache{path: filepath.Join(dir, "tools-x.json"), url: "http://x/mcp"}
	rows := []ToolRow{{Name: "a_b", Target: "a", Description: "d"}}
	if err := c.Store(rows, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, ok, err := c.Load(time.Now())
	if err != nil || !ok || len(got) != 1 || got[0].Name != "a_b" {
		t.Fatalf("round trip: %v %v %v", got, ok, err)
	}
}

func TestToolCacheExpiryAndURL(t *testing.T) {
	dir := t.TempDir()
	c := &ToolCache{path: filepath.Join(dir, "tools-y.json"), url: "http://y/mcp"}
	c.Store([]ToolRow{{Name: "n"}}, time.Now())
	if _, ok, _ := c.Load(time.Now().Add(11 * time.Minute)); ok {
		t.Fatal("expired cache must miss")
	}
	other := &ToolCache{path: c.path, url: "http://z/mcp"}
	if _, ok, _ := other.Load(time.Now()); ok {
		t.Fatal("different URL must miss")
	}
}

func TestToolCacheCorruptFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "tools-z.json")
	os.WriteFile(p, []byte("not json"), 0o600)
	c := &ToolCache{path: p, url: "u"}
	if _, ok, err := c.Load(time.Now()); ok || err != nil {
		t.Fatalf("corrupt cache must be a clean miss, got %v %v", ok, err)
	}
}
