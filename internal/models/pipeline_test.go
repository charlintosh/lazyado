package models

import (
	"strings"
	"testing"
	"time"
)

const pipelineYAML = `
trigger: [main]
parameters:
  - name: environment
    displayName: Environment
    type: string
    default: staging
    values: [dev, staging, prod]
  - name: runE2E
    displayName: Run E2E tests
    type: boolean
    default: true
  - name: shards
    type: number
    default: 4
  - name: matrix
    type: object
    default:
      linux: ubuntu-latest
  - name: steps
    type: stepList
stages:
  - stage: Build
  - stage: DeployProd
    displayName: Deploy prod
  - template: stages/cleanup.yml
`

func TestParsePipelineSpecReadsScalarParametersAndStages(t *testing.T) {
	spec, err := ParsePipelineSpec(pipelineYAML)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Parameters) != 3 {
		t.Fatalf("got %+v", spec.Parameters)
	}
	env, e2e, shards := spec.Parameters[0], spec.Parameters[1], spec.Parameters[2]
	if env.Label() != "Environment" || env.Default != "staging" || strings.Join(env.Values, ",") != "dev,staging,prod" {
		t.Errorf("environment %+v", env)
	}
	if e2e.Type != ParamBoolean || e2e.Default != "true" || shards.Type != ParamNumber || shards.Default != "4" {
		t.Errorf("got %+v %+v", e2e, shards)
	}
	if strings.Join(spec.Fixed, ",") != "matrix,steps" {
		t.Errorf("fixed %v", spec.Fixed)
	}
	if len(spec.Stages) != 2 || spec.Stages[1].Name != "DeployProd" || spec.Stages[1].Label() != "Deploy prod" {
		t.Errorf("stages %+v", spec.Stages)
	}
}

func TestParsePipelineSpecAcceptsTheShortParameterForm(t *testing.T) {
	spec, err := ParsePipelineSpec("parameters:\n  region: westeurope\nsteps:\n- script: echo hi\n")
	if err != nil || len(spec.Parameters) != 1 || spec.Parameters[0].Default != "westeurope" || len(spec.Stages) != 0 {
		t.Fatalf("got %+v, %v", spec, err)
	}
}

func TestParsePipelineSpecReportsBrokenYAML(t *testing.T) {
	if _, err := ParsePipelineSpec("parameters: [\n"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestRunQueryTurnsQualifiersIntoServerCriteria(t *testing.T) {
	pipelines := []Pipeline{{ID: 7, Name: "storefront-web-ci"}, {ID: 9, Name: "storefront-api-ci"}, {ID: 11, Name: "docs"}}
	q := ParseRunQuery(`state:failed branch:main pipeline:docs by:@me guest`)
	c, err := q.Criteria(pipelines, Identity{ID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	want := RunCriteria{PipelineID: 11, Ref: "refs/heads/main", Result: "failed", RequestedFor: "u1"}
	if c != want {
		t.Fatalf("got %+v", c)
	}
	if len(q.Terms) != 1 || q.Terms[0] != "guest" {
		t.Fatalf("terms %v", q.Terms)
	}
	if _, err := ParseRunQuery("pipeline:storefront").Criteria(pipelines, Identity{}); err == nil {
		t.Error("an ambiguous pipeline name should be an error")
	}
	if _, err := ParseRunQuery("state:green").Criteria(pipelines, Identity{}); err == nil {
		t.Error("an unknown state should be an error")
	}
	if c, _ := ParseRunQuery("state:running").Criteria(nil, Identity{}); c.Status != "inProgress" {
		t.Errorf("running maps to %+v", c)
	}
}

func TestRunLabelsAndDurations(t *testing.T) {
	start := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	r := PipelineRun{Status: "completed", Result: "partiallySucceeded", Ref: "refs/pull/4831/merge", Reason: "pullRequest", Started: start, Finished: start.Add(73 * time.Second)}
	if r.State() != RunPartial || r.State().Label() != "PART" || r.Branch() != "PR !4831" || r.Title() != "PR run" {
		t.Fatalf("got %v %q %q", r.State(), r.Branch(), r.Title())
	}
	if got := FormatDuration(r.Duration(time.Now())); got != "1m 13s" {
		t.Fatalf("duration %q", got)
	}
	running := PipelineRun{Status: "inProgress", Started: start}
	if got := FormatDuration(running.Duration(start.Add(42 * time.Second))); got != "42s" {
		t.Fatalf("running duration %q", got)
	}
	if FormatDuration(2*time.Hour+5*time.Minute) != "2h 05m" || RefLabel("refs/tags/v1.2") != "v1.2" || BranchRef("main") != "refs/heads/main" {
		t.Fatal("labels")
	}
}

func TestStatsCountOnlyFinishedRuns(t *testing.T) {
	start := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	done := func(result string, secs int) PipelineRun {
		return PipelineRun{Status: "completed", Result: result, Started: start, Finished: start.Add(time.Duration(secs) * time.Second)}
	}
	s := StatsOf([]PipelineRun{done("succeeded", 60), done("failed", 30), done("succeeded", 90), {Status: "inProgress", Started: start}})
	if s.Completed != 3 || s.Passed != 2 || s.Median != 60*time.Second {
		t.Fatalf("got %+v", s)
	}
}
