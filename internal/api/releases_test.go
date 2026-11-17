package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
)

const releaseJSON = `{"id":64,"name":"Release-64","status":"active","description":"Rates cache","createdOn":"2026-11-17T13:02:00Z",` +
	`"createdBy":{"id":"u1","displayName":"Gopi R"},"releaseDefinition":{"id":12,"name":"smartHotel"},` +
	`"_links":{"web":{"href":"https://dev.azure.com/acme/web/_releaseProgress?releaseId=64"}},` +
	`"artifacts":[{"alias":"_smartHotel-CI","type":"Build","isPrimary":true,"definitionReference":{"version":{"id":"812","name":"20170509.2"},"branch":{"id":"refs/heads/master","name":"refs/heads/master"},"definition":{"id":"7","name":"smartHotel-CI"}}},` +
	`{"alias":"infra","type":"Git","definitionReference":{"version":{"id":"c41b5d2","name":"c41b5d2"},"definition":{"id":"0b8e-guid","name":"infra"}}}],` +
	`"environments":[` +
	`{"id":2,"name":"QA","rank":2,"status":"notStarted","conditions":[{"conditionType":"environmentState","name":"Dev1"}],` +
	`"preDeployApprovals":[{"id":91,"approvalType":"preDeploy","status":"pending","approver":{"id":"g1","displayName":"QA Leads","isContainer":true},"isAutomated":false}],` +
	`"preApprovalsSnapshot":{"approvals":[{"isAutomated":false,"approver":{"displayName":"QA Leads"}}]}},` +
	`{"id":1,"name":"Dev1","rank":1,"status":"succeeded","conditions":[{"conditionType":"event","name":"ReleaseStarted"}],` +
	`"preDeployApprovals":[{"id":90,"approvalType":"preDeploy","status":"approved","isAutomated":true}],` +
	`"deploySteps":[{"attempt":1,"requestedFor":{"displayName":"system"},"queuedOn":"2026-11-17T13:03:00Z","releaseDeployPhases":[{"name":"Agent job","deploymentJobs":[` +
	`{"job":{"id":3,"name":"Agent job","status":"succeeded","agentName":"Hosted Agent","logUrl":"https://vsrm/logs/3"},` +
	`"tasks":[{"id":4,"name":"Deploy","status":"succeeded","logUrl":"https://vsrm/logs/4"},{"id":5,"name":"Smoke tests","status":"failed","logUrl":"https://vsrm/logs/5","issues":[{"issueType":"Error","message":"502"}]}]}]}]}]},` +
	`{"id":3,"name":"Prod","rank":3,"status":"notStarted","conditions":[]}]}`

func TestReleasesMapStagesApprovalsArtifactsAndTasks(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/release/releases": `{"value":[` + releaseJSON + `]}`}, &seen)
	releases, _, err := c.ListReleases(12, 25, "")
	if err != nil || len(releases) != 1 {
		t.Fatalf("got %v, %v", releases, err)
	}
	if !strings.HasPrefix(seen[0].url, "https://vsrm.dev.azure.com/contoso/web/_apis/release/releases?") ||
		!strings.Contains(seen[0].url, "definitionId=12") || !strings.Contains(seen[0].url, "%24expand=environments%2Cartifacts%2Capprovals") {
		t.Fatalf("asked %s", seen[0].url)
	}
	r := releases[0]
	if len(r.Stages) != 3 || r.Stages[0].Name != "Dev1" || r.Stages[2].Name != "Prod" {
		t.Fatalf("stages must follow their rank, got %+v", r.Stages)
	}
	dev, qa, prod := r.Stages[0], r.Stages[1], r.Stages[2]
	if dev.State != models.StageSucceeded || qa.State != models.StageWaitingApproval || prod.State != models.StageNotDeployed {
		t.Fatalf("states %v %v %v", dev.State, qa.State, prod.State)
	}
	if dev.Trigger != "after release" || qa.Trigger != "after Dev1" || prod.Trigger != "manual" {
		t.Fatalf("triggers %q %q %q", dev.Trigger, qa.Trigger, prod.Trigger)
	}
	if a, ok := qa.PendingApproval(); !ok || a.ID != 91 || !a.Group || a.ReleaseName != "Release-64" || a.StageName != "QA" {
		t.Fatalf("pending approval %+v", a)
	}
	if len(dev.Jobs) != 1 || len(dev.Jobs[0].Steps) != 2 || dev.Jobs[0].Steps[1].LogURL != "https://vsrm/logs/5" || dev.Jobs[0].Steps[1].LogID == 0 {
		t.Fatalf("jobs %+v", dev.Jobs)
	}
	if got := r.ArtifactSummary(); got != "_smartHotel-CI · 20170509.2 · master" {
		t.Fatalf("artifact summary %q", got)
	}
	if r.Artifacts[1].RepositoryID != "0b8e-guid" {
		t.Fatalf("git artifact %+v", r.Artifacts[1])
	}
}

func TestReleaseDefinitionsSearchByNameAndKeepTheLastRelease(t *testing.T) {
	var seen []recorded
	body := `{"value":[{"id":12,"name":"smartHotel","path":"\\","lastRelease":{"name":"Release-64","createdOn":"2026-11-17T13:02:00Z"}}]}`
	defs, _, err := clientRouting(t, map[string]string{"/release/definitions": body}, &seen).ListReleaseDefinitions("smart", "")
	if err != nil || len(defs) != 1 || defs[0].LastRelease != "Release-64" {
		t.Fatalf("got %+v, %v", defs, err)
	}
	if !strings.Contains(seen[0].url, "searchText=smart") || !strings.Contains(seen[0].url, "%24expand=lastRelease") {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestApprovalsAndStageDeploysSendTheDocumentedBodies(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{}, &seen)
	if err := c.DecideApproval(91, false, "Regression in booking"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeployReleaseStage(64, 3, "Redeploy"); err != nil {
		t.Fatal(err)
	}
	if seen[0].method != "PATCH" || !strings.Contains(seen[0].url, "/release/approvals/91?api-version=7.1") ||
		!strings.Contains(seen[0].body, `"status":"rejected"`) || !strings.Contains(seen[0].body, `"comments":"Regression in booking"`) {
		t.Errorf("approval %s %s %s", seen[0].method, seen[0].url, seen[0].body)
	}
	if seen[1].method != "PATCH" || !strings.Contains(seen[1].url, "/release/releases/64/environments/3?api-version=7.1-preview.7") ||
		!strings.Contains(seen[1].body, `"status":"inProgress"`) {
		t.Errorf("deploy %s %s %s", seen[1].method, seen[1].url, seen[1].body)
	}
}

func TestMyApprovalsIncludeGroupApprovals(t *testing.T) {
	var seen []recorded
	body := `{"value":[{"id":91,"approvalType":"preDeploy","status":"pending","approver":{"displayName":"QA Leads","isContainer":true},` +
		`"release":{"id":64,"name":"Release-64"},"releaseDefinition":{"id":12,"name":"smartHotel"},"releaseEnvironment":{"id":2,"name":"QA"}},` +
		`{"id":92,"approvalType":"preDeploy","status":"pending","isAutomated":true}]}`
	approvals, err := clientRouting(t, map[string]string{"/release/approvals": body}, &seen).ListReleaseApprovals("dana@example.com")
	if err != nil || len(approvals) != 1 || approvals[0].ReleaseID != 64 || approvals[0].StageName != "QA" {
		t.Fatalf("got %+v, %v", approvals, err)
	}
	for _, want := range []string{"assignedToFilter=dana%40example.com", "includeMyGroupApprovals=true", "statusFilter=pending"} {
		if !strings.Contains(seen[0].url, want) {
			t.Errorf("asked %s, missing %s", seen[0].url, want)
		}
	}
}

func TestReleaseDefinitionSpecReadsArtifactsStageTriggersAndVariables(t *testing.T) {
	var seen []recorded
	body := `{"id":12,"name":"smartHotel","artifacts":[{"alias":"_smartHotel-CI","type":"Build","isPrimary":true,"definitionReference":{"definition":{"id":"7","name":"smartHotel-CI"}}}],` +
		`"environments":[{"name":"Prod","rank":2,"conditions":[]},{"name":"Dev1","rank":1,"conditions":[{"conditionType":"event","name":"ReleaseStarted"}]}],` +
		`"variables":{"Region":{"value":"westeurope","allowOverride":true},"Locked":{"value":"x"}}}`
	spec, err := clientRouting(t, map[string]string{"/release/definitions/12": body}, &seen).GetReleaseDefinition(12)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Artifacts) != 1 || spec.Artifacts[0].DefinitionID != 7 || len(spec.Variables) != 1 || spec.Variables[0].Name != "Region" {
		t.Fatalf("got %+v", spec)
	}
	if len(spec.Stages) != 2 || spec.Stages[0].Name != "Dev1" || !spec.Stages[0].Automated || spec.Stages[1].Automated {
		t.Fatalf("stages %+v", spec.Stages)
	}
}

func TestCreateReleaseSendsArtifactsManualStagesAndVariables(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/release/releases": releaseJSON}, &seen)
	rel, err := c.CreateRelease(models.NewRelease{
		DefinitionID: 12, Description: "Rates cache",
		Artifacts:    []models.ArtifactChoice{{Alias: "_smartHotel-CI", VersionID: "812", Version: "20170509.2", Branch: "refs/heads/master"}},
		ManualStages: []string{"Staging"}, Variables: map[string]string{"Region": "northeurope"},
	})
	if err != nil || rel.ID != 64 {
		t.Fatalf("got %+v, %v", rel, err)
	}
	var body struct {
		DefinitionID int `json:"definitionId"`
		Artifacts    []struct {
			Alias    string            `json:"alias"`
			Instance map[string]string `json:"instanceReference"`
		} `json:"artifacts"`
		Manual    []string                     `json:"manualEnvironments"`
		Variables map[string]map[string]string `json:"variables"`
	}
	if err := json.Unmarshal([]byte(seen[0].body), &body); err != nil {
		t.Fatal(err)
	}
	if body.DefinitionID != 12 || body.Artifacts[0].Instance["id"] != "812" || body.Artifacts[0].Instance["sourceBranch"] != "refs/heads/master" {
		t.Errorf("body %s", seen[0].body)
	}
	if len(body.Manual) != 1 || body.Variables["Region"]["value"] != "northeurope" {
		t.Errorf("body %s", seen[0].body)
	}
}

func TestReleaseLogSkipsLinesAlreadyRead(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/logs/5": "2026-11-17T13:04:00.1Z one\n2026-11-17T13:04:01.1Z two\n2026-11-17T13:04:02.1Z three\n"}, &seen)
	lines, _, err := c.GetReleaseLog("https://vsrm.dev.azure.com/acme/web/_apis/Release/releases/64/logs/5", 3)
	if err != nil || len(lines) != 1 || lines[0] != "three" {
		t.Fatalf("got %q, %v", lines, err)
	}
	if !strings.Contains(seen[0].url, "api-version=7.1-preview") {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestFavoriteReleaseDefinitionsKeepOnlyReleasePipelines(t *testing.T) {
	var seen []recorded
	body := `{"value":[{"artifactId":"p1/12","artifactType":"Microsoft.TeamFoundation.ReleaseManagement.ReleaseDefinition"},` +
		`{"artifactId":"r1","artifactType":"Microsoft.TeamFoundation.Git.Repository"},{"artifactId":"p1/13","artifactType":"ReleaseDefinition","artifactIsDeleted":true}]}`
	ids, err := clientRouting(t, map[string]string{"/favorite/favorites": body}, &seen).GetFavoriteReleaseDefinitionIDs()
	if err != nil || len(ids) != 1 || ids[0] != 12 {
		t.Fatalf("got %v, %v", ids, err)
	}
}

func TestCancelReleaseStageSendsCanceledStatus(t *testing.T) {
	var seen []recorded
	if err := clientRouting(t, map[string]string{}, &seen).CancelReleaseStage(64, 2, "Wrong build"); err != nil {
		t.Fatal(err)
	}
	if seen[0].method != "PATCH" || !strings.Contains(seen[0].url, "/release/releases/64/environments/2?api-version=7.1-preview.7") ||
		!strings.Contains(seen[0].body, `"status":"canceled"`) || !strings.Contains(seen[0].body, `"comment":"Wrong build"`) {
		t.Fatalf("cancel %s %s %s", seen[0].method, seen[0].url, seen[0].body)
	}
}
