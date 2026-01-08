package security

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC008Healthcheck checks for missing HEALTHCHECK
type SEC008Healthcheck struct{}

func (r *SEC008Healthcheck) ID() string          { return "SEC008" }
func (r *SEC008Healthcheck) Name() string        { return "missing-healthcheck" }
func (r *SEC008Healthcheck) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC008Healthcheck) Severity() analyzer.Severity { return analyzer.SeverityInfo }

func (r *SEC008Healthcheck) Description() string {
	return "HEALTHCHECK instruction is missing. Health checks enable container orchestrators to detect unhealthy containers."
}

func (r *SEC008Healthcheck) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	// Only check the final stage (the one that produces the output image)
	if len(df.Stages) == 0 {
		return diags
	}

	finalStage := df.Stages[len(df.Stages)-1]

	// Check if any stage has HEALTHCHECK (could be inherited)
	hasHealthcheck := false
	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			if hc, ok := inst.(*parser.HealthcheckInstruction); ok {
				if !hc.None {
					hasHealthcheck = true
					break
				}
			}
		}
		if hasHealthcheck {
			break
		}
	}

	if !hasHealthcheck {
		diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
			WithSeverity(r.Severity()).
			WithMessage("No HEALTHCHECK instruction found").
			WithPos(finalStage.From.Pos()).
			WithContext(ctx.GetLine(finalStage.From.Pos().Line)).
			WithHelp("Add a HEALTHCHECK instruction, e.g., HEALTHCHECK CMD curl -f http://localhost/ || exit 1").
			WithFix("HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD curl -f http://localhost/ || exit 1").
			Build()
		diags = append(diags, diag)
	}

	return diags
}

func init() {
	Register(&SEC008Healthcheck{})
}
