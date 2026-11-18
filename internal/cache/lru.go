package cache

import (
	"container/list"
	"time"
)

type LRU[K comparable, V any] struct {
	budget int
	cost   func(V) int
	used   int
	order  *list.List
	items  map[K]*list.Element
	now    func() time.Time
}

type entry[K comparable, V any] struct {
	key   K
	value V
	cost  int
	at    time.Time
}

func New[K comparable, V any](budget int, cost func(V) int) *LRU[K, V] {
	if cost == nil {
		cost = func(V) int { return 1 }
	}
	return &LRU[K, V]{budget: budget, cost: cost, order: list.New(), items: map[K]*list.Element{}, now: time.Now}
}

func Count[K comparable, V any](n int) *LRU[K, V] {
	return New[K, V](n, nil)
}

func (c *LRU[K, V]) Get(k K) (V, bool) {
	v, _, ok := c.Lookup(k)
	return v, ok
}

func (c *LRU[K, V]) Lookup(k K) (V, time.Duration, bool) {
	el, ok := c.items[k]
	if !ok {
		var zero V
		return zero, 0, false
	}
	c.order.MoveToFront(el)
	e := el.Value.(*entry[K, V])
	return e.value, c.now().Sub(e.at), true
}

func (c *LRU[K, V]) Put(k K, v V) {
	cost := c.cost(v)
	c.Remove(k)
	if cost > c.budget {
		return
	}
	c.items[k] = c.order.PushFront(&entry[K, V]{key: k, value: v, cost: cost, at: c.now()})
	c.used += cost
	for c.used > c.budget {
		c.removeElement(c.order.Back())
	}
}

func (c *LRU[K, V]) Remove(k K) {
	if el, ok := c.items[k]; ok {
		c.removeElement(el)
	}
}

func (c *LRU[K, V]) Clear() {
	c.order.Init()
	clear(c.items)
	c.used = 0
}

func (c *LRU[K, V]) Len() int { return len(c.items) }

func (c *LRU[K, V]) Used() int { return c.used }

func (c *LRU[K, V]) removeElement(el *list.Element) {
	e := c.order.Remove(el).(*entry[K, V])
	delete(c.items, e.key)
	c.used -= e.cost
}

func (c *LRU[K, V]) SetClock(now func() time.Time) { c.now = now }
