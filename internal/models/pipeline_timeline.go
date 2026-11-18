package models

import (
	"slices"
	"strings"
	"time"
)

const (
	recordStage = "stage"
	recordPhase = "phase"
	recordJob   = "job"
	recordTask  = "task"
	whereJoin   = " › "
)

type TimelineRecord struct {
	ID         string
	ParentID   string
	Type       string
	Name       string
	Identifier string
	Order      int
	State      RunState
	Started    time.Time
	Finished   time.Time
	LogID      int
	LogURL     string
	Worker     string
	Issues     []RunIssue
}

func (r TimelineRecord) Duration(now time.Time) time.Duration {
	if r.Started.IsZero() {
		return 0
	}
	end := r.Finished
	if end.IsZero() {
		end = now
	}
	return max(end.Sub(r.Started), 0)
}

func (r TimelineRecord) kind() string { return strings.ToLower(r.Type) }

type RunIssue struct {
	Error    bool
	Message  string
	Where    string
	RecordID string
}

type RunJob struct {
	TimelineRecord
	Stage string
	Steps []TimelineRecord
}

type RunStage struct {
	TimelineRecord
	Jobs []RunJob
}

func (s RunStage) JobCounts() (done, total int) {
	for _, j := range s.Jobs {
		if !j.State.Active() {
			done++
		}
	}
	return done, len(s.Jobs)
}

func BuildStages(records []TimelineRecord) []RunStage {
	byID := make(map[string]TimelineRecord, len(records))
	for _, r := range records {
		byID[r.ID] = r
	}
	stageOf := func(r TimelineRecord) (TimelineRecord, bool) {
		for p, ok := byID[r.ParentID]; ok; p, ok = byID[p.ParentID] {
			if p.kind() == recordStage {
				return p, true
			}
		}
		return TimelineRecord{}, false
	}
	var stages []RunStage
	index := map[string]int{}
	add := func(r TimelineRecord) int {
		if i, ok := index[r.ID]; ok {
			return i
		}
		index[r.ID] = len(stages)
		stages = append(stages, RunStage{TimelineRecord: r})
		return len(stages) - 1
	}
	for _, r := range sortedRecords(records) {
		if r.kind() == recordStage {
			add(r)
		}
	}
	for _, r := range sortedRecords(records) {
		if r.kind() != recordJob {
			continue
		}
		stage, ok := stageOf(r)
		if !ok {
			stage = TimelineRecord{ID: "", Name: "Jobs", State: r.State}
		}
		i := add(stage)
		stages[i].Jobs = append(stages[i].Jobs, RunJob{TimelineRecord: r, Stage: stage.Name, Steps: stepsOf(records, r.ID)})
	}
	for i := range stages {
		if stages[i].ID == "" {
			stages[i].State = jobsState(stages[i].Jobs)
		}
	}
	return stages
}

func jobsState(jobs []RunJob) RunState {
	worst := RunSucceeded
	for _, j := range jobs {
		switch {
		case j.State == RunRunning:
			return RunRunning
		case j.State == RunFailed:
			worst = RunFailed
		case j.State == RunQueued && worst != RunFailed:
			worst = RunQueued
		}
	}
	return worst
}

func sortedRecords(records []TimelineRecord) []TimelineRecord {
	out := slices.Clone(records)
	slices.SortStableFunc(out, func(a, b TimelineRecord) int { return a.Order - b.Order })
	return out
}

func stepsOf(records []TimelineRecord, jobID string) []TimelineRecord {
	var steps []TimelineRecord
	for _, r := range sortedRecords(records) {
		if r.ParentID == jobID && r.kind() == recordTask {
			steps = append(steps, r)
		}
	}
	return steps
}

func RunIssues(stages []RunStage) []RunIssue {
	var out []RunIssue
	collect := func(r TimelineRecord, where ...string) {
		label := strings.Join(slices.DeleteFunc(where, func(s string) bool { return s == "" }), whereJoin)
		for _, is := range r.Issues {
			is.Where, is.RecordID = label, r.ID
			out = append(out, is)
		}
	}
	for _, s := range stages {
		collect(s.TimelineRecord, s.Name)
		for _, j := range s.Jobs {
			collect(j.TimelineRecord, s.Name, j.Name)
			for _, st := range j.Steps {
				collect(st, s.Name, j.Name, st.Name)
			}
		}
	}
	slices.SortStableFunc(out, func(a, b RunIssue) int {
		switch {
		case a.Error == b.Error:
			return 0
		case a.Error:
			return -1
		}
		return 1
	})
	return out
}
