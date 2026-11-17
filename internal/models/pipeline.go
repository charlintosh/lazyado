package models

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	refHeads = "refs/heads/"
	refTags  = "refs/tags/"
	refPull  = "refs/pull/"

	RootFolder = `\`
)

type RunState int

const (
	RunUnknown RunState = iota
	RunQueued
	RunRunning
	RunSucceeded
	RunPartial
	RunFailed
	RunCanceled
	RunSkipped
)

var runStateLabels = map[RunState]string{
	RunUnknown:   "NONE",
	RunQueued:    "WAIT",
	RunRunning:   "RUN",
	RunSucceeded: "OK",
	RunPartial:   "PART",
	RunFailed:    "FAIL",
	RunCanceled:  "STOP",
	RunSkipped:   "SKIP",
}

var runStateNames = map[RunState]string{
	RunUnknown:   "not run",
	RunQueued:    "queued",
	RunRunning:   "running",
	RunSucceeded: "succeeded",
	RunPartial:   "partially succeeded",
	RunFailed:    "failed",
	RunCanceled:  "canceled",
	RunSkipped:   "skipped",
}

func (s RunState) Label() string { return runStateLabels[s] }

func (s RunState) Name() string { return runStateNames[s] }

func (s RunState) Active() bool { return s == RunQueued || s == RunRunning }

func RunStateOf(status, result string) RunState {
	switch strings.ToLower(result) {
	case "succeeded":
		return RunSucceeded
	case "partiallysucceeded", "succeededwithissues":
		return RunPartial
	case "failed":
		return RunFailed
	case "canceled", "abandoned":
		return RunCanceled
	case "skipped":
		return RunSkipped
	}
	switch strings.ToLower(status) {
	case "notstarted", "pending", "postponed":
		return RunQueued
	case "inprogress", "cancelling":
		return RunRunning
	}
	return RunUnknown
}

type Pipeline struct {
	ID     int
	Name   string
	Folder string
	WebURL string
	Latest *PipelineRun
}

func (p Pipeline) FolderLabel() string {
	if p.Folder == "" {
		return RootFolder
	}
	return p.Folder
}

type PipelineVariable struct {
	Name   string
	Value  string
	Secret bool
}

type PipelineDefinition struct {
	Pipeline
	RepositoryID   string
	RepositoryName string
	RepositoryType string
	DefaultBranch  string
	YAMLPath       string
	Variables      []PipelineVariable
}

func (d PipelineDefinition) YAML() bool { return d.YAMLPath != "" }

func (d PipelineDefinition) AzureRepo() bool { return strings.EqualFold(d.RepositoryType, "TfsGit") }

type PipelineRun struct {
	ID             int
	Number         string
	PipelineID     int
	PipelineName   string
	PipelineFolder string
	Status         string
	Result         string
	Reason         string
	Ref            string
	Commit         string
	Message        string
	RequestedFor   Identity
	RepositoryID   string
	RepositoryName string
	Pool           string
	Queued         time.Time
	Started        time.Time
	Finished       time.Time
	WebURL         string
}

func (r PipelineRun) State() RunState { return RunStateOf(r.Status, r.Result) }

func (r PipelineRun) Branch() string { return RefLabel(r.Ref) }

func (r PipelineRun) ShortCommit() string {
	if len(r.Commit) > 7 {
		return r.Commit[:7]
	}
	return r.Commit
}

var reasonLabels = map[string]string{
	"individualci":    "Individual CI",
	"batchedci":       "Batched CI",
	"manual":          "Manual",
	"pullrequest":     "PR",
	"schedule":        "Scheduled",
	"resourcetrigger": "Resource trigger",
	"buildcompletion": "Build completion",
	"triggered":       "Triggered",
}

func (r PipelineRun) ReasonLabel() string {
	if label, ok := reasonLabels[strings.ToLower(r.Reason)]; ok {
		return label
	}
	return r.Reason
}

func (r PipelineRun) Title() string {
	if r.Message != "" {
		return firstLine(r.Message)
	}
	if r.ReasonLabel() == "Scheduled" {
		return "Scheduled run"
	}
	return r.ReasonLabel() + " run"
}

func (r PipelineRun) Duration(now time.Time) time.Duration {
	start := r.Started
	if start.IsZero() {
		return 0
	}
	end := r.Finished
	if end.IsZero() {
		end = now
	}
	return max(end.Sub(start), 0)
}

func (r PipelineRun) When() time.Time {
	if !r.Queued.IsZero() {
		return r.Queued
	}
	return r.Started
}

func (r PipelineRun) Trigger() string {
	parts := []string{r.ReasonLabel()}
	if r.Reason != "" && strings.EqualFold(r.Reason, "manual") && r.RequestedFor.DisplayName != "" {
		parts = append(parts, r.RequestedFor.DisplayName)
	}
	if b := r.Branch(); b != "" {
		parts = append(parts, b)
	}
	return strings.Join(parts, " · ")
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(line)
}

func RefLabel(ref string) string {
	switch {
	case strings.HasPrefix(ref, refHeads):
		return strings.TrimPrefix(ref, refHeads)
	case strings.HasPrefix(ref, refTags):
		return strings.TrimPrefix(ref, refTags)
	case strings.HasPrefix(ref, refPull):
		id, _, _ := strings.Cut(strings.TrimPrefix(ref, refPull), "/")
		return "PR !" + id
	}
	return ref
}

func BranchRef(name string) string {
	if name == "" || strings.HasPrefix(name, "refs/") {
		return name
	}
	return refHeads + name
}

func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	switch {
	case d <= 0:
		return "<1s"
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + "s"
	case d < time.Hour:
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
}

func LatestPerPipeline(runs []PipelineRun, limit int) []Pipeline {
	seen := map[int]bool{}
	var out []Pipeline
	for i := range runs {
		r := runs[i]
		if seen[r.PipelineID] {
			continue
		}
		seen[r.PipelineID] = true
		out = append(out, Pipeline{ID: r.PipelineID, Name: r.PipelineName, Folder: r.PipelineFolder, Latest: &r})
		if len(out) == limit {
			break
		}
	}
	return out
}

func LatestPerBranch(runs []PipelineRun) []PipelineRun {
	seen := map[string]bool{}
	var out []PipelineRun
	for _, r := range runs {
		if seen[r.Ref] {
			continue
		}
		seen[r.Ref] = true
		out = append(out, r)
	}
	return out
}

type RunStats struct {
	Completed int
	Passed    int
	Median    time.Duration
}

func StatsOf(runs []PipelineRun) RunStats {
	var s RunStats
	var durations []time.Duration
	for _, r := range runs {
		st := r.State()
		if st.Active() || st == RunUnknown {
			continue
		}
		s.Completed++
		if st == RunSucceeded {
			s.Passed++
		}
		durations = append(durations, r.Duration(r.Finished))
	}
	if len(durations) > 0 {
		slices.Sort(durations)
		s.Median = durations[len(durations)/2]
	}
	return s
}

type RunChange struct {
	ID      string
	Message string
	Author  string
}

func (c RunChange) ShortID() string {
	if len(c.ID) > 7 {
		return c.ID[:7]
	}
	return c.ID
}

type GitRef struct {
	Name   string
	Commit string
}

func (r GitRef) Tag() bool { return strings.HasPrefix(r.Name, refTags) }

func (r GitRef) Label() string { return RefLabel(r.Name) }

func (r GitRef) ShortCommit() string {
	if len(r.Commit) > 7 {
		return r.Commit[:7]
	}
	return r.Commit
}

type RunOptions struct {
	Ref          string
	Commit       string
	Parameters   map[string]string
	Variables    map[string]string
	StagesToSkip []string
	Diagnostics  bool
}
