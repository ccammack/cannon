package cache

import (
	"sync"
)

type Status int

const (
	StatusNotFound = iota
	StatusPending
	StatusReady
)

type Payload interface {
	Open()
	Close()
}

type CacheItem struct {
	key     string
	status  Status
	payload Payload
	mu      sync.Mutex
}

func (item *CacheItem) Open() {
	item.mu.Lock()
	defer item.mu.Unlock()
	item.status = StatusPending

	go func() {
		item.payload.Open()
		item.mu.Lock()
		defer item.mu.Unlock()
		item.status = StatusReady
	}()
}

func (item *CacheItem) Close() {
	item.mu.Lock()
	defer item.mu.Unlock()
	item.status = StatusPending

	go func() {
		item.payload.Close()
		item.mu.Lock()
		defer item.mu.Unlock()
		item.status = StatusNotFound
	}()
}

func (item *CacheItem) IsReady() bool {
	item.mu.Lock()
	defer item.mu.Unlock()
	return item.status == StatusReady
}

type Cache struct {
	items map[string]*CacheItem
	mu    sync.RWMutex
}

func New() *Cache {
	return &Cache{
		items: make(map[string]*CacheItem),
	}
}

func (c *Cache) Put(key string, payload Payload) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.items[key]
	if !ok {
		item := &CacheItem{
			key:     key,
			payload: payload,
		}
		item.Open()
		c.items[key] = item
	}
}

func (c *Cache) Get(key string) (Status, Payload) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[key]
	if !ok {
		return StatusNotFound, nil
	}
	if item.IsReady() {
		return StatusReady, item.payload
	}
	return StatusPending, nil
}

func (c *Cache) Evict(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if ok {
		item.Close()
		delete(c.items, key)
	}
}

func (c *Cache) Clear() {
	for hash := range c.items {
		c.Evict(hash)
	}
}
