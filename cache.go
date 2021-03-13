// Copyright (c) 2021 hcolde.
// https://github.com/hcolde/cache.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cache

import (
	"context"
	"math"
	"sync"
	"time"
)

type Cache struct {
	mu   sync.Mutex

	refreshTTL bool

	w int

	size uint64
	head uint64
	tail uint64
	ring []string

	indexPool chan uint64

	items map[string]*item
}

// New create a cache.Cache
func New(ctx context.Context, options Options) (*Cache, error) {
	if options.MaxSize == 0 {
		return nil, ErrMaxSizeIsZero
	}

	cache := &Cache{
		w: int(math.Floor(math.Log2(float64(options.MaxSize)))) + 1,
		size:  options.MaxSize,
		ring:  make([]string, options.MaxSize),
		items: make(map[string]*item),
		tail:  options.MaxSize - 1,
		indexPool: make(chan uint64, options.MaxSize),
		refreshTTL: options.RefreshTTL,
	}

	go cache.timedRemove(ctx)
	return cache, nil
}

// Put the item's index into indexPool, when the ttl has expired.
func (c *Cache) get(key string, forced bool) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok {
		if !item.expired() {
			if c.refreshTTL {
				item.touch()
			}

			return item.Val
		}

		c.indexPool <- item.index
		c.ring[item.index] = ""
		delete(c.items, key)

		if forced {
			return item.Val
		}

		return nil
	}

	return nil
}

// Get gets the value associated with the given key.
// Once the key is not set or timespan has elapsed,
// it returns nil.
func (c *Cache) Get(key string) interface{} {
	return c.get(key, false)
}

// ForcedGet gets the value associated with the given key,
// even if the ttl has expired, when the key is not removed
// by the regular purge rules. But once the key is not set,
// it returns nil.
func (c *Cache) ForcedGet(key string) interface{} {
	return c.get(key, true)
}

// Set returns the value with the given key if present.
// Otherwise, it stores the given value and returns nil.
func (c *Cache) Set(key string, val interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok {
		index := item.index

		_key := c.ring[index]
		if _key == key {
			c.items[key] = newItem(val, index, ttl)
			return
		}

		_item, _ok := c.items[_key]
		if !_ok {
			return
		}

		if _item.expired() {
			delete(c.items, _key)
			return
		}

		index = c.getIndex()

		_item.index = index
		c.ring[index] = _key

		return
	}

	index := c.getIndex()

	c.ring[index] = key
	c.items[key]  = newItem(val, index, ttl)
}

// getIndex returns an index used for Cache.ring.
// Returns the index in indexPool first,
// and returns the index on Cache.head when indexPool is empty,
// otherwise returns the index on Cache.tail and remove the this item.
func (c *Cache) getIndex() uint64 {
	select {
	case index, ok := <-c.indexPool:
		if ok {
			return index
		}
	default:
		break
	}

	index := c.tail

	if (c.head + 1) % c.size != c.tail {
		index = c.head
		c.head = (c.head + 1) % c.size
	} else {
		c.tail = (c.tail + 1) % c.size
		c.removeOnIndex(index)
	}

	return index
}

// remove the item from the given index.
func (c *Cache) removeOnIndex(index uint64) {
	key := c.ring[index]
	if key == "" {
		return
	}

	c.ring[index] = ""

	if _, ok := c.items[key]; !ok {
		return
	}

	delete(c.items, key)
}

// Remove regularly
func (c *Cache) timedRemove(ctx context.Context) {
	for {
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			return

		case <-timer.C:
			c.remove()
		}
	}
}

// check the ttl of log2(Cache.size) item at a time.
// It will continue to check when the remove item exceeds Cache.size * 0.25.
func (c *Cache) remove() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		count := 0

		index := c.tail
		for w := c.w; w > 0; w, index = w - 1, (index + 1) % c.size {
			key := c.ring[index]
			if key == "" {
				continue
			}

			item, ok := c.items[key]
			if !ok {
				c.ring[index] = ""
				continue
			}

			if !item.expired() {
				continue
			}

			c.ring[index] = ""
			delete(c.items, key)
			count++
		}

		c.tail = index

		if float64(count) < float64(c.w) * 0.25 {
			return
		}
	}
}
