package models

import (
	"encoding/json"
	"strings"
	"testing"
)

const definitionFixture = `{"id":7,"name":"Update infra","path":"\\infra","revision":42,"releaseNameFormat":"Release-$(rev:r)",
"modifiedBy":{"displayName":"Ana Example"},"modifiedOn":"2026-11-17T10:00:00.123Z","futureField":{"keep":[1,2.50,"x"]},
"_links":{"web":{"href":"https://dev.azure.com/contoso/web/_release?definitionId=7"}},
"variables":{"LogDays":{"value":"30","allowOverride":true},"Password":{"value":null,"isSecret":true}},
"variableGroups":[3],
"artifacts":[{"alias":"_infra","type":"Build","isPrimary":true,"definitionReference":{"definition":{"id":"9","name":"infra-CI"}}}],
"triggers":[{"triggerType":"artifactSource","artifactAlias":"_infra","triggerConditions":[{"sourceBranch":"refs/heads/main"}]}],
"environments":[
 {"id":2,"name":"prod","rank":2,"conditions":[{"conditionType":"environmentState","name":"test"}],
  "preDeployApprovals":{"approvals":[{"isAutomated":false,"approver":{"displayName":"Ops leads"}}]},
  "postDeployApprovals":{"approvals":[{"isAutomated":true}]},
  "variables":{"Nodes":{"value":"5","allowOverride":true}},"variableGroups":[],
  "deployPhases":[{"name":"Agent job","phaseType":"agentBasedDeployment","deploymentInput":{"queueId":11,"agentSpecification":{"identifier":"ubuntu-latest"}},
   "workflowTasks":[{"taskId":"46e4be58-0000-0000-0000-000000000001","version":"1.*","name":"Deploy","enabled":true,"timeoutInMinutes":0,"definitionType":"task","condition":"succeeded()","inputs":{"scriptLocation":"scriptPath","scriptPath":"deploy.sh","inlineScript":""}}]}]},
 {"id":1,"name":"test","rank":1,"conditions":[{"conditionType":"event","name":"ReleaseStarted"}],
  "preDeployApprovals":{"approvals":[{"isAutomated":true}]},"preDeploymentGates":{"gates":[]},
  "retentionPolicy":{"daysToKeep":30,"releasesToKeep":3},
  "variables":{},"variableGroups":[4],
  "deployPhases":[{"name":"Agent job","phaseType":"agentBasedDeployment","deploymentInput":{"queueId":11},
   "workflowTasks":[
    {"taskId":"6c731c3c-0000-0000-0000-000000000002","version":"3.*","name":"Install","enabled":true,"definitionType":"task","inputs":{"targetType":"inline","script":"echo hi"}},
    {"taskId":"46e4be58-0000-0000-0000-000000000001","version":"1.*","name":"Log Analytics","enabled":false,"definitionType":"task","inputs":{"connectedServiceNameARM":"00000000-aaaa-0000-0000-000000000001","scriptLocation":"scriptPath","scriptPath":"log.sh"}}]}]}]}`

const releaseFixture = `{"id":118,"name":"Release-118","releaseDefinition":{"id":7,"name":"Update infra"},"releaseDefinitionRevision":42,
"variables":{},"variableGroups":[{"id":3,"name":"shared"}],
"environments":[
 {"id":20,"name":"test","rank":1,"status":"succeeded","variables":{},
  "deployPhasesSnapshot":[{"name":"Agent job","phaseType":"agentBasedDeployment","workflowTasks":[{"taskId":"t1","version":"2.*","name":"A","enabled":true,"inputs":{}}]}]},
 {"id":21,"name":"prod","rank":2,"status":"notStarted","variables":{},
  "preApprovalsSnapshot":{"approvals":[{"isAutomated":false,"approver":{"displayName":"Ops leads"}}]},
  "deployPhasesSnapshot":[{"name":"Agent job","phaseType":"agentBasedDeployment","workflowTasks":[{"taskId":"t1","version":"2.*","name":"A","enabled":true,"inputs":{"x":"1"}}]}]}]}`

func azureCLI() *TaskDefinition {
	return &TaskDefinition{
		ID: "46e4be58-0000-0000-0000-000000000001", Name: "AzureCLI", FriendlyName: "Azure CLI", Major: 1,
		Groups: []TaskInputGroup{{Name: "advanced", Label: "Advanced"}},
		Inputs: []TaskInput{
			{Name: "connectedServiceNameARM", Label: "Azure subscription", Type: "connectedService:AzureRM", Required: true},
			{Name: "scriptLocation", Label: "Script Location", Type: "pickList", Default: "scriptPath",
				Options: []TaskOption{{Value: "inlineScript", Label: "Inline script"}, {Value: "scriptPath", Label: "Script path"}}},
			{Name: "scriptPath", Label: "Script Path", Type: "filePath", VisibleRule: "scriptLocation = scriptPath"},
			{Name: "inlineScript", Label: "Inline Script", Type: "multiLine", VisibleRule: "scriptLocation = inlineScript"},
			{Name: "failOnStandardError", Label: "Fail on Standard Error", Type: "boolean", Default: "false", Group: "advanced"},
		},
	}
}

func parseDefinition(t *testing.T) *ReleaseDoc {
	t.Helper()
	d, err := ParseReleaseDoc(DocDefinition, []byte(definitionFixture))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestParseReleaseDefinitionReadsStagesTasksAndVariables(t *testing.T) {
	d := parseDefinition(t)
	if d.Title() != "Update infra" || d.Revision != 42 || d.Folder != `\infra` || d.ModifiedBy != "Ana Example" || d.Modified.IsZero() {
		t.Fatalf("header %+v", d)
	}
	if len(d.Stages) != 2 || d.Stages[0].Name != "test" || d.Stages[0].Index != 1 || d.Stages[1].Name != "prod" {
		t.Fatalf("stages must follow rank and keep their raw index: %+v", d.Stages)
	}
	test, prod := d.Stages[0], d.Stages[1]
	if test.Trigger != "after release" || prod.Trigger != "after test" || test.Approval(true) != "automatic" || prod.Approval(true) != "Ops leads" {
		t.Fatalf("triggers or approvals: %+v %+v", test, prod)
	}
	if test.Summary() != "1 job, 2 tasks" || test.Retention != "30 days, at least 3 releases" {
		t.Fatalf("summary %q retention %q", test.Summary(), test.Retention)
	}
	task := test.Phases[0].Tasks[1]
	if task.Name != "Log Analytics" || task.Enabled || task.Major() != 1 || task.Inputs["scriptPath"] != "log.sh" {
		t.Fatalf("task %+v", task)
	}
	if len(d.Variables) != 3 || d.Variables[0].Name != "LogDays" || !d.Variables[1].Secret || d.Variables[2].Scope != "prod" {
		t.Fatalf("variables %+v", d.Variables)
	}
	if ids := d.GroupIDs(); len(ids) != 2 || ids[0] != 3 || ids[1] != 4 {
		t.Fatalf("groups %v", ids)
	}
	if len(d.Triggers) != 1 || d.Triggers[0] != "Continuous deployment · _infra · main" {
		t.Fatalf("triggers %v", d.Triggers)
	}
	if refs := d.TaskRefs(); len(refs) != 2 {
		t.Fatalf("task refs %v", refs)
	}
	if q := d.QueueIDs(); len(q) != 1 || q[0] != 11 {
		t.Fatalf("queues %v", q)
	}
}

func TestPayloadAppliesEditsAndKeepsEverythingElse(t *testing.T) {
	d := parseDefinition(t)
	test := d.Stages[0]
	path := d.TaskPath(test, test.Phases[0], test.Phases[0].Tasks[1]).Child("inputs", "scriptPath")
	edits := DocEdits{}.With(DocEdit{Path: path, Before: "log.sh", After: "log-v2.sh"})
	secret := DocVariable{Name: "Password", Stage: -1, Secret: true}
	edits = edits.With(DocEdit{Path: d.VariablePath(secret), Before: nil, After: "s3cret"})
	body, err := d.Payload(edits, "Use v2")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got["comment"] != "Use v2" || !strings.Contains(string(body), `"futureField":{"keep":[1,2.50,"x"]}`) {
		t.Fatalf("payload lost fields: %s", body)
	}
	if !strings.Contains(string(body), `"scriptPath":"log-v2.sh"`) || !strings.Contains(string(body), `"value":"s3cret"`) {
		t.Fatalf("edits not applied: %s", body)
	}
	if d.ValueAt(path) != "log.sh" {
		t.Fatal("the payload must not change the loaded document")
	}
}

func TestEditsDropWhenSetBackToTheOriginal(t *testing.T) {
	path := DocPath{"variables", "LogDays", "value"}
	edits := DocEdits{}.With(DocEdit{Path: path, Before: "30", After: "90"})
	if len(edits) != 1 || edits.Value(path, "30") != "90" {
		t.Fatalf("edit %+v", edits)
	}
	edits = edits.With(DocEdit{Path: path, Before: "30", After: "30"})
	if len(edits) != 0 {
		t.Fatalf("setting the original back should drop the edit: %+v", edits)
	}
	task := DocPath{"environments", 1, "deployPhases", 0, "workflowTasks", 1}
	edits = DocEdits{}.With(DocEdit{Path: task.Child("enabled"), Before: false, After: true}).With(DocEdit{Path: task.Child("name"), Before: "a", After: "b"})
	if edits.CountUnder(task) != 2 || len(edits.WithoutUnder(task)) != 0 {
		t.Fatalf("task edits %+v", edits)
	}
}

func TestReapplyKeepsEditsWhoseFieldNobodyChanged(t *testing.T) {
	d := parseDefinition(t)
	test := d.Stages[0]
	task := test.Phases[0].Tasks[1]
	base := d.TaskPath(test, test.Phases[0], task)
	guard := base.Child("taskId")
	mine := DocEdits{
		{Path: base.Child("inputs", "scriptPath"), Field: "Script Path", Before: "log.sh", After: "log-v2.sh", Guard: guard, GuardValue: task.TaskID},
		{Path: base.Child("name"), Field: "Display name", Before: "Old name", After: "New name", Guard: guard, GuardValue: task.TaskID},
		{Path: base.Child("enabled"), Field: "Enabled", Before: false, After: true, Guard: guard, GuardValue: "someone-else"},
	}
	kept, dropped := ReapplyEdits(d, mine)
	if len(kept) != 1 || kept[0].Field != "Script Path" || len(dropped) != 2 {
		t.Fatalf("kept %+v dropped %+v", kept, dropped)
	}
}

func TestInputVisibleRules(t *testing.T) {
	values := map[string]string{"scriptlocation": "scriptPath", "type": "inline", "count": "3"}
	cases := map[string]bool{
		"":                               true,
		"scriptLocation = scriptPath":    true,
		"scriptLocation = inlineScript":  false,
		"scriptLocation != inlineScript": true,
		"type = file && scriptLocation = scriptPath": false,
		"type = file || scriptLocation = scriptPath": true,
		"ScriptLocation endsWith Path":               true,
		"count > 2":                                  true,
		"unknown = \"\"":                             true,
		"weird syntax ~~":                            true,
	}
	for rule, want := range cases {
		if got := InputVisible(rule, values); got != want {
			t.Errorf("%q: got %v, want %v", rule, got, want)
		}
	}
}

func TestTaskFieldsHideInputsByRuleAndFollowEdits(t *testing.T) {
	d := parseDefinition(t)
	test := d.Stages[0]
	phase := test.Phases[0]
	task := phase.Tasks[1]
	fields, hidden := d.TaskFields(test, phase, task, azureCLI(), nil)
	if hidden != 1 {
		t.Fatalf("inline script should be hidden, hidden=%d", hidden)
	}
	var labels []string
	for _, f := range fields {
		labels = append(labels, f.Label)
	}
	want := "Display name,Azure subscription,Script Location,Script Path,Fail on Standard Error,Enabled,Continue on error,Always run,Timeout (minutes),Run this task when"
	if strings.Join(labels, ",") != want {
		t.Fatalf("fields %s", strings.Join(labels, ","))
	}
	if fields[2].Text() != "Script path" || fields[4].Group != "Advanced" || fields[4].Text() != "No" || fields[5].Text() != "No" {
		t.Fatalf("texts %q %q %q %q", fields[2].Text(), fields[4].Group, fields[4].Text(), fields[5].Text())
	}
	edits := DocEdits{}.With(fields[2].Edit("test / Log Analytics", "inlineScript"))
	fields, hidden = d.TaskFields(test, phase, task, azureCLI(), edits)
	if hidden != 1 || fields[3].Label != "Inline Script" || !fields[2].Changed() {
		t.Fatalf("switching location should show the inline script: %+v", fields[3])
	}
	enabled := fields[len(fields)-5]
	if enabled.ToggledValue() != true {
		t.Fatalf("toggle %v", enabled.ToggledValue())
	}
}

func TestTaskFieldsWithoutMetadataListRawInputs(t *testing.T) {
	d := parseDefinition(t)
	test := d.Stages[0]
	fields, hidden := d.TaskFields(test, test.Phases[0], test.Phases[0].Tasks[0], nil, nil)
	if hidden != 0 || fields[1].Label != "script" || fields[2].Label != "targetType" {
		t.Fatalf("raw fields %+v", fields[:3])
	}
}

func TestTaskYAML(t *testing.T) {
	d := parseDefinition(t)
	test := d.Stages[0]
	task := test.Phases[0].Tasks[1]
	fields, _ := d.TaskFields(test, test.Phases[0], task, azureCLI(), nil)
	got := TaskYAML(task, azureCLI(), fields)
	want := "- task: AzureCLI@1\n  displayName: Log Analytics\n  enabled: false\n  inputs:\n" +
		"    connectedServiceNameARM: 00000000-aaaa-0000-0000-000000000001\n    scriptLocation: scriptPath\n    scriptPath: log.sh"
	if got != want {
		t.Fatalf("yaml:\n%s", got)
	}
	if yamlScalar("a: b", "") != "'a: b'" || yamlScalar("it's", "") != "'it''s'" || yamlScalar("true", "") != "'true'" || yamlScalar("x\ny", "  ") != "|\n  x\n  y" {
		t.Fatal("yaml quoting")
	}
}

func TestParseReleaseLocksDeployedStagesAndUsesSnapshots(t *testing.T) {
	d, err := ParseReleaseDoc(DocRelease, []byte(releaseFixture))
	if err != nil {
		t.Fatal(err)
	}
	if d.Title() != "Release-118" || d.Definition != "Update infra" || d.Revision != 42 || !d.Editable() {
		t.Fatalf("release %+v", d)
	}
	test, prod := d.Stages[0], d.Stages[1]
	if !test.Locked || prod.Locked || prod.Approval(true) != "Ops leads" {
		t.Fatalf("locks %+v %+v", test, prod)
	}
	path := d.TaskPath(prod, prod.Phases[0], prod.Phases[0].Tasks[0]).Child("inputs", "x")
	if d.ValueAt(path) != "1" || path[2] != "deployPhasesSnapshot" {
		t.Fatalf("path %v", path)
	}
	if len(d.Groups) != 1 || d.Groups[0].Name != "shared" {
		t.Fatalf("groups %+v", d.Groups)
	}
}

func TestFieldValueValidatesNumbers(t *testing.T) {
	if _, err := FieldValue(InputNumber, json.Number("0"), "ten"); err == nil {
		t.Fatal("want an error")
	}
	if v, _ := FieldValue(InputNumber, json.Number("0"), "15"); v != json.Number("15") {
		t.Fatalf("got %#v", v)
	}
	if v, _ := FieldValue(InputNumber, "0", "15"); v != "15" {
		t.Fatalf("string inputs stay strings, got %#v", v)
	}
}

func TestAzureRequestsResolveTheTaskValuesTheyDependOn(t *testing.T) {
	src := &DataSource{Name: "AzureResourceGroups", EndpointInput: "conn", Parameters: map[string]string{"subscriptionId": "$(sub)", "fixed": "$(System.Other)"}}
	fields := []TaskField{
		{Key: "conn", Label: "Connection", Kind: InputEndpoint, Value: "c-1"},
		{Key: "sub", Label: "Subscription", Kind: InputAzure, Value: "s-1", Source: &DataSource{EndpointInput: "conn"}},
		{Key: "rg", Label: "Resource group", Kind: InputAzure, Source: src},
		{Key: "loc", Label: "Location", Kind: InputAzure, Value: "eastus2", Source: &DataSource{EndpointInput: "conn", Parameters: map[string]string{"s": "$(sub)"}}},
	}
	req, err := AzureRequestFor(fields, fields[2])
	if err != nil || req.EndpointID != "c-1" || req.Source.Parameters["subscriptionId"] != "s-1" || req.Source.Parameters["fixed"] != "$(System.Other)" {
		t.Fatalf("request %+v %v", req, err)
	}
	fields[1].Value = nil
	if _, err := AzureRequestFor(fields, fields[2]); err == nil || err.Error() != "pick Subscription first" {
		t.Fatalf("a missing parent must be named, got %v", err)
	}
	var deps []string
	for _, f := range DependentsOf(fields, "conn") {
		deps = append(deps, f.Key)
	}
	if strings.Join(deps, ",") != "sub,rg,loc" {
		t.Fatalf("dependents %v", deps)
	}
	if got := DependentsOf(fields, "sub"); len(got) != 2 {
		t.Fatalf("subscription feeds resource group and location, got %v", got)
	}
}

func TestPhaseFieldsShowAgentSettingsWithDefaults(t *testing.T) {
	d := parseDefinition(t)
	test := d.Stages[0]
	fields := d.PhaseFields(test, test.Phases[0], map[int]AgentQueue{11: {ID: 11, Name: "Azure Pipelines", Hosted: true}}, nil)
	var labels []string
	for _, f := range fields {
		labels = append(labels, f.Label+"="+f.Text())
	}
	want := "Display name=Agent job,Agent pool=11,Agent specification=,Demands=,Parallelism=None,Timeout (minutes)=0,Job cancel timeout (minutes)=1," +
		"Skip artifact download=No,Allow scripts the OAuth token=No,Run this job when=succeeded()"
	if strings.Join(labels, ",") != want {
		t.Fatalf("fields\n%s", strings.Join(labels, ","))
	}
	selfHosted := d.PhaseFields(test, test.Phases[0], map[int]AgentQueue{11: {ID: 11, Name: "linux"}}, nil)
	if len(selfHosted) != len(fields)-1 {
		t.Fatal("a self-hosted pool with no image has no agent specification")
	}
	if v, err := FieldValue(InputLines, nil, "Agent.OS -equals Linux\n\n  java\n"); err != nil || len(v.([]any)) != 2 {
		t.Fatalf("demands %v %v", v, err)
	}
	spec := fields[2]
	spec.Original, spec.Value = "vs2017-win2016", "vs2017-win2016"
	if opts := agentOptions("vs2017-win2016"); opts[0].Label != "vs2017-win2016 (saved)" {
		t.Fatalf("a retired image stays choosable as the saved value, got %v", opts[0])
	}
}
