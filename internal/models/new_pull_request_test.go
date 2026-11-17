package models

import "testing"

func TestBranchNamesSuggestATitleAndAWorkItem(t *testing.T) {
	cases := []struct {
		branch string
		title  string
		item   int
	}{
		{"feature/1043-guest-checkout", "Guest checkout", 1043},
		{"bugfix/audit_tenant", "Audit tenant", 0},
		{"users/dana/12-fix", "Fix", 0},
		{"main", "Main", 0},
		{"feature/AB#2201_retry-queue", "AB#2201 retry queue", 2201},
	}
	for _, c := range cases {
		if got := TitleFromBranch(c.branch); got != c.title {
			t.Errorf("TitleFromBranch(%q) = %q, want %q", c.branch, got, c.title)
		}
		if got := WorkItemFromBranch(c.branch); got != c.item {
			t.Errorf("WorkItemFromBranch(%q) = %d, want %d", c.branch, got, c.item)
		}
	}
	one := BranchComparison{Commits: []string{"Checkout: allow guests\n\nLonger body"}}
	if got := one.SuggestedTitle("feature/x"); got != "Checkout: allow guests" {
		t.Errorf("a single commit's subject must be the title, got %q", got)
	}
	if got := (BranchComparison{Commits: []string{"a", "b"}}).SuggestedTitle("feature/1043-guest-checkout"); got != "Guest checkout" {
		t.Errorf("several commits must fall back to the branch name, got %q", got)
	}
}
