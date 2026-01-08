package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "keel",
		Short: "Dockerfile linter, analyzer, and optimizer",
		Long: `Keel is a multi-stage Dockerfile linter and optimizer.

It analyzes Dockerfiles for security issues, performance problems,
best practice violations, and style inconsistencies. It can also
automatically fix many issues and rewrite Dockerfiles.`,
		Version: version,
	}

	rootCmd.AddCommand(
		lintCmd(),
		fixCmd(),
		fmtCmd(),
		explainCmd(),
		initCmd(),
	)

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file path (default .keel.yaml)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Only output errors")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Show additional context")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
