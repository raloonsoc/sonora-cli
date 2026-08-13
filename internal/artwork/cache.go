package artwork

import (
	"container/list"
	"image"
	"sync"
)

// defaultCacheSize bounds the in-memory cover art cache. Cover art is
// small relative to typical terminal memory, but an unbounded cache would
// grow with library size across a long session.
const defaultCacheSize = 64

// Cache is an in-memory LRU of decoded cover art images keyed by album ID
// (SPECS §6.3), avoiding a re-fetch and re-decode on every re-render within
// the same session.
type Cache struct {
	mu    sync.Mutex
	size  int
	items map[string]*list.Element
	order *list.List // front = most recently used
}

type cacheEntry struct {
	key string
	img image.Image
}

// NewCache builds a Cache holding at most size images. size <= 0 uses
// defaultCacheSize.
func NewCache(size int) *Cache {
	if size <= 0 {
		size = defaultCacheSize
	}
	return &Cache{
		size:  size,
		items: make(map[string]*list.Element),
		order: list.New(),
	}
}

// Get returns the cached image for albumID, if present, marking it most
// recently used.
func (c *Cache) Get(albumID string) (image.Image, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[albumID]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*cacheEntry).img, true
}

// Put stores img for albumID, evicting the least recently used entry if the
// cache is full.
func (c *Cache) Put(albumID string, img image.Image) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[albumID]; ok {
		el.Value.(*cacheEntry).img = img
		c.order.MoveToFront(el)
		return
	}

	el := c.order.PushFront(&cacheEntry{key: albumID, img: img})
	c.items[albumID] = el

	if c.order.Len() > c.size {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}
}
