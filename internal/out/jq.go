package out

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// EvalJQ runs expr against data (a decoded JSON value) and returns the
// produced values. Errors carry the jq syntax or runtime message.
func EvalJQ(data any, expr string) ([]any, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("--jq parse: %w", err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("--jq compile: %w", err)
	}
	var vals []any
	iter := code.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return nil, fmt.Errorf("--jq run: %w", err)
		}
		vals = append(vals, v)
	}
	return vals, nil
}

// PrintValue renders one jq output like `jq -r`: strings raw, everything
// else as compact JSON.
func PrintValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Sprintf("%v", v)
	}
	return strings_trimEOL(buf.String())
}

func strings_trimEOL(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}
