package formatter

import (
	"strings"
	"testing"
)

func TestFormatter_InstructionCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase from",
			input:    "from ubuntu:22.04\n",
			expected: "FROM ubuntu:22.04\n",
		},
		{
			name:     "lowercase run",
			input:    "FROM alpine\nrun apk add curl\n",
			expected: "FROM alpine\nRUN apk add curl\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(DefaultOptions())
			result, err := f.FormatSource(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Formatted != tt.expected {
				t.Errorf("got:\n%s\nwant:\n%s", result.Formatted, tt.expected)
			}
		})
	}
}

func TestFormatter_MultiValueAlignment(t *testing.T) {
	input := `FROM alpine
ENV FOO=bar BAZ=qux LONGKEY=value
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be formatted with alignment
	if !strings.Contains(result.Formatted, "ENV FOO") {
		t.Errorf("expected ENV instruction, got:\n%s", result.Formatted)
	}
	if !strings.Contains(result.Formatted, "\\\n") {
		t.Errorf("expected line continuation, got:\n%s", result.Formatted)
	}
}

func TestFormatter_SingleEnvNoAlignment(t *testing.T) {
	input := `FROM alpine
ENV FOO=bar
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Single ENV should be on one line
	expected := "FROM alpine\nENV FOO=bar\n"
	if result.Formatted != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result.Formatted, expected)
	}
}

func TestFormatter_LineContinuation(t *testing.T) {
	input := `FROM alpine
RUN apt-get update && apt-get install -y curl && apt-get clean
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have line continuations
	if !strings.Contains(result.Formatted, " \\\n") {
		t.Errorf("expected line continuation, got:\n%s", result.Formatted)
	}
	if !strings.Contains(result.Formatted, "&& apt-get install") {
		t.Errorf("expected && continuation, got:\n%s", result.Formatted)
	}
}

func TestFormatter_ExecForm(t *testing.T) {
	input := `FROM alpine
CMD ["echo", "hello world"]
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "FROM alpine\nCMD [\"echo\", \"hello world\"]\n"
	if result.Formatted != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result.Formatted, expected)
	}
}

func TestFormatter_CopyFlags(t *testing.T) {
	input := `FROM alpine
COPY --chown=app:app --chmod=755 src/ /app/
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Formatted, "--chown=app:app") {
		t.Errorf("expected --chown flag, got:\n%s", result.Formatted)
	}
	if !strings.Contains(result.Formatted, "--chmod=755") {
		t.Errorf("expected --chmod flag, got:\n%s", result.Formatted)
	}
}

func TestFormatter_MultiStage(t *testing.T) {
	input := `FROM golang:1.21 AS builder
WORKDIR /build
COPY . .
RUN go build

FROM alpine
COPY --from=builder /build/app /app
CMD ["/app"]
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Formatted, "FROM golang:1.21 AS builder") {
		t.Errorf("expected first stage, got:\n%s", result.Formatted)
	}
	if !strings.Contains(result.Formatted, "FROM alpine") {
		t.Errorf("expected second stage, got:\n%s", result.Formatted)
	}
	if !strings.Contains(result.Formatted, "--from=builder") {
		t.Errorf("expected --from flag, got:\n%s", result.Formatted)
	}
}

func TestFormatter_Healthcheck(t *testing.T) {
	input := `FROM alpine
HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost/ || exit 1
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Formatted, "HEALTHCHECK") {
		t.Errorf("expected HEALTHCHECK, got:\n%s", result.Formatted)
	}
	if !strings.Contains(result.Formatted, "--interval=30s") {
		t.Errorf("expected --interval flag, got:\n%s", result.Formatted)
	}
}

func TestFormatter_Labels(t *testing.T) {
	input := `FROM alpine
LABEL version="1.0" description="Test image" maintainer="test@example.com"
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Multiple labels should be aligned
	if !strings.Contains(result.Formatted, "LABEL") {
		t.Errorf("expected LABEL, got:\n%s", result.Formatted)
	}
}

func TestFormatter_HasChanges(t *testing.T) {
	input := "FROM alpine\nRUN echo hello\n"
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasChanges {
		t.Errorf("expected no changes, but HasChanges is true")
	}
}

func TestFormatter_NoChanges(t *testing.T) {
	input := "FROM alpine\nRUN echo hello\n"
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Formatted != input {
		t.Errorf("expected input to remain unchanged:\ngot:\n%s\nwant:\n%s",
			result.Formatted, input)
	}
}

func TestFormatter_BlankLines(t *testing.T) {
	input := `FROM alpine


RUN echo hello



RUN echo world
`
	f := New(DefaultOptions())
	result, err := f.FormatSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at most 1 consecutive blank line
	if strings.Contains(result.Formatted, "\n\n\n") {
		t.Errorf("expected no more than 2 consecutive newlines, got:\n%s", result.Formatted)
	}
}

func TestFormatter_ParseError(t *testing.T) {
	input := "INVALID DOCKERFILE SYNTAX {{{"
	f := New(DefaultOptions())
	_, err := f.FormatSource(input)

	// Should return parse error
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestDiff_NoChanges(t *testing.T) {
	diff := Diff("Dockerfile", "FROM alpine\n", "FROM alpine\n")
	if diff != "" {
		t.Errorf("expected empty diff, got:\n%s", diff)
	}
}

func TestDiff_WithChanges(t *testing.T) {
	original := "from alpine\nrun echo hello\n"
	formatted := "FROM alpine\nRUN echo hello\n"

	diff := Diff("Dockerfile", original, formatted)

	if !strings.Contains(diff, "--- Dockerfile") {
		t.Errorf("expected --- header, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++ Dockerfile") {
		t.Errorf("expected +++ header, got:\n%s", diff)
	}
	if !strings.Contains(diff, "-from alpine") {
		t.Errorf("expected removed line, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+FROM alpine") {
		t.Errorf("expected added line, got:\n%s", diff)
	}
}

func TestQuoteIfNeeded(t *testing.T) {
	f := New(DefaultOptions())

	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "\"with space\""},
		{"", "\"\""},
		{"path/to/file", "path/to/file"},
		{"value=123", "\"value=123\""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := f.quoteIfNeeded(tt.input)
			if result != tt.expected {
				t.Errorf("quoteIfNeeded(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
