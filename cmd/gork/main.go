// Package main provides the CLI for Gork development tools.
package main

import (
	"os"

	"github.com/gork-labs/gork/cmd/gork/openapi"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gork",
		Short: "Gork development tools",
	}

	rootCmd.AddCommand(openapi.NewCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
