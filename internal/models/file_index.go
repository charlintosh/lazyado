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
	paths     []string
	lower     []string
	folders   []string
	sorted    []string
}

func NewFileIndex(items []GitItem, commit string, truncated bool) *FileIndex {
	x := &FileIndex{Commit: commit, Truncated: truncated}
	for _, it := range items {
		if it.Folder {
			x.folders = append(x.folders, it.Path)
			continue
		}
		x.paths = append(x.paths, it.Path)
		x.lower = append(x.lower, strings.ToLower(strings.TrimPrefix(it.Path, "/")))
	}
	x.paths, x.lower, x.folders = slices.Clip(x.paths), slices.Clip(x.lower), slices.Clip(x.folders)
	x.sorted = slices.Clone(x.paths)
	slices.Sort(x.sorted)
	return x
}

func (x *FileIndex) Len() int { return len(x.paths) }

func (x *FileIndex) File(i int) string { return strings.TrimPrefix(x.paths[i], "/") }

func (x *FileIndex) Has(path string) bool {
	_, ok := slices.BinarySearch(x.sorted, path)
	return ok
}

func (x *FileIndex) Match(query string, within []int) []FuzzyMatch {
	terms := fuzzyTerms(query)
	if len(terms) == 0 {
		return nil
	}
	needles := make([]string, len(terms))
	for i, t := range terms {
		needles[i] = string(t.text)
	}
	var (
		out         []FuzzyMatch
		text, lower []rune
	)
	try := func(i int) {
		if !containsAllInOrder(x.lower[i], needles) {
			return
		}
		text, lower = appendRunes(text[:0], x.File(i)), appendRunes(lower[:0], x.lower[i])
		if m, ok := fuzzyScoreTerms(terms, text, lower); ok {
			m.Index = i
			out = append(out, m)
		}
	}
	if within != nil {
		for _, i := range within {
			try(i)
		}
	} else {
		for i := range x.paths {
			try(i)
		}
	}
	slices.SortFunc(out, func(a, b FuzzyMatch) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		fa, fb := x.File(a.Index), x.File(b.Index)
		if c := cmp.Compare(len(fa), len(fb)); c != 0 {
			return c
		}
		return strings.Compare(fa, fb)
	})
	return out
}

func (x *FileIndex) Children(folder string) []GitItem {
	prefix := strings.TrimSuffix(folder, "/") + "/"
	direct := func(p string) bool {
		rest, ok := strings.CutPrefix(p, prefix)
		return ok && rest != "" && !strings.Contains(rest, "/")
	}
	var out []GitItem
	for _, p := range x.folders {
		if direct(p) {
			out = append(out, GitItem{Path: p, Folder: true})
		}
	}
	for _, p := range x.paths {
		if direct(p) {
			out = append(out, GitItem{Path: p})
		}
	}
	return SortGitItems(out)
}

func containsAllInOrder(hay string, needles []string) bool {
	for _, n := range needles {
		if !subsequence(hay, n) {
			return false
		}
	}
	return true
}

func subsequence(hay, needle string) bool {
	j := 0
	for i := 0; i < len(hay) && j < len(needle); i++ {
		if hay[i] == needle[j] {
			j++
		}
	}
	return j == len(needle)
}

func appendRunes(dst []rune, s string) []rune {
	for _, r := range s {
		dst = append(dst, r)
	}
	return dst
}
