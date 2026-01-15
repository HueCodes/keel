package transforms

import (
	"path"
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// WorkdirAbsoluteTransform converts relative WORKDIR paths to absolute
type WorkdirAbsoluteTransform struct{}

func (t *WorkdirAbsoluteTransform) Name() string {
	return "workdir-absolute"
}

func (t *WorkdirAbsoluteTransform) Description() string {
	return "Convert relative WORKDIR paths to absolute"
}

func (t *WorkdirAbsoluteTransform) Rules() []string {
	return []string{"BP005"}
}

func (t *WorkdirAbsoluteTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		// Each stage starts with root as the working directory
		currentDir := "/"

		for _, inst := range stage.Instructions {
			wd, ok := inst.(*parser.WorkdirInstruction)
			if !ok {
				continue
			}

			workdirPath := wd.Path

			// Skip variable expansion - we can't resolve these at lint time
			if strings.HasPrefix(workdirPath, "$") || strings.Contains(workdirPath, "${") {
				// Can't resolve, but try to track best-effort
				if strings.HasPrefix(workdirPath, "/") {
					currentDir = workdirPath
				}
				continue
			}

			// If already absolute, just update current directory tracking
			if strings.HasPrefix(workdirPath, "/") {
				currentDir = path.Clean(workdirPath)
				continue
			}

			// Relative path - convert to absolute
			absolutePath := joinPath(currentDir, workdirPath)
			wd.Path = absolutePath
			currentDir = absolutePath
			changed = true
		}
	}

	return changed
}

// joinPath joins a base directory with a relative path
func joinPath(base, rel string) string {
	// Clean up any trailing slashes from base
	base = strings.TrimSuffix(base, "/")

	// Handle case where base is just "/"
	if base == "" {
		base = "/"
	}

	// Join the paths
	joined := base + "/" + rel

	// Clean the path to handle . and ..
	cleaned := path.Clean(joined)

	// Ensure it's still absolute
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}

	return cleaned
}
