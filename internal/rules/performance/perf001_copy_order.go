package performance

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// PERF001CopyOrder checks for COPY/ADD before RUN that could invalidate cache
type PERF001CopyOrder struct{}

func (r *PERF001CopyOrder) ID() string          { return "PERF001" }
func (r *PERF001CopyOrder) Name() string        { return "copy-before-run" }
func (r *PERF001CopyOrder) Category() analyzer.Category { return analyzer.CategoryPerformance }
func (r *PERF001CopyOrder) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *PERF001CopyOrder) Description() string {
	return "COPY/ADD instructions before RUN can invalidate Docker cache. Copy dependency files first, then run install commands, then copy the rest."
}

func (r *PERF001CopyOrder) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		// Look for BAD pattern: COPY . before any RUN install, without prior dependency-only copy
		// GOOD pattern: COPY go.mod -> RUN go mod download -> COPY . -> RUN go build
		// BAD pattern: COPY . -> RUN go mod download (or any install)

		var broadCopy *parser.CopyInstruction
		var hadDependencyInstall bool

		for _, inst := range stage.Instructions {
			switch v := inst.(type) {
			case *parser.CopyInstruction:
				if isBroadCopy(v.Sources) {
					// Broad copy found - only bad if we haven't done dependency install yet
					if !hadDependencyInstall {
						broadCopy = v
					}
				}
			case *parser.RunInstruction:
				if isDependencyInstall(v.Command) {
					if broadCopy != nil {
						// BAD: broad copy happened before dependency install
						diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
							WithSeverity(r.Severity()).
							WithMessage("Broad COPY before dependency install invalidates cache on any file change").
							WithPos(broadCopy.Pos()).
							WithContext(ctx.GetLine(broadCopy.Pos().Line)).
							WithHelp("Copy only dependency files first (package.json, requirements.txt, go.mod, etc.), run install, then COPY the rest").
							Build()
						diags = append(diags, diag)
						broadCopy = nil
					}
					hadDependencyInstall = true
				}
			}
		}
	}

	return diags
}

// isDependencyInstall checks for dependency installation commands (not build commands)
func isDependencyInstall(cmd string) bool {
	installPatterns := []string{
		"npm install", "npm ci", "yarn install", "yarn add",
		"pip install", "pip3 install",
		"go mod download", "go get",
		"bundle install", "gem install",
		"composer install",
		"cargo fetch",
		"apt-get install", "apt install", "apk add", "yum install", "dnf install",
	}

	for _, pattern := range installPatterns {
		if containsSubstring(cmd, pattern) {
			return true
		}
	}
	return false
}

func isBroadCopy(sources []string) bool {
	for _, src := range sources {
		if src == "." || src == "./" || src == "*" || src == "./*" {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func init() {
	Register(&PERF001CopyOrder{})
}
