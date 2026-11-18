package cache

import (
	"testing"
	"time"
)

func TestCountEvictsTheLeastRecentlyUsed(t *testing.T) {
	c := Count[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Get("a")
	c.Put("c", 3)
	if _, ok := c.Get("b"); ok {
		t.Fatal("b was the least recently used and must be evicted")
	}
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatal("a was used recently and must stay")
	}
	if c.Len() != 2 {
		t.Fatalf("len %d, want 2", c.Len())
	}
}

func TestBudgetCountsCost(t *testing.T) {
	c := New[string, string](10, func(s string) int { return len(s) })
	c.Put("a", "aaaa")
	c.Put("b", "bbbb")
	c.Put("c", "cccc")
	if _, ok := c.Get("a"); ok || c.Used() != 8 {
		t.Fatalf("used %d, want 8 with a evicted", c.Used())
	}
	c.Put("huge", "xxxxxxxxxxxx")
	if _, ok := c.Get("huge"); ok || c.Used() != 8 {
		t.Fatal("a value larger than the whole budget must not be kept nor evict others")
	}
}

func TestReplacingAKeyUpdatesCostAndAge(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := New[string, string](100, func(s string) int { return len(s) })
	c.now = func() time.Time { return now }
	c.Put("a", "1234")
	now = now.Add(time.Minute)
	if _, age, _ := c.Lookup("a"); age != time.Minute {
		t.Fatalf("age %v, want 1m", age)
	}
	c.Put("a", "12")
	if _, age, _ := c.Lookup("a"); age != 0 || c.Used() != 2 {
		t.Fatalf("age %v used %d after replacing", age, c.Used())
	}
	c.Remove("a")
	c.Clear()
	if c.Len() != 0 || c.Used() != 0 {
		t.Fatal("clear must empty the cache")
	}
}
