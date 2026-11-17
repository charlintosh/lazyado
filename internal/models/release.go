package models

import (
	"slices"
	"strings"
	"time"
)

const (
	ApprovalPre  = "preDeploy"
	ApprovalPost = "postDeploy"

	ArtifactBuild = "Build"
	ArtifactGit   = "Git"
)

type StageState int

const (
	StageNotDeployed StageState = iota
	StageQueued
	StageScheduled
	StageWaitingApproval
	StageDeploying
	StageSucceeded
	StagePartial
	StageFailed
	StageRejected
	StageCanceled
)

var stageLabels = map[StageState]string{
	StageNotDeployed:     "",
	StageQueued:          "WAIT",
	StageScheduled:       "WAIT",
	StageWaitingApproval: "APPR",
	StageDeploying:       "RUN",
	StageSucceeded:       "OK",
	StagePartial:         "PART",
	StageFailed:          "FAIL",
	StageRejected:        "REJ",
	StageCanceled:        "STOP",
}

var stageNames = map[StageState]string{
	StageNotDeployed:     "not deployed",
	StageQueued:          "queued",
	StageScheduled:       "scheduled",
	StageWaitingApproval: "waiting for approval",
	StageDeploying:       "deploying",
	StageSucceeded:       "deployed",
	StagePartial:         "partially deployed",
	StageFailed:          "failed",
	StageRejected:        "rejected",
	StageCanceled:        "canceled",
}

func (s StageState) Label() string { return stageLabels[s] }

func (s StageState) Name() string { return stageNames[s] }

func (s StageState) Active() bool {
	return s == StageQueued || s == StageDeploying
}

func (s StageState) Cancelable() bool {
	switch s {
	case StageDeploying, StageQueued, StageScheduled, StageWaitingApproval:
		return true
	}
	return false
}

func (s StageState) Deployable() bool {
	switch s {
	case StageNotDeployed, StageSucceeded, StagePartial, StageFailed, StageRejected, StageCanceled:
		return true
	}
	return false
}

type ReleaseDefinition struct {
	ID          int
	Name        string
	Folder      string
	WebURL      string
	LastRelease string
	LastCreated time.Time
}

type ReleaseApproval struct {
	ID          int
	Kind        string
	Status      string
	Approver    Identity
	Group       bool
	Order       int
	Automated   bool
	Created     time.Time
	ReleaseID   int
	ReleaseName string
	Definition  string
	StageID     int
	StageName   string
	WebURL      string
}

func (a ReleaseApproval) Pending() bool { return strings.EqualFold(a.Status, "pending") }

func (a ReleaseApproval) KindLabel() string {
	if a.Kind == ApprovalPost {
		return "POST"
	}
	return "PRE"
}

func (a ReleaseApproval) KindName() string {
	if a.Kind == ApprovalPost {
		return "Post-deployment"
	}
	return "Pre-deployment"
}

type ReleaseArtifact struct {
	Alias        string
	Type         string
	Primary      bool
	Version      string
	VersionID    string
	Branch       string
	DefinitionID int
	Definition   string
	RepositoryID string
}

func (a ReleaseArtifact) BranchName() string { return RefLabel(a.Branch) }

type ReleaseStage struct {
	ID           int
	Name         string
	Rank         int
	Status       string
	State        StageState
	Approvals    []ReleaseApproval
	Approvers    []string
	PostApproval []string
	Trigger      string
	Attempt      int
	Requested    Identity
	Started      time.Time
	Finished     time.Time
	Jobs         []RunJob
}

func (s ReleaseStage) PendingApproval() (ReleaseApproval, bool) {
	for _, a := range s.Approvals {
		if a.Pending() && !a.Automated {
			return a, true
		}
	}
	return ReleaseApproval{}, false
}

func (s ReleaseStage) RunStage() RunStage {
	return RunStage{TimelineRecord: TimelineRecord{ID: s.Name, Name: s.Name, State: s.runState(), Started: s.Started, Finished: s.Finished}, Jobs: s.Jobs}
}

func (s ReleaseStage) runState() RunState {
	switch s.State {
	case StageQueued, StageScheduled, StageWaitingApproval:
		return RunQueued
	case StageDeploying:
		return RunRunning
	case StageSucceeded:
		return RunSucceeded
	case StagePartial:
		return RunPartial
	case StageFailed, StageRejected:
		return RunFailed
	case StageCanceled:
		return RunCanceled
	}
	return RunUnknown
}

func StageStateOf(status string, approvals []ReleaseApproval) StageState {
	for _, a := range approvals {
		if strings.EqualFold(a.Status, "rejected") {
			return StageRejected
		}
	}
	for _, a := range approvals {
		if a.Pending() && !a.Automated {
			return StageWaitingApproval
		}
	}
	switch strings.ToLower(status) {
	case "inprogress":
		return StageDeploying
	case "queued":
		return StageQueued
	case "scheduled":
		return StageScheduled
	case "succeeded":
		return StageSucceeded
	case "partiallysucceeded":
		return StagePartial
	case "rejected":
		return StageFailed
	case "canceled":
		return StageCanceled
	}
	return StageNotDeployed
}

type Release struct {
	ID           int
	Name         string
	Description  string
	Status       string
	DefinitionID int
	Definition   string
	CreatedBy    Identity
	Created      time.Time
	Stages       []ReleaseStage
	Artifacts    []ReleaseArtifact
	WebURL       string
}

func (r Release) Primary() (ReleaseArtifact, bool) {
	for _, a := range r.Artifacts {
		if a.Primary {
			return a, true
		}
	}
	if len(r.Artifacts) > 0 {
		return r.Artifacts[0], true
	}
	return ReleaseArtifact{}, false
}

func (r Release) ArtifactSummary() string {
	a, ok := r.Primary()
	if !ok {
		return "no artifacts"
	}
	parts := []string{a.Alias, a.Version}
	if b := a.BranchName(); b != "" {
		parts = append(parts, b)
	}
	return strings.Join(slices.DeleteFunc(parts, func(s string) bool { return s == "" }), " · ")
}

func (r Release) State() StageState {
	worst := StageNotDeployed
	for _, s := range r.Stages {
		switch s.State {
		case StageDeploying, StageQueued, StageWaitingApproval, StageFailed, StageRejected:
			return s.State
		case StageSucceeded, StagePartial:
			worst = s.State
		}
	}
	return worst
}

func (r Release) Active() bool {
	for _, s := range r.Stages {
		if s.State.Active() {
			return true
		}
	}
	return false
}

func (r Release) RunStages() []RunStage {
	out := make([]RunStage, 0, len(r.Stages))
	for _, s := range r.Stages {
		out = append(out, s.RunStage())
	}
	return out
}

func (r Release) Stage(id int) (ReleaseStage, bool) {
	for _, s := range r.Stages {
		if s.ID == id {
			return s, true
		}
	}
	return ReleaseStage{}, false
}

type ReleaseVariable struct {
	Name   string
	Value  string
	Secret bool
}

type ReleaseStageSpec struct {
	Name      string
	Automated bool
	Trigger   string
}

type ReleaseDefinitionSpec struct {
	ReleaseDefinition
	Artifacts []ReleaseArtifact
	Stages    []ReleaseStageSpec
	Variables []ReleaseVariable
}

type ArtifactChoice struct {
	Alias     string
	VersionID string
	Version   string
	Branch    string
}

type NewRelease struct {
	DefinitionID int
	Description  string
	Artifacts    []ArtifactChoice
	ManualStages []string
	Variables    map[string]string
}

func SortDefinitions(defs []ReleaseDefinition, starred func(int) bool) []ReleaseDefinition {
	out := slices.Clone(defs)
	slices.SortStableFunc(out, func(a, b ReleaseDefinition) int {
		sa, sb := starred(a.ID), starred(b.ID)
		switch {
		case sa && !sb:
			return -1
		case sb && !sa:
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out
}

func (a ReleaseApproval) AssignedTo(me Identity) bool {
	return me.Is(a.Approver.ID, a.Approver.UniqueName)
}

func ApprovalMatches(a ReleaseApproval, terms []string) bool {
	haystack := strings.ToLower(strings.Join([]string{a.StageName, a.ReleaseName, a.Definition, a.Approver.DisplayName}, "\n"))
	for _, t := range terms {
		if !strings.Contains(haystack, t) {
			return false
		}
	}
	return true
}
