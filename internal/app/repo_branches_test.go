package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func sampleBranches() []models.Branch {
	now := time.Now()
	last := func(id, author string, at time.Time) models.CommitRef {
		return models.CommitRef{ID: id, Author: author, Date: at, Comment: "work"}
	}
	return []models.Branch{
		{Name: "main", ObjectID: "a3f9c21", Default: true, Locked: true, Last: last("a3f9c21", "Priya Nair", now.Add(-48*time.Hour))},
		{Name: "feature/1043-guest-checkout", ObjectID: "3f2a91c", Mine: true, Ahead: 5, Behind: 4, PullRequestID: 4826, Last: last("3f2a91c", "Priya Nair", now.Add(-5*time.Hour))},
		{Name: "release/2026.11", ObjectID: "7c1e0b4", Ahead: 0, Behind: 2, Last: last("7c1e0b4", "Tomás Ruiz", now.Add(-12*24*time.Hour))},
		{Name: "spike/old-cache", ObjectID: "9e04d7a", Ahead: 3, Behind: 80, Last: last("9e04d7a", "Jordan Lee", now.AddDate(0, -5, 0))},
	}
}

func openBranchesHub(t testing.TB, width, height int) App {
	t.Helper()
	a := openFilesHub(t, width, height)
	a = feed(t, a, press("V"))
	if !a.inBranches() || !a.branches.loading || a.branches.loadedFor != "r1" {
		t.Fatal("V must open the Branches hub and load the repository's branches")
	}
	return feed(t, a, branchesLoadedMsg{seq: a.branches.seq, branches: sampleBranches()})
}

func TestBranchesHubShowsMineAllAndStale(t *testing.T) {
	a := openBranchesHub(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Repos / Branches / storefront-api", "New branch", "1 Mine 2", "2 All 4", "3 Stale 1", "BRANCHES",
		"main  Default   Locked", "feature/1043-guest-checkout", "     4 | 5", "!4826 active", "ahead and behind main"} {
		if !strings.Contains(out, want) {
			t.Errorf("Branches hub is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "release/2026.11") {
		t.Error("Mine must not list branches created by someone else")
	}
	a = feed(t, a, press("3"))
	if out := ansi.Strip(a.render()); !strings.Contains(out, "spike/old-cache") || strings.Contains(out, "feature/1043") {
		t.Errorf("Stale must list only branches without commits in 3 months:\n%s", out)
	}
	a = feed(t, a, press("2"), press("/"), tea.PasteMsg{Content: "release"})
	if b := a.branches.panel.Selected(); b == nil || b.Name != "release/2026.11" {
		t.Fatalf("search must filter by name, got %+v", b)
	}
}

func TestNewBranchFormValidatesAndCreatesFromTheSelectedBranch(t *testing.T) {
	a := openBranchesHub(t, 160, 45)
	a = feed(t, a, press("j"), press("n"))
	form := topModal[modals.RefForm](t, a)
	if out := ansi.Strip(form.View()); !strings.Contains(out, "New branch") || !strings.Contains(out, "feature/1043-guest-checkout") {
		t.Fatalf("the form must start from the selected branch:\n%s", out)
	}
	a = feed(t, a, tea.PasteMsg{Content: "main"}, ctrl('s'))
	if out := ansi.Strip(topModal[modals.RefForm](t, a).View()); !strings.Contains(out, "a branch named main already exists") {
		t.Fatalf("an existing name must be refused:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyBackspace), special(tea.KeyBackspace), special(tea.KeyBackspace), special(tea.KeyBackspace))
	a = feed(t, a, tea.PasteMsg{Content: "feature/1102-saved-carts"})
	_, msgs := step(t, a, ctrl('s'))
	req, ok := find[modals.CreateBranchRequestMsg](msgs)
	if !ok || req.Name != "feature/1102-saved-carts" || req.Base != models.BranchVersion("feature/1043-guest-checkout") {
		t.Fatalf("got %+v from %v", req, msgs)
	}
}

func TestCreatedBranchClosesTheFormReloadsAndSelectsIt(t *testing.T) {
	a := openBranchesHub(t, 160, 45)
	a = feed(t, a, press("n"))
	a = feed(t, a, refChangedMsg{notice: "Branch feature/x created", name: "feature/x"})
	if a.modals.open() || !a.branches.loading || !a.code.refsLoading {
		t.Fatal("success must close the form and reload the branches and the refs")
	}
	branches := append(sampleBranches(), models.Branch{Name: "feature/x", Mine: true})
	a = feed(t, a, branchesLoadedMsg{seq: a.branches.seq, branches: branches})
	if b := a.branches.panel.Selected(); b == nil || b.Name != "feature/x" {
		t.Fatalf("the new branch must be selected, got %+v", b)
	}
}

func TestFailedRefChangeKeepsTheFormUnderTheError(t *testing.T) {
	a := openBranchesHub(t, 160, 45)
	a = feed(t, a, press("n"), refChangedMsg{err: errors.New("TF401027: You need the Git 'CreateBranch' permission")})
	topModal[modals.ErrorModal](t, a)
	a, msgs := step(t, a, special(tea.KeyEscape))
	a = feed(t, a, msgs...)
	topModal[modals.RefForm](t, a)
}

func TestDeleteBranchConfirmsAndRefusesTheDefaultBranch(t *testing.T) {
	a := openBranchesHub(t, 160, 45)
	a = feed(t, a, press("d"))
	if a.modals.open() {
		t.Fatal("the default branch must not offer deletion")
	}
	a = feed(t, a, press("j"), press("d"))
	confirm := topModal[modals.ConfirmModal](t, a)
	out := ansi.Strip(confirm.View())
	for _, want := range []string{"Delete branch", "feature/1043-guest-checkout", "!4826", "restore"} {
		if !strings.Contains(out, want) {
			t.Errorf("delete dialog is missing %q:\n%s", want, out)
		}
	}
	_, msgs := step(t, a, press("y"))
	req, ok := find[modals.DeleteBranchRequestMsg](msgs)
	if !ok || req.Branch.ObjectID != "3f2a91c" {
		t.Fatalf("y must ask to delete with the branch tip, got %v", msgs)
	}
}

func TestBranchShortcutsOpenFilesCommitsAndAPullRequest(t *testing.T) {
	a := openBranchesHub(t, 160, 45)
	a = feed(t, a, press("j"))
	files := feed(t, a, special(tea.KeyEnter))
	if !files.inFiles() || files.code.version != models.BranchVersion("feature/1043-guest-checkout") {
		t.Fatalf("enter must browse the branch's files, got %+v", files.code.version)
	}
	commits := feed(t, a, press("H"))
	if !commits.inCommits() || commits.code.version.Name != "feature/1043-guest-checkout" {
		t.Fatalf("H must show the branch's commits, got %+v", commits.code.version)
	}
	pr := feed(t, a, press("p"))
	topModal[modals.PullRequestForm](t, pr)
}

func TestBranchesHubFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {120, 32}, {90, 26}, {70, 22}, {50, 14}} {
		a := openBranchesHub(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
		n := feed(t, a, press("n"))
		assertFits(t, n.render(), sz[0], sz[1])
		d := feed(t, a, press("j"), press("d"))
		assertFits(t, d.render(), sz[0], sz[1])
	}
}
