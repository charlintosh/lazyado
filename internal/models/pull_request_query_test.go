package models

import (
	"slices"
	"strings"
	"testing"
)

func queryPullRequests() []PullRequest {
	return []PullRequest{
		{ID: 4821, Title: "Add retry policy to payment webhook client", SourceBranch: "feature/webhook-retry", TargetBranch: "main",
			Author: Identity{ID: "me", DisplayName: "Dana Reyes"}, Reviewers: []Reviewer{{ID: "pn", DisplayName: "Priya Nair"}}},
		{ID: 4799, Title: "Exponential backoff", IsDraft: true, SourceBranch: "feature/backoff", TargetBranch: "release/2.4",
			Author: Identity{ID: "ml", DisplayName: "Marco Lima"}},
	}
}

func matchingIDs(input string, me Identity) []int {
	q := ParsePullRequestQuery(input)
	var ids []int
	for _, pr := range queryPullRequests() {
		if q.Matches(pr, me) {
			ids = append(ids, pr.ID)
		}
	}
	return ids
}

func TestPullRequestQueryMatchesTextAndQualifiers(t *testing.T) {
	me := Identity{ID: "me"}
	cases := map[string][]int{
		"retry":                   {4821},
		"RETRY webhook":           {4821},
		"4799":                    {4799},
		"!4821":                   {4821},
		"backoff":                 {4799},
		"author:@me":              {4821},
		"author:marco":            {4799},
		`reviewer:"Priya Nair"`:   {4821},
		"target:release":          {4799},
		"is:draft":                {4799},
		"source:feature is:draft": {4799},
		"author:@me backoff":      nil,
		"":                        {4821, 4799},
	}
	for input, want := range cases {
		if got := matchingIDs(input, me); !slices.Equal(got, want) {
			t.Errorf("%q matched %v, want %v", input, got, want)
		}
	}
}

func TestPullRequestQueryBuildsServerCriteria(t *testing.T) {
	members := []TeamMember{{ID: "pn", DisplayName: "Priya Nair", UniqueName: "priya@example.com"}}
	q := ParsePullRequestQuery(`author:@me reviewer:priya source:feature/x target:main retry`)
	c, err := q.Criteria(members, Identity{ID: "me"})
	if err != nil {
		t.Fatal(err)
	}
	want := PullRequestCriteria{CreatorID: "me", ReviewerID: "pn", SourceRef: "refs/heads/feature/x", TargetRef: "refs/heads/main"}
	if c != want {
		t.Fatalf("got %+v, want %+v", c, want)
	}
	if _, err := ParsePullRequestQuery("reviewer:nobody").Criteria(members, Identity{}); err == nil || !strings.Contains(err.Error(), `"nobody"`) {
		t.Fatalf("an unknown reviewer must be reported, got %v", err)
	}
	if c, _ := ParsePullRequestQuery("retry is:draft").Criteria(members, Identity{}); !c.Empty() {
		t.Fatalf("text and is:draft never reach the server, got %+v", c)
	}
}

func TestPullRequestQueryKnowsTheTokenBeingTyped(t *testing.T) {
	if q := ParsePullRequestQuery("retry rev"); q.Last != "rev" {
		t.Fatalf("Last = %q, want rev", q.Last)
	}
	if q := ParsePullRequestQuery("retry "); q.Last != "" {
		t.Fatalf("after a space nothing is being typed, got %q", q.Last)
	}
	if id, ok := ParsePullRequestQuery("4511").ID(); !ok || id != 4511 {
		t.Fatalf("a bare number is an ID, got %d %v", id, ok)
	}
}
