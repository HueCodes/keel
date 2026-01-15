package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/HueCodes/keel/internal/formatter"
)

func fmtCmd() *cobra.Command {
	var (
		file  string
		check bool
		diff  bool
		write bool
	)

	cmd := &cobra.Command{
		Use:   "fmt [file]",
		Short: "Format Dockerfile (style only)",
		Long: `Format a Dockerfile for consistent style without making semantic changes.

Formatting includes:
  - Uppercase all instructions (FROM, RUN, COPY, etc.)
  - Align multi-value instructions (ENV, LABEL)
  - Normalize line continuations with aligned backslashes
  - Remove excessive blank lines
  - Preserve comments with their associated instructions

Examples:
  keel fmt                    # Format Dockerfile, output to stdout
  keel fmt -w                 # Format and write back to file
  keel fmt --check            # Check if formatting needed (for CI)
  keel fmt --diff             # Show what would change
  keel fmt Dockerfile.prod    # Format specific file`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				file = args[0]
			}
			if file == "" {
				file = "Dockerfile"
			}

			// Read file
			content, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", file, err)
			}
			source := string(content)

			// Create formatter with default options
			f := formatter.New(formatter.DefaultOptions())

			result, err := f.FormatSource(source)
			if err != nil {
				return fmt.Errorf("failed to format %s: %w", file, err)
			}

			// Handle --check mode (for CI)
			if check {
				if result.HasChanges {
					fmt.Fprintf(os.Stderr, "%s: needs formatting\n", file)
					os.Exit(1)
				}
				fmt.Fprintf(os.Stderr, "%s: already formatted\n", file)
				return nil
			}

			// Handle --diff mode
			if diff {
				if result.HasChanges {
					diffOutput := formatter.Diff(file, result.Original, result.Formatted)
					fmt.Print(diffOutput)
				} else {
					fmt.Println("No changes needed")
				}
				return nil
			}

			// Handle --write mode
			if write {
				if !result.HasChanges {
					fmt.Printf("%s: already formatted\n", file)
					return nil
				}

				if err := os.WriteFile(file, []byte(result.Formatted), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", file, err)
				}
				fmt.Printf("Formatted %s\n", file)
				return nil
			}

			// Default: output to stdout
			fmt.Print(result.Formatted)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Dockerfile path (default \"Dockerfile\")")
	cmd.Flags().BoolVar(&check, "check", false, "Exit non-zero if changes needed (for CI)")
	cmd.Flags().BoolVar(&diff, "diff", false, "Show what would change without writing")
	cmd.Flags().BoolVarP(&write, "write", "w", false, "Write changes back to file")

	return cmd
}
