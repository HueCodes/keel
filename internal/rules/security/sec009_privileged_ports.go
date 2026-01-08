package security

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC009PrivilegedPorts checks for privileged port exposure
type SEC009PrivilegedPorts struct{}

func (r *SEC009PrivilegedPorts) ID() string          { return "SEC009" }
func (r *SEC009PrivilegedPorts) Name() string        { return "privileged-ports" }
func (r *SEC009PrivilegedPorts) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC009PrivilegedPorts) Severity() analyzer.Severity { return analyzer.SeverityInfo }

func (r *SEC009PrivilegedPorts) Description() string {
	return "Privileged ports (< 1024) require root privileges. Consider using unprivileged ports."
}

func (r *SEC009PrivilegedPorts) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			expose, ok := inst.(*parser.ExposeInstruction)
			if !ok {
				continue
			}

			for _, port := range expose.Ports {
				if port.IsPrivilegedPort() {
					diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
						WithSeverity(r.Severity()).
						WithMessagef("Exposing privileged port %s", port.Port).
						WithPos(expose.Pos()).
						WithContext(ctx.GetLine(expose.Pos().Line)).
						WithHelp("Privileged ports require root. Consider using an unprivileged port (>= 1024) and mapping it at runtime with -p 80:8080").
						Build()
					diags = append(diags, diag)
				}
			}
		}
	}

	return diags
}

func init() {
	Register(&SEC009PrivilegedPorts{})
}
