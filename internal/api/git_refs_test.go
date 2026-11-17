package api

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/models"
)

func TestListBranchDetailsJoinsStatsRefsAndMine(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/stats/branches": `{"value":[` +
			`{"name":"feature/1043-guest","aheadCount":5,"behindCount":4,"commit":{"commitId":"3f2a91c","author":{"name":"Priya Nair","date":"2026-11-17T08:00:00Z"},"comment":"Validate guest email"}},` +
			`{"name":"main","isBaseVersion":true,"commit":{"commitId":"a3f9c21"}}]}`,
		"/refs": `{"value":[{"name":"refs/heads/main","objectId":"a3f9c21","isLocked":true,"creator":{"displayName":"Jordan Lee"}},` +
			`{"name":"refs/heads/feature/1043-guest","objectId":"3f2a91c","creator":{"displayName":"Priya Nair"}}]}`,
	}, &seen)
	branches, err := c.ListBranchDetails("r1", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 2 || branches[0].Name != "main" || !branches[0].Default || !branches[0].Locked {
		t.Fatalf("the default branch must come first and carry its lock, got %+v", branches)
	}
	b := branches[1]
	if b.Ahead != 5 || b.Behind != 4 || b.ObjectID != "3f2a91c" || b.Creator != "Priya Nair" || !b.Mine || b.Last.Subject() != "Validate guest email" {
		t.Fatalf("got %+v", b)
	}
	if !strings.Contains(seen[0].url, "baseVersionDescriptor.version=main") || !strings.Contains(seen[2].url, "includeMyBranches=true") {
		t.Errorf("asked %s and %s", seen[0].url, seen[2].url)
	}
	if strings.Contains(seen[2].url, "top=") {
		t.Errorf("Azure DevOps refuses paging with includeMyBranches (400), asked %s", seen[2].url)
	}
}

func TestListBranchDetailsStillListsBranchesWhenMineFails(t *testing.T) {
	c := NewClient(&config.Config{Organization: "contoso", Project: "web", PAT: "test-pat-not-real"})
	c.httpClient.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		reply, status := `{"value":[]}`, http.StatusOK
		switch {
		case strings.Contains(r.URL.RawQuery, "includeMyBranches"):
			reply, status = `{"message":"Pagination is not supported when 'includeMyBranches' is requested."}`, http.StatusBadRequest
		case strings.Contains(r.URL.Path, "/stats/branches"):
			reply = `{"value":[{"name":"main","isBaseVersion":true}]}`
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(reply)), Header: http.Header{}}, nil
	})
	branches, err := c.ListBranchDetails("r1", "main")
	if err != nil || len(branches) != 1 || branches[0].Mine {
		t.Fatalf("a failed Mine lookup must not fail the list, got %+v, %v", branches, err)
	}
}

func TestCreateAndDeleteBranchPostRefUpdates(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/refs": `{"value":[{"success":true}]}`}, &seen)
	if err := c.CreateBranch("r1", "feature/x", "a3f9c21"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteBranch("r1", "feature/x", "a3f9c21"); err != nil {
		t.Fatal(err)
	}
	create, del := seen[0], seen[1]
	if create.method != "POST" || !strings.Contains(create.body, `"name":"refs/heads/feature/x"`) ||
		!strings.Contains(create.body, `"oldObjectId":"`+zeroObjectID+`"`) || !strings.Contains(create.body, `"newObjectId":"a3f9c21"`) {
		t.Fatalf("create sent %s %s", create.method, create.body)
	}
	if !strings.Contains(del.body, `"oldObjectId":"a3f9c21"`) || !strings.Contains(del.body, `"newObjectId":"`+zeroObjectID+`"`) {
		t.Fatalf("delete sent %s", del.body)
	}
}

func TestRefUpdateRefusedByAzureDevOpsIsAnError(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/refs": `{"value":[{"success":false,"updateStatus":"rejectedByPolicy","customMessage":"Branch is locked"}]}`}, &seen)
	err := c.DeleteBranch("r1", "main", "a3f9c21")
	if err == nil || !strings.Contains(err.Error(), "Branch is locked") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveCommitReadsTheTipOfABranchOrTag(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/commits": `{"value":[{"commitId":"7c1e0b4"}]}`}, &seen)
	id, err := c.ResolveCommit("r1", models.GitVersion{Kind: models.VersionTag, Name: "v1.0"})
	if err != nil || id != "7c1e0b4" {
		t.Fatalf("got %q, %v", id, err)
	}
	if !strings.Contains(seen[0].url, "versionType=tag") || !strings.Contains(seen[0].url, "%24top=1") {
		t.Errorf("asked %s", seen[0].url)
	}
	if id, _ := c.ResolveCommit("r1", models.GitVersion{Kind: models.VersionCommit, Name: "abc"}); id != "abc" || len(seen) != 1 {
		t.Error("a commit resolves to itself without a request")
	}
}

func TestListTagsPeelsAnnotatedTagsToTheirCommit(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/refs": `{"value":[` +
		`{"name":"refs/tags/v2026.11.1","objectId":"t111","peeledObjectId":"a3f9c21"},` +
		`{"name":"refs/tags/perf-baseline","objectId":"45532e0"}]}`}, &seen)
	tags, err := c.ListTags("r1")
	if err != nil || len(tags) != 2 {
		t.Fatalf("got %v, %v", tags, err)
	}
	if tags[0].Name != "perf-baseline" || tags[0].Annotated() || tags[0].CommitID != "45532e0" {
		t.Fatalf("a lightweight tag points at its commit, got %+v", tags[0])
	}
	if !tags[1].Annotated() || tags[1].CommitID != "a3f9c21" {
		t.Fatalf("an annotated tag must be peeled, got %+v", tags[1])
	}
	if !strings.Contains(seen[0].url, "peelTags=true") {
		t.Errorf("asked %s", seen[0].url)
	}
}

func TestCreateTagPostsAnAnnotatedTag(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/annotatedtags": `{"name":"v1.4.0","message":"Hotfix","taggedBy":{"name":"Priya Nair"},"taggedObject":{"objectId":"a3f9c21"}}`}, &seen)
	d, err := c.CreateTag("r1", "v1.4.0", "a3f9c21", "Hotfix")
	if err != nil || d.Name != "v1.4.0" || d.Tagger != "Priya Nair" {
		t.Fatalf("got %+v, %v", d, err)
	}
	body := seen[0].body
	for _, want := range []string{`"name":"v1.4.0"`, `"taggedObject":{"objectId":"a3f9c21"}`, `"message":"Hotfix"`} {
		if seen[0].method != "POST" || !strings.Contains(body, want) {
			t.Errorf("sent %s %s, missing %s", seen[0].method, body, want)
		}
	}
}

func TestGetAnnotatedTagAcceptsATaggerDateWithoutTimezone(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/annotatedtags/t120": `{"name":"v.1.2.0","message":"Release","taggedBy":{"name":"Priya Nair","date":"2026-02-03T15:46:59"},"taggedObject":{"objectId":"a3f9c21"}}`}, &seen)
	d, err := c.GetAnnotatedTag("r1", "t120")
	if err != nil {
		t.Fatalf("Azure DevOps sends tagger dates without a zone; got %v", err)
	}
	if want := time.Date(2026, 2, 3, 15, 46, 59, 0, time.UTC); !d.Date.Equal(want) {
		t.Fatalf("got %v, want %v", d.Date, want)
	}
}

func TestGitTimeParsesTheFormatsAzureDevOpsSends(t *testing.T) {
	for raw, want := range map[string]time.Time{
		`"2026-11-15T15:03:00Z"`:          time.Date(2026, 11, 15, 15, 3, 0, 0, time.UTC),
		`"2026-11-15T15:03:00.123+01:00"`: time.Date(2026, 11, 15, 14, 3, 0, 123e6, time.UTC),
		`"2026-02-03T15:46:59"`:           time.Date(2026, 2, 3, 15, 46, 59, 0, time.UTC),
		`"2026-02-03T15:46:59.5"`:         time.Date(2026, 2, 3, 15, 46, 59, 5e8, time.UTC),
		`""`:                              {},
	} {
		var got gitTime
		if err := got.UnmarshalJSON([]byte(raw)); err != nil || !got.Equal(want) {
			t.Errorf("%s: got %v, %v", raw, got.Time, err)
		}
	}
	var bad gitTime
	if bad.UnmarshalJSON([]byte(`"yesterday"`)) == nil {
		t.Error("an unknown format must still be an error")
	}
}
