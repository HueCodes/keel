package security

import (
	"regexp"
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC002SecretsEnv checks for secrets in ENV or ARG instructions
type SEC002SecretsEnv struct{}

func (r *SEC002SecretsEnv) ID() string          { return "SEC002" }
func (r *SEC002SecretsEnv) Name() string        { return "secrets-in-env" }
func (r *SEC002SecretsEnv) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC002SecretsEnv) Severity() analyzer.Severity { return analyzer.SeverityError }

func (r *SEC002SecretsEnv) Description() string {
	return "Secrets should not be passed via ENV or ARG instructions as they are visible in image history."
}

var secretPatterns = []struct {
	pattern *regexp.Regexp
	name    string
}{
	{regexp.MustCompile(`(?i)(password|passwd|pwd)`), "password"},
	{regexp.MustCompile(`(?i)(secret|api_?key|apikey|auth_?token)`), "secret/API key"},
	{regexp.MustCompile(`(?i)(private_?key|priv_?key)`), "private key"},
	{regexp.MustCompile(`(?i)(access_?key|secret_?key)`), "access key"},
	{regexp.MustCompile(`(?i)(credentials?|creds?)`), "credentials"},
	{regexp.MustCompile(`(?i)(token)$`), "token"},
	{regexp.MustCompile(`(?i)^(aws_|azure_|gcp_|github_|gitlab_)`), "cloud/service credential"},
}

func (r *SEC002SecretsEnv) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			switch v := inst.(type) {
			case *parser.EnvInstruction:
				for _, kv := range v.Variables {
					if secretType := isSecretKey(kv.Key); secretType != "" {
						diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
							WithSeverity(r.Severity()).
							WithMessagef("ENV variable %q appears to contain a %s", kv.Key, secretType).
							WithPos(v.Pos()).
							WithContext(ctx.GetLine(v.Pos().Line)).
							WithHelp("Use Docker secrets, BuildKit secrets (--mount=type=secret), or runtime environment variables instead").
							Build()
						diags = append(diags, diag)
					}
				}
			case *parser.ArgInstruction:
				if secretType := isSecretKey(v.Name); secretType != "" {
					diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
						WithSeverity(r.Severity()).
						WithMessagef("ARG %q appears to contain a %s", v.Name, secretType).
						WithPos(v.Pos()).
						WithContext(ctx.GetLine(v.Pos().Line)).
						WithHelp("ARG values are visible in image history. Use BuildKit secrets (--mount=type=secret) instead").
						Build()
					diags = append(diags, diag)
				}
			}
		}
	}

	return diags
}

func isSecretKey(key string) string {
	key = strings.ToLower(key)
	for _, p := range secretPatterns {
		if p.pattern.MatchString(key) {
			return p.name
		}
	}
	return ""
}

func init() {
	Register(&SEC002SecretsEnv{})
}
