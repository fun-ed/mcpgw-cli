package cli

import (
	"errors"
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

var opts struct {
	url           string
	timeout       time.Duration
	expectTargets int
	verbose       bool
}

func NewRoot() *cobraRoot {
	root := &cobraRoot{}
	cmd := &cobra.Command{
		Use:           "agwctl",
		Short:         "Shell client for the local agentgateway",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.verbose {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
			} else {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&opts.url, "url", envOr("AGWCTL_URL", "http://127.0.0.1:8083/mcp"), "gateway MCP endpoint")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", 120*time.Second, "initialize and call timeout")
	cmd.PersistentFlags().IntVar(&opts.expectTargets, "expect-targets", 8, "warn and fail doctor when fewer targets answer (0 disables)")
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "log protocol details to stderr")

	cmd.AddCommand(newToolsCmd())
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
		if errors.Is(err, gwErrConnect) {
			return ExitConnect
		}
		if errors.Is(err, errProtocol) {
			return ExitProtocol
		}
		return ExitProtocol
	}
	return ExitOK
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}