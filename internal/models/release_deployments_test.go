package models

import (
	"testing"
	"time"
)

func TestDeploymentStateReadsStatusAndOperation(t *testing.T) {
	cases := []struct {
		status, op string
		want       StageState
	}{
		{"succeeded", "Approved", StageSucceeded},
		{"partiallySucceeded", "", StagePartial},
		{"failed", "PhaseFailed", StageFailed},
		{"failed", "Rejected", StageRejected},
		{"failed", "Canceled", StageCanceled},
		{"inProgress", "Pending", StageWaitingApproval},
		{"inProgress", "EvaluatingGates", StageWaitingApproval},
		{"inProgress", "QueuedForAgent", StageQueued},
		{"inProgress", "PhaseInProgress", StageDeploying},
		{"notDeployed", "Canceled", StageCanceled},
		{"undefined", "", StageNotDeployed},
	}
	for _, c := range cases {
		if got := DeploymentState(c.status, c.op); got != c.want {
			t.Errorf("%s/%s: got %v, want %v", c.status, c.op, got, c.want)
		}
	}
}

func TestCurrentDeploymentIsTheLatestSuccessPerStage(t *testing.T) {
	now := time.Now()
	releases := []Release{
		{ID: 3, Created: now, Stages: []ReleaseStage{{Name: "Dev", State: StageSucceeded, Finished: now}, {Name: "Prod", State: StageFailed, Finished: now}}},
		{ID: 2, Created: now.Add(-time.Hour), Stages: []ReleaseStage{{Name: "Dev", State: StageSucceeded, Finished: now.Add(-time.Hour)}, {Name: "Prod", State: StagePartial, Finished: now.Add(-time.Minute)}}},
		{ID: 1, Created: now.Add(-2 * time.Hour), Stages: []ReleaseStage{{Name: "Dev", State: StageSucceeded}, {Name: "Prod", State: StageSucceeded, Finished: now.Add(-2 * time.Hour)}}},
	}
	got := CurrentDeployments(releases)
	if !got[3]["Dev"] || !got[2]["Prod"] || got[1]["Prod"] || got[3]["Prod"] || len(got[1]) != 0 {
		t.Fatalf("current %v", got)
	}
}
