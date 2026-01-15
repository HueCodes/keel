package lexer

import (
	"os"
	"path/filepath"
	"testing"
)

// loadBenchFixture loads a benchmark Dockerfile fixture
func loadBenchFixture(name string) string {
	path := filepath.Join("..", "..", "testdata", "bench", name+".dockerfile")
	data, err := os.ReadFile(path)
	if err != nil {
		// Fall back to inline fixtures for CI
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

func BenchmarkLexer_Simple(b *testing.B) {
	input := loadBenchFixture("simple")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := New(input)
		l.Tokenize()
	}
}

func BenchmarkLexer_Medium(b *testing.B) {
	input := loadBenchFixture("medium")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := New(input)
		l.Tokenize()
	}
}

func BenchmarkLexer_Complex(b *testing.B) {
	input := loadBenchFixture("complex")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := New(input)
		l.Tokenize()
	}
}

func BenchmarkLexer_NextToken(b *testing.B) {
	input := loadBenchFixture("medium")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == TokenEOF {
				break
			}
		}
	}
}

func BenchmarkLexer_LargeFile(b *testing.B) {
	// Create a large Dockerfile by repeating medium content
	base := loadBenchFixture("medium")
	var large string
	for i := 0; i < 10; i++ {
		large += base
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := New(large)
		l.Tokenize()
	}
}
