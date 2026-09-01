package cli

import "testing"

func TestParseArgValue(t *testing.T) {
	if v := parseArgValue("5"); v != float64(5) {
		t.Fatalf("number: %#v", v)
	}
	if v := parseArgValue("true"); v != true {
		t.Fatalf("bool: %#v", v)
	}
	if v := parseArgValue(`"x"`); v != "x" {
		t.Fatalf("json string: %#v", v)
	}
	if v := parseArgValue("hello world"); v != "hello world" {
		t.Fatalf("plain string: %#v", v)
	}
}

func TestBuildArgsMutuallyExclusive(t *testing.T) {
	if _, err := buildArgs([]string{"a=1"}, `{"b":2}`); err == nil {
		t.Fatal("--arg and --json must be exclusive")
	}
	if _, err := buildArgs([]string{"noequals"}, ""); err == nil {
		t.Fatal("missing = must fail")
	}
	if _, err := buildArgs([]string{"a=1"}, "@/nonexistent.json"); err == nil {
		t.Fatal("missing file must fail")
	}
	v, err := buildArgs([]string{"q=web search", "n=3"}, "")
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["q"] != "web search" || m["n"] != float64(3) {
		t.Fatalf("args = %#v", m)
	}
	if _, err := buildArgs(nil, ""); err != nil {
		t.Fatalf("no args should be empty object, got %v", err)
	}
}
