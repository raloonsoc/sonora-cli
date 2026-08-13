package artwork

import (
	"image"
	"testing"
)

func testImage() image.Image {
	return image.NewRGBA(image.Rect(0, 0, 4, 4))
}

func TestCache_putGet(t *testing.T) {
	c := NewCache(2)
	img := testImage()

	if _, ok := c.Get("missing"); ok {
		t.Fatal("Get on empty cache should report false")
	}

	c.Put("album1", img)
	got, ok := c.Get("album1")
	if !ok || got != img {
		t.Fatalf("Get(album1) = %v, %v, want the stored image", got, ok)
	}
}

func TestCache_evictsLeastRecentlyUsed(t *testing.T) {
	c := NewCache(2)
	c.Put("a", testImage())
	c.Put("b", testImage())
	c.Put("c", testImage()) // evicts "a", the least recently used

	if _, ok := c.Get("a"); ok {
		t.Error("expected a to be evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Error("expected b to survive eviction")
	}
	if _, ok := c.Get("c"); !ok {
		t.Error("expected c to survive eviction")
	}
}

func TestCache_getRefreshesRecency(t *testing.T) {
	c := NewCache(2)
	c.Put("a", testImage())
	c.Put("b", testImage())

	c.Get("a") // touch a so it's no longer the least recently used

	c.Put("c", testImage()) // should evict b, not a

	if _, ok := c.Get("a"); !ok {
		t.Error("expected a to survive eviction after being touched")
	}
	if _, ok := c.Get("b"); ok {
		t.Error("expected b to be evicted")
	}
}

func TestCache_putOverwritesExisting(t *testing.T) {
	c := NewCache(2)
	img1 := testImage()
	img2 := testImage()

	c.Put("a", img1)
	c.Put("a", img2)

	got, ok := c.Get("a")
	if !ok || got != img2 {
		t.Errorf("Get(a) = %v, want the overwritten image", got)
	}
}

func TestNewCache_defaultsSizeWhenNonPositive(t *testing.T) {
	c := NewCache(0)
	if c.size != defaultCacheSize {
		t.Errorf("size = %d, want %d", c.size, defaultCacheSize)
	}
}
