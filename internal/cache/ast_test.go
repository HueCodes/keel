package cache

import (
	"testing"
	"time"

	"github.com/HueCodes/keel/internal/parser"
)

func TestASTCache_GetPut(t *testing.T) {
	cache := NewASTCache()

	content := "FROM alpine\nRUN echo hello\n"
	df, parseErrors := parser.Parse(content)

	cache.Put("Dockerfile", content, df, parseErrors)

	// Should hit cache
	entry, ok := cache.Get("Dockerfile", content)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.Dockerfile == nil {
		t.Error("expected Dockerfile in cache entry")
	}
}

func TestASTCache_ContentChange(t *testing.T) {
	cache := NewASTCache()

	content1 := "FROM alpine\n"
	content2 := "FROM ubuntu\n"

	df, parseErrors := parser.Parse(content1)
	cache.Put("Dockerfile", content1, df, parseErrors)

	// Should miss with different content
	_, ok := cache.Get("Dockerfile", content2)
	if ok {
		t.Error("expected cache miss for changed content")
	}
}

func TestASTCache_LRUEviction(t *testing.T) {
	cache := NewASTCache(WithMaxEntries(2))

	content := "FROM alpine\n"
	df, parseErrors := parser.Parse(content)

	cache.Put("file1", content, df, parseErrors)
	cache.Put("file2", content, df, parseErrors)
	cache.Put("file3", content, df, parseErrors) // Should evict file1

	if _, ok := cache.Get("file1", content); ok {
		t.Error("expected file1 to be evicted")
	}
	if _, ok := cache.Get("file2", content); !ok {
		t.Error("expected file2 to still be cached")
	}
	if _, ok := cache.Get("file3", content); !ok {
		t.Error("expected file3 to still be cached")
	}
}

func TestASTCache_Invalidate(t *testing.T) {
	cache := NewASTCache()

	content := "FROM alpine\n"
	df, parseErrors := parser.Parse(content)

	cache.Put("Dockerfile", content, df, parseErrors)
	cache.Invalidate("Dockerfile")

	if _, ok := cache.Get("Dockerfile", content); ok {
		t.Error("expected cache miss after invalidation")
	}
}

func TestASTCache_Clear(t *testing.T) {
	cache := NewASTCache()

	content := "FROM alpine\n"
	df, parseErrors := parser.Parse(content)

	cache.Put("file1", content, df, parseErrors)
	cache.Put("file2", content, df, parseErrors)
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected empty cache, got %d entries", cache.Size())
	}
}

func TestASTCache_Expiration(t *testing.T) {
	cache := NewASTCache(WithMaxAge(10 * time.Millisecond))

	content := "FROM alpine\n"
	df, parseErrors := parser.Parse(content)

	cache.Put("Dockerfile", content, df, parseErrors)

	// Should hit immediately
	if _, ok := cache.Get("Dockerfile", content); !ok {
		t.Error("expected cache hit")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should miss after expiration
	if _, ok := cache.Get("Dockerfile", content); ok {
		t.Error("expected cache miss after expiration")
	}
}

func TestASTCache_Stats(t *testing.T) {
	cache := NewASTCache(WithMaxEntries(50))

	content := "FROM alpine\n"
	df, parseErrors := parser.Parse(content)

	cache.Put("file1", content, df, parseErrors)
	cache.Put("file2", content, df, parseErrors)

	stats := cache.Stats()
	if stats.Entries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.Entries)
	}
	if stats.MaxEntries != 50 {
		t.Errorf("expected max 50 entries, got %d", stats.MaxEntries)
	}
}

func TestCachedParser_Parse(t *testing.T) {
	cache := NewASTCache()
	cp := NewCachedParser(cache)

	content := "FROM alpine\nRUN echo hello\n"

	// First parse - cache miss
	df1, _ := cp.Parse("Dockerfile", content)
	if df1 == nil {
		t.Fatal("expected parsed Dockerfile")
	}

	// Second parse - cache hit
	df2, _ := cp.Parse("Dockerfile", content)
	if df2 == nil {
		t.Fatal("expected parsed Dockerfile from cache")
	}

	// Cache should have 1 entry
	if cache.Size() != 1 {
		t.Errorf("expected 1 cache entry, got %d", cache.Size())
	}
}

func TestHashContent(t *testing.T) {
	hash1 := hashContent("FROM alpine\n")
	hash2 := hashContent("FROM alpine\n")
	hash3 := hashContent("FROM ubuntu\n")

	if hash1 != hash2 {
		t.Error("expected same hash for same content")
	}
	if hash1 == hash3 {
		t.Error("expected different hash for different content")
	}
}
