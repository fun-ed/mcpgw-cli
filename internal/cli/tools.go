package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"agwctl/internal/gw"
)

// gwErrConnect aliases the gateway connect marker so root.go can map exit
// codes without importing gw here twice.
var gwErrConnect = gw.ErrConnect

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "List, search and describe gateway tools",
	}
	cmd.AddCommand(newToolsListCmd())
	return cmd
}

func newToolsListCmd() *cobra.Command {
	var (
		asJSON  bool
		target  string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "All tools as name + one-line description (no schema)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), asJSON, target)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "minimal JSON output")
	cmd.Flags().StringVar(&target, "target", "", "only tools of this target")
	return cmd
}

func runList(ctx context.Context, asJSON bool, target string) error {
	c, err := gw.Connect(ctx, opts.url, opts.timeout)
	if err != nil {
		return err
	}
	defer c.Close()

	tools, err := c.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("%w: tools/list: %v", errProtocol, err)
	}
	slog.Debug("gateway session ready", "protocol", c.ProtocolVersion(), "tools", len(tools))

	rows := Rows(tools)
	if opts.expectTargets > 0 {
		if got := countTargets(rows); got < opts.expectTargets {
			slog.Warn("fewer targets than expected; a failOpen target may be down",
				"got", got, "want", opts.expectTargets)
		}
	}
	if target != "" {
		rows = filterTarget(rows, target)
	}

	out := os.Stdout
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		return enc.Encode(rows)
	}
	for _, r := range rows {
		fmt.Fprintf(out, "%s\t%s\n", r.Name, r.Description)
	}
	return nil
}

func filterTarget(rows []ToolRow, target string) []ToolRow {
	kept := rows[:0:0]
	for _, r := range rows {
		if r.Target == target {
			kept = append(kept, r)
		}
	}
	return kept
}

func countTargets(rows []ToolRow) int {
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Target != "" {
			seen[r.Target] = true
		}
	}
	return len(seen)
}