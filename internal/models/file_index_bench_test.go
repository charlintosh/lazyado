package models

import (
	"fmt"
	"runtime"
	"testing"
)

func largeIndexItems() []GitItem {
	items := make([]GitItem, 0, MaxIndexedItems)
	for i := range MaxIndexedItems {
		path := fmt.Sprintf("/services/svc%03d/internal/pkg%02d/handler_file_%05d.go", i%400, i%37, i)
		if i%5 == 0 {
			path = fmt.Sprintf("/web/src/Components/Svc%03d/HandlerView%05d.tsx", i%400, i)
		}
		items = append(items, GitItem{Path: path})
	}
	return items
}

func BenchmarkFileIndexBuild(b *testing.B) {
	items := largeIndexItems()
	b.ReportAllocs()
	for b.Loop() {
		NewFileIndex(items, "abc", false)
	}
	b.ReportMetric(retainedMB(func() any { return NewFileIndex(largeIndexItems(), "abc", false) }), "MB-retained")
}

func retainedMB(build func() any) float64 {
	heap := func() int64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return int64(m.HeapAlloc)
	}
	before := heap()
	kept := build()
	after := heap()
	runtime.KeepAlive(kept)
	return float64(after-before) / 1e6
}

func BenchmarkFileIndexMatch(b *testing.B) {
	x := NewFileIndex(largeIndexItems(), "abc", false)
	for _, q := range []string{"h", "handler", "svc12 pkg file9", "components/ view"} {
		b.Run(q, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				x.Match(q, nil)
			}
		})
	}
}
