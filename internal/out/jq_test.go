package out

import (
	"encoding/json"
	"testing"
)

func TestEvalJQField(t *testing.T) {
	var data any
	json.Unmarshal([]byte(`{"content":[{"type":"text","text":"hi"}]}`), &data)
	vals, err := EvalJQ(data, ".content[0].text")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || PrintValue(vals[0]) != "hi" {
		t.Fatalf("got %v", vals)
	}
}

func TestEvalJQParseError(t *testing.T) {
	if _, err := EvalJQ(map[string]any{}, ".["); err == nil {
		t.Fatal("want parse error")
	}
}
