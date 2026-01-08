package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/reporter"
	"github.com/HueCodes/keel/internal/rules/bestpractice"
	"github.com/HueCodes/keel/internal/rules/performance"
	"github.com/HueCodes/keel/internal/rules/security"
	"github.com/HueCodes/keel/internal/rules/style"
)

func lintCmd() *cobra.Command {
	var (
		file       string
		output     string
		severity   string
		ignore     []string
		only       []string
	)

	cmd := &cobra.Command{
		Use:   "lint [file]",
		Short: "Analyze Dockerfile and report issues",
		Long:  "Analyze a Dockerfile for security, performance, best practice, and style issues.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine file to lint
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

			// Parse severity
			minSeverity := parseSeverity(severity)

			// Create analyzer options
			opts := []analyzer.Option{
				analyzer.WithRules(rules...),
				analyzer.WithMinSeverity(minSeverity),
			}

			if len(only) > 0 {
				opts = append(opts, analyzer.WithEnabled(only...))
			}
			if len(ignore) > 0 {
				opts = append(opts, analyzer.WithDisabled(ignore...))
			}

			// Analyze
			a := analyzer.New(opts...)
			result, parseErrors := a.AnalyzeSource(string(content), file)

			// Report parse errors
			for _, pe := range parseErrors {
				fmt.Fprintf(os.Stderr, "Parse error: %s\n", pe)
			}

			// Determine output format
			noColor, _ := cmd.Flags().GetBool("no-color")
			format := reporter.Format(output)

			rep := reporter.New(format, os.Stdout, reporter.WithColors(!noColor))
			if err := rep.Report(result, string(content)); err != nil {
				return fmt.Errorf("failed to report: %w", err)
			}

			// Exit with error if there are errors
			if result.HasErrors() {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Dockerfile path (default \"Dockerfile\")")
	cmd.Flags().StringVarP(&output, "output", "o", "terminal", "Output format: terminal|json|sarif|markdown|github")
	cmd.Flags().StringVar(&severity, "severity", "warning", "Minimum severity: error|warning|info|hint")
	cmd.Flags().StringSliceVar(&ignore, "ignore", nil, "Rules to ignore (e.g., --ignore SEC001,PERF004)")
	cmd.Flags().StringSliceVar(&only, "only", nil, "Only run these rules")

	return cmd
}

func parseSeverity(s string) analyzer.Severity {
	switch s {
	case "error":
		return analyzer.SeverityError
	case "warning":
		return analyzer.SeverityWarning
	case "info":
		return analyzer.SeverityInfo
	case "hint":
		return analyzer.SeverityHint
	default:
		return analyzer.SeverityWarning
	}
}
