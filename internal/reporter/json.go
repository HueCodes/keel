package reporter

import (
	"encoding/json"

	"github.com/HueCodes/keel/internal/analyzer"
)

// JSONReporter outputs results as JSON
type JSONReporter struct {
	cfg *Config
}

// JSONOutput is the JSON output structure
type JSONOutput struct {
	Filename    string           `json:"filename"`
	Diagnostics []JSONDiagnostic `json:"diagnostics"`
	Summary     JSONSummary      `json:"summary"`
}

// JSONDiagnostic represents a diagnostic in JSON format
type JSONDiagnostic struct {
	Rule     string `json:"rule"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	EndLine  int    `json:"end_line,omitempty"`
	EndColumn int   `json:"end_column,omitempty"`
	Context  string `json:"context,omitempty"`
	Help     string `json:"help,omitempty"`
	Fixable  bool   `json:"fixable"`
	Fix      string `json:"fix,omitempty"`
}

// JSONSummary contains summary counts
type JSONSummary struct {
	Total    int `json:"total"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
	Hints    int `json:"hints"`
}

// Report outputs the analysis results as JSON
func (r *JSONReporter) Report(result *analyzer.Result, source string) error {
	output := JSONOutput{
		Filename:    result.Filename,
		Diagnostics: make([]JSONDiagnostic, 0, len(result.Diagnostics)),
	}

	counts := result.CountBySeverity()
	output.Summary = JSONSummary{
		Total:    len(result.Diagnostics),
		Errors:   counts[analyzer.SeverityError],
		Warnings: counts[analyzer.SeverityWarning],
		Info:     counts[analyzer.SeverityInfo],
		Hints:    counts[analyzer.SeverityHint],
	}

	for _, diag := range result.Diagnostics {
		jd := JSONDiagnostic{
			Rule:      diag.Rule,
			Category:  string(diag.Category),
			Severity:  diag.Severity.String(),
			Message:   diag.Message,
			Line:      diag.Pos.Line,
			Column:    diag.Pos.Column,
			EndLine:   diag.EndPos.Line,
			EndColumn: diag.EndPos.Column,
			Context:   diag.Context,
			Help:      diag.Help,
			Fixable:   diag.Fixable,
			Fix:       diag.FixSuggestion,
		}
		output.Diagnostics = append(output.Diagnostics, jd)
	}

	encoder := json.NewEncoder(r.cfg.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
