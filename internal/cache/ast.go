package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/HueCodes/keel/internal/parser"
)

// ASTEntry represents a cached AST entry
type ASTEntry struct {
	Dockerfile   *parser.Dockerfile
	ParseErrors  []parser.ParseError
	Hash         string
	LastAccessed time.Time
}

// ASTCache provides an LRU cache for parsed Dockerfiles
type ASTCache struct {
	mu         sync.RWMutex
	cache      map[string]*list.Element
	lru        *list.List
	maxEntries int
	maxAge     time.Duration
}

// entry stores the key and value in the LRU list
type entry struct {
	key   string
	value *ASTEntry
}

// Option configures the ASTCache
type Option func(*ASTCache)

// NewASTCache creates a new AST cache
func NewASTCache(opts ...Option) *ASTCache {
	c := &ASTCache{
		cache:      make(map[string]*list.Element),
		lru:        list.New(),
		maxEntries: 100,
		maxAge:     5 * time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithMaxEntries sets the maximum number of cached entries
func WithMaxEntries(n int) Option {
	return func(c *ASTCache) {
		if n > 0 {
			c.maxEntries = n
		}
	}
}

// WithMaxAge sets the maximum age of cached entries
func WithMaxAge(d time.Duration) Option {
	return func(c *ASTCache) {
		if d > 0 {
			c.maxAge = d
		}
	}
}

// Get retrieves a cached AST if it exists and the content hash matches
func (c *ASTCache) Get(filename, content string) (*ASTEntry, bool) {
	hash := hashContent(content)

	c.mu.RLock()
	elem, ok := c.cache[filename]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	ent := elem.Value.(*entry)

	// Check if hash matches
	if ent.value.Hash != hash {
		// Content changed, remove stale entry
		c.mu.Lock()
		c.removeElement(elem)
		c.mu.Unlock()
		return nil, false
	}

	// Check if expired
	if time.Since(ent.value.LastAccessed) > c.maxAge {
		c.mu.Lock()
		c.removeElement(elem)
		c.mu.Unlock()
		return nil, false
	}

	// Move to front of LRU and update access time
	c.mu.Lock()
	c.lru.MoveToFront(elem)
	ent.value.LastAccessed = time.Now()
	c.mu.Unlock()

	return ent.value, true
}

// Put stores an AST in the cache
func (c *ASTCache) Put(filename, content string, df *parser.Dockerfile, parseErrors []parser.ParseError) {
	hash := hashContent(content)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if entry already exists
	if elem, ok := c.cache[filename]; ok {
		c.lru.MoveToFront(elem)
		ent := elem.Value.(*entry)
		ent.value = &ASTEntry{
			Dockerfile:   df,
			ParseErrors:  parseErrors,
			Hash:         hash,
			LastAccessed: time.Now(),
		}
		return
	}

	// Create new entry
	ent := &entry{
		key: filename,
		value: &ASTEntry{
			Dockerfile:   df,
			ParseErrors:  parseErrors,
			Hash:         hash,
			LastAccessed: time.Now(),
		},
	}
	elem := c.lru.PushFront(ent)
	c.cache[filename] = elem

	// Evict if over capacity
	for c.lru.Len() > c.maxEntries {
		c.removeOldest()
	}
}

// Invalidate removes an entry from the cache
func (c *ASTCache) Invalidate(filename string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[filename]; ok {
		c.removeElement(elem)
	}
}

// Clear removes all entries from the cache
func (c *ASTCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*list.Element)
	c.lru.Init()
}

// Size returns the number of entries in the cache
func (c *ASTCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Stats returns cache statistics
func (c *ASTCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return CacheStats{
		Entries:    len(c.cache),
		MaxEntries: c.maxEntries,
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Entries    int
	MaxEntries int
	Hits       int64
	Misses     int64
}

func (c *ASTCache) removeOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *ASTCache) removeElement(elem *list.Element) {
	c.lru.Remove(elem)
	ent := elem.Value.(*entry)
	delete(c.cache, ent.key)
}

// hashContent computes a SHA256 hash of the content
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
