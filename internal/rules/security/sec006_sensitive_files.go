package security

import (
	"path/filepath"
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/lexer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC006SensitiveFiles checks for copying sensitive files
type SEC006SensitiveFiles struct{}

func (r *SEC006SensitiveFiles) ID() string          { return "SEC006" }
func (r *SEC006SensitiveFiles) Name() string        { return "sensitive-files" }
func (r *SEC006SensitiveFiles) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC006SensitiveFiles) Severity() analyzer.Severity { return analyzer.SeverityError }

func (r *SEC006SensitiveFiles) Description() string {
	return "Sensitive files should not be copied into Docker images."
}

var sensitivePatterns = []struct {
	pattern string
	desc    string
}{
	{".env", "environment file"},
	{".env.*", "environment file"},
	{"*.pem", "PEM certificate/key"},
	{"*.key", "private key"},
	{"*.p12", "PKCS12 certificate"},
	{"*.pfx", "PKCS12 certificate"},
	{"id_rsa", "SSH private key"},
	{"id_dsa", "SSH private key"},
	{"id_ecdsa", "SSH private key"},
	{"id_ed25519", "SSH private key"},
	{".ssh/*", "SSH files"},
	{".git/*", "Git repository"},
	{".gitconfig", "Git config"},
	{"*.log", "log file"},
	{".dockerenv", "Docker environment"},
	{"docker-compose*.yml", "Docker Compose file"},
	{"docker-compose*.yaml", "Docker Compose file"},
	{".aws/*", "AWS credentials"},
	{".kube/*", "Kubernetes config"},
	{"credentials.json", "credentials file"},
	{"secrets.json", "secrets file"},
	{"*.secret", "secret file"},
	{".npmrc", "NPM config (may contain tokens)"},
	{".pypirc", "PyPI config (may contain tokens)"},
}

func (r *SEC006SensitiveFiles) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			var sources []string
			var pos lexer.Position

			switch v := inst.(type) {
			case *parser.CopyInstruction:
				sources = v.Sources
				pos = v.Pos()
			case *parser.AddInstruction:
				sources = v.Sources
				pos = v.Pos()
			default:
				continue
			}

			for _, src := range sources {
				if sensitive, desc := isSensitiveFile(src); sensitive {
					diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
						WithSeverity(r.Severity()).
						WithMessagef("Copying %s (%s) into image", src, desc).
						WithPos(pos).
						WithContext(ctx.GetLine(pos.Line)).
						WithHelp("Add this file to .dockerignore or use Docker secrets/BuildKit secrets for sensitive data").
						Build()
					diags = append(diags, diag)
				}
			}
		}
	}

	return diags
}

func isSensitiveFile(path string) (bool, string) {
	base := filepath.Base(path)

	for _, p := range sensitivePatterns {
		// Check exact match or glob pattern
		if matched, _ := filepath.Match(p.pattern, base); matched {
			return true, p.desc
		}

		// Check if path contains the pattern
		if strings.Contains(path, strings.TrimPrefix(p.pattern, "*")) {
			return true, p.desc
		}
	}

	return false, ""
}

func init() {
	Register(&SEC006SensitiveFiles{})
}
