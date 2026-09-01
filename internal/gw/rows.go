package gw

import (
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const descCap = 100

// ToolRow is the minimal per-tool record shared by list, search and describe.
type ToolRow struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	Description string `json:"description"`
}

// Rows converts gateway tools into output records. The gateway prefixes tool
// names with the target name and an underscore; names without that shape keep
// an empty target.
func Rows(tools []*mcp.Tool) []ToolRow {
	rows := make([]ToolRow, 0, len(tools))
	for _, t := range tools {
		target := ""
		if before, _, found := strings.Cut(t.Name, "_"); found {
			target = before
		}
		rows = append(rows, ToolRow{
			Name:        t.Name,
			Target:      target,
			Description: strings.TrimSpace(FirstLine(t.Description)),
		})
	}
	return rows
}

func FirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}