package security

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/lexer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC001RootUser checks for containers running as root
type SEC001RootUser struct{}

func (r *SEC001RootUser) ID() string          { return "SEC001" }
func (r *SEC001RootUser) Name() string        { return "root-user" }
func (r *SEC001RootUser) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC001RootUser) Severity() analyzer.Severity { return analyzer.SeverityError }

func (r *SEC001RootUser) Description() string {
	return "Container runs as root user. Running containers as root is a security risk."
}

func (r *SEC001RootUser) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	// Only check the final stage (the one that produces the output image)
	// Build stages running as root is generally acceptable
	if len(df.Stages) == 0 {
		return diags
	}

	finalStage := df.Stages[len(df.Stages)-1]
	hasUser := false
	var lastUserIsRoot bool
	var lastUserPos lexer.Position

	for _, inst := range finalStage.Instructions {
		if user, ok := inst.(*parser.UserInstruction); ok {
			hasUser = true
			// Check if USER is root or 0
			lastUserIsRoot = user.User == "root" || user.User == "0"
			lastUserPos = user.Pos()
		}
	}

	// No USER instruction at all
	if !hasUser {
		diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
			WithSeverity(r.Severity()).
			WithMessage("Container runs as root (no USER instruction found)").
			WithPos(finalStage.From.Pos()).
			WithContext(ctx.GetLine(finalStage.From.Pos().Line)).
			WithHelp("Add a USER instruction to run as a non-root user, e.g., USER nobody").
			WithFix("USER nobody").
			Build()
		diags = append(diags, diag)
	} else if lastUserIsRoot {
		// Last USER instruction sets root
		diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
			WithSeverity(r.Severity()).
			WithMessage("Container explicitly runs as root user").
			WithPos(lastUserPos).
			WithContext(ctx.GetLine(lastUserPos.Line)).
			WithHelp("Change to a non-root user for better security").
			Build()
		diags = append(diags, diag)
	}

	return diags
}

type Position = lexer.Position

func init() {
	Register(&SEC001RootUser{})
}
