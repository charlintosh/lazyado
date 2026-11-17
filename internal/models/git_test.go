package models

import (
	"testing"
	"time"
)

func TestValidateRefNameFollowsGitRules(t *testing.T) {
	for _, ok := range []string{"main", "feature/1102-saved-carts", "release/2026.11", "v1.0.0"} {
		if err := ValidateRefName(ok); err != nil {
			t.Errorf("%q must be valid, got %v", ok, err)
		}
	}
	for _, bad := range []string{"", "has space", "a..b", "/lead", "trail/", "x.lock", "a:b", "-x", "a//b", "end."} {
		if ValidateRefName(bad) == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
}

func TestParseCommitQuerySplitsQualifiersFromText(t *testing.T) {
	c, text := ParseCommitQuery("rounding author:Priya path:src/cart.go fix")
	if c.Author != "Priya" || c.Path != "/src/cart.go" || text != "rounding fix" {
		t.Fatalf("got %+v %q", c, text)
	}
	if c.Query() != "author:Priya path:/src/cart.go" {
		t.Fatalf("got %q", c.Query())
	}
}

func TestBranchIsStaleAfterThreeMonthsWithoutCommits(t *testing.T) {
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	if (Branch{Last: CommitRef{Date: now.AddDate(0, -2, 0)}}).Stale(now) {
		t.Error("two months is not stale")
	}
	if !(Branch{Last: CommitRef{Date: now.AddDate(0, -4, 0)}}).Stale(now) {
		t.Error("four months is stale")
	}
}

func TestSortGitItemsPutsFoldersFirst(t *testing.T) {
	got := SortGitItems([]GitItem{{Path: "/b.go"}, {Path: "/z", Folder: true}, {Path: "/A.md"}})
	if got[0].Path != "/z" || got[1].Path != "/A.md" || got[2].Path != "/b.go" {
		t.Fatalf("got %+v", got)
	}
}
