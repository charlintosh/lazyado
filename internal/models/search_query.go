package models

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type SearchKind int

const (
	SearchAny SearchKind = iota
	SearchWorkItem
	SearchPullRequest
)

const (
	pullRequestMark = "!"
	workItemMark    = "#"
	azureHost       = "dev.azure.com"
	legacyHostTail  = ".visualstudio.com"
)

var (
	ErrEmptySearch   = errors.New("enter an ID, !PR, #work item or link")
	ErrInvalidSearch = errors.New("use a number, !PR, #work item, or an Azure DevOps work item or pull request link")
)

type SearchQuery struct {
	ID           int
	Kind         SearchKind
	Organization string
	Project      string
}

func (q SearchQuery) WantsWorkItem() bool {
	return q.Kind != SearchPullRequest
}

func (q SearchQuery) WantsPullRequest() bool {
	return q.Kind != SearchWorkItem
}

func (q SearchQuery) Foreign(organization, project string) bool {
	if q.Organization == "" {
		return false
	}
	return !strings.EqualFold(q.Organization, organization) || (q.Project != "" && !strings.EqualFold(q.Project, project))
}

func (q SearchQuery) Label() string {
	switch q.Kind {
	case SearchPullRequest:
		return "pull request " + pullRequestMark + strconv.Itoa(q.ID)
	case SearchWorkItem:
		return "work item " + workItemMark + strconv.Itoa(q.ID)
	}
	return strconv.Itoa(q.ID)
}

func ParseSearchQuery(input string) (SearchQuery, error) {
	text := strings.TrimSpace(input)
	switch {
	case text == "":
		return SearchQuery{}, ErrEmptySearch
	case strings.HasPrefix(text, pullRequestMark):
		return searchID(strings.TrimPrefix(text, pullRequestMark), SearchPullRequest)
	case strings.HasPrefix(text, workItemMark):
		return searchID(strings.TrimPrefix(text, workItemMark), SearchWorkItem)
	case isLink(text):
		return searchLink(text)
	}
	return searchID(text, SearchAny)
}

func searchID(text string, kind SearchKind) (SearchQuery, error) {
	id, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || id <= 0 {
		return SearchQuery{}, ErrInvalidSearch
	}
	return SearchQuery{ID: id, Kind: kind}, nil
}

func isLink(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "://") || strings.HasPrefix(lower, azureHost) || strings.Contains(lower, legacyHostTail)
}

func searchLink(text string) (SearchQuery, error) {
	if !strings.Contains(text, "://") {
		text = "https://" + text
	}
	u, err := url.Parse(text)
	if err != nil {
		return SearchQuery{}, ErrInvalidSearch
	}
	segments := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	var q SearchQuery
	host := strings.ToLower(u.Hostname())
	switch {
	case host == azureHost && len(segments) > 0:
		q.Organization, segments = segments[0], segments[1:]
	case strings.HasSuffix(host, legacyHostTail):
		q.Organization = strings.TrimSuffix(host, legacyHostTail)
	default:
		return SearchQuery{}, ErrInvalidSearch
	}
	if len(segments) > 0 && !strings.HasPrefix(segments[0], "_") {
		q.Project = segments[0]
	}
	q.ID, q.Kind = linkTarget(segments, u.Query())
	if q.ID <= 0 {
		return SearchQuery{}, ErrInvalidSearch
	}
	return q, nil
}

func linkTarget(segments []string, query url.Values) (int, SearchKind) {
	for i := 0; i+1 < len(segments); i++ {
		id, err := strconv.Atoi(segments[i+1])
		if err != nil {
			continue
		}
		switch strings.ToLower(segments[i]) {
		case "pullrequest":
			return id, SearchPullRequest
		case "edit":
			if i > 0 && strings.EqualFold(segments[i-1], "_workitems") {
				return id, SearchWorkItem
			}
		}
	}
	if id, err := strconv.Atoi(query.Get("workitem")); err == nil {
		return id, SearchWorkItem
	}
	return 0, SearchAny
}
