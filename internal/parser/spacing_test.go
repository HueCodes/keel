package parser

import (
	"testing"
)

func TestCollectRestOfLineSpacing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double ampersand",
			input:    "FROM ubuntu\nRUN apt-get update && apt-get install -y curl",
			expected: "apt-get update && apt-get install -y curl",
		},
		{
			name:     "with cleanup pattern",
			input:    "FROM ubuntu\nRUN apt-get install -y curl && rm -rf /var/lib/apt/lists/*",
			expected: "apt-get install -y curl && rm -rf /var/lib/apt/lists/*",
		},
		{
			name:     "pipe",
			input:    "FROM ubuntu\nRUN cat file | grep pattern",
			expected: "cat file | grep pattern",
		},
		{
			name:     "semicolon",
			input:    "FROM ubuntu\nRUN echo hello; echo world",
			expected: "echo hello; echo world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, errs := Parse(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse errors: %v", errs)
			}
			if len(df.Stages) == 0 || len(df.Stages[0].Instructions) == 0 {
				t.Fatal("Expected at least one instruction")
			}
			run, ok := df.Stages[0].Instructions[0].(*RunInstruction)
			if !ok {
				t.Fatal("Expected RunInstruction")
			}
			if run.Command != tt.expected {
				t.Errorf("got %q, want %q", run.Command, tt.expected)
			}
		})
	}
}
