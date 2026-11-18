package models

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

func sampleIndex() *FileIndex {
	paths := []string{
		"/internal/release/gate.go",
		"/internal/release/gate_test.go",
		"/internal/release/gates/retry.go",
		"/web/src/pages/ReleaseGate.tsx",
		"/cmd/relay/gateway.go",
		"/docs/release-gates.md",
		"/internal/registry/legacy/template.go",
		"/README.md",
	}
	items := []GitItem{{Path: "/internal", Folder: true}, {Path: "/internal/release", Folder: true}}
	for _, p := range paths {
		items = append(items, GitItem{Path: p})
	}
	return NewFileIndex(items, "2b4b23c", false)
}

func names(x *FileIndex, ms []FuzzyMatch) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = x.File(m.Index)
	}
	return out
}

func TestFuzzyRanksFileNameMatchesFirst(t *testing.T) {
	x := sampleIndex()
	got := names(x, x.Match("gate", nil))
	if len(got) < 3 || got[0] != "internal/release/gate.go" || got[1] != "internal/release/gate_test.go" {
		t.Fatalf("gate ranked %v", got)
	}
	got = names(x, x.Match("relgate", nil))
	if !slices.Contains(got[:4], "internal/release/gate.go") || got[len(got)-1] != "internal/registry/legacy/template.go" {
		t.Fatalf("relgate ranked %v", got)
	}
	for _, n := range got {
		if n == "README.md" {
			t.Fatalf("README.md must not match relgate: %v", got)
		}
	}
}

func TestFuzzyTermsMatchInAnyOrderAndFoldersNarrow(t *testing.T) {
	x := sampleIndex()
	if got := names(x, x.Match("test gate", nil)); len(got) == 0 || got[0] != "internal/release/gate_test.go" {
		t.Fatalf("test gate matched %v", got)
	}
	if got := names(x, x.Match("web/ gate", nil)); len(got) != 1 || got[0] != "web/src/pages/ReleaseGate.tsx" {
		t.Fatalf("web/ gate matched %v", got)
	}
	if got := x.Match("relgatz", nil); len(got) != 0 {
		t.Fatalf("relgatz should not match, got %v", names(x, got))
	}
}

func TestFuzzyPositionsPointAtTheMatchedLetters(t *testing.T) {
	m, ok := FuzzyScore("gate", "internal/release/gate.go")
	if !ok || fmt.Sprint(m.Positions) != "[17 18 19 20]" {
		t.Fatalf("got %+v", m)
	}
}

func TestNarrowingKeepsTheSameResults(t *testing.T) {
	x := sampleIndex()
	wide := x.Match("re", nil)
	within := make([]int, len(wide))
	for i, m := range wide {
		within[i] = m.Index
	}
	if a, b := names(x, x.Match("regate", within)), names(x, x.Match("regate", nil)); fmt.Sprint(a) != fmt.Sprint(b) {
		t.Fatalf("narrowing changed the results: %v vs %v", a, b)
	}
}

func TestIndexChildrenListsOneLevel(t *testing.T) {
	x := sampleIndex()
	got := x.Children("/internal/release")
	if len(got) != 2 || got[0].Path != "/internal/release/gate.go" {
		t.Fatalf("children %+v", got)
	}
}

func TestFuzzyIsFastOnALargeIndex(t *testing.T) {
	items := make([]GitItem, 0, MaxIndexedItems)
	for i := range MaxIndexedItems {
		items = append(items, GitItem{Path: fmt.Sprintf("/services/svc%03d/internal/pkg%02d/file_%05d.go", i%400, i%37, i)})
	}
	x := NewFileIndex(items, "", false)
	start := time.Now()
	x.Match("svc12 pkg file9", nil)
	if took := time.Since(start); took > 400*time.Millisecond {
		t.Errorf("matching %d files took %v", MaxIndexedItems, took)
	}
}

func TestCompactIndexKeepsMatchingCaseAndUnicode(t *testing.T) {
	x := NewFileIndex([]GitItem{
		{Path: "/docs", Folder: true},
		{Path: "/docs/Résumé Notes.md"},
		{Path: "/web/src/ReleaseGate.tsx"},
		{Path: "/README.md"},
	}, "", false)
	cases := map[string]string{
		"résumé":    "docs/Résumé Notes.md",
		"README":    "README.md",
		"rgate":     "web/src/ReleaseGate.tsx",
		"src/ gate": "web/src/ReleaseGate.tsx",
	}
	for q, want := range cases {
		got := names(x, x.Match(q, nil))
		if len(got) == 0 || got[0] != want {
			t.Errorf("Match(%q) = %v, want %q first", q, got, want)
		}
	}
	if got := x.Match("resume", nil); len(got) != 0 {
		t.Errorf("accents are significant, got %v", names(x, got))
	}
	if !x.Has("/README.md") || x.Has("/docs") || x.Has("/missing.go") {
		t.Error("Has must know every file and nothing else")
	}
	if got := x.Children("/docs"); len(got) != 1 || got[0].Path != "/docs/Résumé Notes.md" {
		t.Errorf("children %+v", got)
	}
	if got := x.Children("/"); len(got) != 2 || !got[0].Folder || got[1].Path != "/README.md" {
		t.Errorf("root children %+v, want the docs folder first then files", got)
	}
}
