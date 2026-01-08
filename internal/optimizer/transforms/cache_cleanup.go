package transforms

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// AddCacheCleanupTransform adds package manager cache cleanup
type AddCacheCleanupTransform struct{}

func (t *AddCacheCleanupTransform) Name() string {
	return "add-cache-cleanup"
}

func (t *AddCacheCleanupTransform) Description() string {
	return "Add package manager cache cleanup to reduce image size"
}

func (t *AddCacheCleanupTransform) Rules() []string {
	return []string{"PERF003"}
}

type pkgManagerCleanup struct {
	detect  string
	cleanup string
}

var cleanupCommands = []pkgManagerCleanup{
	{
		detect:  "apt-get install",
		cleanup: " && rm -rf /var/lib/apt/lists/*",
	},
	{
		detect:  "apt install",
		cleanup: " && rm -rf /var/lib/apt/lists/*",
	},
	{
		detect:  "yum install",
		cleanup: " && yum clean all && rm -rf /var/cache/yum",
	},
	{
		detect:  "dnf install",
		cleanup: " && dnf clean all",
	},
}

// For apk, we modify the command to use --no-cache
var apkPattern = "apk add"

func (t *AddCacheCleanupTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if !ok {
				continue
			}

			// Skip heredocs and exec form
			if run.Heredoc != nil || run.IsExec {
				continue
			}

			newCmd := addCleanupToCommand(run.Command, &changed)
			if newCmd != run.Command {
				run.Command = newCmd
			}
		}
	}

	return changed
}

func addCleanupToCommand(cmd string, changed *bool) string {
	// Handle apk specially - add --no-cache flag
	if strings.Contains(cmd, apkPattern) && !strings.Contains(cmd, "--no-cache") {
		cmd = strings.Replace(cmd, apkPattern, "apk add --no-cache", 1)
		*changed = true
	}

	// Handle other package managers - add cleanup at end
	for _, pm := range cleanupCommands {
		if strings.Contains(cmd, pm.detect) {
			// Check if cleanup already exists
			hasCleanup := false
			for _, check := range []string{
				"rm -rf /var/lib/apt/lists",
				"apt-get clean",
				"yum clean all",
				"dnf clean all",
			} {
				if strings.Contains(cmd, check) {
					hasCleanup = true
					break
				}
			}

			if !hasCleanup {
				cmd = strings.TrimRight(cmd, " \t\n") + pm.cleanup
				*changed = true
				break // Only add one cleanup
			}
		}
	}

	return cmd
}
