package performance

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// Rule interface for performance rules
type Rule interface {
	ID() string
	Name() string
	Description() string
	Category() analyzer.Category
	Severity() analyzer.Severity
	Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic
}

var rules []Rule

// Register adds a rule to the performance rules list
func Register(rule Rule) {
	rules = append(rules, rule)
}

// All returns all performance rules
func All() []Rule {
	return rules
}
