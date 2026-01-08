package security

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC005Sudo checks for sudo usage in RUN instructions
type SEC005Sudo struct{}

func (r *SEC005Sudo) ID() string          { return "SEC005" }
func (r *SEC005Sudo) Name() string        { return "sudo-usage" }
func (r *SEC005Sudo) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC005Sudo) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *SEC005Sudo) Description() string {
	return "sudo should not be used in Dockerfiles. RUN commands execute as root by default."
}

func (r *SEC005Sudo) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if !ok {
				continue
			}

			cmd := run.Command
			if run.Heredoc != nil {
				cmd = run.Heredoc.Content
			}

			// Check for sudo
			if containsSudo(cmd) {
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessage("sudo usage detected in RUN instruction").
					WithPos(run.Pos()).
					WithContext(ctx.GetLine(run.Pos().Line)).
					WithHelp("Remove sudo - RUN commands execute as root by default. If you need to run as non-root, use USER instruction.").
					Build()
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

func containsSudo(cmd string) bool {
	// Split by common separators
	parts := strings.FieldsFunc(cmd, func(r rune) bool {
		return r == ' ' || r == '\t' || r == ';' || r == '&' || r == '|' || r == '\n'
	})

	for _, part := range parts {
		if part == "sudo" {
			return true
		}
	}
	return false
}

func init() {
	Register(&SEC005Sudo{})
}
