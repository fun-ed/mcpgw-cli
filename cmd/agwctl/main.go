package main

import (
	"os"

	"github.com/fun-ed/mcpgw-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
