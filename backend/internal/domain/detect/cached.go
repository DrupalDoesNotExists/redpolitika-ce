package detect

import "sync"

// CachedNode wraps a Node and caches Detect results by input text.
//
// When multiple rules reference the same detect tree via ref, CachedNode
// ensures the tree runs once per unique text — subsequent calls reuse
// the cached result.
//
// Thread-safe: uses RWMutex for concurrent access.
type CachedNode struct {
	node  Node
	mu    sync.RWMutex
	cache map[string][]MatchRange
}

// NewCachedNode creates a CachedNode wrapping the given node.
func NewCachedNode(node Node) *CachedNode {
	return &CachedNode{
		node:  node,
		cache: make(map[string][]MatchRange),
	}
}

// Detect returns cached result if available, otherwise delegates and caches.
func (c *CachedNode) Detect(text string) []MatchRange {
	c.mu.RLock()
	result, ok := c.cache[text]
	c.mu.RUnlock()
	if ok {
		return result
	}

	c.mu.Lock()
	// Double-check after acquiring write lock
	if result, ok = c.cache[text]; ok {
		c.mu.Unlock()
		return result
	}
	result = c.node.Detect(text)
	c.cache[text] = result
	c.mu.Unlock()
	return result
}
