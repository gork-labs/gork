// Package cli provides the command-line interface for Gork development tools.
package cli

import (
	"github.com/spf13/cobra"
)

// Execute creates and runs the root command.
func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "gork",
		Short: "Gork development tools",
	}

	rootCmd.AddCommand(newOpenAPICommand())
	
	return rootCmd.Execute()
}