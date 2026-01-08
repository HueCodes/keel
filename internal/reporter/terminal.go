package reporter

import (
	"fmt"
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
)

// TerminalReporter outputs results to the terminal with colors
type TerminalReporter struct {
	cfg *Config
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func (r *TerminalReporter) color(c, s string) string {
	if !r.cfg.UseColors {
		return s
	}
	return c + s + colorReset
}

func (r *TerminalReporter) severityColor(s analyzer.Severity) string {
	switch s {
	case analyzer.SeverityError:
		return colorRed
	case analyzer.SeverityWarning:
		return colorYellow
	case analyzer.SeverityInfo:
		return colorBlue
	case analyzer.SeverityHint:
		return colorCyan
	default:
		return ""
	}
}

// Report outputs the analysis results
func (r *TerminalReporter) Report(result *analyzer.Result, source string) error {
	w := r.cfg.Writer
	lines := strings.Split(source, "\n")

	for _, diag := range result.Diagnostics {
		// Location and rule
		loc := fmt.Sprintf("%s:%d:%d", result.Filename, diag.Pos.Line, diag.Pos.Column)
		severity := r.color(r.severityColor(diag.Severity), diag.Severity.String())
		rule := r.color(colorGray, "["+diag.Rule+"]")

		fmt.Fprintf(w, "%s %s %s: %s\n", loc, rule, severity, diag.Message)

		// Source context
		if diag.Pos.Line > 0 && diag.Pos.Line <= len(lines) {
			lineNum := diag.Pos.Line
			line := lines[lineNum-1]

			// Print line number gutter
			gutter := fmt.Sprintf("%4d", lineNum)
			fmt.Fprintf(w, "  %s │ %s\n", r.color(colorGray, gutter), line)

			// Print underline
			if diag.Pos.Column > 0 {
				padding := strings.Repeat(" ", diag.Pos.Column-1)
				underline := "^"
				if diag.EndPos.Column > diag.Pos.Column {
					underline = strings.Repeat("─", diag.EndPos.Column-diag.Pos.Column)
				}
				fmt.Fprintf(w, "       │ %s%s\n", padding, r.color(r.severityColor(diag.Severity), underline))
			}
		}

		// Help message
		if diag.Help != "" {
			fmt.Fprintf(w, "       │\n")
			fmt.Fprintf(w, "       = %s: %s\n", r.color(colorCyan, "help"), diag.Help)
		}

		fmt.Fprintln(w)
	}

	// Summary
	counts := result.CountBySeverity()
	var parts []string
	if c := counts[analyzer.SeverityError]; c > 0 {
		parts = append(parts, r.color(colorRed, fmt.Sprintf("%d error(s)", c)))
	}
	if c := counts[analyzer.SeverityWarning]; c > 0 {
		parts = append(parts, r.color(colorYellow, fmt.Sprintf("%d warning(s)", c)))
	}
	if c := counts[analyzer.SeverityInfo]; c > 0 {
		parts = append(parts, r.color(colorBlue, fmt.Sprintf("%d info", c)))
	}
	if c := counts[analyzer.SeverityHint]; c > 0 {
		parts = append(parts, r.color(colorCyan, fmt.Sprintf("%d hint(s)", c)))
	}

	if len(parts) > 0 {
		fmt.Fprintf(w, "Found %s in %s\n", strings.Join(parts, ", "), result.Filename)
	} else {
		fmt.Fprintf(w, "%s No issues found in %s\n", r.color(colorGray, "✓"), result.Filename)
	}

	return nil
}
