package security

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC007AddRemote checks for ADD with remote URLs
type SEC007AddRemote struct{}

func (r *SEC007AddRemote) ID() string          { return "SEC007" }
func (r *SEC007AddRemote) Name() string        { return "add-remote-url" }
func (r *SEC007AddRemote) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC007AddRemote) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *SEC007AddRemote) Description() string {
	return "ADD with remote URL downloads without verification. Use curl/wget with checksum verification instead."
}

func (r *SEC007AddRemote) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			add, ok := inst.(*parser.AddInstruction)
			if !ok {
				continue
			}

			for _, src := range add.Sources {
				if isRemoteURL(src) {
					// Check if checksum is provided
					if add.Checksum == "" {
						diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
							WithSeverity(r.Severity()).
							WithMessagef("ADD fetches remote URL %q without checksum verification", src).
							WithPos(add.Pos()).
							WithContext(ctx.GetLine(add.Pos().Line)).
							WithHelp("Use ADD --checksum=sha256:... or prefer: RUN curl -o file URL && echo 'CHECKSUM file' | sha256sum -c -").
							Build()
						diags = append(diags, diag)
					}
				}
			}
		}
	}

	return diags
}

func isRemoteURL(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "ftp://")
}

func init() {
	Register(&SEC007AddRemote{})
}
