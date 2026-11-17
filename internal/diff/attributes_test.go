package diff

import (
	"fmt"
	"strings"
	"testing"
)

func TestAttributesFollowGitattributesPatterns(t *testing.T) {
	a := ParseAttributes(strings.Join([]string{
		"# lockfiles are generated",
		"package-lock.json linguist-generated=true",
		"*.min.js -diff",
		"/dist/** linguist-generated",
		"docs/**/*.svg binary",
		"vendor/keep.go -linguist-generated",
		"vendor/** linguist-generated",
		"src/schema.json text eol=lf",
	}, "\n")).WithPatterns([]string{"**/*.snap", ""})
	cases := map[string]Reason{
		"/package-lock.json":         Generated,
		"/web/app/package-lock.json": Generated,
		"/static/app.min.js":         NoDiff,
		"/dist/bundle.js":            Generated,
		"/src/dist/bundle.js":        Shown,
		"/docs/img/logo/mark.svg":    NoDiff,
		"/vendor/keep.go":            Generated,
		"/vendor/lib/a.go":           Generated,
		"/src/schema.json":           Shown,
		"/src/__snapshots__/ui.snap": Configured,
		"/README.md":                 Shown,
		"/Package-Lock.JSON":         Shown,
	}
	for path, want := range cases {
		if got := a.Reason(path); got != want {
			t.Errorf("Reason(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestLaterAttributeLinesOverrideEarlierOnes(t *testing.T) {
	a := ParseAttributes("*.json linguist-generated\nconfig.json -linguist-generated")
	if a.Reason("/a/config.json") != Shown || a.Reason("/a/data.json") != Generated {
		t.Fatal("the last matching line must win")
	}
}

func TestEstimateCountsOnlyTheChangedRegion(t *testing.T) {
	var lines []string
	for i := range 5000 {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	old := strings.Join(lines, "\n")
	lines[2500] = "changed"
	if got := Estimate(old, strings.Join(lines, "\n")); got != 2 {
		t.Fatalf("one changed line in a long file estimated %d, want 2", got)
	}
	if got := Estimate("", old); got != 5000 {
		t.Fatalf("a new 5000-line file estimated %d, want 5000", got)
	}
}
