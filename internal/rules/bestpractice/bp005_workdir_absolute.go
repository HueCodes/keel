package bestpractice

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// BP005WorkdirAbsolute checks for WORKDIR with relative paths
type BP005WorkdirAbsolute struct{}

func (r *BP005WorkdirAbsolute) ID() string          { return "BP005" }
func (r *BP005WorkdirAbsolute) Name() string        { return "workdir-absolute" }
func (r *BP005WorkdirAbsolute) Category() analyzer.Category { return analyzer.CategoryBestPractice }
func (r *BP005WorkdirAbsolute) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *BP005WorkdirAbsolute) Description() string {
	return "WORKDIR should use absolute paths for clarity and predictability."
}

func (r *BP005WorkdirAbsolute) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			wd, ok := inst.(*parser.WorkdirInstruction)
			if !ok {
				continue
			}

			path := wd.Path
			// Skip variable expansion
			if strings.HasPrefix(path, "$") {
				continue
			}

			// Check for absolute path
			if !strings.HasPrefix(path, "/") {
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessagef("WORKDIR uses relative path: %s", path).
					WithPos(wd.Pos()).
					WithContext(ctx.GetLine(wd.Pos().Line)).
					WithHelp("Use an absolute path for WORKDIR: WORKDIR /" + path).
					WithFix("WORKDIR /" + path).
					Build()
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

func init() {
	Register(&BP005WorkdirAbsolute{})
}
