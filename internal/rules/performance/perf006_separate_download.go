package performance

import (
	"regexp"
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// PERF006SeparateDownload checks for download and extract in separate layers
type PERF006SeparateDownload struct{}

func (r *PERF006SeparateDownload) ID() string          { return "PERF006" }
func (r *PERF006SeparateDownload) Name() string        { return "separate-download-extract" }
func (r *PERF006SeparateDownload) Category() analyzer.Category { return analyzer.CategoryPerformance }
func (r *PERF006SeparateDownload) Severity() analyzer.Severity { return analyzer.SeverityInfo }

func (r *PERF006SeparateDownload) Description() string {
	return "Download and extract should be in the same RUN instruction to avoid storing the archive in a layer."
}

var downloadPattern = regexp.MustCompile(`(curl|wget)\s+.*\.(tar|tar\.gz|tgz|tar\.bz2|tar\.xz|zip)`)
var extractPattern = regexp.MustCompile(`(tar\s+(-x|x)|unzip|gunzip)`)

func (r *PERF006SeparateDownload) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		var downloadRun *parser.RunInstruction

		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if !ok {
				continue
			}

			cmd := run.Command
			if run.Heredoc != nil {
				cmd = run.Heredoc.Content
			}

			hasDownload := downloadPattern.MatchString(cmd) || strings.Contains(cmd, "curl") && containsArchiveExt(cmd)
			hasExtract := extractPattern.MatchString(cmd)

			if hasDownload && !hasExtract {
				// Download without extract in same command
				downloadRun = run
			} else if hasExtract && downloadRun != nil {
				// Extract in different RUN than download
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessage("Download and extract are in separate RUN instructions").
					WithPos(downloadRun.Pos()).
					WithContext(ctx.GetLine(downloadRun.Pos().Line)).
					WithHelp("Combine download and extract in the same RUN instruction, then remove the archive: curl -o file.tar.gz URL && tar xf file.tar.gz && rm file.tar.gz").
					Build()
				diags = append(diags, diag)
				downloadRun = nil
			}
		}
	}

	return diags
}

func containsArchiveExt(s string) bool {
	exts := []string{".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tar.xz", ".zip"}
	for _, ext := range exts {
		if strings.Contains(s, ext) {
			return true
		}
	}
	return false
}

func init() {
	Register(&PERF006SeparateDownload{})
}
