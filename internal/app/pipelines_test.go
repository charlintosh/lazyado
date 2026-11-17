package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func sampleRuns() []models.PipelineRun {
	now := time.Now()
	run := func(id int, number, pipeline string, pid int, status, result, ref, message string, ago time.Duration) models.PipelineRun {
		r := models.PipelineRun{
			ID: id, Number: number, PipelineID: pid, PipelineName: pipeline, PipelineFolder: `\CI`, Status: status, Result: result,
			Reason: "individualCI", Ref: ref, Commit: "3f9c2ab8e1d", Message: message, RequestedFor: models.Identity{DisplayName: "Dana Ruiz"},
			Queued: now.Add(-ago), Started: now.Add(-ago + 5*time.Second), WebURL: "https://dev.azure.com/acme/storefront/_build/results?buildId=" + number,
		}
		if status == "completed" {
			r.Finished = r.Started.Add(4 * time.Minute)
		}
		return r
	}
	return []models.PipelineRun{
		run(812, "20261117.6", "storefront-web-ci", 7, "inProgress", "", "refs/heads/feature/1043-guest-checkout", "Add guest checkout feature flag", 3*time.Minute),
		run(811, "20261117.3", "payments-api-ci", 9, "completed", "succeeded", "refs/pull/4831/merge", "Retry idempotency key on 409", 9*time.Minute),
		run(805, "20261117.1", "nightly-e2e", 11, "completed", "failed", "refs/heads/main", "", time.Hour),
		run(799, "20261116.9", "storefront-web-ci", 7, "completed", "canceled", "refs/heads/main", "Rework header nav", 26*time.Hour),
	}
}

func openPipelinesHub(t testing.TB, width, height int) App {
	t.Helper()
	a := newTestAppAs(t, width, height, reviewer)
	a = feed(t, a, press("D"))
	if !a.inPipelines() || !a.pl.loading {
		t.Fatal("R must open the Pipelines › Builds hub and load the recent pipelines")
	}
	return feed(t, a, pipelineListMsg{seq: a.pl.seq, mode: panels.ListRecent, pipelines: models.LatestPerPipeline(sampleRuns(), 10)})
}

func TestPipelinesHubListsRecentlyRunPipelines(t *testing.T) {
	a := openPipelinesHub(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Recently run pipelines 3", "storefront-web-ci", "#20261117.6", "Add guest checkout feature flag", "nightly-e2e", "FAIL", "1 Recent", "2 All", "3 Runs", "Run pipeline"} {
		if !strings.Contains(out, want) {
			t.Errorf("recent view is missing %q", want)
		}
	}
	if !strings.Contains(out, "Pipelines / Builds / Recent") || !strings.Contains(out, "BUILDS") {
		t.Error("the breadcrumb and status bar should name the Pipelines › Builds hub")
	}
}

func TestPipelinesHubPollsWhileARunIsActive(t *testing.T) {
	a := newTestAppAs(t, 160, 45, reviewer)
	a = feed(t, a, press("D"))
	m, cmd := a.Update(pipelineListMsg{seq: a.pl.seq, mode: panels.ListRecent, pipelines: models.LatestPerPipeline(sampleRuns(), 10)})
	a = m.(App)
	if cmd == nil || a.pl.pollSeq == 0 {
		t.Fatal("an active run must schedule a refresh")
	}
	a = feed(t, a, pipelinePollMsg{seq: a.pl.pollSeq - 1})
	if a.pl.loading {
		t.Fatal("an outdated poll must be ignored")
	}
}

func TestSearchingRecentSearchesEveryPipelineOnTheServer(t *testing.T) {
	a := openPipelinesHub(t, 160, 45)
	a = feed(t, a, panels.SearchPipelinesMsg{Text: "pay"})
	if a.pl.pivot != pipelinesAll || a.pl.pattern != "*pay*" || !a.pl.loading {
		t.Fatalf("pivot %d pattern %q loading %v", a.pl.pivot, a.pl.pattern, a.pl.loading)
	}
	a = feed(t, a, pipelineListMsg{seq: a.pl.seq, mode: panels.ListAll, pipelines: []models.Pipeline{
		{ID: 9, Name: "payments-api-ci", Folder: `\CI`, Latest: &sampleRuns()[1]},
		{ID: 21, Name: "payments-db-backup", Folder: `\Infra`},
	}})
	out := ansi.Strip(a.render())
	for _, want := range []string{`▾ \CI`, `▾ \Infra`, "payments-api-ci", "Never run", "/ pay"} {
		if !strings.Contains(out, want) {
			t.Errorf("All view is missing %q", want)
		}
	}
}

func TestRunsSearchReportsAnUnknownState(t *testing.T) {
	a := openPipelinesHub(t, 160, 45)
	a = feed(t, a, press("3"))
	a = feed(t, a, pipelineListMsg{seq: a.pl.seq, mode: panels.ListRuns, runs: sampleRuns()})
	a, msgs := step(t, a, panels.SearchPipelinesMsg{Text: "state:green"})
	list, ok := find[pipelineListMsg](msgs)
	if !ok || list.searchErr == "" {
		t.Fatalf("an unknown state must not reach Azure DevOps, got %v", msgs)
	}
	a = feed(t, a, list)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "state: can be running") {
		t.Errorf("the search error should be shown:\n%s", out)
	}
}

func TestPipelinePageShowsRunsAndBranches(t *testing.T) {
	a := openPipelinesHub(t, 160, 45)
	a, msgs := step(t, a, special(tea.KeyEnter))
	open, ok := find[panels.OpenPipelineMsg](msgs)
	if !ok {
		t.Fatalf("enter must ask to open the pipeline, got %v", msgs)
	}
	a = feed(t, a, open)
	if a.pl.page == nil || a.pl.page.Name != "storefront-web-ci" {
		t.Fatalf("enter must open the selected pipeline, got %+v", a.pl.page)
	}
	runs := []models.PipelineRun{sampleRuns()[0], sampleRuns()[3]}
	extra := sampleRuns()[3]
	extra.ID, extra.Number = 790, "20261116.2"
	runs = append(runs, extra)
	a = feed(t, a, pipelineListMsg{seq: a.pl.seq, mode: panels.ListPipelineRuns, runs: runs, token: "next"})
	out := ansi.Strip(a.render())
	for _, want := range []string{"Runs 3", "#20261116.9", "1 Runs", "2 Branches", "More load as you scroll"} {
		if !strings.Contains(out, want) {
			t.Errorf("pipeline page is missing %q", want)
		}
	}
	a = feed(t, a, press("2"))
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Branches 2") || !strings.Contains(out, "feature/1043-guest-checkout") {
		t.Errorf("branches view should list the latest run per branch:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyEscape))
	if a.pl.page != nil || !a.pl.loading {
		t.Fatal("esc must go back to the hub list")
	}
}

func sampleTimeline() []models.TimelineRecord {
	start := time.Now().Add(-2 * time.Minute)
	return []models.TimelineRecord{
		{ID: "s1", Type: "Stage", Name: "Build", Order: 1, State: models.RunSucceeded, Started: start, Finished: start.Add(time.Minute)},
		{ID: "s2", Type: "Stage", Name: "Test", Order: 2, State: models.RunRunning, Started: start.Add(time.Minute)},
		{ID: "p1", ParentID: "s1", Type: "Phase", Name: "Build", Order: 1},
		{ID: "j1", ParentID: "p1", Type: "Job", Name: "Compile", Order: 1, State: models.RunSucceeded, Worker: "Hosted Agent", LogID: 3},
		{ID: "t1", ParentID: "j1", Type: "Task", Name: "npm ci", Order: 1, State: models.RunSucceeded, LogID: 4},
		{ID: "p2", ParentID: "s2", Type: "Phase", Name: "Test", Order: 1},
		{ID: "j2", ParentID: "p2", Type: "Job", Name: "Integration tests", Order: 1, State: models.RunRunning, Worker: "Hosted Agent 14", LogID: 8},
		{ID: "t2", ParentID: "j2", Type: "Task", Name: "Checkout", Order: 1, State: models.RunSucceeded, LogID: 9,
			Issues: []models.RunIssue{{Message: "shallow fetch"}}},
		{ID: "t3", ParentID: "j2", Type: "Task", Name: "Run integration suite", Order: 2, State: models.RunRunning, Started: start, LogID: 12},
	}
}

func openRunView(t testing.TB, width, height int) App {
	t.Helper()
	a := openPipelinesHub(t, width, height)
	run := sampleRuns()[0]
	a = feed(t, a, panels.OpenRunMsg{Run: run})
	if !a.viewingRun() {
		t.Fatal("opening a run must show the run view")
	}
	return feed(t, a, runTimelineMsg{id: run.ID, records: sampleTimeline()}, runChangesMsg{id: run.ID, changes: []models.RunChange{{ID: "3f9c2ab8e1", Message: "Add guest checkout flag"}}})
}

func TestRunViewSummaryShowsStagesJobsIssuesAndDetails(t *testing.T) {
	a := openRunView(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{"#20261117.6", "Stages and jobs", "Build", "Integration tests", "Errors 0 · warnings 1", "shallow fetch", "Details", "Changes 1", "1 Summary", "2 Logs", "RUN"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary is missing %q", want)
		}
	}
}

func TestRunViewLogsLoadTheRunningStepAndFollowIt(t *testing.T) {
	a := openRunView(t, 160, 45)
	m, cmd := a.Update(press("2"))
	a = m.(App)
	if cmd == nil {
		t.Fatal("the logs tab must ask for the selected step's log")
	}
	if logID, _, ok := a.pl.view.WantedLog(); ok {
		t.Fatalf("the log %d was requested twice", logID)
	}
	a = feed(t, a, runLogMsg{id: 812, logID: 12, from: 1, lines: []string{"##[section]Starting: Run integration suite", "PASS checkout/guest.int.ts"}})
	out := ansi.Strip(a.render())
	for _, want := range []string{"Run integration suite", "Agent: Hosted Agent 14", "PASS checkout/guest.int.ts", "FOLLOWING", "LOGS"} {
		if !strings.Contains(out, want) {
			t.Errorf("logs tab is missing %q", want)
		}
	}
	a.pl.view.MarkStale()
	if logID, from, ok := a.pl.view.WantedLog(); !ok || logID != 12 || from != 3 {
		t.Fatalf("a running step must be tailed from the next line, got %d %d %v", logID, from, ok)
	}
}

func TestRunViewFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {120, 32}, {80, 24}, {50, 14}} {
		a := openRunView(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
		a = feed(t, a, press("2"), runLogMsg{id: 812, logID: 12, from: 1, lines: []string{strings.Repeat("x", 300)}})
		assertFits(t, a.render(), sz[0], sz[1])
	}
}

func TestPipelinesHubFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {120, 32}, {90, 26}, {70, 22}, {50, 14}} {
		a := openPipelinesHub(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
		a = feed(t, a, press("3"), pipelineListMsg{seq: a.pl.seq + 1, mode: panels.ListRuns, runs: sampleRuns()})
		a = feed(t, a, pipelineListMsg{seq: a.pl.seq, mode: panels.ListRuns, runs: sampleRuns()})
		assertFits(t, a.render(), sz[0], sz[1])
	}
}

func runDefinition() models.PipelineDefinition {
	return models.PipelineDefinition{
		Pipeline:     models.Pipeline{ID: 7, Name: "storefront-web-ci"},
		RepositoryID: "r1", RepositoryType: "TfsGit", DefaultBranch: "refs/heads/main", YAMLPath: "/azure-pipelines.yml",
		Variables: []models.PipelineVariable{{Name: "LOG_LEVEL", Value: "info"}},
	}
}

func runSpec() models.PipelineSpec {
	return models.PipelineSpec{
		Parameters: []models.PipelineParameter{
			{Name: "environment", DisplayName: "Environment", Type: models.ParamString, Default: "staging", Values: []string{"dev", "staging", "prod"}},
			{Name: "runE2E", DisplayName: "Run E2E tests", Type: models.ParamBoolean, Default: "true"},
		},
		Stages: []models.PipelineStageSpec{{Name: "Build"}, {Name: "DeployProd", DisplayName: "Deploy prod"}},
	}
}

func openRunForm(t *testing.T, width, height int) App {
	t.Helper()
	a := openPipelinesHub(t, width, height)
	a, msgs := step(t, a, press("n"))
	form := topModal[modals.RunPipelineForm](t, a)
	if form.PipelineID() != 7 {
		t.Fatalf("the form must run the selected pipeline, got %d", form.PipelineID())
	}
	if _, ok := find[modals.RunDefinitionRequestMsg](msgs); !ok {
		t.Fatalf("the form must ask for the pipeline definition, got %v", msgs)
	}
	refs := []models.GitRef{{Name: "refs/heads/main", Commit: "5d22e18aa"}, {Name: "refs/heads/release/2.4"}, {Name: "refs/tags/v2.4.0"}}
	a, msgs = step(t, a, modals.RunDefinitionMsg{PipelineID: 7, Definition: runDefinition(), Refs: refs})
	req, ok := find[modals.RunSpecRequestMsg](msgs)
	if !ok || req.Ref != "refs/heads/main" || req.Path != "/azure-pipelines.yml" {
		t.Fatalf("the form must read the YAML at the default branch, got %+v", msgs)
	}
	return feed(t, a, modals.RunSpecMsg{PipelineID: 7, Ref: "refs/heads/main", Spec: runSpec()})
}

func TestRunFormShowsParametersStagesAndVariables(t *testing.T) {
	a := openRunForm(t, 160, 60)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Run pipeline", "storefront-web-ci", "Branch or tag", "Parameters", "Environment", "staging", "Run E2E tests", "Stages to run", "Deploy prod", "LOG_LEVEL", "System diagnostics", "azure-pipelines.yml at main"} {
		if !strings.Contains(out, want) {
			t.Errorf("run form is missing %q", want)
		}
	}
}

func TestRunFormSendsTheChosenOptions(t *testing.T) {
	a := openRunForm(t, 160, 60)
	tab := special(tea.KeyTab)
	a = feed(t, a, tab, tab, tab, special(tea.KeyEnter), tab, special(tea.KeyDown), special(tea.KeyEnter))
	_, msgs := step(t, a, ctrl('s'))
	req, ok := find[modals.RunPipelineRequestMsg](msgs)
	if !ok {
		t.Fatalf("ctrl+s must ask to run the pipeline, got %v", msgs)
	}
	o := req.Options
	if o.Ref != "refs/heads/main" || o.Parameters["environment"] != "staging" || o.Parameters["runE2E"] != "false" {
		t.Errorf("options %+v", o)
	}
	if len(o.StagesToSkip) != 1 || o.StagesToSkip[0] != "DeployProd" {
		t.Errorf("stages to skip %v", o.StagesToSkip)
	}
	if len(o.Variables) != 0 || o.Diagnostics {
		t.Errorf("unchanged variables must not be sent: %+v", o)
	}
}

func TestStartedRunClosesTheFormAndOpensItsLogs(t *testing.T) {
	a := openRunForm(t, 160, 60)
	run := sampleRuns()[0]
	run.ID, run.Number, run.Status = 900, "20261117.7", "notStarted"
	a = feed(t, a, runStartedMsg{run: run})
	if a.modals.open() || !a.viewingRun() || a.pl.view.Tab() != panels.RunLogsTab || a.pl.view.Run().ID != 900 {
		t.Fatal("a started run must close the form and open its logs")
	}
}

func TestRunFormFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{160, 45}, {80, 24}, {50, 14}} {
		a := openRunForm(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
	}
}

func TestPipelinePageKeepsALoadThatArrivesAfterSwitchingToBranches(t *testing.T) {
	a := openPipelinesHub(t, 160, 45)
	a = feed(t, a, panels.OpenPipelineMsg{Pipeline: models.Pipeline{ID: 7, Name: "storefront-web-ci"}}, press("2"))
	a = feed(t, a, pipelineListMsg{seq: a.pl.seq, mode: panels.ListPipelineRuns, runs: sampleRuns()[:1]})
	if a.pl.loading || a.pl.panel.Mode() != panels.ListBranches {
		t.Fatalf("loading %v mode %v", a.pl.loading, a.pl.panel.Mode())
	}
}
