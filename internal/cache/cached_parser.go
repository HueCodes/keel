package cache

import (
	"github.com/HueCodes/keel/internal/parser"
)

// CachedParser wraps the parser with AST caching
type CachedParser struct {
	cache *ASTCache
}

// NewCachedParser creates a new cached parser
func NewCachedParser(cache *ASTCache) *CachedParser {
	return &CachedParser{cache: cache}
}

// Parse parses the input, using the cache if available
func (p *CachedParser) Parse(filename, content string) (*parser.Dockerfile, []parser.ParseError) {
	// Try cache first
	if entry, ok := p.cache.Get(filename, content); ok {
		return entry.Dockerfile, entry.ParseErrors
	}

	// Cache miss - parse and cache
	df, parseErrors := parser.Parse(content)
	p.cache.Put(filename, content, df, parseErrors)
	return df, parseErrors
}

// Invalidate removes a file from the cache
func (p *CachedParser) Invalidate(filename string) {
	p.cache.Invalidate(filename)
}
