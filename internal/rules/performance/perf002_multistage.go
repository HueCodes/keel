package performance

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/lexer"
	"github.com/HueCodes/keel/internal/parser"
)

// PERF002MultiStage checks for builds that could benefit from multi-stage
type PERF002MultiStage struct{}

func (r *PERF002MultiStage) ID() string          { return "PERF002" }
func (r *PERF002MultiStage) Name() string        { return "missing-multistage" }
func (r *PERF002MultiStage) Category() analyzer.Category { return analyzer.CategoryPerformance }
func (r *PERF002MultiStage) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *PERF002MultiStage) Description() string {
	return "Build tools in the final image increase size. Use multi-stage builds to separate build and runtime environments."
}

// Build tools that shouldn't be in final images
var buildTools = []string{
	"gcc", "g++", "make", "cmake", "cargo", "rustc",
	"go build", "go install", "go mod",
	"npm run build", "yarn build",
	"mvn ", "gradle ", "./gradlew",
	"dotnet build", "dotnet publish",
}

// Base images that are typically build environments
var buildImages = []string{
	"golang", "rust", "node", "maven", "gradle", "dotnet/sdk",
}

func (r *PERF002MultiStage) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	// Only relevant if single stage
	if len(df.Stages) != 1 {
		return diags
	}

	stage := df.Stages[0]
	hasBuildCommand := false
	var buildPos lexer.Position

	// Check if using a build base image
	isBuildImage := false
	for _, img := range buildImages {
		if strings.Contains(strings.ToLower(stage.From.Image), img) {
			isBuildImage = true
			break
		}
	}

	// Check for build commands
	for _, inst := range stage.Instructions {
		run, ok := inst.(*parser.RunInstruction)
		if !ok {
			continue
		}

		for _, tool := range buildTools {
			if strings.Contains(run.Command, tool) {
				hasBuildCommand = true
				buildPos = run.Pos()
				break
			}
		}
	}

	if isBuildImage && hasBuildCommand {
		diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
			WithSeverity(r.Severity()).
			WithMessage("Single-stage build with build tools will produce a large image").
			WithPos(buildPos).
			WithContext(ctx.GetLine(buildPos.Line)).
			WithHelp("Use multi-stage build: build in one stage, copy only the artifact to a minimal base image (e.g., alpine, distroless)").
			Build()
		diags = append(diags, diag)
	}

	return diags
}

func init() {
	Register(&PERF002MultiStage{})
}
