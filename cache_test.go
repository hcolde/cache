// Copyright (c) 2021 hcolde.
// https://github.com/hcolde/cache.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cache

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func newCache(size uint64) (*Cache, error) {
	opt := Options{
		MaxSize: size,
		RefreshTTL: true,
	}

	return New(context.Background(), opt)
}

func TestCache(t *testing.T) {
	c, err := newCache(2)
	if err != nil {
		t.Fatal(err)
	}

	v := c.Get("a")
	if v != nil {
		t.Fatalf("get value %v but not set", v)
	}

	c.Set("a", 1, 2 * time.Second)
	c.Set("b", 2, 2 * time.Second)

	v = c.Get("a")
	if v != 1 {
		t.Fatalf("%v is not equal 1", v)
	}

	v = c.Get("b")
	if v != 2 {
		t.Fatalf("%v is not equal 1", v)
	}

	t.Log("test success")
}

func BenchmarkCache_Set(b *testing.B) {
	n := b.N
	c, err := newCache(uint64(n))
	if err != nil {
		b.Fatal(err)
	}

	m := make(map[string]int)

	for i := 0; i < n; i++ {
		k := strconv.Itoa(i)
		c.Set(k, i, 0)
		m[k] = i
	}

	for i := 0; i < n; i++ {
		k := strconv.Itoa(i)
		v1 := m[k]
		v2 := c.Get(k)
		if v1 != v2 {
			b.Fatalf("%v is not equal %v", v1, v2)
		}
	}

	b.Logf("Set test success(bench N:%v)", n)
}

func BenchmarkCache_SetOnPool(b *testing.B) {
	n := 10000
	c, err := newCache(uint64(n))
	if err != nil {
		b.Fatal(err)
	}

	m := make(map[string]int)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num := r.Intn(n)

	for i := 0; i < n; i++ {
		k := strconv.Itoa(i)
		if i == num {
			c.Set(k, i, 1000 * time.Millisecond)
			continue
		}
		m[k] = i
		c.Set(k, i, 0)
	}

	time.Sleep(1 * time.Second)
	for i := 0; i < n; i++ {
		k := strconv.Itoa(i)
		if i == num {
			if c.Get(k) != nil {
				b.Fatal("timed remove failed")
			}
			continue
		}

		v1 := m[k]
		v2 := c.Get(k)
		if v1 != v2 {
			b.Fatalf("%v is not equal %v", v1, v2)
		}
	}

	c.Set("a", "aaa", 0)
	if c.ring[num] != "a" {
		b.Fatal("cache index pool not malfunction")
	}

	if c.Get("a") != "aaa" {
		b.Fatal("value error")
	}

	b.Logf("Set on pool test success(bench N:%v)", n)
}
