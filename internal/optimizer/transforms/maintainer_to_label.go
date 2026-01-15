package transforms

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// MaintainerToLabelTransform converts deprecated MAINTAINER to LABEL maintainer
type MaintainerToLabelTransform struct{}

func (t *MaintainerToLabelTransform) Name() string {
	return "maintainer-to-label"
}

func (t *MaintainerToLabelTransform) Description() string {
	return "Convert deprecated MAINTAINER to LABEL maintainer="
}

func (t *MaintainerToLabelTransform) Rules() []string {
	return []string{"BP004"}
}

func (t *MaintainerToLabelTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		newInstructions := make([]parser.Instruction, 0, len(stage.Instructions))

		for _, inst := range stage.Instructions {
			maint, ok := inst.(*parser.MaintainerInstruction)
			if !ok {
				newInstructions = append(newInstructions, inst)
				continue
			}

			// Convert MAINTAINER to LABEL
			label := &parser.LabelInstruction{
				BaseInstruction: maint.BaseInstruction,
				Labels: []parser.KeyValue{
					{Key: "maintainer", Value: maint.Maintainer},
				},
			}
			newInstructions = append(newInstructions, label)
			changed = true
		}

		stage.Instructions = newInstructions
	}

	return changed
}
