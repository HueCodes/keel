package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HueCodes/keel/internal/parser"
)

// loadBenchFixture loads a benchmark Dockerfile fixture
func loadBenchFixture(name string) string {
	path := filepath.Join("..", "..", "testdata", "bench", name+".dockerfile")
	data, err := os.ReadFile(path)
	if err != nil {
		return getInlineFixture(name)
	}
	return string(data)
}

// getInlineFixture returns inline fixtures for CI environments
func getInlineFixture(name string) string {
	switch name {
	case "simple":
		return `FROM alpine:3.18
RUN apk add --no-cache curl
WORKDIR /app
CMD ["curl", "-s", "http://example.com"]
`
	case "medium":
		return `FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
ENV APP_HOME=/app
RUN apt-get update && apt-get install -y --no-install-recommends curl wget git
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY --chown=node:node src/ ./src/
RUN npm run build
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost:3000/health || exit 1
USER node
CMD ["node", "dist/server.js"]
`
	case "complex":
		return `ARG GO_VERSION=1.21
FROM golang:${GO_VERSION}-alpine AS builder
RUN apk add --no-cache git make gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server

FROM node:18-alpine AS frontend
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM alpine:3.18 AS production
LABEL org.opencontainers.image.title="App" maintainer="team@example.com"
RUN apk add --no-cache ca-certificates tzdata tini
RUN addgroup -g 1000 app && adduser -u 1000 -G app -D app
WORKDIR /app
COPY --from=builder --chown=app:app /app/server ./
COPY --from=frontend --chown=app:app /build/dist ./public/
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s CMD curl -f http://localhost:8080/health || exit 1
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./server"]
`
	default:
		return `FROM alpine:3.18
RUN echo "default"
`
	}
}

// mockRule is a simple rule for benchmarking
type mockRule struct {
	id string
}

func (r *mockRule) ID() string       { return r.id }
func (r *mockRule) Category() Category { return CategoryBestPractice }
func (r *mockRule) Severity() Severity { return SeverityWarning }

func (r *mockRule) Check(df *parser.Dockerfile, ctx *RuleContext) []Diagnostic {
	var diags []Diagnostic
	// Simple check - iterate all instructions
	for _, stage := range df.Stages {
		for range stage.Instructions {
			// Just iterate, no actual diagnostics for benchmarking overhead
		}
	}
	return diags
}

// mockRuleWithDiags creates diagnostics for benchmarking
type mockRuleWithDiags struct {
	id string
}

func (r *mockRuleWithDiags) ID() string       { return r.id }
func (r *mockRuleWithDiags) Category() Category { return CategoryBestPractice }
func (r *mockRuleWithDiags) Severity() Severity { return SeverityWarning }

func (r *mockRuleWithDiags) Check(df *parser.Dockerfile, ctx *RuleContext) []Diagnostic {
	var diags []Diagnostic
	// Create a diagnostic for each instruction
	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			diags = append(diags, Diagnostic{
				Rule:     r.id,
				Category: CategoryBestPractice,
				Severity: SeverityWarning,
				Message:  "Mock diagnostic",
				Pos:      inst.Pos(),
			})
		}
	}
	return diags
}

func BenchmarkAnalyzer_SingleRule_Simple(b *testing.B) {
	source := loadBenchFixture("simple")
	df, _ := parser.Parse(source)
	a := New(WithRules(&mockRule{id: "MOCK001"}))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.Analyze(df, "Dockerfile", source)
	}
}

func BenchmarkAnalyzer_SingleRule_Complex(b *testing.B) {
	source := loadBenchFixture("complex")
	df, _ := parser.Parse(source)
	a := New(WithRules(&mockRule{id: "MOCK001"}))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.Analyze(df, "Dockerfile", source)
	}
}

func BenchmarkAnalyzer_MultipleRules(b *testing.B) {
	source := loadBenchFixture("complex")
	df, _ := parser.Parse(source)

	// Create 10 mock rules
	rules := make([]Rule, 10)
	for i := range rules {
		rules[i] = &mockRule{id: "MOCK" + string(rune('0'+i))}
	}
	a := New(WithRules(rules...))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.Analyze(df, "Dockerfile", source)
	}
}

func BenchmarkAnalyzer_WithDiagnostics(b *testing.B) {
	source := loadBenchFixture("complex")
	df, _ := parser.Parse(source)
	a := New(WithRules(&mockRuleWithDiags{id: "MOCK001"}))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.Analyze(df, "Dockerfile", source)
	}
}

func BenchmarkAnalyzer_DiagnosticSorting(b *testing.B) {
	source := loadBenchFixture("complex")
	df, _ := parser.Parse(source)

	// Create 5 rules that all create diagnostics
	rules := make([]Rule, 5)
	for i := range rules {
		rules[i] = &mockRuleWithDiags{id: "MOCK" + string(rune('0'+i))}
	}
	a := New(WithRules(rules...))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.Analyze(df, "Dockerfile", source)
	}
}

func BenchmarkAnalyzer_FullPipeline(b *testing.B) {
	source := loadBenchFixture("complex")

	// Simulate a realistic analyzer with multiple rules
	rules := make([]Rule, 10)
	for i := range rules {
		if i%2 == 0 {
			rules[i] = &mockRule{id: "MOCK" + string(rune('A'+i))}
		} else {
			rules[i] = &mockRuleWithDiags{id: "MOCK" + string(rune('A'+i))}
		}
	}
	a := New(WithRules(rules...))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.AnalyzeSource(source, "Dockerfile")
	}
}

func BenchmarkAnalyzer_SeverityFiltering(b *testing.B) {
	source := loadBenchFixture("complex")
	df, _ := parser.Parse(source)

	a := New(
		WithRules(&mockRuleWithDiags{id: "MOCK001"}),
		WithMinSeverity(SeverityError),
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.Analyze(df, "Dockerfile", source)
	}
}

func BenchmarkAnalyzer_RuleFiltering(b *testing.B) {
	source := loadBenchFixture("complex")
	df, _ := parser.Parse(source)

	rules := make([]Rule, 20)
	for i := range rules {
		rules[i] = &mockRuleWithDiags{id: "MOCK" + string(rune('A'+i))}
	}

	// Only enable 5 of 20 rules
	a := New(
		WithRules(rules...),
		WithEnabled("MOCKA", "MOCKC", "MOCKE", "MOCKG", "MOCKI"),
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		a.Analyze(df, "Dockerfile", source)
	}
}
