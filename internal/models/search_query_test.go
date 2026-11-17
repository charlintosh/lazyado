package models

import (
	"errors"
	"testing"
)

func TestParseSearchQueryReadsIDsMarksAndLinks(t *testing.T) {
	cases := []struct {
		input string
		want  SearchQuery
	}{
		{"1234", SearchQuery{ID: 1234, Kind: SearchAny}},
		{"  !88 ", SearchQuery{ID: 88, Kind: SearchPullRequest}},
		{"#42", SearchQuery{ID: 42, Kind: SearchWorkItem}},
		{"https://dev.azure.com/contoso/Web/_git/lazyado/pullrequest/1234",
			SearchQuery{ID: 1234, Kind: SearchPullRequest, Organization: "contoso", Project: "Web"}},
		{"https://dev.azure.com/contoso/Web/_git/lazyado/pullrequest/1234?_a=files&path=/a.go",
			SearchQuery{ID: 1234, Kind: SearchPullRequest, Organization: "contoso", Project: "Web"}},
		{"dev.azure.com/contoso/My%20Project/_workitems/edit/77",
			SearchQuery{ID: 77, Kind: SearchWorkItem, Organization: "contoso", Project: "My Project"}},
		{"https://contoso.visualstudio.com/Web/_workitems/edit/9/",
			SearchQuery{ID: 9, Kind: SearchWorkItem, Organization: "contoso", Project: "Web"}},
		{"https://dev.azure.com/contoso/Web/_boards/board/t/Team/Stories?workitem=31",
			SearchQuery{ID: 31, Kind: SearchWorkItem, Organization: "contoso", Project: "Web"}},
	}
	for _, c := range cases {
		got, err := ParseSearchQuery(c.input)
		if err != nil || got != c.want {
			t.Errorf("ParseSearchQuery(%q) = %+v, %v; want %+v", c.input, got, err, c.want)
		}
	}
}

func TestParseSearchQueryRejectsUnusableInput(t *testing.T) {
	cases := map[string]error{
		"":                                  ErrEmptySearch,
		"abc":                               ErrInvalidSearch,
		"!0":                                ErrInvalidSearch,
		"#-3":                               ErrInvalidSearch,
		"https://github.com/o/r/pull/12":    ErrInvalidSearch,
		"https://dev.azure.com/contoso/Web": ErrInvalidSearch,
	}
	for input, want := range cases {
		if _, err := ParseSearchQuery(input); !errors.Is(err, want) {
			t.Errorf("ParseSearchQuery(%q) error = %v, want %v", input, err, want)
		}
	}
}

func TestSearchQueryFlagsLinksFromAnotherProject(t *testing.T) {
	q := SearchQuery{ID: 1, Kind: SearchPullRequest, Organization: "Fabrikam", Project: "Ops"}
	if !q.Foreign("contoso", "Web") {
		t.Fatal("a link from another organization must be foreign")
	}
	if (SearchQuery{Organization: "CONTOSO", Project: "web"}).Foreign("contoso", "Web") {
		t.Fatal("organization and project must match without regard to case")
	}
	if (SearchQuery{ID: 5}).Foreign("contoso", "Web") {
		t.Fatal("a bare ID is never foreign")
	}
}
