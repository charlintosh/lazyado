package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	FieldGroupJob       = "Agent job"
	FieldGroupExecution = "Execution plan"
	FieldGroupArtifacts = "Artifacts"
	FieldGroupOptions   = "Additional options"
)

var AgentImages = []TaskOption{
	{Value: "ubuntu-latest", Label: "ubuntu-latest"},
	{Value: "ubuntu-24.04", Label: "ubuntu-24.04"},
	{Value: "ubuntu-22.04", Label: "ubuntu-22.04"},
	{Value: "windows-latest", Label: "windows-latest"},
	{Value: "windows-2025", Label: "windows-2025"},
	{Value: "windows-2022", Label: "windows-2022"},
	{Value: "macos-latest", Label: "macos-latest"},
	{Value: "macos-15", Label: "macos-15"},
	{Value: "macos-14", Label: "macos-14"},
}

var parallelism = []TaskOption{
	{Value: "none", Label: "None"},
	{Value: "multiConfiguration", Label: "Multi-configuration"},
	{Value: "multiMachine", Label: "Multi-agent"},
}

func (d *ReleaseDoc) PhasePath(stage DocStage, phase DocPhase) DocPath {
	return DocPath{"environments", stage.Index, d.phasesKey(), phase.Index}
}

func (p DocPhase) AgentJob() bool { return p.Kind == "Agent job" }

func (d *ReleaseDoc) PhaseFields(stage DocStage, phase DocPhase, queues map[int]AgentQueue, edits DocEdits) []TaskField {
	base := d.PhasePath(stage, phase)
	input := base.Child("deploymentInput")
	guard := DocPath{"environments", stage.Index, "name"}
	field := func(key, label, group string, kind InputKind, path DocPath) TaskField {
		original := d.ValueAt(path)
		return TaskField{
			Key: key, Label: label, Group: group, Kind: kind, Path: path, Original: original,
			Value: edits.Value(path, original), Guard: guard, GuardValue: stage.Name,
		}
	}
	name := field("$name", "Display name", FieldGroupJob, InputText, base.Child("name"))
	name.Required = true
	fields := []TaskField{name}
	timeout := withDefault(field("$timeout", "Timeout (minutes)", FieldGroupExecution, InputNumber, input.Child("timeoutInMinutes")), "0")
	condition := withDefault(field("$condition", "Run this job when", FieldGroupOptions, InputText, input.Child("condition")), "succeeded()")
	if !phase.AgentJob() {
		return append(fields, timeout, condition)
	}
	pool := field("$queue", "Agent pool", FieldGroupJob, InputQueue, input.Child("queueId"))
	pool.Required = true
	fields = append(fields, pool)
	agent := field("$agent", "Agent specification", FieldGroupJob, InputChoice, input.Child("agentSpecification", "identifier"))
	agent.Options = agentOptions(FormatDocValue(agent.Original))
	agent.Required = true
	queueID, _ := asInt(pool.Value)
	if queues[queueID].Hosted || FormatDocValue(agent.Value) != "" {
		fields = append(fields, agent)
	}
	parallel := field("$parallel", "Parallelism", FieldGroupExecution, InputChoice, input.Child("parallelExecution", "parallelExecutionType"))
	parallel.Options, parallel.Default = parallelism, "none"
	if _, numeric := parallel.Original.(json.Number); numeric {
		parallel.ReadOnly = true
	}
	return append(fields,
		field("$demands", "Demands", FieldGroupJob, InputLines, input.Child("demands")),
		parallel,
		timeout,
		withDefault(field("$cancelTimeout", "Job cancel timeout (minutes)", FieldGroupExecution, InputNumber, input.Child("jobCancelTimeoutInMinutes")), "1"),
		field("$skipArtifacts", "Skip artifact download", FieldGroupArtifacts, InputBool, input.Child("skipArtifactsDownload")),
		field("$accessToken", "Allow scripts the OAuth token", FieldGroupOptions, InputBool, input.Child("enableAccessToken")),
		condition,
	)
}

func (p DocPhase) RunsOn() string {
	switch p.Kind {
	case "Agentless job":
		return "run on server"
	case "Deployment group job":
		return "deployment group"
	case "Gates":
		return "gates"
	}
	return "run on agent"
}

func agentOptions(current string) []TaskOption {
	for _, o := range AgentImages {
		if strings.EqualFold(o.Value, current) {
			return AgentImages
		}
	}
	if current == "" {
		return AgentImages
	}
	return append([]TaskOption{{Value: current, Label: current + " (saved)"}}, AgentImages...)
}

func asInt(v any) (int, bool) {
	var n int
	_, err := fmt.Sscan(FormatDocValue(v), &n)
	return n, err == nil
}

func QueueID(v any) int {
	n, _ := asInt(v)
	return n
}

func fieldValue(fields []TaskField, key string) string {
	for _, f := range fields {
		if strings.EqualFold(f.Key, key) {
			return f.EditText()
		}
	}
	return ""
}

func fieldLabel(fields []TaskField, key string) string {
	for _, f := range fields {
		if strings.EqualFold(f.Key, key) {
			return f.Label
		}
	}
	return key
}

func AzureRequestFor(fields []TaskField, f TaskField) (AzureListRequest, error) {
	if f.Source == nil {
		return AzureListRequest{}, fmt.Errorf("%s isn't listed by Azure", f.Label)
	}
	src := *f.Source
	req := AzureListRequest{}
	if src.EndpointInput != "" {
		req.EndpointID = strings.TrimSpace(fieldValue(fields, src.EndpointInput))
		if req.EndpointID == "" {
			return req, fmt.Errorf("pick %s first", fieldLabel(fields, src.EndpointInput))
		}
	}
	var missing string
	resolve := func(text string) string {
		return inputRef.ReplaceAllStringFunc(text, func(ref string) string {
			name := inputRef.FindStringSubmatch(ref)[1]
			v := fieldValue(fields, name)
			if v == "" && missing == "" && hasField(fields, name) {
				missing = fieldLabel(fields, name)
			}
			if v == "" && !hasField(fields, name) {
				return ref
			}
			return v
		})
	}
	params := make(map[string]string, len(src.Parameters))
	for k, v := range src.Parameters {
		params[k] = resolve(v)
	}
	src.Parameters = params
	src.EndpointURL = resolve(src.EndpointURL)
	if missing != "" {
		return req, fmt.Errorf("pick %s first", missing)
	}
	req.Source = src
	return req, nil
}

func hasField(fields []TaskField, key string) bool {
	for _, f := range fields {
		if strings.EqualFold(f.Key, key) {
			return true
		}
	}
	return false
}

func DependentsOf(fields []TaskField, key string) []TaskField {
	var out []TaskField
	seen := map[string]bool{strings.ToLower(key): true}
	queue := []string{key}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		for _, f := range fields {
			k := strings.ToLower(f.Key)
			if !seen[k] && f.DependsOn(next) {
				seen[k] = true
				out = append(out, f)
				queue = append(queue, f.Key)
			}
		}
	}
	return out
}

type StaleField struct {
	Path   DocPath
	Field  string
	Where  string
	Parent DocPath
	Cause  string
}

type AzureValue struct {
	Value string
	Label string
}
