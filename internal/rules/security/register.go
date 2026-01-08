package security

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// Rule interface for security rules
type Rule interface {
	ID() string
	Name() string
	Description() string
	Category() analyzer.Category
	Severity() analyzer.Severity
	Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic
}

var rules []Rule

// Register adds a rule to the security rules list
func Register(rule Rule) {
	rules = append(rules, rule)
}

// All returns all security rules
func All() []Rule {
	return rules
}
