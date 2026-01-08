package security

import (
	"regexp"
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC004CurlPipe checks for curl/wget piped to shell
type SEC004CurlPipe struct{}

func (r *SEC004CurlPipe) ID() string          { return "SEC004" }
func (r *SEC004CurlPipe) Name() string        { return "curl-pipe-shell" }
func (r *SEC004CurlPipe) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC004CurlPipe) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *SEC004CurlPipe) Description() string {
	return "curl/wget piped to shell is dangerous. Downloads should be verified before execution."
}

var curlPipePattern = regexp.MustCompile(`(curl|wget)\s+[^|]+\|\s*(sh|bash|zsh|dash|ksh)`)
var curlBashPattern = regexp.MustCompile(`(bash|sh)\s+-c\s+["']?\$\((curl|wget)`)
var curlBashPattern2 = regexp.MustCompile(`(bash|sh)\s+<\(\s*(curl|wget)`)

func (r *SEC004CurlPipe) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
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

			// Check various patterns
			if curlPipePattern.MatchString(cmd) ||
				curlBashPattern.MatchString(cmd) ||
				curlBashPattern2.MatchString(cmd) ||
				isCurlPipe(cmd) {

				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessage("curl/wget output piped directly to shell").
					WithPos(run.Pos()).
					WithContext(ctx.GetLine(run.Pos().Line)).
					WithHelp("Download the script first, verify its checksum, then execute. Example: curl -o script.sh URL && sha256sum -c script.sha256 && sh script.sh").
					Build()
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

func isCurlPipe(cmd string) bool {
	// Look for patterns like: curl URL | bash
	parts := strings.Split(cmd, "|")
	if len(parts) < 2 {
		return false
	}

	for i := 0; i < len(parts)-1; i++ {
		left := strings.TrimSpace(parts[i])
		right := strings.TrimSpace(parts[i+1])

		// Check if left side has curl/wget
		if strings.Contains(left, "curl") || strings.Contains(left, "wget") {
			// Check if right side is a shell
			shells := []string{"sh", "bash", "zsh", "dash", "ksh"}
			for _, shell := range shells {
				if strings.HasPrefix(right, shell) || right == shell {
					return true
				}
			}
		}
	}

	return false
}

func init() {
	Register(&SEC004CurlPipe{})
}
