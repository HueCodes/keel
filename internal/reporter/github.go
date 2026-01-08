package reporter

import (
	"fmt"

	"github.com/HueCodes/keel/internal/analyzer"
)

// GitHubReporter outputs results as GitHub Actions workflow commands
type GitHubReporter struct {
	cfg *Config
}

// Report outputs the analysis results as GitHub workflow commands
func (r *GitHubReporter) Report(result *analyzer.Result, source string) error {
	w := r.cfg.Writer

	for _, diag := range result.Diagnostics {
		level := githubLevel(diag.Severity)
		// Format: ::warning file={name},line={line},col={col}::{message}
		fmt.Fprintf(w, "::%s file=%s,line=%d,col=%d,title=%s::%s\n",
			level,
			result.Filename,
			diag.Pos.Line,
			diag.Pos.Column,
			diag.Rule,
			diag.Message,
		)
	}

	// Summary
	counts := result.CountBySeverity()
	if counts[analyzer.SeverityError] > 0 || counts[analyzer.SeverityWarning] > 0 {
		fmt.Fprintf(w, "::group::Summary\n")
		fmt.Fprintf(w, "Found %d issue(s) in %s\n", len(result.Diagnostics), result.Filename)
		fmt.Fprintf(w, "::endgroup::\n")
	}

	return nil
}

func githubLevel(s analyzer.Severity) string {
	switch s {
	case analyzer.SeverityError:
		return "error"
	case analyzer.SeverityWarning:
		return "warning"
	default:
		return "notice"
	}
}
