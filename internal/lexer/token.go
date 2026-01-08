package lexer

import "fmt"

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenNewline
	TokenComment
	TokenWhitespace

	// Dockerfile instructions
	TokenFrom
	TokenRun
	TokenCmd
	TokenLabel
	TokenMaintainer // deprecated but still parsed
	TokenExpose
	TokenEnv
	TokenAdd
	TokenCopy
	TokenEntrypoint
	TokenVolume
	TokenUser
	TokenWorkdir
	TokenArg
	TokenOnbuild
	TokenStopsignal
	TokenHealthcheck
	TokenShell

	// Literals and identifiers
	TokenString       // quoted string
	TokenWord         // unquoted word/identifier
	TokenVariable     // $VAR or ${VAR}
	TokenHeredoc      // <<EOF ... EOF
	TokenHeredocStart // <<EOF or <<-EOF
	TokenHeredocEnd   // EOF (the delimiter)

	// Operators and punctuation
	TokenEquals       // =
	TokenColon        // :
	TokenAt           // @
	TokenComma        // ,
	TokenLeftBracket  // [
	TokenRightBracket // ]
	TokenBackslash    // \ (line continuation)

	// Flags (--flag=value)
	TokenFlag

	// Escape directive
	TokenEscapeDirective
)

var tokenNames = map[TokenType]string{
	TokenEOF:             "EOF",
	TokenNewline:         "NEWLINE",
	TokenComment:         "COMMENT",
	TokenWhitespace:      "WHITESPACE",
	TokenFrom:            "FROM",
	TokenRun:             "RUN",
	TokenCmd:             "CMD",
	TokenLabel:           "LABEL",
	TokenMaintainer:      "MAINTAINER",
	TokenExpose:          "EXPOSE",
	TokenEnv:             "ENV",
	TokenAdd:             "ADD",
	TokenCopy:            "COPY",
	TokenEntrypoint:      "ENTRYPOINT",
	TokenVolume:          "VOLUME",
	TokenUser:            "USER",
	TokenWorkdir:         "WORKDIR",
	TokenArg:             "ARG",
	TokenOnbuild:         "ONBUILD",
	TokenStopsignal:      "STOPSIGNAL",
	TokenHealthcheck:     "HEALTHCHECK",
	TokenShell:           "SHELL",
	TokenString:          "STRING",
	TokenWord:            "WORD",
	TokenVariable:        "VARIABLE",
	TokenHeredoc:         "HEREDOC",
	TokenHeredocStart:    "HEREDOC_START",
	TokenHeredocEnd:      "HEREDOC_END",
	TokenEquals:          "EQUALS",
	TokenColon:           "COLON",
	TokenAt:              "AT",
	TokenComma:           "COMMA",
	TokenLeftBracket:     "LEFT_BRACKET",
	TokenRightBracket:    "RIGHT_BRACKET",
	TokenBackslash:       "BACKSLASH",
	TokenFlag:            "FLAG",
	TokenEscapeDirective: "ESCAPE_DIRECTIVE",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// instructionKeywords maps instruction names to token types
var instructionKeywords = map[string]TokenType{
	"FROM":        TokenFrom,
	"RUN":         TokenRun,
	"CMD":         TokenCmd,
	"LABEL":       TokenLabel,
	"MAINTAINER":  TokenMaintainer,
	"EXPOSE":      TokenExpose,
	"ENV":         TokenEnv,
	"ADD":         TokenAdd,
	"COPY":        TokenCopy,
	"ENTRYPOINT":  TokenEntrypoint,
	"VOLUME":      TokenVolume,
	"USER":        TokenUser,
	"WORKDIR":     TokenWorkdir,
	"ARG":         TokenArg,
	"ONBUILD":     TokenOnbuild,
	"STOPSIGNAL":  TokenStopsignal,
	"HEALTHCHECK": TokenHealthcheck,
	"SHELL":       TokenShell,
}

// LookupKeyword returns the token type for an instruction keyword,
// or TokenWord if not a keyword
func LookupKeyword(ident string) TokenType {
	if tok, ok := instructionKeywords[ident]; ok {
		return tok
	}
	return TokenWord
}

// Position represents a position in the source
type Position struct {
	Line   int // 1-based line number
	Column int // 1-based column number
	Offset int // 0-based byte offset
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string   // the actual text
	Pos     Position // start position
	EndPos  Position // end position
}

func (t Token) String() string {
	if len(t.Literal) > 20 {
		return fmt.Sprintf("%s(%q...) at %s", t.Type, t.Literal[:20], t.Pos)
	}
	return fmt.Sprintf("%s(%q) at %s", t.Type, t.Literal, t.Pos)
}

// IsInstruction returns true if the token is a Dockerfile instruction
func (t Token) IsInstruction() bool {
	switch t.Type {
	case TokenFrom, TokenRun, TokenCmd, TokenLabel, TokenMaintainer,
		TokenExpose, TokenEnv, TokenAdd, TokenCopy, TokenEntrypoint,
		TokenVolume, TokenUser, TokenWorkdir, TokenArg, TokenOnbuild,
		TokenStopsignal, TokenHealthcheck, TokenShell:
		return true
	}
	return false
}
