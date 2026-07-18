// Package cache provides CacheRepository implementations (bounded LRU + TTL).
package cache

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

type cacheKey struct {
	textHash   uint64
	configHash uint64
}

type cacheEntry struct {
	key       cacheKey
	value     *model.Analysis
	expiresAt time.Time
}

// ErrCacheMiss is returned when a cache entry is not found or expired.
var ErrCacheMiss = &CacheError{"cache miss"}

// CacheError represents a cache operation error.
type CacheError struct {
	msg string
}

func (e *CacheError) Error() string { return e.msg }

// LRUCache implements ports.CacheRepository with bounded LRU eviction + TTL.
type LRUCache struct {
	mu        sync.Mutex
	items     map[cacheKey]*list.Element
	order     *list.List
	maxSize   int
	ttl       time.Duration
	stopClean chan struct{}
}

// NewLRUCache creates an LRU analysis cache with TTL.
func NewLRUCache(maxSize int, ttl time.Duration) *LRUCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	c := &LRUCache{
		items:     make(map[cacheKey]*list.Element),
		order:     list.New(),
		maxSize:   maxSize,
		ttl:       ttl,
		stopClean: make(chan struct{}),
	}
	go c.cleanLoop()
	return c
}

// StopCleanup stops the background TTL cleanup goroutine.
func (c *LRUCache) StopCleanup() {
	close(c.stopClean)
}

func (c *LRUCache) cleanLoop() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.evictExpired()
		case <-c.stopClean:
			return
		}
	}
}

func (c *LRUCache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, elem := range c.items {
		if now.After(elem.Value.(*cacheEntry).expiresAt) {
			c.order.Remove(elem)
			delete(c.items, k)
		}
	}
}

func (c *LRUCache) Get(ctx context.Context, textHash, configHash uint64) (*model.Analysis, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey{textHash: textHash, configHash: configHash}
	elem, ok := c.items[key]
	if !ok {
		return nil, ErrCacheMiss
	}

	entry := elem.Value.(*cacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.order.Remove(elem)
		delete(c.items, key)
		return nil, ErrCacheMiss
	}

	c.order.MoveToFront(elem)
	return entry.value, nil
}

func (c *LRUCache) Set(ctx context.Context, textHash, configHash uint64, analysis *model.Analysis) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey{textHash: textHash, configHash: configHash}
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.value = analysis
		entry.expiresAt = time.Now().Add(c.ttl)
		return nil
	}

	if c.order.Len() >= c.maxSize {
		c.evictOldest()
	}

	entry := &cacheEntry{
		key:       key,
		value:     analysis,
		expiresAt: time.Now().Add(c.ttl),
	}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
	return nil
}

func (c *LRUCache) Invalidate(ctx context.Context, configHash uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, elem := range c.items {
		if k.configHash == configHash {
			c.order.Remove(elem)
			delete(c.items, k)
		}
	}
	return nil
}

func (c *LRUCache) evictOldest() {
	elem := c.order.Back()
	if elem == nil {
		return
	}
	entry := elem.Value.(*cacheEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
}

// Size returns current number of cached entries.
func (c *LRUCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}
