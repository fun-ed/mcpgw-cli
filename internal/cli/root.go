package cli

import (
	"context"
	"errors"
	"strconv"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// Exit codes, see PLAN.md section 3.
const (
	ExitOK       = 0
	ExitToolErr  = 1
	ExitUsage    = 2
	ExitConnect  = 3
	ExitTimeout  = 4
	ExitProtocol = 5
	ExitTargets  = 6
)

// errUsage marks argument and flag problems, handled by cobra itself.
var errUsage = errors.New("usage error")

// errProtocol marks failures after a session is established.
var errProtocol = errors.New("gateway call failed")

// errToolResult marks a completed call whose result isError.
var errToolResult = errors.New("tool returned an error")

// errTargets marks a doctor --expect-targets mismatch.
var errTargets = errors.New("target count mismatch")


var opts struct {
	url           string
	timeoutRaw    string
	timeout       time.Duration
	expectTargets int
	verbose       bool
	refresh       bool
	maxChars      int
	jqExpr        string
}

func NewRoot() *cobraRoot {
	root := &cobraRoot{}
	cmd := &cobra.Command{
		Use:           "agwctl",
		Short:         "Shell client for the local agentgateway",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			d, err := parseTimeout(opts.timeoutRaw)
			if err != nil {
				return fmt.Errorf("%w: --timeout: %v", errUsage, err)
			}
			opts.timeout = d
			if opts.verbose {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
			} else {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
			}
			return nil
		},
	}
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%w: %v", errUsage, err)
	})
	cmd.PersistentFlags().StringVar(&opts.url, "url", envOr("AGWCTL_URL", "http://127.0.0.1:8083/mcp"), "gateway MCP endpoint")
	cmd.PersistentFlags().StringVar(&opts.timeoutRaw, "timeout", "120s", "initialize and call timeout; plain number means seconds")
	cmd.PersistentFlags().IntVar(&opts.expectTargets, "expect-targets", 8, "warn when fewer targets answer (0 disables)")
	cmd.PersistentFlags().BoolVar(&opts.refresh, "refresh", false, "bypass the tool-list cache and refetch")
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "log protocol details to stderr")

	cmd.AddCommand(newToolsCmd())
	cmd.AddCommand(newCallCmd())
	cmd.AddCommand(newDoctorCmd())
	root.cmd = cmd
	return root
}

type cobraRoot struct{ cmd *cobra.Command }

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	root := NewRoot()
	if err := root.cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "agwctl: %v\n", err)
		if errors.Is(err, errUsage) {
			return ExitUsage
		}
		if errors.Is(err, errToolResult) {
			return ExitToolErr
		}
		if errors.Is(err, gwErrConnect) {
			return ExitConnect
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return ExitTimeout
		}
		if errors.Is(err, errTargets) {
			return ExitTargets
		}
		if errors.Is(err, errProtocol) {
			return ExitProtocol
		}
		return ExitProtocol
	}
	return ExitOK
}

// parseTimeout accepts Go durations and bare integers as seconds, so both
// --timeout 300 and --timeout 300s work.
func parseTimeout(raw string) (time.Duration, error) {
	if n, err := strconv.Atoi(raw); err == nil {
		return time.Duration(n) * time.Second, nil
	}
	return time.ParseDuration(raw)
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}