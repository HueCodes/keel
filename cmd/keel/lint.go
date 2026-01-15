package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parallel"
	"github.com/HueCodes/keel/internal/reporter"
	"github.com/HueCodes/keel/internal/rules/bestpractice"
	"github.com/HueCodes/keel/internal/rules/performance"
	"github.com/HueCodes/keel/internal/rules/security"
	"github.com/HueCodes/keel/internal/rules/style"
)

func lintCmd() *cobra.Command {
	var (
		file          string
		output        string
		severity      string
		ignore        []string
		only          []string
		runParallel   bool
		workers       int
		parallelRules bool
	)

	cmd := &cobra.Command{
		Use:   "lint [files...]",
		Short: "Analyze Dockerfile(s) and report issues",
		Long: `Analyze Dockerfile(s) for security, performance, best practice, and style issues.

Supports glob patterns for multiple files:
  keel lint                           # Lint ./Dockerfile
  keel lint Dockerfile.prod           # Lint specific file
  keel lint Dockerfile*               # Lint all matching files
  keel lint --parallel **/Dockerfile  # Lint in parallel`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine files to lint
			var files []string
			if len(args) > 0 {
				for _, pattern := range args {
					matches, err := filepath.Glob(pattern)
					if err != nil {
						return fmt.Errorf("invalid pattern %s: %w", pattern, err)
					}
					if len(matches) == 0 {
						// Treat as literal file path
						files = append(files, pattern)
					} else {
						files = append(files, matches...)
					}
				}
			} else if file != "" {
				files = append(files, file)
			} else {
				files = append(files, "Dockerfile")
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
			if parallelRules {
				opts = append(opts, analyzer.WithParallelRules(true))
			}
			if workers > 0 {
				opts = append(opts, analyzer.WithMaxWorkers(workers))
			}

			// Determine output format
			noColor, _ := cmd.Flags().GetBool("no-color")
			format := reporter.Format(output)
			rep := reporter.New(format, os.Stdout, reporter.WithColors(!noColor))

			var hasErrors bool

			// Process files
			if runParallel && len(files) > 1 {
				hasErrors = lintFilesParallel(files, opts, rep, workers)
			} else {
				hasErrors = lintFilesSequential(files, opts, rep)
			}

			if hasErrors {
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
	cmd.Flags().BoolVar(&runParallel, "parallel", false, "Process multiple files in parallel")
	cmd.Flags().IntVar(&workers, "workers", 0, "Number of parallel workers (default: number of CPUs)")
	cmd.Flags().BoolVar(&parallelRules, "parallel-rules", false, "Run rules in parallel for each file")

	return cmd
}

// lintFilesSequential processes files one at a time
func lintFilesSequential(files []string, opts []analyzer.Option, rep reporter.Reporter) bool {
	var hasErrors bool

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			hasErrors = true
			continue
		}

		a := analyzer.New(opts...)
		result, parseErrors := a.AnalyzeSource(string(content), file)

		for _, pe := range parseErrors {
			fmt.Fprintf(os.Stderr, "Parse error in %s: %s\n", file, pe)
		}

		if err := rep.Report(result, string(content)); err != nil {
			fmt.Fprintf(os.Stderr, "Error reporting %s: %v\n", file, err)
		}

		if result.HasErrors() {
			hasErrors = true
		}
	}

	return hasErrors
}

// lintFilesParallel processes files concurrently
func lintFilesParallel(files []string, opts []analyzer.Option, rep reporter.Reporter, workers int) bool {
	type lintResult struct {
		result      *analyzer.Result
		content     string
		parseErrors []string
	}

	p := parallel.New(parallel.WithWorkers(workers))
	results := p.Process(context.Background(), files, func(ctx context.Context, file string) (interface{}, error) {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		a := analyzer.New(opts...)
		result, parseErrors := a.AnalyzeSource(string(content), file)

		var errStrs []string
		for _, pe := range parseErrors {
			errStrs = append(errStrs, pe.Error())
		}

		return &lintResult{
			result:      result,
			content:     string(content),
			parseErrors: errStrs,
		}, nil
	})

	var hasErrors bool
	for _, r := range results {
		if r.Error != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", r.Filename, r.Error)
			hasErrors = true
			continue
		}

		lr := r.Result.(*lintResult)
		for _, pe := range lr.parseErrors {
			fmt.Fprintf(os.Stderr, "Parse error in %s: %s\n", r.Filename, pe)
		}

		if err := rep.Report(lr.result, lr.content); err != nil {
			fmt.Fprintf(os.Stderr, "Error reporting %s: %v\n", r.Filename, err)
		}

		if lr.result.HasErrors() {
			hasErrors = true
		}
	}

	return hasErrors
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
