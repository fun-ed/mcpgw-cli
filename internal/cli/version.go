package cli

import (
	"fmt"

	"github.com/fun-ed/mcpgw-cli/internal/gw"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version and repo URL",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "agwctl %s\nhttps://github.com/fun-ed/mcpgw-cli\n", gw.ClientVersion)
			return err
		},
	}
}