package transforms

import (
	"context"
	"errors"
	"testing"

	"github.com/HueCodes/keel/internal/parser"
)

// mockRegistryClient is a mock implementation of RegistryClient for testing
type mockRegistryClient struct {
	digests map[string]string // image:tag -> digest
	err     error
}

func (m *mockRegistryClient) GetDigest(ctx context.Context, image, tag string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	key := image + ":" + tag
	if digest, ok := m.digests[key]; ok {
		return digest, nil
	}
	return "", errors.New("image not found")
}

func TestPinImageTagTransform_Name(t *testing.T) {
	tr := &PinImageTagTransform{}
	if tr.Name() != "pin-image-tag" {
		t.Errorf("expected name 'pin-image-tag', got %s", tr.Name())
	}
}

func TestPinImageTagTransform_Rules(t *testing.T) {
	tr := &PinImageTagTransform{}
	rules := tr.Rules()
	if len(rules) != 1 || rules[0] != "SEC003" {
		t.Errorf("expected rules ['SEC003'], got %v", rules)
	}
}

func TestPinImageTagTransform_NoClient(t *testing.T) {
	// Without a client, no changes should be made
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "ubuntu",
					Tag:   "latest",
				},
			},
		},
	}

	tr := &PinImageTagTransform{} // No client
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes without client")
	}
}

func TestPinImageTagTransform_LatestTag(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "ubuntu",
					Tag:   "latest",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{
				"ubuntu:latest": "sha256:abc123",
			},
		},
	}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	from := df.Stages[0].From
	if from.Digest != "sha256:abc123" {
		t.Errorf("expected digest 'sha256:abc123', got '%s'", from.Digest)
	}
}

func TestPinImageTagTransform_NoTag(t *testing.T) {
	// No tag defaults to latest
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "alpine",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{
				"alpine:latest": "sha256:def456",
			},
		},
	}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	from := df.Stages[0].From
	if from.Digest != "sha256:def456" {
		t.Errorf("expected digest 'sha256:def456', got '%s'", from.Digest)
	}
}

func TestPinImageTagTransform_AlreadyPinned(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image:  "ubuntu",
					Tag:    "22.04",
					Digest: "sha256:existing",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{
				"ubuntu:22.04": "sha256:new",
			},
		},
	}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes for already pinned image")
	}

	from := df.Stages[0].From
	if from.Digest != "sha256:existing" {
		t.Errorf("expected digest to remain 'sha256:existing', got '%s'", from.Digest)
	}
}

func TestPinImageTagTransform_Scratch(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "scratch",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{},
		},
	}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes for scratch image")
	}
}

func TestPinImageTagTransform_ArgVariable(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "$BASE_IMAGE",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{},
		},
	}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes for variable image")
	}
}

func TestPinImageTagTransform_StageReference(t *testing.T) {
	// Multi-stage build with FROM builder
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Name: "builder",
				From: &parser.FromInstruction{
					Image: "golang",
					Tag:   "1.21",
				},
			},
			{
				From: &parser.FromInstruction{
					Image: "builder",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{
				"golang:1.21": "sha256:golang123",
			},
		},
	}
	changed := tr.Transform(df, nil)

	// Only first stage should be pinned
	if !changed {
		t.Error("expected transform to report changes")
	}

	from1 := df.Stages[0].From
	if from1.Digest != "sha256:golang123" {
		t.Errorf("first stage: expected digest, got '%s'", from1.Digest)
	}

	from2 := df.Stages[1].From
	if from2.Digest != "" {
		t.Errorf("second stage: expected no digest (stage ref), got '%s'", from2.Digest)
	}
}

func TestPinImageTagTransform_NetworkError(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "ubuntu",
					Tag:   "latest",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			err: errors.New("network error"),
		},
	}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes on network error")
	}

	from := df.Stages[0].From
	if from.Digest != "" {
		t.Errorf("expected no digest on error, got '%s'", from.Digest)
	}
}

func TestPinImageTagTransform_SpecificTag(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "node",
					Tag:   "18-alpine",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{
				"node:18-alpine": "sha256:node18alpine",
			},
		},
	}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	from := df.Stages[0].From
	if from.Digest != "sha256:node18alpine" {
		t.Errorf("expected digest 'sha256:node18alpine', got '%s'", from.Digest)
	}
	// Tag should be preserved
	if from.Tag != "18-alpine" {
		t.Errorf("expected tag '18-alpine', got '%s'", from.Tag)
	}
}

func TestPinImageTagTransform_MultipleStages(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "golang",
					Tag:   "1.21",
				},
			},
			{
				From: &parser.FromInstruction{
					Image: "alpine",
					Tag:   "3.18",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{
				"golang:1.21": "sha256:golang",
				"alpine:3.18": "sha256:alpine",
			},
		},
	}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	if df.Stages[0].From.Digest != "sha256:golang" {
		t.Errorf("stage 0: expected 'sha256:golang', got '%s'", df.Stages[0].From.Digest)
	}

	if df.Stages[1].From.Digest != "sha256:alpine" {
		t.Errorf("stage 1: expected 'sha256:alpine', got '%s'", df.Stages[1].From.Digest)
	}
}

func TestPinImageTagTransform_PartialFailure(t *testing.T) {
	// First image succeeds, second fails
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				From: &parser.FromInstruction{
					Image: "golang",
					Tag:   "1.21",
				},
			},
			{
				From: &parser.FromInstruction{
					Image: "unknown-image",
					Tag:   "latest",
				},
			},
		},
	}

	tr := &PinImageTagTransform{
		Client: &mockRegistryClient{
			digests: map[string]string{
				"golang:1.21": "sha256:golang",
				// unknown-image not in map, will fail
			},
		},
	}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes (first stage)")
	}

	if df.Stages[0].From.Digest != "sha256:golang" {
		t.Errorf("stage 0: expected 'sha256:golang', got '%s'", df.Stages[0].From.Digest)
	}

	if df.Stages[1].From.Digest != "" {
		t.Errorf("stage 1: expected no digest (failed), got '%s'", df.Stages[1].From.Digest)
	}
}
