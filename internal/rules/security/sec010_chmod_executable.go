package security

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/lexer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC010ChmodExecutable checks for COPY --chmod with executable permissions
type SEC010ChmodExecutable struct{}

func (r *SEC010ChmodExecutable) ID() string          { return "SEC010" }
func (r *SEC010ChmodExecutable) Name() string        { return "chmod-executable" }
func (r *SEC010ChmodExecutable) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC010ChmodExecutable) Severity() analyzer.Severity { return analyzer.SeverityInfo }

func (r *SEC010ChmodExecutable) Description() string {
	return "COPY with --chmod granting execute permissions should be reviewed."
}

func (r *SEC010ChmodExecutable) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			var chmod string
			var pos lexer.Position

			switch v := inst.(type) {
			case *parser.CopyInstruction:
				chmod = v.Chmod
				pos = v.Pos()
			case *parser.AddInstruction:
				chmod = v.Chmod
				pos = v.Pos()
			default:
				continue
			}

			if chmod == "" {
				continue
			}

			// Check for executable permissions
			if hasExecutePermission(chmod) {
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessagef("--chmod=%s grants execute permissions", chmod).
					WithPos(pos).
					WithContext(ctx.GetLine(pos.Line)).
					WithHelp("Ensure execute permissions are intentional. Only scripts and binaries should be executable.").
					Build()
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

func hasExecutePermission(chmod string) bool {
	// Octal format
	if len(chmod) >= 3 {
		// Check for execute bit in any position
		for _, c := range chmod {
			if c >= '0' && c <= '7' {
				val := int(c - '0')
				if val&1 != 0 { // execute bit
					return true
				}
			}
		}
	}

	// Symbolic format
	if strings.Contains(chmod, "+x") || strings.Contains(chmod, "=x") {
		return true
	}

	return false
}

func init() {
	Register(&SEC010ChmodExecutable{})
}
