package performance

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// PERF004ConsecutiveRun checks for multiple consecutive RUN instructions
type PERF004ConsecutiveRun struct{}

func (r *PERF004ConsecutiveRun) ID() string          { return "PERF004" }
func (r *PERF004ConsecutiveRun) Name() string        { return "consecutive-run" }
func (r *PERF004ConsecutiveRun) Category() analyzer.Category { return analyzer.CategoryPerformance }
func (r *PERF004ConsecutiveRun) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *PERF004ConsecutiveRun) Description() string {
	return "Consecutive RUN instructions create multiple layers. Merge them to reduce image size."
}

func (r *PERF004ConsecutiveRun) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	// Get configurable threshold (default 2)
	threshold := 2
	if v, ok := ctx.Config["max_consecutive"].(int); ok {
		threshold = v
	}

	for _, stage := range df.Stages {
		consecutiveRuns := []*parser.RunInstruction{}

		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if ok {
				consecutiveRuns = append(consecutiveRuns, run)
			} else {
				// Non-RUN instruction breaks the sequence
				if len(consecutiveRuns) >= threshold {
					reportConsecutiveRuns(consecutiveRuns, ctx, &diags, r)
				}
				consecutiveRuns = []*parser.RunInstruction{}
			}
		}

		// Check remaining
		if len(consecutiveRuns) >= threshold {
			reportConsecutiveRuns(consecutiveRuns, ctx, &diags, r)
		}
	}

	return diags
}

func reportConsecutiveRuns(runs []*parser.RunInstruction, ctx *analyzer.RuleContext, diags *[]analyzer.Diagnostic, r *PERF004ConsecutiveRun) {
	if len(runs) < 2 {
		return
	}

	firstRun := runs[0]
	lastRun := runs[len(runs)-1]

	diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
		WithSeverity(r.Severity()).
		WithMessagef("%d consecutive RUN instructions could be merged", len(runs)).
		WithRange(firstRun.Pos(), lastRun.End()).
		WithContext(ctx.GetLine(firstRun.Pos().Line)).
		WithHelp("Merge into a single RUN with && between commands to reduce layers").
		WithFix("merge-run").
		Build()
	*diags = append(*diags, diag)
}

func init() {
	Register(&PERF004ConsecutiveRun{})
}
