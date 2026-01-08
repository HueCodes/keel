package bestpractice

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// BP004DeprecatedMaintainer checks for deprecated MAINTAINER instruction
type BP004DeprecatedMaintainer struct{}

func (r *BP004DeprecatedMaintainer) ID() string          { return "BP004" }
func (r *BP004DeprecatedMaintainer) Name() string        { return "deprecated-maintainer" }
func (r *BP004DeprecatedMaintainer) Category() analyzer.Category { return analyzer.CategoryBestPractice }
func (r *BP004DeprecatedMaintainer) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *BP004DeprecatedMaintainer) Description() string {
	return "MAINTAINER is deprecated. Use LABEL maintainer=\"...\" instead."
}

func (r *BP004DeprecatedMaintainer) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			maint, ok := inst.(*parser.MaintainerInstruction)
			if !ok {
				continue
			}

			diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
				WithSeverity(r.Severity()).
				WithMessage("MAINTAINER instruction is deprecated").
				WithPos(maint.Pos()).
				WithContext(ctx.GetLine(maint.Pos().Line)).
				WithHelp("Use LABEL instead: LABEL maintainer=\"" + maint.Maintainer + "\"").
				WithFix("LABEL maintainer=\"" + maint.Maintainer + "\"").
				Build()
			diags = append(diags, diag)
		}
	}

	return diags
}

func init() {
	Register(&BP004DeprecatedMaintainer{})
}
