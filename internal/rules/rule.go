package rules

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// Rule is the interface that all linting rules must implement
type Rule interface {
	// ID returns the unique rule identifier (e.g., SEC001)
	ID() string

	// Name returns a human-readable name for the rule
	Name() string

	// Description returns a detailed description of what the rule checks
	Description() string

	// Category returns the rule category
	Category() analyzer.Category

	// Severity returns the default severity
	Severity() analyzer.Severity

	// Check analyzes the Dockerfile and returns diagnostics
	Check(df *parser.Dockerfile, ctx *Context) []analyzer.Diagnostic
}

// Context provides context for rule checking
type Context struct {
	// Filename is the name of the Dockerfile being analyzed
	Filename string

	// Source is the original source code
	Source string

	// SourceLines is the source split by lines
	SourceLines []string

	// Config holds rule-specific configuration
	Config map[string]interface{}
}

// NewContext creates a new context
func NewContext(filename, source string) *Context {
	lines := splitLines(source)
	return &Context{
		Filename:    filename,
		Source:      source,
		SourceLines: lines,
		Config:      make(map[string]interface{}),
	}
}

// GetLine returns the source line at the given line number (1-based)
func (c *Context) GetLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(c.SourceLines) {
		return ""
	}
	return c.SourceLines[lineNum-1]
}

// GetLines returns a range of source lines (1-based, inclusive)
func (c *Context) GetLines(startLine, endLine int) []string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(c.SourceLines) {
		endLine = len(c.SourceLines)
	}
	if startLine > endLine {
		return nil
	}
	return c.SourceLines[startLine-1 : endLine]
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

// BaseRule provides common functionality for rules
type BaseRule struct {
	RuleID          string
	RuleName        string
	RuleDescription string
	RuleCategory    analyzer.Category
	RuleSeverity    analyzer.Severity
}

func (r *BaseRule) ID() string                  { return r.RuleID }
func (r *BaseRule) Name() string                { return r.RuleName }
func (r *BaseRule) Description() string         { return r.RuleDescription }
func (r *BaseRule) Category() analyzer.Category { return r.RuleCategory }
func (r *BaseRule) Severity() analyzer.Severity { return r.RuleSeverity }

// NewDiagnostic creates a diagnostic for this rule
func (r *BaseRule) NewDiagnostic() *analyzer.DiagnosticBuilder {
	return analyzer.NewDiagnostic(r.RuleID, r.RuleCategory).
		WithSeverity(r.RuleSeverity)
}
