package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
)

const buildJSON = `{"id":812,"buildNumber":"20261117.6","status":"inProgress","result":"none","reason":"individualCI",` +
	`"sourceBranch":"refs/heads/feature/1043-guest","sourceVersion":"3f9c2ab8e1","queueTime":"2026-11-17T10:41:00.12Z",` +
	`"startTime":"2026-11-17T10:41:05Z","triggerInfo":{"ci.message":"Add guest checkout flag\n\nDetails"},` +
	`"requestedFor":{"id":"u1","displayName":"Dana Ruiz"},"definition":{"id":7,"name":"storefront-web-ci","path":"\\CI"},` +
	`"repository":{"id":"r1","name":"web"},"queue":{"name":"Azure Pipelines"},"_links":{"web":{"href":"https://dev.azure.com/acme/web/_build/results?buildId=812"}}}`

func TestListPipelineRunsSendsTheFiltersAndMapsRuns(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/build/builds": `{"value":[` + buildJSON + `]}`}, &seen)
	runs, _, err := c.ListPipelineRuns(models.RunCriteria{PipelineID: 7, Ref: "refs/heads/main", Result: "failed", RequestedFor: "u1"}, 50, "tok")
	if err != nil || len(runs) != 1 {
		t.Fatalf("got %v, %v", runs, err)
	}
	r := runs[0]
	if r.Number != "20261117.6" || r.PipelineName != "storefront-web-ci" || r.Title() != "Add guest checkout flag" || r.State() != models.RunRunning {
		t.Fatalf("mapped %+v", r)
	}
	if r.Branch() != "feature/1043-guest" || r.ShortCommit() != "3f9c2ab" || r.ReasonLabel() != "Individual CI" || r.WebURL == "" {
		t.Fatalf("mapped %+v", r)
	}
	for _, want := range []string{"definitions=7", "branchName=refs%2Fheads%2Fmain", "resultFilter=failed", "requestedFor=u1", "continuationToken=tok", "queryOrder=queueTimeDescending", "%24top=50"} {
		if !strings.Contains(seen[0].url, want) {
			t.Errorf("asked %s, missing %s", seen[0].url, want)
		}
	}
}

func TestRecentPipelinesKeepsTheLatestRunOfEachPipeline(t *testing.T) {
	var seen []recorded
	other := strings.Replace(strings.Replace(buildJSON, `"id":812`, `"id":800`, 1), `"buildNumber":"20261117.6"`, `"buildNumber":"20261117.1"`, 1)
	third := strings.Replace(strings.Replace(buildJSON, `"id":812`, `"id":790`, 1), `"definition":{"id":7`, `"definition":{"id":9`, 1)
	c := clientRouting(t, map[string]string{"/build/builds": `{"value":[` + buildJSON + `,` + other + `,` + third + `]}`}, &seen)
	pipelines, err := c.RecentPipelines(10)
	if err != nil || len(pipelines) != 2 {
		t.Fatalf("got %v, %v", pipelines, err)
	}
	if pipelines[0].ID != 7 || pipelines[0].Latest.ID != 812 || pipelines[1].ID != 9 {
		t.Fatalf("got %+v", pipelines)
	}
}

func TestListPipelinesSearchesByNameOnTheServer(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/build/definitions": `{"value":[{"id":7,"name":"payments-api-ci","path":"\\CI","latestBuild":` + buildJSON + `}]}`}, &seen)
	pipelines, _, err := c.ListPipelines("*pay*", "")
	if err != nil || len(pipelines) != 1 || pipelines[0].Latest == nil || pipelines[0].Latest.PipelineName != "payments-api-ci" {
		t.Fatalf("got %+v, %v", pipelines, err)
	}
	if !strings.Contains(seen[0].url, "name=%2Apay%2A") || !strings.Contains(seen[0].url, "includeLatestBuilds=true") {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestRunTimelineGroupsStagesJobsAndSteps(t *testing.T) {
	var seen []recorded
	body := `{"records":[` +
		`{"id":"s1","type":"Stage","name":"Build","order":1,"state":"completed","result":"succeeded"},` +
		`{"id":"s2","type":"Stage","name":"Test","order":2,"state":"inProgress"},` +
		`{"id":"p1","parentId":"s2","type":"Phase","name":"Test","order":1},` +
		`{"id":"j1","parentId":"p1","type":"Job","name":"Integration tests","order":1,"state":"inProgress","log":{"id":9}},` +
		`{"id":"t2","parentId":"j1","type":"Task","name":"Run suite","order":2,"state":"inProgress","log":{"id":12}},` +
		`{"id":"t1","parentId":"j1","type":"Task","name":"Checkout","order":1,"state":"completed","result":"succeeded",` +
		`"issues":[{"type":"warning","message":"shallow clone"}]}]}`
	c := clientRouting(t, map[string]string{"/timeline": body}, &seen)
	records, err := c.GetRunTimeline(812)
	if err != nil {
		t.Fatal(err)
	}
	stages := models.BuildStages(records)
	if len(stages) != 2 || stages[0].Name != "Build" || len(stages[1].Jobs) != 1 {
		t.Fatalf("got %+v", stages)
	}
	job := stages[1].Jobs[0]
	if job.Stage != "Test" || len(job.Steps) != 2 || job.Steps[0].Name != "Checkout" || job.Steps[1].LogID != 12 {
		t.Fatalf("got %+v", job)
	}
	issues := models.RunIssues(stages)
	if len(issues) != 1 || issues[0].Error || issues[0].Where != "Test › Integration tests › Checkout" {
		t.Fatalf("got %+v", issues)
	}
}

func TestRunLogDropsTimestampsAndAsksFromALine(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/logs/12": "2026-11-17T10:42:31.1234567Z Starting: Run suite\n2026-11-17T10:42:32.0000000Z \x1b[32mPASS\x1b[0m cart\n"}, &seen)
	lines, truncated, err := c.GetRunLog(812, 12, 40)
	if err != nil || truncated {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "Starting: Run suite" || lines[1] != "PASS cart" {
		t.Fatalf("got %q", lines)
	}
	if !strings.Contains(seen[0].url, "/build/builds/812/logs/12?startLine=40") {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestPipelineDefinitionKeepsOnlyQueueTimeVariables(t *testing.T) {
	var seen []recorded
	body := `{"id":7,"name":"web","path":"\\CI","repository":{"id":"r1","name":"web","type":"TfsGit","defaultBranch":"refs/heads/main"},` +
		`"process":{"yamlFilename":"azure-pipelines.yml"},"variables":{"LOG_LEVEL":{"value":"info","allowOverride":true},"TOKEN":{"value":"x"}}}`
	def, err := clientRouting(t, map[string]string{"/build/definitions/7": body}, &seen).GetPipelineDefinition(7)
	if err != nil || !def.YAML() || !def.AzureRepo() || def.DefaultBranch != "refs/heads/main" {
		t.Fatalf("got %+v, %v", def, err)
	}
	if len(def.Variables) != 1 || def.Variables[0].Name != "LOG_LEVEL" {
		t.Fatalf("got %+v", def.Variables)
	}
}

func TestFileAtRefUsesTheTagVersionTypeForTags(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/items": "parameters: []"}, &seen)
	if _, err := c.GetFileAtRef("r1", "/azure-pipelines.yml", "refs/tags/v2.4.0"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(seen[0].url, "versionDescriptor.versionType=tag") || !strings.Contains(seen[0].url, "versionDescriptor.version=v2.4.0") {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestRunPipelineSendsRefParametersVariablesAndSkippedStages(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/pipelines/7/runs": `{"id":813}`}, &seen)
	def := models.PipelineDefinition{Pipeline: models.Pipeline{ID: 7, Name: "web"}, YAMLPath: "azure-pipelines.yml"}
	id, err := c.RunPipeline(def, models.RunOptions{
		Ref: "refs/tags/v2.4.0", Commit: "8be41c0", Parameters: map[string]string{"environment": "staging", "runE2E": "true"},
		Variables: map[string]string{"LOG_LEVEL": "debug"}, StagesToSkip: []string{"DeployProd"}, Diagnostics: true,
	})
	if err != nil || id != 813 {
		t.Fatalf("got %d, %v", id, err)
	}
	if seen[0].method != "POST" || !strings.Contains(seen[0].url, "/pipelines/7/runs?api-version=7.1") {
		t.Fatalf("asked %s %s", seen[0].method, seen[0].url)
	}
	var body struct {
		Resources struct {
			Repositories struct {
				Self struct {
					RefName string `json:"refName"`
					Version string `json:"version"`
				} `json:"self"`
			} `json:"repositories"`
		} `json:"resources"`
		TemplateParameters map[string]string            `json:"templateParameters"`
		Variables          map[string]map[string]string `json:"variables"`
		StagesToSkip       []string                     `json:"stagesToSkip"`
	}
	if err := json.Unmarshal([]byte(seen[0].body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Resources.Repositories.Self.RefName != "refs/tags/v2.4.0" || body.Resources.Repositories.Self.Version != "8be41c0" {
		t.Errorf("resources %+v", body.Resources)
	}
	if body.TemplateParameters["environment"] != "staging" || body.Variables["LOG_LEVEL"]["value"] != "debug" || body.Variables["system.debug"]["value"] != "true" {
		t.Errorf("body %s", seen[0].body)
	}
	if len(body.StagesToSkip) != 1 || body.StagesToSkip[0] != "DeployProd" {
		t.Errorf("stages %v", body.StagesToSkip)
	}
}

func TestRunClassicPipelineQueuesABuild(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/build/builds": `{"id":900}`}, &seen)
	def := models.PipelineDefinition{Pipeline: models.Pipeline{ID: 3, Name: "legacy"}}
	id, err := c.RunPipeline(def, models.RunOptions{Ref: "refs/heads/main", Variables: map[string]string{"A": "1"}})
	if err != nil || id != 900 {
		t.Fatalf("got %d, %v", id, err)
	}
	if !strings.Contains(seen[0].body, `"sourceBranch":"refs/heads/main"`) || !strings.Contains(seen[0].body, `"parameters":"{\"A\":\"1\"}"`) {
		t.Fatalf("body %s", seen[0].body)
	}
}
