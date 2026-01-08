package bestpractice

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// Rule interface for best practice rules
type Rule interface {
	ID() string
	Name() string
	Description() string
	Category() analyzer.Category
	Severity() analyzer.Severity
	Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic
}

var rules []Rule

// Register adds a rule to the best practice rules list
func Register(rule Rule) {
	rules = append(rules, rule)
}

// All returns all best practice rules
func All() []Rule {
	return rules
}
