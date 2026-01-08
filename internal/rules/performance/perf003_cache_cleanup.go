package performance

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// PERF003CacheCleanup checks for package manager cache not cleaned in same layer
type PERF003CacheCleanup struct{}

func (r *PERF003CacheCleanup) ID() string          { return "PERF003" }
func (r *PERF003CacheCleanup) Name() string        { return "cache-not-cleaned" }
func (r *PERF003CacheCleanup) Category() analyzer.Category { return analyzer.CategoryPerformance }
func (r *PERF003CacheCleanup) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *PERF003CacheCleanup) Description() string {
	return "Package manager cache should be cleaned in the same RUN instruction to reduce layer size."
}

type pkgManager struct {
	install   string
	cleanup   []string
}

var packageManagers = []pkgManager{
	{
		install: "apt-get install",
		cleanup: []string{"rm -rf /var/lib/apt/lists/*", "apt-get clean"},
	},
	{
		install: "apt install",
		cleanup: []string{"rm -rf /var/lib/apt/lists/*", "apt-get clean"},
	},
	{
		install: "apk add",
		cleanup: []string{"--no-cache", "rm -rf /var/cache/apk/*"},
	},
	{
		install: "yum install",
		cleanup: []string{"yum clean all", "rm -rf /var/cache/yum"},
	},
	{
		install: "dnf install",
		cleanup: []string{"dnf clean all"},
	},
	{
		install: "pip install",
		cleanup: []string{"--no-cache-dir", "rm -rf ~/.cache/pip"},
	},
	{
		install: "pip3 install",
		cleanup: []string{"--no-cache-dir", "rm -rf ~/.cache/pip"},
	},
	{
		install: "npm install",
		cleanup: []string{"npm cache clean", "rm -rf ~/.npm"},
	},
	{
		install: "yarn",
		cleanup: []string{"yarn cache clean"},
	},
}

func (r *PERF003CacheCleanup) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
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

			for _, pm := range packageManagers {
				if strings.Contains(cmd, pm.install) {
					hasCleanup := false
					for _, cleanup := range pm.cleanup {
						if strings.Contains(cmd, cleanup) {
							hasCleanup = true
							break
						}
					}

					if !hasCleanup {
						diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
							WithSeverity(r.Severity()).
							WithMessagef("Package manager cache not cleaned after %s", pm.install).
							WithPos(run.Pos()).
							WithContext(ctx.GetLine(run.Pos().Line)).
							WithHelp("Add cache cleanup in the same RUN instruction: " + strings.Join(pm.cleanup, " or ")).
							Build()
						diags = append(diags, diag)
					}
				}
			}
		}
	}

	return diags
}

func init() {
	Register(&PERF003CacheCleanup{})
}
