package models

import (
	"slices"
	"strings"
	"unicode"
)

const (
	fuzzyCharScore     = 16
	fuzzyBoundaryBonus = 10
	fuzzySlashBonus    = 14
	fuzzyCamelBonus    = 8
	fuzzyFirstBonus    = 12
	fuzzyConsecutive   = 14
	fuzzyGapStart      = 4
	fuzzyGapExtend     = 1
	fuzzyBasenameBonus = 48
	fuzzyPrefixBonus   = 32
	fuzzyExactBonus    = 64
	fuzzyWordBonus     = 44
	fuzzyNameChar      = 6
	fuzzyFolderBonus   = 8
	fuzzyLengthDivisor = 12
)

type FuzzyMatch struct {
	Index     int
	Score     int
	Positions []int
}

type fuzzyTerm struct {
	text   []rune
	folder bool
}

func fuzzyTerms(query string) []fuzzyTerm {
	var terms []fuzzyTerm
	for _, f := range strings.Fields(strings.ToLower(query)) {
		if f == "/" {
			continue
		}
		terms = append(terms, fuzzyTerm{text: []rune(f), folder: strings.HasSuffix(f, "/")})
	}
	return terms
}

func FuzzyScore(query, text string) (FuzzyMatch, bool) {
	return fuzzyScoreTerms(fuzzyTerms(query), []rune(text), []rune(strings.ToLower(text)))
}

func fuzzyScoreTerms(terms []fuzzyTerm, text, lower []rune) (FuzzyMatch, bool) {
	if len(terms) == 0 {
		return FuzzyMatch{}, false
	}
	base := lastSlash(lower) + 1
	var total FuzzyMatch
	for _, t := range terms {
		score, positions, ok := scoreTerm(t, text, lower, base)
		if !ok {
			return FuzzyMatch{}, false
		}
		total.Score += score
		total.Positions = append(total.Positions, positions...)
	}
	total.Score -= len(text) / fuzzyLengthDivisor
	slices.Sort(total.Positions)
	total.Positions = slices.Compact(total.Positions)
	return total, true
}

func lastSlash(r []rune) int {
	for i := len(r) - 1; i >= 0; i-- {
		if r[i] == '/' {
			return i
		}
	}
	return -1
}

func scoreTerm(t fuzzyTerm, text, lower []rune, base int) (int, []int, bool) {
	if t.folder {
		i := indexRunes(lower[:max(base, 0)], t.text)
		if i < 0 {
			return 0, nil, false
		}
		positions := make([]int, 0, len(t.text)-1)
		for j := range len(t.text) - 1 {
			positions = append(positions, i+j)
		}
		return fuzzyFolderBonus + fuzzyCharScore*len(t.text), positions, true
	}
	if score, positions, ok := scoreWindow(t.text, text, lower, base); ok {
		score += fuzzyBasenameBonus
		return score + nameBonus(t.text, text[base:], lower[base:]), positions, true
	}
	score, positions, ok := scoreWindow(t.text, text, lower, 0)
	for _, p := range positions {
		if p >= base {
			score += fuzzyNameChar
		}
	}
	return score, positions, ok
}

func nameBonus(term, name, lower []rune) int {
	n := len(term)
	if len(lower) < n || !slices.Equal(lower[:n], term) {
		return 0
	}
	switch {
	case len(lower) == n || lower[n] == '.':
		return fuzzyExactBonus
	case strings.ContainsRune("_-. ", lower[n]) || (unicode.IsLower(name[n-1]) && unicode.IsUpper(name[n])):
		return fuzzyWordBonus
	}
	return fuzzyPrefixBonus
}

func indexRunes(hay, needle []rune) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if slices.Equal(hay[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func scoreWindow(pattern, text, lower []rune, from int) (int, []int, bool) {
	if from > len(lower) {
		return 0, nil, false
	}
	end, pi := -1, 0
	for i := from; i < len(lower) && pi < len(pattern); i++ {
		if lower[i] == pattern[pi] {
			pi++
			if pi == len(pattern) {
				end = i
			}
		}
	}
	if end < 0 {
		return 0, nil, false
	}
	start, pi := end, len(pattern)-1
	for i := end; i >= from && pi >= 0; i-- {
		if lower[i] == pattern[pi] {
			pi--
			start = i
		}
	}
	positions := make([]int, 0, len(pattern))
	score, prev, pi := 0, -1, 0
	for i := start; i <= end && pi < len(pattern); i++ {
		if lower[i] != pattern[pi] {
			continue
		}
		score += fuzzyCharScore + boundaryBonus(text, i)
		switch {
		case prev >= 0 && i == prev+1:
			score += fuzzyConsecutive
		case prev >= 0:
			score -= fuzzyGapStart + fuzzyGapExtend*(i-prev-2)
		}
		if pi == 0 && i == from {
			score += fuzzyFirstBonus
		}
		positions = append(positions, i)
		prev = i
		pi++
	}
	return score, positions, true
}

func boundaryBonus(text []rune, i int) int {
	if i == 0 {
		return fuzzyBoundaryBonus
	}
	prev, cur := text[i-1], text[i]
	switch {
	case prev == '/':
		return fuzzySlashBonus
	case prev == '_' || prev == '-' || prev == '.' || prev == ' ':
		return fuzzyBoundaryBonus
	case unicode.IsLower(prev) && unicode.IsUpper(cur):
		return fuzzyCamelBonus
	case !unicode.IsDigit(prev) && unicode.IsDigit(cur):
		return fuzzyCamelBonus
	}
	return 0
}
