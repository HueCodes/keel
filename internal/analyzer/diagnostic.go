package analyzer

import (
	"fmt"

	"github.com/HueCodes/keel/internal/lexer"
)

// Severity represents the severity of a diagnostic
type Severity int

const (
	SeverityHint Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityHint:
		return "hint"
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Category represents the category of a rule
type Category string

const (
	CategorySecurity    Category = "security"
	CategoryPerformance Category = "performance"
	CategoryBestPractice Category = "bestpractice"
	CategoryStyle       Category = "style"
)

// Diagnostic represents a linting issue
type Diagnostic struct {
	Rule       string         // rule ID (e.g., SEC001)
	Category   Category       // rule category
	Severity   Severity       // issue severity
	Message    string         // human-readable message
	Pos        lexer.Position // start position
	EndPos     lexer.Position // end position
	Context    string         // source context (the problematic line)
	Help       string         // help message with suggestion
	Fixable    bool           // whether this can be auto-fixed
	FixSuggestion string      // suggested fix text
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("[%s] %s: %s at %s", d.Rule, d.Severity, d.Message, d.Pos)
}

// DiagnosticBuilder helps construct diagnostics
type DiagnosticBuilder struct {
	diag Diagnostic
}

// NewDiagnostic creates a new diagnostic builder
func NewDiagnostic(rule string, category Category) *DiagnosticBuilder {
	return &DiagnosticBuilder{
		diag: Diagnostic{
			Rule:     rule,
			Category: category,
			Severity: SeverityWarning, // default
		},
	}
}

// WithSeverity sets the severity
func (b *DiagnosticBuilder) WithSeverity(s Severity) *DiagnosticBuilder {
	b.diag.Severity = s
	return b
}

// WithMessage sets the message
func (b *DiagnosticBuilder) WithMessage(msg string) *DiagnosticBuilder {
	b.diag.Message = msg
	return b
}

// WithMessagef sets a formatted message
func (b *DiagnosticBuilder) WithMessagef(format string, args ...interface{}) *DiagnosticBuilder {
	b.diag.Message = fmt.Sprintf(format, args...)
	return b
}

// WithPos sets the position
func (b *DiagnosticBuilder) WithPos(pos lexer.Position) *DiagnosticBuilder {
	b.diag.Pos = pos
	return b
}

// WithRange sets the position range
func (b *DiagnosticBuilder) WithRange(pos, endPos lexer.Position) *DiagnosticBuilder {
	b.diag.Pos = pos
	b.diag.EndPos = endPos
	return b
}

// WithContext sets the source context
func (b *DiagnosticBuilder) WithContext(ctx string) *DiagnosticBuilder {
	b.diag.Context = ctx
	return b
}

// WithHelp sets the help message
func (b *DiagnosticBuilder) WithHelp(help string) *DiagnosticBuilder {
	b.diag.Help = help
	return b
}

// WithFix marks this as fixable with a suggestion
func (b *DiagnosticBuilder) WithFix(suggestion string) *DiagnosticBuilder {
	b.diag.Fixable = true
	b.diag.FixSuggestion = suggestion
	return b
}

// Build returns the constructed diagnostic
func (b *DiagnosticBuilder) Build() Diagnostic {
	return b.diag
}

// Result holds the results of analyzing a Dockerfile
type Result struct {
	Diagnostics []Diagnostic
	Filename    string
}

// HasErrors returns true if there are any error-level diagnostics
func (r *Result) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// CountBySeverity returns the count of diagnostics by severity
func (r *Result) CountBySeverity() map[Severity]int {
	counts := make(map[Severity]int)
	for _, d := range r.Diagnostics {
		counts[d.Severity]++
	}
	return counts
}

// FilterBySeverity returns diagnostics at or above the given severity
func (r *Result) FilterBySeverity(minSeverity Severity) []Diagnostic {
	var filtered []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity >= minSeverity {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// FilterByCategory returns diagnostics of the given category
func (r *Result) FilterByCategory(category Category) []Diagnostic {
	var filtered []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Category == category {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
