package main

import (
	"os"

	"agwctl/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
