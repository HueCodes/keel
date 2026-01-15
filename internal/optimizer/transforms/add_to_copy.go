package transforms

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// AddToCopyTransform replaces ADD with COPY when ADD features aren't needed
type AddToCopyTransform struct{}

func (t *AddToCopyTransform) Name() string {
	return "add-to-copy"
}

func (t *AddToCopyTransform) Description() string {
	return "Replace ADD with COPY when special features aren't used"
}

func (t *AddToCopyTransform) Rules() []string {
	return []string{"BP002"}
}

func (t *AddToCopyTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		newInstructions := make([]parser.Instruction, 0, len(stage.Instructions))

		for _, inst := range stage.Instructions {
			add, ok := inst.(*parser.AddInstruction)
			if !ok {
				newInstructions = append(newInstructions, inst)
				continue
			}

			// Check if ADD features are needed (URL or tar extraction)
			if needsAddFeatures(add.Sources) {
				newInstructions = append(newInstructions, inst)
				continue
			}

			// Convert ADD to COPY
			copy := &parser.CopyInstruction{
				BaseInstruction: add.BaseInstruction,
				Sources:         add.Sources,
				Destination:     add.Destination,
				Chown:           add.Chown,
				Chmod:           add.Chmod,
			}
			newInstructions = append(newInstructions, copy)
			changed = true
		}

		stage.Instructions = newInstructions
	}

	return changed
}

// needsAddFeatures returns true if any source requires ADD features
func needsAddFeatures(sources []string) bool {
	for _, src := range sources {
		if isRemoteURL(src) || isCompressedArchive(src) {
			return true
		}
	}
	return false
}

// isRemoteURL checks if the source is a URL
func isRemoteURL(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "ftp://")
}

// isCompressedArchive checks if the source is a tar file (ADD auto-extracts these)
func isCompressedArchive(s string) bool {
	lower := strings.ToLower(s)
	tarSuffixes := []string{
		".tar",
		".tar.gz",
		".tgz",
		".tar.bz2",
		".tbz2",
		".tar.xz",
		".txz",
		".tar.zst",
		".tar.lz",
		".tar.lzma",
	}
	for _, suffix := range tarSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}
