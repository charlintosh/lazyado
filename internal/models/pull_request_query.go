package models

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	QualifierAuthor   = "author:"
	QualifierReviewer = "reviewer:"
	QualifierSource   = "source:"
	QualifierTarget   = "target:"
	QualifierIs       = "is:"
	MeAlias           = "@me"
	DraftValue        = "draft"
	branchRefPrefix   = "refs/heads/"
)

var Qualifiers = []string{QualifierAuthor, QualifierReviewer, QualifierSource, QualifierTarget, QualifierIs + DraftValue}

type PullRequestCriteria struct {
	CreatorID  string
	ReviewerID string
	SourceRef  string
	TargetRef  string
}

func (c PullRequestCriteria) Empty() bool {
	return c == PullRequestCriteria{}
}

type PullRequestQuery struct {
	Terms    []string
	Author   string
	Reviewer string
	Source   string
	Target   string
	Draft    bool
	Last     string
}

func ParsePullRequestQuery(input string) PullRequestQuery {
	var q PullRequestQuery
	tokens := searchTokens(input)
	if len(tokens) > 0 && !strings.HasSuffix(input, " ") {
		q.Last = tokens[len(tokens)-1]
	}
	for _, t := range tokens {
		lower := strings.ToLower(t)
		switch {
		case strings.HasPrefix(lower, QualifierAuthor):
			q.Author = unquote(t[len(QualifierAuthor):])
		case strings.HasPrefix(lower, QualifierReviewer):
			q.Reviewer = unquote(t[len(QualifierReviewer):])
		case strings.HasPrefix(lower, QualifierSource):
			q.Source = unquote(t[len(QualifierSource):])
		case strings.HasPrefix(lower, QualifierTarget):
			q.Target = unquote(t[len(QualifierTarget):])
		case lower == QualifierIs+DraftValue:
			q.Draft = true
		default:
			q.Terms = append(q.Terms, strings.ToLower(unquote(t)))
		}
	}
	return q
}

func (q PullRequestQuery) Active() bool {
	return len(q.Terms) > 0 || q.Author != "" || q.Reviewer != "" || q.Source != "" || q.Target != "" || q.Draft
}

func (q PullRequestQuery) ID() (int, bool) {
	if len(q.Terms) != 1 || q.Author != "" || q.Reviewer != "" || q.Source != "" || q.Target != "" {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimPrefix(q.Terms[0], "!"))
	return id, err == nil && id > 0
}

func (q PullRequestQuery) Matches(pr PullRequest, me Identity) bool {
	if q.Draft && !pr.IsDraft {
		return false
	}
	if q.Author != "" && !matchesPerson(q.Author, pr.Author, me) {
		return false
	}
	if q.Reviewer != "" && !hasReviewer(pr, q.Reviewer, me) {
		return false
	}
	if q.Source != "" && !containsFold(pr.SourceBranch, q.Source) {
		return false
	}
	if q.Target != "" && !containsFold(pr.TargetBranch, q.Target) {
		return false
	}
	haystack := strings.ToLower(strings.Join([]string{
		strconv.Itoa(pr.ID), pr.Title, pr.SourceBranch, pr.TargetBranch, pr.Author.DisplayName, pr.Author.UniqueName, pr.RepositoryName,
	}, "\n"))
	for _, term := range q.Terms {
		if !strings.Contains(haystack, strings.TrimPrefix(term, "!")) {
			return false
		}
	}
	return true
}

func (q PullRequestQuery) Criteria(members []TeamMember, me Identity) (PullRequestCriteria, error) {
	var c PullRequestCriteria
	var err error
	if c.CreatorID, err = memberID(q.Author, members, me); err != nil {
		return c, err
	}
	if c.ReviewerID, err = memberID(q.Reviewer, members, me); err != nil {
		return c, err
	}
	if q.Source != "" {
		c.SourceRef = branchRefPrefix + strings.TrimPrefix(q.Source, branchRefPrefix)
	}
	if q.Target != "" {
		c.TargetRef = branchRefPrefix + strings.TrimPrefix(q.Target, branchRefPrefix)
	}
	return c, nil
}

func MatchMembers(name string, members []TeamMember) []TeamMember {
	var out []TeamMember
	for _, m := range members {
		if containsFold(m.DisplayName, name) || containsFold(m.UniqueName, name) {
			out = append(out, m)
		}
	}
	return out
}

func memberID(name string, members []TeamMember, me Identity) (string, error) {
	switch {
	case name == "":
		return "", nil
	case strings.EqualFold(name, MeAlias):
		if me.ID == "" {
			return "", fmt.Errorf("couldn't confirm who you are, so %s can't be searched", MeAlias)
		}
		return me.ID, nil
	}
	for _, m := range MatchMembers(name, members) {
		if m.ID != "" {
			return m.ID, nil
		}
	}
	return "", fmt.Errorf("no team member matches %q", name)
}

func matchesPerson(name string, who Identity, me Identity) bool {
	if strings.EqualFold(name, MeAlias) {
		return me.Is(who.ID, who.UniqueName)
	}
	return containsFold(who.DisplayName, name) || containsFold(who.UniqueName, name)
}

func hasReviewer(pr PullRequest, name string, me Identity) bool {
	for _, r := range pr.Reviewers {
		if matchesPerson(name, Identity{ID: r.ID, DisplayName: r.DisplayName, UniqueName: r.UniqueName}, me) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func unquote(s string) string {
	return strings.Trim(s, `"`)
}

func searchTokens(input string) []string {
	var tokens []string
	var b strings.Builder
	quoted := false
	for _, r := range input {
		switch {
		case r == '"':
			quoted = !quoted
			b.WriteRune(r)
		case r == ' ' && !quoted:
			if b.Len() > 0 {
				tokens = append(tokens, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens
}
