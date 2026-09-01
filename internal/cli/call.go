package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"agwctl/internal/gw"
	"agwctl/internal/out"
)

func newCallCmd() *cobra.Command {
	var (
		stdin    bool
		argKVs   []string
		jsonArgs string
		jqExpr   string
		maxChars int
		outFile  string
	)
	cmd := &cobra.Command{
		Use:   "call [NAME]",
		Short: "Call a gateway tool; --stdin batches JSONL requests over one session",
		Args: func(cmd *cobra.Command, args []string) error {
			if stdin {
				if len(args) != 0 {
					return fmt.Errorf("%w: --stdin takes no NAME", errUsage)
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("%w: call requires exactly one tool name (or --stdin)", errUsage)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if stdin {
				opts.maxChars = maxChars
				opts.jqExpr = jqExpr
				return runBatch(ctx)
			}
			return runCall(ctx, args[0], argKVs, jsonArgs, jqExpr, maxChars, outFile)
		},
	}
	cmd.Flags().BoolVar(&stdin, "stdin", false, "read JSONL {name,arguments} requests from stdin, one session")
	cmd.Flags().StringArrayVar(&argKVs, "arg", nil, "tool argument key=value, repeatable")
	cmd.Flags().StringVar(&jsonArgs, "json", "", "full arguments as JSON object, or @file.json")
	cmd.Flags().StringVar(&jqExpr, "jq", "", "jq expression over the raw CallToolResult")
	cmd.Flags().IntVar(&maxChars, "max-chars", 20000, "truncate text output to N chars, 0 unlimited")
	cmd.Flags().StringVar(&outFile, "out", "", "write full result to file; stdout only notes the path")
	return cmd
}

func buildArgs(argKVs []string, jsonArgs string) (any, error) {
	if jsonArgs != "" {
		if len(argKVs) > 0 {
			return nil, fmt.Errorf("%w: --json and --arg are mutually exclusive", errUsage)
		}
		raw := jsonArgs
		if strings.HasPrefix(jsonArgs, "@") {
			b, err := os.ReadFile(strings.TrimPrefix(jsonArgs, "@"))
			if err != nil {
				return nil, fmt.Errorf("%w: --json file: %v", errUsage, err)
			}
			raw = string(b)
		}
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, fmt.Errorf("%w: --json is not valid JSON: %v", errUsage, err)
		}
		return v, nil
	}
	if len(argKVs) == 0 {
		return map[string]any{}, nil
	}
	args := map[string]any{}
	for _, kv := range argKVs {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("%w: --arg expects key=value, got %q", errUsage, kv)
		}
		args[k] = parseArgValue(v)
	}
	return args, nil
}

// parseArgValue tries JSON first: 5 is a number, true a bool, "x" a string;
// anything else stays a plain string.
func parseArgValue(v string) any {
	var j any
	if err := json.Unmarshal([]byte(v), &j); err == nil {
		return j
	}
	return v
}

func runCall(ctx context.Context, name string, argKVs []string, jsonArgs, jqExpr string, maxChars int, outFile string) error {
	arguments, err := buildArgs(argKVs, jsonArgs)
	if err != nil {
		return err
	}
	c, err := gw.Connect(ctx, opts.url, opts.timeout)
	if err != nil {
		return err
	}
	defer c.Close()

	res, err := c.CallTool(ctx, name, arguments)
	if err != nil {
		return fmt.Errorf("%w: tools/call %s: %v", errProtocol, name, err)
	}

	// Pipeline per PLAN.md: raw result -> --jq -> --max-chars -> --out/stdout.
	// isError skips --jq on purpose; the error text must survive intact.
	// --jq "." is the raw-result mode for scripts.
	var stdoutBody string
	switch {
	case jqExpr != "" && !res.IsError:
		stdoutBody, err = applyJQ(res, jqExpr)
	default:
		stdoutBody, err = textOf(res)
	}
	if err != nil {
		return err
	}

	if outFile != "" {
		// Full fidelity for scripts: post-jq content when --jq is set, the
		// raw CallToolResult JSON otherwise, so downstream jq always works.
		fileBody := stdoutBody
		if jqExpr == "" || res.IsError {
			raw, marshalErr := json.Marshal(res)
			if marshalErr != nil {
				return marshalErr
			}
			fileBody = string(raw)
		}
		if err := os.WriteFile(outFile, []byte(fileBody), 0o600); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "wrote %s (%d bytes)\n", outFile, len(fileBody))
	} else {
		fmt.Fprintln(os.Stdout, out.Truncate(stdoutBody, maxChars))
	}
	if res.IsError {
		return errToolResult
	}
	return nil
}

func runBatch(ctx context.Context) error {
	// Batch results are raw JSONL by contract; scripts filter with jq
	// themselves, so result-shaping flags have nothing to act on.
	if opts.maxChars != 0 || opts.jqExpr != "" {
		fmt.Fprintln(os.Stderr, "agwctl: --max-chars and --jq are ignored with --stdin; pipe through jq instead")
	}
	c, err := gw.Connect(ctx, opts.url, opts.timeout)
	if err != nil {
		return err
	}
	defer c.Close()

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	anyToolErr := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var req struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(line), &req); err != nil || req.Name == "" {
			return fmt.Errorf("%w: batch line wants {\"name\",\"arguments\"}: %v", errUsage, err)
		}
		if req.Arguments == nil {
			req.Arguments = map[string]any{}
		}
		res, err := c.CallTool(ctx, req.Name, req.Arguments)
		if err != nil {
			return fmt.Errorf("%w: tools/call %s: %v", errProtocol, req.Name, err)
		}
		raw, err := json.Marshal(res)
		if err != nil {
			return err
		}
		w.Write(raw)
		w.WriteByte('\n')
		w.Flush()
		if res.IsError {
			anyToolErr = true
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if anyToolErr {
		return errToolResult
	}
	return nil
}

// textOf renders the human-readable body: text content joined by newlines,
// falling back to structured content as JSON.
func textOf(res *mcp.CallToolResult) (string, error) {
	var parts []string
	for _, item := range res.Content {
		if tc, ok := item.(*mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		} else {
			raw, err := json.Marshal(item)
			if err != nil {
				return "", err
			}
			parts = append(parts, string(raw))
		}
	}
	if len(parts) == 0 && res.StructuredContent != nil {
		raw, err := json.Marshal(res.StructuredContent)
		if err != nil {
			return "", err
		}
		parts = append(parts, string(raw))
	}
	return strings.Join(parts, "\n"), nil
}

// applyJQ evaluates the expression over the full result object.
func applyJQ(res *mcp.CallToolResult, expr string) (string, error) {
	raw, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", err
	}
	vals, err := out.EvalJQ(data, expr)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(vals))
	for _, v := range vals {
		lines = append(lines, out.PrintValue(v))
	}
	if len(lines) == 0 {
		return "", nil
	}
	return strings.Join(lines, "\n"), nil
}

