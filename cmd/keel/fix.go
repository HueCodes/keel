package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/optimizer"
	"github.com/HueCodes/keel/internal/parser"
	"github.com/HueCodes/keel/internal/rules/bestpractice"
	"github.com/HueCodes/keel/internal/rules/performance"
	"github.com/HueCodes/keel/internal/rules/security"
	"github.com/HueCodes/keel/internal/rules/style"
)

func fixCmd() *cobra.Command {
	var (
		file    string
		diff    bool
		dryRun  bool
		write   bool
	)

	cmd := &cobra.Command{
		Use:   "fix [file]",
		Short: "Auto-fix issues and write corrected Dockerfile",
		Long:  "Analyze a Dockerfile, apply automatic fixes, and write the corrected version.",
		Args:  cobra.MaximumNArgs(1),
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

			// Parse
			df, parseErrors := parser.Parse(source)
			if len(parseErrors) > 0 {
				for _, pe := range parseErrors {
					fmt.Fprintf(os.Stderr, "Parse error: %s\n", pe)
				}
			}

			// Collect all rules
			var rules []analyzer.Rule
			for _, r := range security.All() {
				rules = append(rules, r)
			}
			for _, r := range performance.All() {
				rules = append(rules, r)
			}
			for _, r := range bestpractice.All() {
				rules = append(rules, r)
			}
			for _, r := range style.All() {
				rules = append(rules, r)
			}

			// Analyze to find issues
			a := analyzer.New(analyzer.WithRules(rules...))
			result := a.Analyze(df, file, source)

			// Create optimizer with all transforms
			opt := optimizer.New(
				optimizer.WithTransforms(optimizer.AllTransforms()...),
				optimizer.WithDryRun(dryRun),
			)

			// Optimize
			optResult := opt.Optimize(df, result.Diagnostics)

			if !optResult.HasChanges() && !dryRun {
				fmt.Println("No fixable issues found.")
				return nil
			}

			// Rewrite
			rewriter := optimizer.NewRewriter()
			fixed := rewriter.Rewrite(df)

			if dryRun {
				fmt.Println("Dry run - changes that would be applied:")
				for _, c := range optResult.ChangesMade {
					fmt.Printf("  - %s: %s\n", c.Transform, c.Description)
				}
				return nil
			}

			if diff {
				// Show diff
				fmt.Println("--- " + file + " (original)")
				fmt.Println("+++ " + file + " (fixed)")
				showDiff(source, fixed)
				return nil
			}

			if write {
				// Write back to file
				if err := os.WriteFile(file, []byte(fixed), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", file, err)
				}
				fmt.Printf("Fixed %s\n", file)
				for _, c := range optResult.ChangesMade {
					if c.Applied {
						fmt.Printf("  - %s: %s\n", c.Transform, c.Description)
					}
				}
			} else {
				// Print to stdout
				fmt.Print(fixed)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Dockerfile path (default \"Dockerfile\")")
	cmd.Flags().BoolVar(&diff, "diff", false, "Show diff instead of writing")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without making changes")
	cmd.Flags().BoolVarP(&write, "write", "w", false, "Write changes back to file")

	return cmd
}

func showDiff(original, fixed string) {
	// Simple line-by-line diff
	origLines := splitLines(original)
	fixedLines := splitLines(fixed)

	// Very simple diff - just show all lines with +/-
	// A real implementation would use a proper diff algorithm
	maxLines := len(origLines)
	if len(fixedLines) > maxLines {
		maxLines = len(fixedLines)
	}

	for i := 0; i < maxLines; i++ {
		var origLine, fixedLine string
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(fixedLines) {
			fixedLine = fixedLines[i]
		}

		if origLine != fixedLine {
			if origLine != "" {
				fmt.Printf("\033[31m- %s\033[0m\n", origLine)
			}
			if fixedLine != "" {
				fmt.Printf("\033[32m+ %s\033[0m\n", fixedLine)
			}
		} else if origLine != "" {
			fmt.Printf("  %s\n", origLine)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
