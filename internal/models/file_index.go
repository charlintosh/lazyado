package models

import (
	"cmp"
	"slices"
	"strings"
)

const MaxIndexedItems = 50000

type FileIndex struct {
	Commit    string
	Truncated bool
	items     []GitItem
	files     []string
	runes     [][]rune
	lower     [][]rune
	present   map[string]bool
}

func NewFileIndex(items []GitItem, commit string, truncated bool) *FileIndex {
	x := &FileIndex{Commit: commit, Truncated: truncated, items: items, present: map[string]bool{}}
	for _, it := range items {
		if it.Folder {
			continue
		}
		p := strings.TrimPrefix(it.Path, "/")
		x.files = append(x.files, p)
		x.runes = append(x.runes, []rune(p))
		x.lower = append(x.lower, []rune(strings.ToLower(p)))
		x.present[it.Path] = true
	}
	return x
}

func (x *FileIndex) Len() int { return len(x.files) }

func (x *FileIndex) File(i int) string { return x.files[i] }

func (x *FileIndex) Has(path string) bool { return x.present[path] }

func (x *FileIndex) Match(query string, within []int) []FuzzyMatch {
	terms := fuzzyTerms(query)
	if len(terms) == 0 {
		return nil
	}
	var out []FuzzyMatch
	try := func(i int) {
		if m, ok := fuzzyScoreTerms(terms, x.runes[i], x.lower[i]); ok {
			m.Index = i
			out = append(out, m)
		}
	}
	if within != nil {
		for _, i := range within {
			try(i)
		}
	} else {
		for i := range x.files {
			try(i)
		}
	}
	slices.SortFunc(out, func(a, b FuzzyMatch) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		if c := cmp.Compare(len(x.files[a.Index]), len(x.files[b.Index])); c != 0 {
			return c
		}
		return strings.Compare(x.files[a.Index], x.files[b.Index])
	})
	return out
}

func (x *FileIndex) Children(folder string) []GitItem {
	prefix := strings.TrimSuffix(folder, "/") + "/"
	var out []GitItem
	for _, it := range x.items {
		rest, ok := strings.CutPrefix(it.Path, prefix)
		if ok && rest != "" && !strings.Contains(rest, "/") {
			out = append(out, it)
		}
	}
	return SortGitItems(out)
}
