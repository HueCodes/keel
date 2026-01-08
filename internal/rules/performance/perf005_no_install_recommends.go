package performance

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// PERF005NoInstallRecommends checks for apt-get install without --no-install-recommends
type PERF005NoInstallRecommends struct{}

func (r *PERF005NoInstallRecommends) ID() string          { return "PERF005" }
func (r *PERF005NoInstallRecommends) Name() string        { return "no-install-recommends" }
func (r *PERF005NoInstallRecommends) Category() analyzer.Category { return analyzer.CategoryPerformance }
func (r *PERF005NoInstallRecommends) Severity() analyzer.Severity { return analyzer.SeverityInfo }

func (r *PERF005NoInstallRecommends) Description() string {
	return "apt-get install without --no-install-recommends installs unnecessary packages."
}

func (r *PERF005NoInstallRecommends) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
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

			// Check for apt-get install without --no-install-recommends
			if strings.Contains(cmd, "apt-get install") && !strings.Contains(cmd, "--no-install-recommends") {
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessage("apt-get install without --no-install-recommends").
					WithPos(run.Pos()).
					WithContext(ctx.GetLine(run.Pos().Line)).
					WithHelp("Add --no-install-recommends to avoid installing unnecessary packages: apt-get install --no-install-recommends").
					Build()
				diags = append(diags, diag)
			}

			// Check for apt install without --no-install-recommends
			if strings.Contains(cmd, "apt install") && !strings.Contains(cmd, "--no-install-recommends") {
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessage("apt install without --no-install-recommends").
					WithPos(run.Pos()).
					WithContext(ctx.GetLine(run.Pos().Line)).
					WithHelp("Add --no-install-recommends to avoid installing unnecessary packages").
					Build()
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

func init() {
	Register(&PERF005NoInstallRecommends{})
}
