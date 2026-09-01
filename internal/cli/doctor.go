package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/fun-ed/mcpgw-cli/internal/gw"
)

const slowInitialize = 15 * time.Second

type doctorReport struct {
	URL          string         `json:"url"`
	Reachable    bool           `json:"reachable"`
	Protocol     string         `json:"protocol"`
	InitializeMS int64          `json:"initializeMs"`
	CloseMS      int64          `json:"closeMs"`
	Targets      map[string]int `json:"targets"`
	Tools        int            `json:"tools"`
	Warnings     []string       `json:"warnings"`
}

func newDoctorCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Gateway reachability, initialize latency, per-target tool counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd.Context(), asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "structured report")
	return cmd
}

func runDoctor(ctx context.Context, asJSON bool) error {
	rep := doctorReport{URL: opts.url, Targets: map[string]int{}}

	if err := dialGateway(opts.url, 10*time.Second); err != nil {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("gateway unreachable: %v", err))
		return finishDoctor(asJSON, rep, gwErrConnect)
	}
	rep.Reachable = true

	connectStart := time.Now()
	c, err := gw.Connect(ctx, opts.url, opts.timeout)
	if err != nil {
		rep.Warnings = append(rep.Warnings, err.Error())
		return finishDoctor(asJSON, rep, gwErrConnect)
	}
	rep.InitializeMS = time.Since(connectStart).Milliseconds()
	rep.Protocol = c.ProtocolVersion()

	tools, err := c.ListTools(ctx)
	if err != nil {
		rep.Warnings = append(rep.Warnings, err.Error())
		return finishDoctor(asJSON, rep, errProtocol)
	}
	closeStart := time.Now()
	if cerr := c.Close(); cerr != nil {
		slog.Debug("session close", "err", cerr)
	}
	rep.CloseMS = time.Since(closeStart).Milliseconds()

	for _, row := range gw.Rows(tools) {
		if row.Target != "" {
			rep.Targets[row.Target]++
		}
	}
	rep.Tools = len(tools)

	if rep.InitializeMS > slowInitialize.Milliseconds() {
		rep.Warnings = append(rep.Warnings,
			fmt.Sprintf("initialize took %dms; agents time out at 30s (cold sidecar?)", rep.InitializeMS))
	}
	var dead []string
	for t, n := range rep.Targets {
		if n == 0 {
			dead = append(dead, t)
		}
	}
	if len(dead) > 0 {
		sort.Strings(dead)
		rep.Warnings = append(rep.Warnings, "targets with zero tools: "+fmt.Sprint(dead))
	}
	mismatch := false
	if opts.expectTargets > 0 {
		if got := len(rep.Targets); got < opts.expectTargets {
			mismatch = true
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"got %d targets, want %d; a failOpen target may be down", got, opts.expectTargets))
		}
	}
	return finishDoctor(asJSON, rep, func() error {
		if mismatch {
			return errTargets
		}
		return nil
	}())
}

// dialGateway checks TCP reachability with a bounded timeout, independent of
// the MCP session so a hung fanout still reports something.
func dialGateway(raw string, timeout time.Duration) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("bad url: %w", err)
	}
	host := u.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		if u.Scheme == "https" {
			host = net.JoinHostPort(host, "443")
		} else {
			host = net.JoinHostPort(host, "80")
		}
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	slog.Debug("tcp dial", "host", host, "ms", time.Since(start).Milliseconds())
	return nil
}

func finishDoctor(asJSON bool, rep doctorReport, ret error) error {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(rep)
	} else {
		printDoctor(os.Stdout, rep)
	}
	for _, w := range rep.Warnings {
		fmt.Fprintf(os.Stderr, "agwctl: warning: %s\n", w)
	}
	return ret
}

func printDoctor(w *os.File, rep doctorReport) {
	status := "ok"
	if !rep.Reachable {
		status = "FAIL"
	}
	fmt.Fprintf(w, "gateway\t%s\t%s\n", status, rep.URL)
	if !rep.Reachable {
		return
	}
	fmt.Fprintf(w, "initialize\t%dms\tprotocol %s\n", rep.InitializeMS, rep.Protocol)
	names := make([]string, 0, len(rep.Targets))
	for t := range rep.Targets {
		names = append(names, t)
	}
	sort.Strings(names)
	fmt.Fprintf(w, "targets\t%d\t", len(rep.Targets))
	for i, t := range names {
		if i > 0 {
			fmt.Fprint(w, ", ")
		}
		fmt.Fprintf(w, "%s %d", t, rep.Targets[t])
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "tools\t%d\t\n", rep.Tools)
	fmt.Fprintf(w, "session close\t%dms\tDELETE\n", rep.CloseMS)
}
