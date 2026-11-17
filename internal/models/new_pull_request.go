package models

import (
	"strings"
	"unicode"
)

type NewReviewer struct {
	ID       string
	Required bool
}

type NewPullRequest struct {
	Source      string
	Target      string
	Title       string
	Description string
	Draft       bool
	Reviewers   []NewReviewer
	WorkItems   []int
	Labels      []string
}

type BranchComparison struct {
	Ahead         int
	Behind        int
	Files         int
	Commits       []string
	ExistingID    int
	ExistingTitle string
	Template      string
}

func (c BranchComparison) SuggestedTitle(source string) string {
	if len(c.Commits) == 1 {
		return strings.TrimSpace(strings.SplitN(c.Commits[0], "\n", 2)[0])
	}
	return TitleFromBranch(source)
}

func TitleFromBranch(branch string) string {
	name := branch[strings.LastIndex(branch, "/")+1:]
	name = strings.TrimLeftFunc(name, func(r rune) bool { return unicode.IsDigit(r) || r == '-' || r == '_' })
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
	if len(words) == 0 {
		return branch
	}
	title := strings.Join(words, " ")
	return strings.ToUpper(title[:1]) + title[1:]
}

func WorkItemFromBranch(branch string) int {
	name := branch[strings.LastIndex(branch, "/")+1:]
	for _, part := range strings.FieldsFunc(name, func(r rune) bool { return !unicode.IsDigit(r) }) {
		if len(part) >= 3 {
			n := 0
			for _, r := range part {
				n = n*10 + int(r-'0')
			}
			return n
		}
	}
	return 0
}
