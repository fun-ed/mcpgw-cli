package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"agwctl/internal/gw"
	"agwctl/internal/search"
)

var gwErrConnect = gw.ErrConnect

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "List, search and describe gateway tools",
	}
	cmd.AddCommand(newToolsListCmd())
	cmd.AddCommand(newToolsSearchCmd())
	cmd.AddCommand(newToolsDescribeCmd())
	return cmd
}

func newToolsListCmd() *cobra.Command {
	var (
		asJSON bool
		target string
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

func newToolsSearchCmd() *cobra.Command {
	var (
		asJSON bool
		limit  int
	)
	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Lexical tool search, top 5 by default",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd.Context(), asJSON, limit, args[0])
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "minimal JSON output")
	cmd.Flags().IntVar(&limit, "limit", 5, "max results")
	return cmd
}

func newToolsDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe NAME",
		Short: "Name, description and inputSchema of one tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDescribe(cmd.Context(), args[0])
		},
	}
	return cmd
}

// listRows returns name/description rows, honoring --refresh and the cache.
// The schema is never cached, so describe always goes live.
func listRows(ctx context.Context) ([]gw.ToolRow, error) {
	if cache, err := gw.NewToolCache(opts.url); err == nil && !opts.refresh {
		if rows, ok, err := cache.Load(time.Now()); err == nil && ok {
			slog.Debug("tool list from cache", "rows", len(rows))
			return rows, nil
		}
	}
	c, err := gw.Connect(ctx, opts.url, opts.timeout)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	tools, err := c.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tools/list: %v", errProtocol, err)
	}
	rows := gw.Rows(tools)
	if cache, err := gw.NewToolCache(opts.url); err == nil {
		if err := cache.Store(rows, time.Now()); err != nil {
			slog.Debug("cache store failed", "err", err)
		}
	}
	return rows, nil
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

	rows := gw.Rows(tools)
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

func runSearch(ctx context.Context, asJSON bool, limit int, query string) error {
	rows, err := listRows(ctx)
	if err != nil {
		return err
	}
	ranked := search.Search(rows, query, limit)

	out := os.Stdout
	if asJSON {
		type hit struct {
			Name   string  `json:"name"`
			Target string  `json:"target"`
			Score  float64 `json:"score"`
		}
		hits := make([]hit, 0, len(ranked))
		for _, r := range ranked {
			hits = append(hits, hit{Name: r.Row.Name, Target: r.Row.Target, Score: r.Score})
		}
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		return enc.Encode(hits)
	}
	for _, r := range ranked {
		fmt.Fprintf(out, "%0.2f\t%s\t%s\n", r.Score, r.Row.Name, r.Row.Description)
	}
	return nil
}

func runDescribe(ctx context.Context, name string) error {
	c, err := gw.Connect(ctx, opts.url, opts.timeout)
	if err != nil {
		return err
	}
	defer c.Close()

	tools, err := c.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("%w: tools/list: %v", errProtocol, err)
	}
	for _, t := range tools {
		if t.Name != name {
			continue
		}
		payload := map[string]any{
			"name":        t.Name,
			"target":      targetOf(t.Name),
			"description": t.Description,
			"inputSchema": t.InputSchema,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	fmt.Fprintf(os.Stderr, "agwctl: tool %q not found; run: agwctl tools search\n", name)
	return errToolResult
}

func filterTarget(rows []gw.ToolRow, target string) []gw.ToolRow {
	kept := make([]gw.ToolRow, 0, len(rows))
	for _, r := range rows {
		if r.Target == target {
			kept = append(kept, r)
		}
	}
	return kept
}

func countTargets(rows []gw.ToolRow) int {
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Target != "" {
			seen[r.Target] = true
		}
	}
	return len(seen)
}

func targetOf(name string) string {
	if before, _, found := strings.Cut(name, "_"); found {
		return before
	}
	return ""
}
