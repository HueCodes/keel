package lexer

import (
	"testing"
)

func TestLexerBasicInstructions(t *testing.T) {
	input := `FROM ubuntu:22.04
RUN apt-get update
`
	l := New(input)
	tokens := l.Tokenize()

	expected := []struct {
		typ     TokenType
		literal string
	}{
		{TokenFrom, "FROM"},
		{TokenWord, "ubuntu"},
		{TokenColon, ":"},
		{TokenWord, "22.04"},
		{TokenNewline, "\n"},
		{TokenRun, "RUN"},
		{TokenWord, "apt-get"},
		{TokenWord, "update"},
		{TokenNewline, "\n"},
		{TokenEOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.typ {
			t.Errorf("token %d: expected type %s, got %s", i, exp.typ, tokens[i].Type)
		}
		if tokens[i].Literal != exp.literal {
			t.Errorf("token %d: expected literal %q, got %q", i, exp.literal, tokens[i].Literal)
		}
	}
}

func TestLexerComment(t *testing.T) {
	input := `# This is a comment
FROM alpine`
	l := New(input)
	tokens := l.Tokenize()

	if tokens[0].Type != TokenComment {
		t.Errorf("expected comment, got %s", tokens[0].Type)
	}
	if tokens[0].Literal != "# This is a comment" {
		t.Errorf("expected comment text, got %q", tokens[0].Literal)
	}
}

func TestLexerEscapeDirective(t *testing.T) {
	input := "# escape=`\nFROM alpine"
	l := New(input)
	tokens := l.Tokenize()

	if tokens[0].Type != TokenEscapeDirective {
		t.Errorf("expected escape directive, got %s", tokens[0].Type)
	}
	if l.escapeChar != '`' {
		t.Errorf("expected escape char to be backtick, got %c", l.escapeChar)
	}
}

func TestLexerLineContinuation(t *testing.T) {
	input := `RUN apt-get update \
    && apt-get install -y curl`
	l := New(input)
	tokens := l.Tokenize()

	// Line continuation should be transparent
	types := []TokenType{}
	for _, tok := range tokens {
		if tok.Type != TokenEOF {
			types = append(types, tok.Type)
		}
	}

	// Should see: RUN, apt-get, update, &&, apt-get, install, -y, curl
	if len(types) < 5 {
		t.Errorf("expected at least 5 tokens, got %d", len(types))
	}
	if types[0] != TokenRun {
		t.Errorf("expected RUN, got %s", types[0])
	}
}

func TestLexerQuotedString(t *testing.T) {
	input := `ENV MESSAGE="hello world"`
	l := New(input)
	tokens := l.Tokenize()

	found := false
	for _, tok := range tokens {
		if tok.Type == TokenString && tok.Literal == `"hello world"` {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find quoted string token")
		for _, tok := range tokens {
			t.Logf("  %s", tok)
		}
	}
}

func TestLexerVariable(t *testing.T) {
	input := `RUN echo $HOME ${PATH}`
	l := New(input)
	tokens := l.Tokenize()

	varCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenVariable {
			varCount++
		}
	}
	if varCount != 2 {
		t.Errorf("expected 2 variables, got %d", varCount)
	}
}

func TestLexerFlag(t *testing.T) {
	input := `COPY --from=builder --chmod=755 /app /app`
	l := New(input)
	tokens := l.Tokenize()

	flagCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenFlag {
			flagCount++
		}
	}
	if flagCount != 2 {
		t.Errorf("expected 2 flags, got %d", flagCount)
	}
}

func TestLexerJSON(t *testing.T) {
	input := `CMD ["echo", "hello"]`
	l := New(input)
	tokens := l.Tokenize()

	hasLeftBracket := false
	hasRightBracket := false
	stringCount := 0

	for _, tok := range tokens {
		switch tok.Type {
		case TokenLeftBracket:
			hasLeftBracket = true
		case TokenRightBracket:
			hasRightBracket = true
		case TokenString:
			stringCount++
		}
	}

	if !hasLeftBracket || !hasRightBracket {
		t.Error("expected brackets for JSON array")
	}
	if stringCount != 2 {
		t.Errorf("expected 2 strings, got %d", stringCount)
	}
}

func TestLexerHeredoc(t *testing.T) {
	input := `RUN <<EOF
echo hello
echo world
EOF
`
	l := New(input)
	tokens := l.Tokenize()

	heredocFound := false
	for _, tok := range tokens {
		if tok.Type == TokenHeredoc {
			heredocFound = true
			break
		}
	}
	if !heredocFound {
		t.Error("expected heredoc token")
		for _, tok := range tokens {
			t.Logf("  %s", tok)
		}
	}
}

func TestLexerMultiStage(t *testing.T) {
	input := `FROM golang:1.21 AS builder
RUN go build -o /app

FROM alpine:3.18
COPY --from=builder /app /app
`
	l := New(input)
	tokens := l.Tokenize()

	fromCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenFrom {
			fromCount++
		}
	}
	if fromCount != 2 {
		t.Errorf("expected 2 FROM instructions, got %d", fromCount)
	}
}

func TestLexerPosition(t *testing.T) {
	input := `FROM alpine
RUN echo`
	l := New(input)
	tokens := l.Tokenize()

	// First token should be at line 1, column 1
	if tokens[0].Pos.Line != 1 || tokens[0].Pos.Column != 1 {
		t.Errorf("expected FROM at 1:1, got %s", tokens[0].Pos)
	}

	// RUN should be at line 2
	for _, tok := range tokens {
		if tok.Type == TokenRun {
			if tok.Pos.Line != 2 {
				t.Errorf("expected RUN at line 2, got line %d", tok.Pos.Line)
			}
			break
		}
	}
}

func TestLexerAllInstructions(t *testing.T) {
	input := `FROM base
RUN cmd
CMD ["x"]
LABEL a=b
MAINTAINER old
EXPOSE 80
ENV K=V
ADD src dst
COPY src dst
ENTRYPOINT ["x"]
VOLUME /data
USER nobody
WORKDIR /app
ARG VAR
ONBUILD RUN x
STOPSIGNAL SIGTERM
HEALTHCHECK CMD curl
SHELL ["/bin/sh"]
`
	l := New(input)
	tokens := l.Tokenize()

	instructions := map[TokenType]bool{}
	for _, tok := range tokens {
		if tok.IsInstruction() {
			instructions[tok.Type] = true
		}
	}

	expectedInstructions := []TokenType{
		TokenFrom, TokenRun, TokenCmd, TokenLabel, TokenMaintainer,
		TokenExpose, TokenEnv, TokenAdd, TokenCopy, TokenEntrypoint,
		TokenVolume, TokenUser, TokenWorkdir, TokenArg, TokenOnbuild,
		TokenStopsignal, TokenHealthcheck, TokenShell,
	}

	for _, exp := range expectedInstructions {
		if !instructions[exp] {
			t.Errorf("missing instruction: %s", exp)
		}
	}
}
