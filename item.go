// Copyright (c) 2021 hcolde.
// https://github.com/hcolde/cache.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cache

import "time"

type item struct {
	Val interface{}

	index uint64

	ttl      time.Duration
	expireAt time.Time
}

// newItem create new an item.
func newItem(val interface{}, index uint64, ttl time.Duration) *item {
	i := &item{
		Val:      val,
		index:    index,
		ttl:      ttl,
	}

	i.touch()
	return i
}

// Refresh the ttl
func (i *item) touch() {
	if i.ttl > 0 {
		i.expireAt = time.Now().Add(i.ttl)
	}
}

// Returns true if ths item.ttl is expired.
// Otherwise returns false.
func (i *item) expired() bool {
	if i.ttl <= 0 {
		return false
	}

	return i.expireAt.Before(time.Now())
}
