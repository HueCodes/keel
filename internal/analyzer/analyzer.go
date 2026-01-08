package analyzer

import (
	"sort"

	"github.com/HueCodes/keel/internal/parser"
)

// Rule is the interface that linting rules must implement
// This is duplicated here to avoid circular imports
type Rule interface {
	ID() string
	Category() Category
	Severity() Severity
	Check(df *parser.Dockerfile, ctx *RuleContext) []Diagnostic
}

// RuleContext provides context for rule checking
type RuleContext struct {
	Filename    string
	Source      string
	SourceLines []string
	Config      map[string]interface{}
}

// Analyzer runs rules against Dockerfiles
type Analyzer struct {
	rules       []Rule
	enabled     map[string]bool
	disabled    map[string]bool
	minSeverity Severity
	config      map[string]map[string]interface{}
}

// Option is a function that configures an Analyzer
type Option func(*Analyzer)

// New creates a new Analyzer with the given options
func New(opts ...Option) *Analyzer {
	a := &Analyzer{
		enabled:     make(map[string]bool),
		disabled:    make(map[string]bool),
		minSeverity: SeverityWarning,
		config:      make(map[string]map[string]interface{}),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// WithRules sets the rules to run
func WithRules(rules ...Rule) Option {
	return func(a *Analyzer) {
		a.rules = append(a.rules, rules...)
	}
}

// WithEnabled sets specific rules to enable (if set, only these run)
func WithEnabled(ids ...string) Option {
	return func(a *Analyzer) {
		for _, id := range ids {
			a.enabled[id] = true
		}
	}
}

// WithDisabled sets specific rules to disable
func WithDisabled(ids ...string) Option {
	return func(a *Analyzer) {
		for _, id := range ids {
			a.disabled[id] = true
		}
	}
}

// WithMinSeverity sets the minimum severity to report
func WithMinSeverity(s Severity) Option {
	return func(a *Analyzer) {
		a.minSeverity = s
	}
}

// WithRuleConfig sets configuration for a specific rule
func WithRuleConfig(ruleID string, config map[string]interface{}) Option {
	return func(a *Analyzer) {
		a.config[ruleID] = config
	}
}

// Analyze runs all enabled rules against the Dockerfile
func (a *Analyzer) Analyze(df *parser.Dockerfile, filename, source string) *Result {
	ctx := &RuleContext{
		Filename:    filename,
		Source:      source,
		SourceLines: splitLines(source),
		Config:      make(map[string]interface{}),
	}

	var diagnostics []Diagnostic

	for _, rule := range a.rules {
		// Check if rule should run
		if !a.shouldRun(rule) {
			continue
		}

		// Set rule-specific config
		if cfg, ok := a.config[rule.ID()]; ok {
			ctx.Config = cfg
		} else {
			ctx.Config = make(map[string]interface{})
		}

		// Run rule
		diags := rule.Check(df, ctx)

		// Filter by severity
		for _, d := range diags {
			if d.Severity >= a.minSeverity {
				diagnostics = append(diagnostics, d)
			}
		}
	}

	// Sort diagnostics by position
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Pos.Line != diagnostics[j].Pos.Line {
			return diagnostics[i].Pos.Line < diagnostics[j].Pos.Line
		}
		return diagnostics[i].Pos.Column < diagnostics[j].Pos.Column
	})

	return &Result{
		Diagnostics: diagnostics,
		Filename:    filename,
	}
}

// shouldRun checks if a rule should be run
func (a *Analyzer) shouldRun(rule Rule) bool {
	// If disabled, don't run
	if a.disabled[rule.ID()] {
		return false
	}

	// If enabled set is specified, only run those
	if len(a.enabled) > 0 {
		return a.enabled[rule.ID()]
	}

	return true
}

// GetLine returns the source line at the given line number (1-based)
func (c *RuleContext) GetLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(c.SourceLines) {
		return ""
	}
	return c.SourceLines[lineNum-1]
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

// AnalyzeSource parses and analyzes source code
func (a *Analyzer) AnalyzeSource(source, filename string) (*Result, []parser.ParseError) {
	df, parseErrors := parser.Parse(source)
	if len(parseErrors) > 0 {
		// Still try to analyze what we can
		result := a.Analyze(df, filename, source)
		return result, parseErrors
	}
	return a.Analyze(df, filename, source), nil
}
