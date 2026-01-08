package bestpractice

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// BP003MultipleCmd checks for multiple CMD instructions
type BP003MultipleCmd struct{}

func (r *BP003MultipleCmd) ID() string          { return "BP003" }
func (r *BP003MultipleCmd) Name() string        { return "multiple-cmd" }
func (r *BP003MultipleCmd) Category() analyzer.Category { return analyzer.CategoryBestPractice }
func (r *BP003MultipleCmd) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *BP003MultipleCmd) Description() string {
	return "Only the last CMD instruction takes effect. Multiple CMDs are likely a mistake."
}

func (r *BP003MultipleCmd) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		var cmds []*parser.CmdInstruction

		for _, inst := range stage.Instructions {
			if cmd, ok := inst.(*parser.CmdInstruction); ok {
				cmds = append(cmds, cmd)
			}
		}

		if len(cmds) > 1 {
			// Report all but the last CMD
			for i := 0; i < len(cmds)-1; i++ {
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessage("This CMD instruction is overridden by a later CMD").
					WithPos(cmds[i].Pos()).
					WithContext(ctx.GetLine(cmds[i].Pos().Line)).
					WithHelp("Remove this CMD or combine the commands. Only the last CMD takes effect.").
					Build()
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

func init() {
	Register(&BP003MultipleCmd{})
}
