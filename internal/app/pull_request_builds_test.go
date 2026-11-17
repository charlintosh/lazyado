package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func buildChecks(ids ...int) []models.PullRequestCheck {
	checks := sampleChecks()
	names := []string{"Build: storefront-web-ci", "ci/e2e"}
	for i, id := range ids {
		checks = append(checks, models.PullRequestCheck{Name: names[i], State: models.CheckFailed, Required: true, BuildID: id})
	}
	return checks
}

func TestReviewOpensTheOnlyBuildAndEscComesBack(t *testing.T) {
	a := openReview(t, 160, 45)
	pr := *a.pr.view.PullRequest()
	a = feed(t, a, checksLoadedMsg{id: pr.ID, checks: buildChecks(812)})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "Policies b") || !strings.Contains(out, "b open build") {
		t.Errorf("the review should offer b when a check points to a build:\n%s", out)
	}
	a, msgs := step(t, a, press("b"))
	builds, ok := find[panels.PullRequestBuildsMsg](msgs)
	if !ok || len(builds.Builds) != 1 {
		t.Fatalf("b must ask to open the builds, got %v", msgs)
	}
	a = feed(t, a, builds)
	if !a.viewingRun() || a.pl.view.Run().ID != 812 {
		t.Fatal("a single build must open in the run view")
	}
	a = feed(t, a, panels.CloseRunViewMsg{})
	if !a.reviewing() {
		t.Fatal("Esc from the build must return to the pull request")
	}
}

func TestReviewAsksWhichBuildWhenThereAreSeveral(t *testing.T) {
	a := openReview(t, 160, 45)
	pr := *a.pr.view.PullRequest()
	a = feed(t, a, checksLoadedMsg{id: pr.ID, checks: buildChecks(812, 900)})
	a, msgs := step(t, a, press("b"))
	builds, _ := find[panels.PullRequestBuildsMsg](msgs)
	a = feed(t, a, builds)
	topModal[modals.ChoiceModal](t, a)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Builds") || !strings.Contains(out, "ci/e2e") {
		t.Errorf("the picker should list the builds:\n%s", out)
	}
	_, msgs = step(t, a, special(tea.KeyEnter))
	chosen, ok := find[modals.BuildChosenMsg](msgs)
	if !ok || chosen.BuildID != 812 {
		t.Fatalf("Enter must choose the highlighted build, got %v", msgs)
	}
	a = feed(t, a, chosen)
	if a.modals.open() || !a.viewingRun() || a.pl.view.Run().ID != 812 {
		t.Fatal("the chosen build must open")
	}
}

func TestPullRequestListOpensTheSelectedPullRequestsBuild(t *testing.T) {
	a := openPullRequestHub(t, 160, 45)
	a = feed(t, a, checksLoadedMsg{id: a.pr.checksFor, checks: buildChecks(812)})
	if out := ansi.Strip(a.render()); !strings.Contains(out, "b open build") {
		t.Errorf("the list should offer b for the selected pull request:\n%s", out)
	}
	a = feed(t, a, press("b"))
	if !a.viewingRun() || a.pl.view.Run().ID != 812 {
		t.Fatal("b in the list must open the build")
	}
	a = feed(t, a, panels.CloseRunViewMsg{})
	if a.viewMode != ViewMain || !a.inPullRequests() {
		t.Fatal("Esc must return to the pull request list")
	}
}

func TestPullRequestWithoutBuildsSaysSo(t *testing.T) {
	a := openPullRequestHub(t, 160, 45)
	a = feed(t, a, press("b"))
	if a.viewingRun() || !strings.Contains(ansi.Strip(a.render()), "No policy or status check") {
		t.Fatal("b without a build must explain why nothing opens")
	}
}
