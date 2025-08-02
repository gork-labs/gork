// Package main provides the CLI for Gork development tools.
package main

import (
	"fmt"
	"os"

	"github.com/gork-labs/gork/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
