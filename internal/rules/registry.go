package rules

import (
	"sort"
	"sync"

	"github.com/HueCodes/keel/internal/analyzer"
)

// Registry holds all registered rules
type Registry struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

// Global registry instance
var globalRegistry = &Registry{
	rules: make(map[string]Rule),
}

// Register adds a rule to the global registry
func Register(rule Rule) {
	globalRegistry.Register(rule)
}

// Get returns a rule by ID from the global registry
func Get(id string) (Rule, bool) {
	return globalRegistry.Get(id)
}

// All returns all rules from the global registry
func All() []Rule {
	return globalRegistry.All()
}

// ByCategory returns rules filtered by category from the global registry
func ByCategory(category analyzer.Category) []Rule {
	return globalRegistry.ByCategory(category)
}

// Register adds a rule to the registry
func (r *Registry) Register(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID()] = rule
}

// Get returns a rule by ID
func (r *Registry) Get(id string) (Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.rules[id]
	return rule, ok
}

// All returns all registered rules, sorted by ID
func (r *Registry) All() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		rules = append(rules, rule)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID() < rules[j].ID()
	})

	return rules
}

// ByCategory returns rules filtered by category
func (r *Registry) ByCategory(category analyzer.Category) []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var rules []Rule
	for _, rule := range r.rules {
		if rule.Category() == category {
			rules = append(rules, rule)
		}
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID() < rules[j].ID()
	})

	return rules
}

// IDs returns all rule IDs
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.rules))
	for id := range r.rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Count returns the number of registered rules
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.rules)
}
