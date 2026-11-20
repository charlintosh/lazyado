package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

type ReleaseDocKind int

const (
	DocDefinition ReleaseDocKind = iota
	DocRelease
)

const ReleaseScope = "Release"

type ReleaseDoc struct {
	Kind         ReleaseDocKind
	ID           int
	Name         string
	DefinitionID int
	Definition   string
	Folder       string
	Revision     int
	NameFormat   string
	ModifiedBy   string
	Modified     time.Time
	WebURL       string
	Artifacts    []DocArtifact
	Triggers     []string
	Stages       []DocStage
	Variables    []DocVariable
	Groups       []DocGroup
	raw          map[string]any
}

type DocArtifact struct {
	Alias   string
	Type    string
	Source  string
	Primary bool
}

type DocStage struct {
	Index         int
	ID            int
	Name          string
	Rank          int
	Trigger       string
	Status        string
	Locked        bool
	PreApprovers  []string
	PostApprovers []string
	Gates         bool
	Retention     string
	Phases        []DocPhase
}

type DocPhase struct {
	Index   int
	Name    string
	Kind    string
	QueueID int
	Agent   string
	Tasks   []DocTask
}

type DocTask struct {
	Index           int
	TaskID          string
	Version         string
	Name            string
	DefinitionType  string
	Condition       string
	Enabled         bool
	ContinueOnError bool
	AlwaysRun       bool
	Timeout         int
	Inputs          map[string]string
}

type DocVariable struct {
	Name     string
	Value    string
	Secret   bool
	Settable bool
	Stage    int
	Scope    string
}

type DocGroup struct {
	ID    int
	Name  string
	Stage int
	Scope string
}

func (t DocTask) Major() int {
	major, _, _ := strings.Cut(t.Version, ".")
	n, err := strconv.Atoi(major)
	if err != nil {
		return 0
	}
	return n
}

func (t DocTask) TaskGroup() bool { return strings.EqualFold(t.DefinitionType, "metaTask") }

func (s DocStage) TaskCount() int {
	n := 0
	for _, p := range s.Phases {
		n += len(p.Tasks)
	}
	return n
}

func (s DocStage) Summary() string {
	return plural(len(s.Phases), "job") + ", " + plural(s.TaskCount(), "task")
}

func (s DocStage) Approval(pre bool) string {
	names := s.PostApprovers
	if pre {
		names = s.PreApprovers
	}
	if len(names) == 0 {
		return "automatic"
	}
	return strings.Join(names, ", ")
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func (d *ReleaseDoc) Title() string {
	if d.Kind == DocRelease {
		return d.Name
	}
	return d.Definition
}

func (d *ReleaseDoc) Editable() bool {
	for _, s := range d.Stages {
		if !s.Locked {
			return true
		}
	}
	return false
}

func (d *ReleaseDoc) StageVariables(stage int) []DocVariable {
	var out []DocVariable
	for _, v := range d.Variables {
		if v.Stage == stage {
			out = append(out, v)
		}
	}
	return out
}

func (d *ReleaseDoc) StageGroups(stage int) []DocGroup {
	var out []DocGroup
	for _, g := range d.Groups {
		if g.Stage == stage {
			out = append(out, g)
		}
	}
	return out
}

func (d *ReleaseDoc) GroupIDs() []int {
	var ids []int
	for _, g := range d.Groups {
		if g.ID > 0 && !slices.Contains(ids, g.ID) {
			ids = append(ids, g.ID)
		}
	}
	return ids
}

func (d *ReleaseDoc) QueueIDs() []int {
	var ids []int
	for _, s := range d.Stages {
		for _, p := range s.Phases {
			if p.QueueID > 0 && !slices.Contains(ids, p.QueueID) {
				ids = append(ids, p.QueueID)
			}
		}
	}
	return ids
}

func (d *ReleaseDoc) TaskRefs() []TaskRef {
	var refs []TaskRef
	for _, s := range d.Stages {
		for _, p := range s.Phases {
			for _, t := range p.Tasks {
				ref := TaskRef{ID: t.TaskID, Major: t.Major()}
				if t.TaskID != "" && !t.TaskGroup() && !slices.Contains(refs, ref) {
					refs = append(refs, ref)
				}
			}
		}
	}
	return refs
}

func ParseReleaseDoc(kind ReleaseDocKind, body []byte) (*ReleaseDoc, error) {
	root, err := decodeTree(body)
	if err != nil {
		return nil, err
	}
	d := &ReleaseDoc{Kind: kind, raw: root, ID: intAt(root, "id"), Name: strAt(root, "name")}
	d.ModifiedBy = strAt(objAt(root, "modifiedBy"), "displayName")
	d.Modified = timeAt(root, "modifiedOn")
	d.WebURL = strAt(objAt(objAt(root, "_links"), "web"), "href")
	if kind == DocRelease {
		def := objAt(root, "releaseDefinition")
		d.DefinitionID, d.Definition = intAt(def, "id"), strAt(def, "name")
		d.Revision = intAt(root, "releaseDefinitionRevision")
	} else {
		d.DefinitionID, d.Definition = d.ID, d.Name
		d.Folder, d.Revision, d.NameFormat = strAt(root, "path"), intAt(root, "revision"), strAt(root, "releaseNameFormat")
		d.Triggers = parseTriggers(listAt(root, "triggers"))
	}
	for _, a := range listAt(root, "artifacts") {
		art, _ := a.(map[string]any)
		ref := objAt(art, "definitionReference")
		d.Artifacts = append(d.Artifacts, DocArtifact{
			Alias: strAt(art, "alias"), Type: strAt(art, "type"), Primary: boolAt(art, "isPrimary"),
			Source: strAt(objAt(ref, "definition"), "name"),
		})
	}
	d.Variables = parseVariables(objAt(root, "variables"), -1, ReleaseScope)
	d.Groups = parseGroups(listAt(root, "variableGroups"), -1, ReleaseScope)
	for i, e := range listAt(root, "environments") {
		env, _ := e.(map[string]any)
		s := parseStage(kind, i, env)
		d.Stages = append(d.Stages, s)
		d.Variables = append(d.Variables, parseVariables(objAt(env, "variables"), i, s.Name)...)
		d.Groups = append(d.Groups, parseGroups(listAt(env, "variableGroups"), i, s.Name)...)
	}
	slices.SortStableFunc(d.Stages, func(a, b DocStage) int { return a.Rank - b.Rank })
	return d, nil
}

func (d *ReleaseDoc) phasesKey() string {
	if d.Kind == DocRelease {
		return "deployPhasesSnapshot"
	}
	return "deployPhases"
}

func parseStage(kind ReleaseDocKind, index int, env map[string]any) DocStage {
	s := DocStage{Index: index, ID: intAt(env, "id"), Name: strAt(env, "name"), Rank: intAt(env, "rank"), Status: strAt(env, "status")}
	s.Trigger = stageTrigger(listAt(env, "conditions"))
	phases := "deployPhases"
	if kind == DocRelease {
		phases = "deployPhasesSnapshot"
		s.PreApprovers = approverNames(objAt(env, "preApprovalsSnapshot"))
		s.PostApprovers = approverNames(objAt(env, "postApprovalsSnapshot"))
		s.Gates = len(listAt(objAt(env, "preDeploymentGatesSnapshot"), "gates")) > 0
		s.Locked = !stageEditable(s.Status)
	} else {
		s.PreApprovers = approverNames(objAt(env, "preDeployApprovals"))
		s.PostApprovers = approverNames(objAt(env, "postDeployApprovals"))
		s.Gates = len(listAt(objAt(env, "preDeploymentGates"), "gates")) > 0
	}
	if ret := objAt(env, "retentionPolicy"); ret != nil {
		s.Retention = fmt.Sprintf("%d days, at least %s", intAt(ret, "daysToKeep"), plural(intAt(ret, "releasesToKeep"), "release"))
	}
	for j, p := range listAt(env, phases) {
		phase, _ := p.(map[string]any)
		input := objAt(phase, "deploymentInput")
		ph := DocPhase{
			Index: j, Name: strAt(phase, "name"), Kind: phaseKind(strAt(phase, "phaseType")),
			QueueID: intAt(input, "queueId"), Agent: strAt(objAt(input, "agentSpecification"), "identifier"),
		}
		for k, t := range listAt(phase, "workflowTasks") {
			task, _ := t.(map[string]any)
			ph.Tasks = append(ph.Tasks, parseTask(k, task))
		}
		s.Phases = append(s.Phases, ph)
	}
	return s
}

func stageEditable(status string) bool {
	switch strings.ToLower(status) {
	case "", "undefined", "notstarted":
		return true
	}
	return false
}

func parseTask(index int, t map[string]any) DocTask {
	task := DocTask{
		Index: index, TaskID: strAt(t, "taskId"), Version: strAt(t, "version"), Name: strAt(t, "name"),
		DefinitionType: strAt(t, "definitionType"), Condition: strAt(t, "condition"), Enabled: boolAt(t, "enabled"),
		ContinueOnError: boolAt(t, "continueOnError"), AlwaysRun: boolAt(t, "alwaysRun"), Timeout: intAt(t, "timeoutInMinutes"),
		Inputs: map[string]string{},
	}
	for name, v := range objAt(t, "inputs") {
		task.Inputs[name] = scalar(v)
	}
	return task
}

func phaseKind(kind string) string {
	switch strings.ToLower(kind) {
	case "runonserver", "2":
		return "Agentless job"
	case "machinegroupbaseddeployment", "4":
		return "Deployment group job"
	case "deploymentgates", "8":
		return "Gates"
	}
	return "Agent job"
}

func stageTrigger(conditions []any) string {
	for _, c := range conditions {
		cond, _ := c.(map[string]any)
		switch strings.ToLower(strAt(cond, "conditionType")) {
		case "event", "1":
			return "after release"
		case "environmentstate", "2":
			return "after " + strAt(cond, "name")
		}
	}
	return "manual"
}

func approverNames(approvals map[string]any) []string {
	var out []string
	for _, a := range listAt(approvals, "approvals") {
		ap, _ := a.(map[string]any)
		if name := strAt(objAt(ap, "approver"), "displayName"); !boolAt(ap, "isAutomated") && name != "" {
			out = append(out, name)
		}
	}
	return out
}

func parseTriggers(triggers []any) []string {
	var out []string
	for _, t := range triggers {
		trig, _ := t.(map[string]any)
		switch strings.ToLower(strAt(trig, "triggerType")) {
		case "artifactsource", "1":
			text := "Continuous deployment · " + strAt(trig, "artifactAlias")
			var branches []string
			for _, c := range listAt(trig, "triggerConditions") {
				cond, _ := c.(map[string]any)
				if b := strAt(cond, "sourceBranch"); b != "" {
					branches = append(branches, strings.TrimPrefix(b, "refs/heads/"))
				}
			}
			if len(branches) > 0 {
				text += " · " + strings.Join(branches, ", ")
			}
			out = append(out, text)
		case "schedule", "2":
			sched := objAt(trig, "schedule")
			out = append(out, fmt.Sprintf("Scheduled · %02d:%02d %s", intAt(sched, "startHours"), intAt(sched, "startMinutes"), strAt(sched, "timeZoneId")))
		case "pullrequest", "4":
			out = append(out, "Pull request · "+strAt(trig, "artifactAlias"))
		}
	}
	return out
}

func parseVariables(vars map[string]any, stage int, scope string) []DocVariable {
	out := make([]DocVariable, 0, len(vars))
	for name, raw := range vars {
		v, _ := raw.(map[string]any)
		out = append(out, DocVariable{
			Name: name, Value: strAt(v, "value"), Secret: boolAt(v, "isSecret"), Settable: boolAt(v, "allowOverride"),
			Stage: stage, Scope: scope,
		})
	}
	slices.SortFunc(out, func(a, b DocVariable) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out
}

func parseGroups(groups []any, stage int, scope string) []DocGroup {
	var out []DocGroup
	for _, g := range groups {
		group := DocGroup{Stage: stage, Scope: scope}
		switch v := g.(type) {
		case json.Number:
			n, _ := v.Int64()
			group.ID = int(n)
		case map[string]any:
			group.ID, group.Name = intAt(v, "id"), strAt(v, "name")
		default:
			continue
		}
		out = append(out, group)
	}
	return out
}

func decodeTree(body []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var root map[string]any
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("reading the release pipeline: %w", err)
	}
	if root == nil {
		return nil, fmt.Errorf("reading the release pipeline: empty response")
	}
	return root, nil
}

func objAt(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func listAt(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	return v
}

func strAt(m map[string]any, key string) string { return scalar(m[key]) }

func scalar(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}

func intAt(m map[string]any, key string) int {
	n, _ := strconv.Atoi(strAt(m, key))
	return n
}

func boolAt(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func timeAt(m map[string]any, key string) time.Time {
	t, _ := time.Parse(time.RFC3339, strAt(m, key))
	return t
}
