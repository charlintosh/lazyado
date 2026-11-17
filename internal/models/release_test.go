package models

import "testing"

func TestStageStateOfPrefersRejectionsThenPendingApprovals(t *testing.T) {
	pending := ReleaseApproval{Status: "pending"}
	automated := ReleaseApproval{Status: "pending", Automated: true}
	cases := []struct {
		status    string
		approvals []ReleaseApproval
		want      StageState
	}{
		{"notStarted", []ReleaseApproval{pending}, StageWaitingApproval},
		{"notStarted", []ReleaseApproval{automated}, StageNotDeployed},
		{"rejected", []ReleaseApproval{{Status: "rejected"}}, StageRejected},
		{"rejected", nil, StageFailed},
		{"inProgress", nil, StageDeploying},
		{"succeeded", nil, StageSucceeded},
		{"partiallySucceeded", nil, StagePartial},
		{"canceled", nil, StageCanceled},
		{"queued", nil, StageQueued},
	}
	for _, c := range cases {
		if got := StageStateOf(c.status, c.approvals); got != c.want {
			t.Errorf("%s %v: got %v, want %v", c.status, c.approvals, got, c.want)
		}
	}
	if StageWaitingApproval.Deployable() || StageDeploying.Deployable() || !StageFailed.Deployable() || !StageNotDeployed.Deployable() {
		t.Error("only idle stages can be deployed")
	}
}

func TestSortDefinitionsPutsStarredFirstThenByName(t *testing.T) {
	defs := []ReleaseDefinition{{ID: 1, Name: "beta"}, {ID: 2, Name: "Alpha"}, {ID: 3, Name: "zeta"}}
	got := SortDefinitions(defs, func(id int) bool { return id == 3 })
	if got[0].ID != 3 || got[1].ID != 2 || got[2].ID != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestReleaseStateShowsWhatNeedsAttention(t *testing.T) {
	r := Release{Stages: []ReleaseStage{{State: StageSucceeded}, {State: StageWaitingApproval}, {State: StageNotDeployed}}}
	if r.State() != StageWaitingApproval || r.Active() {
		t.Fatalf("got %v active %v", r.State(), r.Active())
	}
	r.Stages[1].State = StageDeploying
	if r.State() != StageDeploying || !r.Active() {
		t.Fatalf("got %v active %v", r.State(), r.Active())
	}
}

func TestOnlyStartedDeploymentsCanBeCanceled(t *testing.T) {
	for _, s := range []StageState{StageDeploying, StageQueued, StageScheduled, StageWaitingApproval} {
		if !s.Cancelable() {
			t.Errorf("%v should be cancelable", s)
		}
	}
	for _, s := range []StageState{StageNotDeployed, StageSucceeded, StageFailed, StageRejected, StageCanceled, StagePartial} {
		if s.Cancelable() {
			t.Errorf("%v should not be cancelable", s)
		}
	}
}
