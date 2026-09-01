package out

import (
	"fmt"
	"strings"
)

// TruncMarker is appended when stdout output is cut at --max-chars. Scripts
// and agents should treat it as a signal to narrow with --jq instead.
func TruncMarker(omitted int) string {
	return fmt.Sprintf("[agwctl: truncated, %d chars omitted; rerun with --jq or --max-chars 0]", omitted)
}

// Truncate cuts s at n chars and appends the marker. n <= 0 means no limit.
func Truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	omitted := len(s) - n
	return s[:n] + "\n" + TruncMarker(omitted)
}

func FirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
