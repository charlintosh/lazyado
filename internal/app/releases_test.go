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

var releaseApprover = models.Identity{ID: "u7", DisplayName: "Dana Reyes", UniqueName: "dana@example.com"}

func sampleDefinitions() []models.ReleaseDefinition {
	now := time.Now()
	return []models.ReleaseDefinition{
		{ID: 3, Name: "Analytics - Prod Install", LastRelease: "Release-12", LastCreated: now.Add(-3 * time.Hour)},
		{ID: 12, Name: "smartHotel", LastRelease: "Release-64", LastCreated: now.Add(-2 * time.Hour)},
		{ID: 20, Name: "payments-api - Prod", LastRelease: "Release-207", LastCreated: now.Add(-time.Hour)},
	}
}

func sampleRelease() models.Release {
	now := time.Now()
	qaApproval := models.ReleaseApproval{ID: 91, Kind: models.ApprovalPre, Status: "pending", Approver: models.Identity{DisplayName: "QA Leads"}, Group: true,
		Created: now.Add(-2 * time.Hour), ReleaseID: 64, ReleaseName: "Release-64", DefinitionID: 12, Definition: "smartHotel", StageID: 2, StageName: "QA"}
	deployJob := models.RunJob{TimelineRecord: models.TimelineRecord{ID: "3", Name: "Agent job", State: models.RunSucceeded, LogID: 103, LogURL: "https://vsrm/logs/3", Worker: "Hosted Agent"}, Stage: "Dev1",
		Steps: []models.TimelineRecord{{ID: "4", Name: "Azure App Service deploy", State: models.RunSucceeded, LogID: 104, LogURL: "https://vsrm/logs/4"}}}
	return models.Release{
		ID: 64, Name: "Release-64", DefinitionID: 12, Definition: "smartHotel", CreatedBy: models.Identity{DisplayName: "Gopi R"}, Created: now.Add(-2 * time.Hour),
		Artifacts: []models.ReleaseArtifact{{Alias: "_smartHotel-CI", Type: models.ArtifactBuild, Primary: true, Version: "20170509.2", Branch: "refs/heads/master"}},
		Stages: []models.ReleaseStage{
			{ID: 1, Name: "Dev1", State: models.StageSucceeded, Trigger: "after release", Attempt: 1, Jobs: []models.RunJob{deployJob}},
			{ID: 2, Name: "QA", State: models.StageWaitingApproval, Trigger: "after Dev1", Approvals: []models.ReleaseApproval{qaApproval}},
			{ID: 3, Name: "Prod", State: models.StageNotDeployed, Trigger: "manual", Approvers: []string{"Ops on-call"}},
		},
	}
}

func openReleasesHub(t testing.TB, width, height int) App {
	t.Helper()
	a := newTestAppAs(t, width, height, releaseApprover)
	a = feed(t, a, press("R"))
	if !a.inReleases() || !a.rl.loading {
		t.Fatal("R must open the Releases hub and load the release pipelines")
	}
	a = feed(t, a, releaseDefinitionsMsg{seq: a.rl.seq, defs: sampleDefinitions(), starred: []int{12}})
	if a.rl.listFor != 12 {
		t.Fatalf("the starred pipeline must be selected first, got %d", a.rl.listFor)
	}
	a = feed(t, a, releasesDueMsg{definition: 12})
	return feed(t, a, releasesLoadedMsg{seq: a.rl.listSeq, definition: 12, releases: []models.Release{sampleRelease()}})
}

func TestReleasesHubListsStarredPipelinesFirstWithTheirReleases(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	out := ansi.Strip(a.render())
	starred, all := strings.Index(out, "★ STARRED 1"), strings.Index(out, "ALL RELEASE PIPELINES 2")
	if starred < 0 || all < 0 || starred > all {
		t.Fatalf("starred pipelines must come first:\n%s", out)
	}
	for _, want := range []string{"Pipelines / Releases / smartHotel", "smartHotel · Releases 1", "Release-64", "Releases", "Created", "Stages", "✓ Dev1", "◐ QA", "○ Prod", "_smartHotel-CI · 20170509.2 · master", "Gopi R", "smartHotel ", "1 Releases", "2 Approvals", "Create release", "RELEASES"} {
		if !strings.Contains(out, want) {
			t.Errorf("releases hub is missing %q", want)
		}
	}
}

func TestReleaseSearchFiltersThenEnterGoesIntoTheMatchAndEscClearsIt(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, press("/"), tea.PasteMsg{Content: "pay"})
	if d := a.rl.defs.Selected(); d == nil || d.ID != 20 {
		t.Fatalf("typing must select the first match, got %+v", d)
	}
	a, msgs := step(t, a, special(tea.KeyEnter))
	focus, ok := find[panels.FocusReleasesMsg](msgs)
	if !ok {
		t.Fatalf("Enter must go into the match's releases, got %v", msgs)
	}
	a = feed(t, a, focus)
	if a.rl.defs.Searching() || a.rl.pane != paneReleases {
		t.Fatal("Enter must close the bar and focus the releases")
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "· /pay") || strings.Contains(out, "smartHotel") {
		t.Errorf("the filter must stay applied and visible:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyEscape))
	if a.rl.pane != paneDefinitions || a.rl.defs.Filter() != "pay" {
		t.Fatal("Esc in the releases must go back to the filtered pipelines")
	}
	a = feed(t, a, special(tea.KeyEscape))
	if a.rl.defs.Filter() != "" || a.rl.defs.Selected().ID != 20 {
		t.Fatal("Esc in the pipelines must clear the filter and keep the selection")
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "smartHotel") {
		t.Errorf("clearing the filter must show every pipeline again:\n%s", out)
	}
}

func TestReleaseSearchAsksAzureDevOpsOnlyWhenMorePipelinesExist(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, press("/"), tea.PasteMsg{Content: "rates"}, panels.ReleaseSearchTypedMsg{Text: "rates"})
	if a.rl.defs.Loading() {
		t.Fatal("with every pipeline loaded, nothing should be sent to Azure DevOps")
	}
	a = feed(t, a, releaseDefinitionsMsg{seq: a.rl.seq, defs: sampleDefinitions(), token: "next", starred: []int{12}})
	a = feed(t, a, panels.ReleaseSearchTypedMsg{Text: "rates"})
	if !a.rl.defs.Loading() {
		t.Fatal("with more pipelines on the server, a search must start")
	}
	a = feed(t, a, releaseSearchMsg{seq: a.rl.searchSeq, text: "rates", defs: []models.ReleaseDefinition{{ID: 31, Name: "smartHotel - rates", LastRelease: "Release-31"}}})
	out := ansi.Strip(a.render())
	for _, want := range []string{"None of the 3 loaded match.", "FOUND ON AZURE DEVOPS 1", "smartHotel - rates"} {
		if !strings.Contains(out, want) {
			t.Errorf("server results are missing %q:\n%s", want, out)
		}
	}
	if d := a.rl.defs.Selected(); d == nil || d.ID != 31 {
		t.Fatalf("a server result must be selectable, got %+v", d)
	}
}

func TestStarredGroupShowsEvenWhenEmpty(t *testing.T) {
	a := newTestAppAs(t, 160, 45, releaseApprover)
	a = feed(t, a, press("R"))
	a = feed(t, a, releaseDefinitionsMsg{seq: a.rl.seq, defs: sampleDefinitions()})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "★ STARRED 0") || !strings.Contains(out, "None yet. Ctrl+t stars a pipeline.") {
		t.Errorf("the Starred group must explain how to star:\n%s", out)
	}
	a = feed(t, a, panels.StarReleaseDefinitionMsg{Definition: models.ReleaseDefinition{ID: 20, Name: "payments-api - Prod"}, Star: true})
	if s := a.savedState(); len(s.FavoriteReleases) != 1 || s.FavoriteReleases[0] != 20 {
		t.Fatalf("a local star must be saved, got %v", s.FavoriteReleases)
	}
	if out := ansi.Strip(a.render()); strings.Contains(out, "None yet") {
		t.Errorf("the hint must go once a pipeline is starred:\n%s", out)
	}
}

func TestViewOpensTheLatestReleaseOfAPipeline(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a, msgs := step(t, a, press("v"))
	open, ok := find[panels.OpenLatestReleaseMsg](msgs)
	if !ok || open.DefinitionID != 12 {
		t.Fatalf("v must ask for the latest release, got %v", msgs)
	}
	a = feed(t, a, open)
	if !a.viewingRelease() || a.rl.view.ID() != 64 {
		t.Fatal("v must open the newest loaded release")
	}
	b := openReleasesHub(t, 160, 45)
	b = feed(t, b, panels.OpenLatestReleaseMsg{DefinitionID: 3, Name: "Analytics - Prod Install"})
	if !b.loading {
		t.Fatal("for a pipeline whose releases aren't loaded, v must look the latest one up")
	}
	b = feed(t, b, latestReleaseMsg{id: 12, name: "Analytics - Prod Install"})
	if !b.viewingRelease() || b.rl.view.ID() != 12 {
		t.Fatal("the looked-up release must open")
	}
}

func TestApprovalsPivotListsMineFirstAndOpensTheDialog(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, press("2"))
	if !a.rl.approvalsLoading {
		t.Fatal("the Approvals view must load pending approvals")
	}
	mine := sampleRelease().Stages[1].Approvals[0]
	other := models.ReleaseApproval{ID: 95, Kind: models.ApprovalPre, Status: "pending", Approver: models.Identity{DisplayName: "Ops on-call"}, ReleaseID: 63, ReleaseName: "Release-63", DefinitionID: 12, Definition: "smartHotel", StageName: "Staging"}
	elsewhere := models.ReleaseApproval{ID: 96, Kind: models.ApprovalPre, Status: "pending", Approver: models.Identity{DisplayName: "Payments leads"}, ReleaseID: 207, ReleaseName: "Release-207", DefinitionID: 20, Definition: "payments-api - Prod", StageName: "Prod"}
	a = feed(t, a, approvalsLoadedMsg{seq: a.rl.approvalsSeq, mine: []models.ReleaseApproval{mine}, others: []models.ReleaseApproval{other, elsewhere}})
	out := ansi.Strip(a.render())
	for _, want := range []string{"Approvals · smartHotel 2", "Waiting on you 1", "Waiting on others 1", "QA · Release-64", "you can approve", "2 Approvals  1", "A all pipelines"} {
		if !strings.Contains(out, want) {
			t.Errorf("approvals view is missing %q", want)
		}
	}
	for _, unwanted := range []string{"Prod · Release-207", "Release-64 · smartHotel"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("approvals scoped to smartHotel must not show %q:\n%s", unwanted, out)
		}
	}
	if c := a.rl.approvals.Counts(); c[12] != (panels.ApprovalCount{Mine: 1, Others: 1}) || c[20] != (panels.ApprovalCount{Others: 1}) {
		t.Errorf("each pipeline must count its own approvals, got %v", c)
	}
	all := feed(t, a, press("A"))
	out = ansi.Strip(all.render())
	for _, want := range []string{"Approvals 3 · all pipelines", "Prod · Release-207 · payments-api - Prod", "QA · Release-64 · smartHotel", "A this pipeline"} {
		if !strings.Contains(out, want) {
			t.Errorf("A must widen approvals to every pipeline, missing %q", want)
		}
	}
	other20 := feed(t, a, press("h"), press("j"))
	if d := other20.rl.defs.Selected(); d == nil || d.ID == 12 {
		t.Fatalf("j on the pipeline pane must pick another pipeline, got %+v", d)
	}
	a, msgs := step(t, a, press("a"))
	req, ok := find[panels.ApproveReleaseMsg](msgs)
	if !ok || req.Approval.ID != 91 {
		t.Fatalf("a must ask to decide the selected approval, got %v", msgs)
	}
	a = feed(t, a, req)
	topModal[modals.ApprovalModal](t, a)
	_, msgs = step(t, a, ctrl('s'))
	decision, ok := find[modals.ReleaseApprovalRequestMsg](msgs)
	if !ok || !decision.Approve || decision.Approval.ID != 91 {
		t.Fatalf("Ctrl+s must approve, got %v", msgs)
	}
}

func TestRejectingNeedsAComment(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, panels.ApproveReleaseMsg{Approval: sampleRelease().Stages[1].Approvals[0]}, press("l"))
	_, msgs := step(t, a, ctrl('s'))
	if _, ok := find[modals.ReleaseApprovalRequestMsg](msgs); ok {
		t.Fatal("a rejection without a comment must not be sent")
	}
	a = feed(t, a, special(tea.KeyTab), tea.PasteMsg{Content: "Booking regression"})
	_, msgs = step(t, a, ctrl('s'))
	decision, ok := find[modals.ReleaseApprovalRequestMsg](msgs)
	if !ok || decision.Approve || decision.Comment != "Booking regression" {
		t.Fatalf("got %+v %v", decision, ok)
	}
}

func openReleaseView(t testing.TB, width, height int) App {
	t.Helper()
	a := openReleasesHub(t, width, height)
	a = feed(t, a, panels.OpenReleaseMsg{ReleaseID: 64})
	if !a.viewingRelease() {
		t.Fatal("opening a release must show the release view")
	}
	return feed(t, a, releaseLoadedMsg{release: sampleRelease()})
}

func TestReleaseViewShowsStagesApprovalAndDeploysAStage(t *testing.T) {
	a := openReleaseView(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Release-64 · smartHotel", "Stages", "Dev1", "QA", "Prod", "pre-deployment approval", "QA Leads (any one member)", "press a to approve or reject", "Artifacts", "Stage triggers", "1 Pipeline", "2 Logs"} {
		if !strings.Contains(out, want) {
			t.Errorf("release view is missing %q", want)
		}
	}
	a = feed(t, a, press("l"))
	a, msgs := step(t, a, press("d"))
	deploy, ok := find[panels.DeployStageMsg](msgs)
	if !ok || deploy.Stage.Name != "Prod" {
		t.Fatalf("d must deploy the selected stage, got %v", msgs)
	}
	a = feed(t, a, deploy)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Deploy Prod") || !strings.Contains(out, "Ops on-call must approve") {
		t.Errorf("the deploy dialog should warn about the approval:\n%s", out)
	}
	_, msgs = step(t, a, ctrl('s'))
	req, ok := find[modals.DeployStageRequestMsg](msgs)
	if !ok || req.ReleaseID != 64 || req.StageID != 3 {
		t.Fatalf("Ctrl+s must deploy Prod, got %v", msgs)
	}
}

func TestReleaseViewRefusesToDeployAStageWaitingForApproval(t *testing.T) {
	a := openReleaseView(t, 160, 45)
	_, msgs := step(t, a, press("d"))
	notice, ok := find[panels.ReleaseNoticeMsg](msgs)
	if !ok || !strings.Contains(notice.Text, "waiting for approval") {
		t.Fatalf("got %v", msgs)
	}
}

func TestReleaseLogsLoadFromTheTaskLogURL(t *testing.T) {
	a := openReleaseView(t, 160, 45)
	a = feed(t, a, press("h"), special(tea.KeyEnter))
	if a.rl.view.Tab() != panels.ReleaseLogsTab {
		t.Fatal("enter on a deployed stage must open its logs")
	}
	if _, _, _, ok := a.rl.view.WantedLog(); ok {
		t.Fatal("reconcile must have requested the selected step's log")
	}
	a = feed(t, a, releaseLogMsg{release: 64, logID: 104, from: 1, lines: []string{"##[section]Starting: Azure App Service deploy", "Deployed to smarthotel-dev"}})
	out := ansi.Strip(a.render())
	for _, want := range []string{"Agent job", "Azure App Service deploy", "Deployed to smarthotel-dev", "› JOBS"} {
		if !strings.Contains(out, want) {
			t.Errorf("release logs are missing %q", want)
		}
	}
}

func TestReleaseLogsShowOnlyTheSelectedStage(t *testing.T) {
	a := openReleaseView(t, 160, 45)
	if s, _ := a.rl.view.SelectedStage(); s.Name != "QA" {
		t.Fatalf("the stage waiting for approval must be picked first, got %q", s.Name)
	}
	a = feed(t, a, press("2"))
	out := ansi.Strip(a.render())
	for _, want := range []string{"Stage", "Jobs · QA", "No jobs in this stage yet.", "< > switch stage"} {
		if !strings.Contains(out, want) {
			t.Errorf("logs for QA are missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Agent job") || strings.Contains(out, "Jobs in run") {
		t.Errorf("logs for QA must not list Dev1's jobs:\n%s", out)
	}
	a = feed(t, a, press("<"))
	if s, _ := a.rl.view.SelectedStage(); s.Name != "Dev1" {
		t.Fatalf("< must move to the previous stage, got %q", s.Name)
	}
	out = ansi.Strip(a.render())
	for _, want := range []string{"Jobs · Dev1", "Agent job", "1 job"} {
		if !strings.Contains(out, want) {
			t.Errorf("logs for Dev1 are missing %q:\n%s", want, out)
		}
	}
	a = feed(t, a, press("1"))
	if s, _ := a.rl.view.SelectedStage(); s.Name != "Dev1" {
		t.Fatalf("the Pipeline tab must keep the stage chosen in Logs, got %q", s.Name)
	}
}

func releaseSpec() models.ReleaseDefinitionSpec {
	return models.ReleaseDefinitionSpec{
		ReleaseDefinition: models.ReleaseDefinition{ID: 12, Name: "smartHotel"},
		Artifacts:         []models.ReleaseArtifact{{Alias: "_smartHotel-CI", Type: models.ArtifactBuild, Primary: true, DefinitionID: 7}},
		Stages:            []models.ReleaseStageSpec{{Name: "Dev1", Automated: true, Trigger: "after release"}, {Name: "QA", Automated: true, Trigger: "after Dev1"}, {Name: "Prod", Trigger: "manual"}},
		Variables:         []models.ReleaseVariable{{Name: "Region", Value: "westeurope"}},
	}
}

func openCreateRelease(t *testing.T, width, height int) App {
	t.Helper()
	a := openReleasesHub(t, width, height)
	a, msgs := step(t, a, press("n"))
	form := topModal[modals.CreateReleaseForm](t, a)
	if form.DefinitionID() != 12 {
		t.Fatalf("the form must create a release of the selected pipeline, got %d", form.DefinitionID())
	}
	if _, ok := find[modals.ReleaseDefinitionRequestMsg](msgs); !ok {
		t.Fatalf("the form must load the release pipeline, got %v", msgs)
	}
	builds := []models.PipelineRun{
		{ID: 812, Number: "20170509.2", Status: "completed", Result: "succeeded", Ref: "refs/heads/master", Commit: "3f9c2ab8e1"},
		{ID: 800, Number: "20170508.3", Status: "completed", Result: "partiallySucceeded", Ref: "refs/heads/hotfix/rates"},
	}
	return feed(t, a, modals.ReleaseDefinitionMsg{DefinitionID: 12, Spec: releaseSpec(), Builds: map[string][]models.PipelineRun{"_smartHotel-CI": builds}})
}

func TestCreateReleaseShowsArtifactsStagesAndVariables(t *testing.T) {
	a := openCreateRelease(t, 160, 60)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Create a new release", "Artifacts", "_smartHotel-CI · branch", "Any branch", "_smartHotel-CI · version", "20170509.2", "Deploy automatically", "Dev1 (after release)", "Always manual: Prod", "Region", "Description"} {
		if !strings.Contains(out, want) {
			t.Errorf("create release form is missing %q", want)
		}
	}
}

func TestCreateReleaseSendsTheChosenVersionAndManualStages(t *testing.T) {
	a := openCreateRelease(t, 160, 60)
	tab := special(tea.KeyTab)
	a = feed(t, a, tab, tab, special(tea.KeyEnter))
	_, msgs := step(t, a, ctrl('s'))
	req, ok := find[modals.CreateReleaseRequestMsg](msgs)
	if !ok {
		t.Fatalf("Ctrl+s must create the release, got %v", msgs)
	}
	r := req.Release
	if r.DefinitionID != 12 || len(r.Artifacts) != 1 || r.Artifacts[0].VersionID != "812" || r.Artifacts[0].Branch != "refs/heads/master" {
		t.Errorf("artifacts %+v", r.Artifacts)
	}
	if len(r.ManualStages) != 1 || r.ManualStages[0] != "Dev1" || len(r.Variables) != 0 {
		t.Errorf("release %+v", r)
	}
}

func TestCreatedReleaseOpensItsPipeline(t *testing.T) {
	a := openCreateRelease(t, 160, 60)
	created := sampleRelease()
	created.ID, created.Name = 65, "Release-65"
	a = feed(t, a, releaseCreatedMsg{release: created})
	if a.modals.open() || !a.viewingRelease() || a.rl.view.ID() != 65 {
		t.Fatal("a created release must close the form and open the release")
	}
}

func TestReleasesLayoutsFitTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {120, 32}, {80, 24}, {60, 20}, {50, 14}} {
		a := openReleasesHub(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
		approvals := feed(t, a, press("2"))
		approvals = feed(t, approvals, approvalsLoadedMsg{seq: approvals.rl.approvalsSeq, mine: sampleRelease().Stages[1].Approvals})
		assertFits(t, approvals.render(), sz[0], sz[1])
		view := openReleaseView(t, sz[0], sz[1])
		assertFits(t, view.render(), sz[0], sz[1])
		logs := feed(t, view, press("2"))
		assertFits(t, logs.render(), sz[0], sz[1])
		deploy := feed(t, view, panels.DeployStageMsg{Release: sampleRelease(), Stage: sampleRelease().Stages[2]})
		assertFits(t, deploy.render(), sz[0], sz[1])
	}
}

func TestReleaseDialogsFitTerminal(t *testing.T) {
	for _, sz := range [][2]int{{160, 45}, {80, 24}, {50, 14}} {
		form := openCreateRelease(t, sz[0], sz[1])
		assertFits(t, form.render(), sz[0], sz[1])
		a := feed(t, openReleasesHub(t, sz[0], sz[1]), panels.ApproveReleaseMsg{Approval: sampleRelease().Stages[1].Approvals[0]})
		assertFits(t, a.render(), sz[0], sz[1])
	}
}

func TestHubPanesOnlyNumberTheMainViewWhereTheKeyWorks(t *testing.T) {
	for name, open := range map[string]func(testing.TB, int, int) App{
		"pull requests": openPullRequestHub,
		"builds":        openPipelinesHub,
		"releases":      openReleasesHub,
	} {
		a := open(t, 120, 32)
		if out := ansi.Strip(a.render()); strings.Contains(out, " 5  ") {
			t.Errorf("%s: a pane title still shows the 5 key:\n%s", name, out)
		}
	}
	backlog := newTestApp(t, 120, 32)
	if out := ansi.Strip(backlog.render()); !strings.Contains(out, " 5  ") {
		t.Error("the backlog still focuses its main view with 5, so its pane should keep the key")
	}
}

func TestReleaseViewCancelsAStageThatIsWaitingOrDeploying(t *testing.T) {
	a := openReleaseView(t, 160, 45)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "press x to cancel this deployment") {
		t.Errorf("a waiting stage should offer to cancel:\n%s", out)
	}
	a, msgs := step(t, a, press("x"))
	cancel, ok := find[panels.CancelDeploymentMsg](msgs)
	if !ok || cancel.Stage.Name != "QA" {
		t.Fatalf("x must ask to cancel the selected stage, got %v", msgs)
	}
	a = feed(t, a, cancel)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Cancel the QA deployment") || !strings.Contains(out, "pending approval is withdrawn") {
		t.Errorf("the cancel dialog should explain what happens:\n%s", out)
	}
	a, msgs = step(t, a, ctrl('s'))
	req, ok := find[modals.CancelDeploymentRequestMsg](msgs)
	if !ok || req.ReleaseID != 64 || req.StageID != 2 {
		t.Fatalf("Ctrl+s must cancel QA, got %v", msgs)
	}
	a = feed(t, a, req)
	if a.modals.open() || !a.loading {
		t.Fatal("confirming must close the dialog and send the cancel")
	}
}

func TestReleaseViewWontCancelAStageThatIsNotDeploying(t *testing.T) {
	a := openReleaseView(t, 160, 45)
	a = feed(t, a, press("l"))
	_, msgs := step(t, a, press("x"))
	notice, ok := find[panels.ReleaseNoticeMsg](msgs)
	if !ok || !strings.Contains(notice.Text, "no deployment to cancel") {
		t.Fatalf("got %v", msgs)
	}
}

func TestCancelDeploymentDialogFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{160, 45}, {80, 24}, {50, 14}} {
		a := feed(t, openReleaseView(t, sz[0], sz[1]), panels.CancelDeploymentMsg{Release: sampleRelease(), Stage: sampleRelease().Stages[1]})
		assertFits(t, a.render(), sz[0], sz[1])
	}
}

func TestCtrlTStarsWhileSearchingAndFromTheReleasesPane(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, press("/"), tea.PasteMsg{Content: "pay"})
	a, msgs := step(t, a, ctrl('t'))
	star, ok := find[panels.StarReleaseDefinitionMsg](msgs)
	if !ok || star.Definition.ID != 20 || !star.Star {
		t.Fatalf("Ctrl+t in the search bar must star the highlighted match, got %v", msgs)
	}
	a = feed(t, a, star)
	if !a.rl.defs.Searching() || !a.rl.defs.Starred(20) {
		t.Fatal("starring keeps the search open and stars the match")
	}
	if !strings.Contains(ansi.Strip(a.render()), "Ctrl+t unstar") {
		t.Error("the search bar must offer Ctrl+t")
	}
	a, msgs = step(t, a, special(tea.KeyEnter))
	a = feed(t, a, msgs...)
	if a.rl.pane != paneReleases {
		t.Fatal("enter goes into the match's releases")
	}
	_, msgs = step(t, a, ctrl('t'))
	if unstar, ok := find[panels.StarReleaseDefinitionMsg](msgs); !ok || unstar.Definition.ID != 20 || unstar.Star {
		t.Fatalf("Ctrl+t in the releases pane must toggle their pipeline, got %v", msgs)
	}
}

func TestStarringAPipelineThatIsNotLoadedKeepsItInStarred(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	far := models.ReleaseDefinition{ID: 99, Name: "pay-legacy - Archive"}
	a = feed(t, a, panels.StarReleaseDefinitionMsg{Definition: far, Star: true})
	if !strings.Contains(ansi.Strip(a.render()), "pay-legacy - Archive") {
		t.Fatal("a pipeline starred from Found on Azure DevOps must show in Starred")
	}
	a.rl.localStars = append(a.rl.localStars, 77)
	if missing := a.missingStarred(); len(missing) != 1 || missing[0] != 77 {
		t.Fatalf("starred pipelines that aren't loaded must be fetched, got %v", missing)
	}
	a = feed(t, a, starredDefinitionsMsg{seq: a.rl.seq, defs: []models.ReleaseDefinition{{ID: 77, Name: "infra - Nightly"}, {ID: 12, Name: "smartHotel"}}})
	if len(a.missingStarred()) != 0 || !strings.Contains(ansi.Strip(a.render()), "infra - Nightly") {
		t.Fatal("fetched starred pipelines join the list once")
	}
	count := 0
	for _, d := range a.rl.definitions {
		if d.ID == 12 {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("pipelines must not be listed twice, got %d", count)
	}
}

func TestReleaseListMarksWhatIsDeployedNowAndTheLatestDeploymentOfEachPipeline(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	now := time.Now()
	older := sampleRelease()
	older.ID, older.Name = 63, "Release-63"
	older.Stages = []models.ReleaseStage{
		{Name: "Dev1", State: models.StageSucceeded, Finished: now.Add(-3 * time.Hour)},
		{Name: "QA", State: models.StageSucceeded, Finished: now.Add(-2 * time.Hour)},
		{Name: "Prod", State: models.StageFailed},
	}
	a = feed(t, a, releasesLoadedMsg{seq: a.rl.listSeq, definition: 12, releases: []models.Release{sampleRelease(), older}})
	current := models.CurrentDeployments(a.rl.releases)
	if !current[64]["Dev1"] || !current[63]["QA"] || current[63]["Dev1"] || current[63]["Prod"] {
		t.Fatalf("the newest successful deployment of each stage is current, got %v", current)
	}
	a.rl.defs.SetLatest(map[int]models.DefinitionDeployment{20: {DefinitionID: 20, Release: "Release-207", Stage: "Prod-EU", State: models.StageFailed, When: now.Add(-time.Hour)}})
	out := ansi.Strip(a.render())
	for _, want := range []string{"✗ Prod-EU · Release-207 · 1h ago", "✗ Prod", "Release-63"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}
