package reporter

import (
	"fmt"

	"github.com/HueCodes/keel/internal/analyzer"
)

// MarkdownReporter outputs results as Markdown
type MarkdownReporter struct {
	cfg *Config
}

// Report outputs the analysis results as Markdown
func (r *MarkdownReporter) Report(result *analyzer.Result, source string) error {
	w := r.cfg.Writer

	if len(result.Diagnostics) == 0 {
		fmt.Fprintf(w, "## ✅ No issues found\n\nDockerfile `%s` passed all checks.\n", result.Filename)
		return nil
	}

	counts := result.CountBySeverity()
	fmt.Fprintf(w, "## Dockerfile Linting Results: `%s`\n\n", result.Filename)

	// Summary
	fmt.Fprintf(w, "| Severity | Count |\n")
	fmt.Fprintf(w, "|----------|-------|\n")
	if c := counts[analyzer.SeverityError]; c > 0 {
		fmt.Fprintf(w, "| 🔴 Error | %d |\n", c)
	}
	if c := counts[analyzer.SeverityWarning]; c > 0 {
		fmt.Fprintf(w, "| 🟡 Warning | %d |\n", c)
	}
	if c := counts[analyzer.SeverityInfo]; c > 0 {
		fmt.Fprintf(w, "| 🔵 Info | %d |\n", c)
	}
	if c := counts[analyzer.SeverityHint]; c > 0 {
		fmt.Fprintf(w, "| 💡 Hint | %d |\n", c)
	}
	fmt.Fprintln(w)

	// Details
	fmt.Fprintf(w, "### Issues\n\n")

	for _, diag := range result.Diagnostics {
		emoji := severityEmoji(diag.Severity)
		fmt.Fprintf(w, "#### %s `%s` - Line %d\n\n", emoji, diag.Rule, diag.Pos.Line)
		fmt.Fprintf(w, "%s\n\n", diag.Message)

		if diag.Context != "" {
			fmt.Fprintf(w, "```dockerfile\n%s\n```\n\n", diag.Context)
		}

		if diag.Help != "" {
			fmt.Fprintf(w, "> 💡 %s\n\n", diag.Help)
		}
	}

	return nil
}

func severityEmoji(s analyzer.Severity) string {
	switch s {
	case analyzer.SeverityError:
		return "🔴"
	case analyzer.SeverityWarning:
		return "🟡"
	case analyzer.SeverityInfo:
		return "🔵"
	default:
		return "💡"
	}
}
