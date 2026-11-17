package models

import "testing"

func TestIdentityOwnsOnlyItsOwnComments(t *testing.T) {
	me := Identity{ID: "6f1c2e7a-0000-4b1d-9c55-1f0e2d3c4b5a", DisplayName: "Ana Ruiz", UniqueName: "ana@contoso.com"}
	cases := []struct {
		name    string
		me      Identity
		comment Comment
		want    bool
	}{
		{"same identity id", me, Comment{CreatedByID: "6F1C2E7A-0000-4B1D-9C55-1F0E2D3C4B5A", CreatedBy: "Ana Ruiz"}, true},
		{"id wins over a matching upn", me, Comment{CreatedByID: "another-id", CreatedByUniqueName: "ana@contoso.com"}, false},
		{"upn when the comment has no id", me, Comment{CreatedByUniqueName: "ANA@contoso.com"}, true},
		{"display name alone is never enough", me, Comment{CreatedBy: "Ana Ruiz"}, false},
		{"someone else", me, Comment{CreatedByID: "other", CreatedByUniqueName: "ben@contoso.com", CreatedBy: "Ben Okafor"}, false},
		{"unknown user owns nothing", Identity{}, Comment{CreatedByID: "", CreatedBy: ""}, false},
	}
	for _, c := range cases {
		if got := c.me.Owns(c.comment); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestReviewerNameDropsTheGroupScope(t *testing.T) {
	cases := map[string]string{
		`[DLLG]\Release Approvers`: "Release Approvers",
		"[storefront]/Web Team":    "Web Team",
		"Dana Reyes":               "Dana Reyes",
		"[TEAM FOUNDATION]":        "[TEAM FOUNDATION]",
	}
	for in, want := range cases {
		if got := (Reviewer{DisplayName: in}).Name(); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
