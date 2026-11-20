package ui

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

const editorDefinitionJSON = `{"id":12,"name":"smartHotel","path":"\\web","revision":%REV%,"releaseNameFormat":"Release-$(rev:r)",
"modifiedBy":{"displayName":"Gopi R"},"modifiedOn":"2026-11-17T10:00:00Z",
"variables":{"LogDays":{"value":"30","allowOverride":true},"DbPassword":{"value":null,"isSecret":true}},
"variableGroups":[3],
"artifacts":[{"alias":"_smartHotel-CI","type":"Build","isPrimary":true,"definitionReference":{"definition":{"id":"7","name":"smartHotel-CI"}}}],
"environments":[
 {"id":1,"name":"Dev1","rank":1,"conditions":[{"conditionType":"event","name":"ReleaseStarted"}],"variables":{"Nodes":{"value":"2"}},
  "deployPhases":[{"name":"Agent job","phaseType":"agentBasedDeployment","deploymentInput":{"queueId":11,"agentSpecification":{"identifier":"ubuntu-latest"}},
   "workflowTasks":[
    {"taskId":"6c731c3c-0000-0000-0000-000000000002","version":"3.*","name":"Install tools","enabled":true,"definitionType":"task","inputs":{"script":"make tools"}},
    {"taskId":"46e4be58-0000-0000-0000-000000000001","version":"1.*","name":"Log Analytics","enabled":true,"timeoutInMinutes":0,"definitionType":"task","condition":"succeeded()",
     "inputs":{"connectedServiceNameARM":"00000000-aaaa-0000-0000-000000000001","scriptLocation":"scriptPath","scriptPath":"scripts/log.sh","failOnStandardError":"false"}}]}]},
 {"id":2,"name":"Prod","rank":2,"conditions":[{"conditionType":"environmentState","name":"Dev1"}],
  "preDeployApprovals":{"approvals":[{"isAutomated":false,"approver":{"displayName":"Ops on-call"}}]},"variables":{},
  "deployPhases":[{"name":"Agent job","phaseType":"agentBasedDeployment","deploymentInput":{"queueId":11},"workflowTasks":[
    {"taskId":"46e4be58-0000-0000-0000-000000000001","version":"1.*","name":"Log Analytics","enabled":true,"inputs":{"scriptLocation":"scriptPath","scriptPath":"scripts/log.sh"}}]}]}]}`

const editorReleaseJSON = `{"id":64,"name":"Release-64","releaseDefinition":{"id":12,"name":"smartHotel"},"releaseDefinitionRevision":7,"variables":{},
"environments":[
 {"id":1,"name":"Dev1","rank":1,"status":"succeeded","variables":{},"deployPhasesSnapshot":[{"name":"Agent job","workflowTasks":[{"taskId":"t1","version":"1.*","name":"Deploy","enabled":true,"inputs":{"a":"1"}}]}]},
 {"id":3,"name":"Prod","rank":2,"status":"notStarted","variables":{},"deployPhasesSnapshot":[{"name":"Agent job","workflowTasks":[{"taskId":"t1","version":"1.*","name":"Deploy","enabled":true,"inputs":{"a":"1"}}]}]}]}`

func editorDoc(t testing.TB, revision string) *models.ReleaseDoc {
	t.Helper()
	d, err := models.ParseReleaseDoc(models.DocDefinition, []byte(strings.Replace(editorDefinitionJSON, "%REV%", revision, 1)))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func editorRefs() models.ReleaseDocRefs {
	cli := &models.TaskDefinition{
		ID: "46e4be58-0000-0000-0000-000000000001", Name: "AzureCLI", FriendlyName: "Azure CLI", Major: 1,
		Groups: []models.TaskInputGroup{{Name: "advanced", Label: "Advanced"}},
		Inputs: []models.TaskInput{
			{Name: "connectedServiceNameARM", Label: "Azure subscription", Type: "connectedService:AzureRM", Required: true},
			{Name: "scriptLocation", Label: "Script Location", Type: "pickList", Default: "scriptPath",
				Options: []models.TaskOption{{Value: "inlineScript", Label: "Inline script"}, {Value: "scriptPath", Label: "Script path"}}},
			{Name: "scriptPath", Label: "Script Path", Type: "filePath", VisibleRule: "scriptLocation = scriptPath"},
			{Name: "inlineScript", Label: "Inline Script", Type: "multiLine", VisibleRule: "scriptLocation = inlineScript"},
			{Name: "failOnStandardError", Label: "Fail on Standard Error", Type: "boolean", Default: "false", Group: "advanced"},
		},
	}
	return models.ReleaseDocRefs{
		Tasks:     map[string]*models.TaskDefinition{models.TaskRef{ID: cli.ID, Major: 1}.Key(): cli},
		Endpoints: map[string]string{"00000000-aaaa-0000-0000-000000000001": "Prod subscription"},
		Queues:    map[int]models.AgentQueue{11: {ID: 11, Name: "Azure Pipelines", Hosted: true}},
		Groups:    map[int]models.VariableGroupInfo{3: {Name: "shared-secrets", Variables: 4}},
	}
}

func openEditor(t testing.TB, width, height int) App {
	t.Helper()
	a := openReleasesHub(t, width, height)
	a, msgs := step(t, a, press("e"))
	open, ok := find[panels.OpenReleaseEditorMsg](msgs)
	if !ok || open.Kind != models.DocDefinition || open.ID != 12 {
		t.Fatalf("e on a release pipeline must open its definition, got %v", msgs)
	}
	a = feed(t, a, open)
	if !a.viewingReleaseEditor() || !a.rl.editor.Loading() {
		t.Fatal("the editor must open and load")
	}
	return feed(t, a, releaseDocLoadedMsg{seq: a.rl.editorSeq, doc: editorDoc(t, "42"), refs: editorRefs()})
}

func pressAll(t testing.TB, a App, presses ...tea.Msg) (App, []tea.Msg) {
	t.Helper()
	var all []tea.Msg
	for _, p := range presses {
		var msgs []tea.Msg
		a, msgs = step(t, a, p)
		all = append(all, msgs...)
	}
	return a, all
}

func TestReleaseEditorShowsThePipelineTasksAndVariables(t *testing.T) {
	a := openEditor(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{"PIPELINE", "smartHotel", "Revision 42", "1 Pipeline", "2 Tasks", "3 Variables", "4 History",
		"Dev1", "1 job, 2 tasks", "after release", "_smartHotel-CI", "Prod", "1 approver", "Agent job · Azure Pipelines · ubuntu-latest · 2 tasks"} {
		if !strings.Contains(out, want) {
			t.Errorf("pipeline tab is missing %q", want)
		}
	}
	a, _ = pressAll(t, a, press("2"), press("j"), press("j"), special(tea.KeyEnter))
	out = ansi.Strip(a.render())
	for _, want := range []string{"Deployment process", "Install tools", "Log Analytics", "AzureCLI@1", "Azure CLI · 1.*",
		"Azure subscription", "Prod subscription", "from Azure", "Script Location", "Script path", "scripts/log.sh", "ADVANCED", "CONTROL OPTIONS",
		"◌ 1 input hidden by the values above.", "▾ Agent job", "├ ● Install tools", "└ ● Log Analytics"} {
		if !strings.Contains(out, want) {
			t.Errorf("tasks tab is missing %q", want)
		}
	}
	if strings.Contains(out, "Inline Script") {
		t.Error("inputs hidden by their visible rule must not show")
	}
	a, _ = pressAll(t, a, press("3"))
	out = ansi.Strip(a.render())
	for _, want := range []string{"All 3", "Release 2", "Dev1 1", "LogDays", "●●●●●●●●", "secret", "■ Release", "● yes", "⊘ shared-secrets", "4 variables"} {
		if !strings.Contains(out, want) {
			t.Errorf("variables tab is missing %q", want)
		}
	}
}

func TestReleaseEditorRequiresEditModeBeforeChanging(t *testing.T) {
	a := openEditor(t, 160, 45)
	a, msgs := pressAll(t, a, press("2"), press("j"), press("j"), special(tea.KeyEnter), special(tea.KeyEnter))
	if n, ok := find[panels.ReleaseNoticeMsg](msgs); !ok || !strings.Contains(n.Text, "Press e") {
		t.Fatalf("enter outside edit mode must say how to edit, got %v", msgs)
	}
	if _, ok := find[panels.EditReleaseFieldMsg](msgs); ok {
		t.Fatal("no field may be edited outside edit mode")
	}
}

func TestReleaseEditorEditReviewAndSave(t *testing.T) {
	a := openEditor(t, 160, 45)
	a, _ = pressAll(t, a, press("2"), press("j"), press("j"), special(tea.KeyEnter), press("e"))
	if !a.rl.editor.Editing() || !strings.Contains(ansi.Strip(a.render()), "EDITING") || a.modeColor() == nil {
		t.Fatal("e must start editing")
	}
	a, msgs := pressAll(t, a, press("j"), press("j"), press("j"), special(tea.KeyEnter))
	edit, ok := find[panels.EditReleaseFieldMsg](msgs)
	if !ok || edit.Field.Label != "Script Path" || edit.Where != "Dev1 / Log Analytics" {
		t.Fatalf("enter on Script Path must ask to edit it, got %v", msgs)
	}
	a = feed(t, a, edit)
	topModal[modals.ReleaseFieldModal](t, a)
	a = feed(t, a, modals.ReleaseFieldEditedMsg{Field: edit.Field, Where: edit.Where, Value: "scripts/log-v2.sh"})
	if a.modals.open() || len(a.rl.editor.Edits()) != 1 {
		t.Fatalf("the change must be kept and the dialog closed, edits %+v", a.rl.editor.Edits())
	}
	a, _ = pressAll(t, a, press(" "))
	if len(a.rl.editor.Edits()) != 2 {
		t.Fatalf("space must turn the task off, edits %+v", a.rl.editor.Edits())
	}
	out := ansi.Strip(a.render())
	for _, want := range []string{"● 2 unsaved", "saving creates revision 43", "2 Tasks 2", "− scripts/log.sh", "OFF", "○ Log Analytics"} {
		if !strings.Contains(out, want) {
			t.Errorf("edit mode is missing %q", want)
		}
	}
	a, msgs = pressAll(t, a, ctrl('s'))
	review, ok := find[panels.ReviewReleaseEditsMsg](msgs)
	if !ok {
		t.Fatalf("ctrl+s must open the review, got %v", msgs)
	}
	a = feed(t, a, review)
	topModal[modals.ReleaseSaveModal](t, a)
	out = ansi.Strip(a.render())
	for _, want := range []string{"Save smartHotel", "2 changes · saved as revision 43", "Dev1 / Log Analytics", "- scripts/log.sh", "+ scripts/log-v2.sh", "- Yes", "+ No"} {
		if !strings.Contains(out, want) {
			t.Errorf("review is missing %q", want)
		}
	}
	a, msgs = pressAll(t, a, ctrl('s'))
	if _, ok := find[modals.SaveReleaseEditsRequestMsg](msgs); !ok {
		t.Fatalf("ctrl+s in the review must save, got %v", msgs)
	}
	a = feed(t, a, releaseDocSavedMsg{doc: editorDoc(t, "43")})
	if a.modals.open() || a.rl.editor.Editing() || a.rl.editor.Dirty() || a.rl.editor.Doc().Revision != 43 {
		t.Fatal("saving must close the review, leave edit mode and show the new revision")
	}
}

func TestReleaseEditorConflictReloadsAndReappliesUntouchedChanges(t *testing.T) {
	a := openEditor(t, 160, 45)
	a, _ = pressAll(t, a, press("2"), press("j"), press("j"), special(tea.KeyEnter), press("e"), press(" "))
	a, msgs := pressAll(t, a, ctrl('s'))
	a = feed(t, a, msgs[0])
	a = feed(t, a, releaseDocConflictMsg{latest: editorDoc(t, "44")})
	if !strings.Contains(strings.Join(strings.Fields(ansi.Strip(a.render())), " "), "revision 44 was saved by Gopi R") {
		t.Fatal("the review must explain the conflict")
	}
	a, msgs = pressAll(t, a, press("r"))
	reapply, ok := find[modals.ReapplyReleaseEditsMsg](msgs)
	if !ok {
		t.Fatalf("r must reload and reapply, got %v", msgs)
	}
	a = feed(t, a, reapply)
	if a.modals.open() || a.rl.editor.Doc().Revision != 44 || len(a.rl.editor.Edits()) != 1 || !a.rl.editor.Editing() {
		t.Fatalf("reapplying must keep the change on the new revision, edits %+v", a.rl.editor.Edits())
	}
}

func TestReleaseEditorEscAsksBeforeDiscarding(t *testing.T) {
	a := openEditor(t, 160, 45)
	a, _ = pressAll(t, a, press("2"), press("j"), press("e"), press(" "))
	a, msgs := pressAll(t, a, special(tea.KeyEscape))
	discard, ok := find[panels.DiscardReleaseEditsMsg](msgs)
	if !ok || discard.Count != 1 {
		t.Fatalf("esc with changes must ask first, got %v", msgs)
	}
	a = feed(t, a, discard)
	a, msgs = pressAll(t, a, press("y"))
	a = feed(t, a, msgs...)
	if a.modals.open() || a.rl.editor.Dirty() || a.rl.editor.Editing() {
		t.Fatal("confirming must discard the changes and leave edit mode")
	}
	a, msgs = pressAll(t, a, special(tea.KeyEscape), special(tea.KeyEscape))
	a = feed(t, a, msgs...)
	if a.viewingReleaseEditor() || !a.inReleases() {
		t.Fatal("esc outside edit mode must go back to the hub")
	}
}

func TestReleaseEditorKeepsDeployedStagesOfAReleaseReadOnly(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, panels.OpenReleaseEditorMsg{Kind: models.DocRelease, ID: 64})
	doc, err := models.ParseReleaseDoc(models.DocRelease, []byte(editorReleaseJSON))
	if err != nil {
		t.Fatal(err)
	}
	a = feed(t, a, releaseDocLoadedMsg{seq: a.rl.editorSeq, doc: doc, refs: models.ReleaseDocRefs{}})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "Editing this release, not its pipeline") || !strings.Contains(out, "read-only") {
		t.Fatalf("a release must say it's edited on its own:\n%s", out)
	}
	a, msgs := pressAll(t, a, press("2"), press("e"), press(" "))
	if n, ok := find[panels.ReleaseNoticeMsg](msgs); !ok || !strings.Contains(n.Text, "already deployed") || a.rl.editor.Dirty() {
		t.Fatalf("a deployed stage must stay read-only, got %v", msgs)
	}
	a, _ = pressAll(t, a, press(">"), press("j"), press(" "))
	if len(a.rl.editor.Edits()) != 1 {
		t.Fatal("a stage that hasn't deployed can change")
	}
}

func TestReleaseScreenOpensTheReleaseEditor(t *testing.T) {
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, panels.OpenReleaseMsg{ReleaseID: 64}, releaseLoadedMsg{release: sampleRelease()})
	_, msgs := pressAll(t, a, press("e"))
	if open, ok := find[panels.OpenReleaseEditorMsg](msgs); !ok || open.Kind != models.DocRelease || open.ID != 64 {
		t.Fatalf("e on the release screen must edit the release, got %v", msgs)
	}
}

func TestReleaseEditorFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {120, 32}, {90, 26}, {70, 22}, {55, 18}, {50, 14}} {
		a := openEditor(t, sz[0], sz[1])
		for _, tab := range []string{"1", "2", "3", "4"} {
			a, _ = pressAll(t, a, press(tab))
			if tab == "4" {
				a = feed(t, a, definitionRevisionsMsg{id: 12, revs: []models.DefinitionRevision{{Revision: 42, ChangedBy: "Gopi R", ChangeType: "update", Comment: "Point at v2"}}})
			}
			assertFits(t, a.render(), sz[0], sz[1])
		}
		a, _ = pressAll(t, a, press("2"), press("j"), special(tea.KeyEnter), press("e"), press(" "))
		assertFits(t, a.render(), sz[0], sz[1])
		a, msgs := pressAll(t, a, ctrl('s'))
		a = feed(t, a, msgs...)
		assertFits(t, a.render(), sz[0], sz[1])
		a, _ = pressAll(t, a, special(tea.KeyEscape))
		for _, field := range []string{"Script Location", "Script Path"} {
			f := editorField(t, a, field)
			b := feed(t, a, panels.EditReleaseFieldMsg{Field: f, Where: "Dev1 / Log Analytics"})
			topModal[modals.ReleaseFieldModal](t, b)
			assertFits(t, b.render(), sz[0], sz[1])
		}
	}
}

func editorField(t *testing.T, a App, label string) models.TaskField {
	t.Helper()
	d := a.rl.editor.Doc()
	s := d.Stages[0]
	fields, _ := d.TaskFields(s, s.Phases[0], s.Phases[0].Tasks[1], editorRefs().Tasks[models.TaskRef{ID: "46e4be58-0000-0000-0000-000000000001", Major: 1}.Key()], nil)
	for _, f := range fields {
		if f.Label == label {
			return f
		}
	}
	t.Fatalf("no field %q", label)
	return models.TaskField{}
}

func TestReleaseEditorPicksAServiceConnectionByName(t *testing.T) {
	a := openEditor(t, 160, 45)
	a, _ = pressAll(t, a, press("2"), press("j"), press("j"), special(tea.KeyEnter), press("e"))
	f := editorField(t, a, "Azure subscription")
	a = feed(t, a, panels.EditReleaseFieldMsg{Field: f, Where: "Dev1 / Log Analytics"})
	if a.modals.open() || !a.loading {
		t.Fatal("the service connections must load before the picker opens")
	}
	staging := models.ServiceEndpoint{ID: "00000000-bbbb-0000-0000-000000000002", Name: "Staging subscription", Type: "azurerm"}
	a = feed(t, a, serviceEndpointsMsg{kind: "azurerm", endpoints: []models.ServiceEndpoint{staging}, field: f, where: "Dev1 / Log Analytics"})
	topModal[modals.ReleaseFieldModal](t, a)
	a = feed(t, a, modals.ReleaseFieldEditedMsg{Field: f, Where: "Dev1 / Log Analytics", Value: staging.ID})
	edits := a.rl.editor.Edits()
	if len(edits) != 1 || edits[0].BeforeText != "Prod subscription" || edits[0].AfterText != "Staging subscription" {
		t.Fatalf("the change must read by name, got %+v", edits)
	}
	if !strings.Contains(ansi.Strip(a.render()), "Staging subscription") {
		t.Fatal("the task must show the new connection by name")
	}
	a = feed(t, a, panels.EditReleaseFieldMsg{Field: f, Where: "Dev1 / Log Analytics"})
	topModal[modals.ReleaseFieldModal](t, a)
}

func TestReleaseEditorNeverTypesAServiceConnection(t *testing.T) {
	a := openEditor(t, 160, 45)
	f := editorField(t, a, "Azure subscription")
	a = feed(t, a, serviceEndpointsMsg{kind: "azurerm", field: f, where: "Dev1 / Log Analytics"})
	a, msgs := pressAll(t, a, press("x"), special(tea.KeyEnter))
	if _, ok := find[modals.ReleaseFieldEditedMsg](msgs); ok {
		t.Fatal("with no connections listed nothing may be typed")
	}
	if !strings.Contains(ansi.Strip(a.render()), "Nothing to choose from") {
		t.Fatal("the dialog must say there is nothing to choose")
	}
}

func TestReleaseEditorChangesTheAgentPoolAndSpecification(t *testing.T) {
	a := openEditor(t, 160, 45)
	a, _ = pressAll(t, a, press("2"), special(tea.KeyEnter), press("e"))
	out := ansi.Strip(a.render())
	for _, want := range []string{"AGENT JOB", "run on agent", "Agent pool *", "Azure Pipelines", "Agent specification *", "ubuntu-latest", "EXECUTION PLAN", "None", "ARTIFACTS", "ADDITIONAL OPTIONS", "succeeded()"} {
		if !strings.Contains(out, want) {
			t.Errorf("agent job settings miss %q", want)
		}
	}
	a, msgs := pressAll(t, a, press("j"), special(tea.KeyEnter))
	pool, ok := find[panels.EditReleaseFieldMsg](msgs)
	if !ok || pool.Field.Kind != models.InputQueue {
		t.Fatalf("enter on the pool must ask to change it, got %v", msgs)
	}
	a = feed(t, a, pool)
	if a.modals.open() || !a.loading {
		t.Fatal("the agent pools must load before the picker opens")
	}
	pools := []models.AgentQueue{{ID: 11, Name: "Azure Pipelines", Hosted: true}, {ID: 12, Name: "linux-agents"}}
	a = feed(t, a, agentQueuesMsg{queues: pools, field: pool.Field, where: pool.Where})
	topModal[modals.ReleaseFieldModal](t, a)
	a = feed(t, a, modals.ReleaseFieldEditedMsg{Field: pool.Field, Where: pool.Where, Value: json.Number("12")})
	edits := a.rl.editor.Edits()
	if len(edits) != 1 || edits[0].BeforeText != "Azure Pipelines" || edits[0].AfterText != "linux-agents" {
		t.Fatalf("the pool change must read by name, got %+v", edits)
	}
	if !strings.Contains(ansi.Strip(a.render()), "linux-agents") {
		t.Fatal("the job must show its new pool")
	}
	a, msgs = pressAll(t, a, press("j"), special(tea.KeyEnter))
	spec, ok := find[panels.EditReleaseFieldMsg](msgs)
	if !ok || spec.Field.Label != "Agent specification" || len(spec.Field.Options) != len(models.AgentImages) {
		t.Fatalf("the specification is picked from the hosted images, got %+v", spec.Field)
	}
}

const azureDefinitionJSON = `{"id":30,"name":"connection-infra","revision":5,"variables":{"ResourceGroup":{"value":"rg-1"}},
"environments":[{"id":1,"name":"testing","rank":1,"variables":{},"deployPhases":[{"name":"Agent job","phaseType":"agentBasedDeployment","deploymentInput":{"queueId":11},"workflowTasks":[
 {"taskId":"94a74903-0000-0000-0000-000000000003","version":"3.*","name":"Deploy ARM","enabled":true,"inputs":{
  "ConnectedServiceName":"00000000-aaaa-0000-0000-000000000001","subscriptionName":"sub-1","resourceGroupName":"rg-1","location":"eastus2"}}]}]}]}`

func armDefinition() *models.TaskDefinition {
	byConnection := func(name string, params map[string]string) *models.DataSource {
		return &models.DataSource{Name: name, EndpointInput: "ConnectedServiceName", Parameters: params}
	}
	sub := map[string]string{"subscriptionId": "$(subscriptionName)"}
	return &models.TaskDefinition{
		ID: "94a74903-0000-0000-0000-000000000003", Name: "AzureResourceManagerTemplateDeployment", FriendlyName: "ARM template deployment", Major: 3,
		Inputs: []models.TaskInput{
			{Name: "ConnectedServiceName", Label: "Azure Resource Manager connection", Type: "connectedService:AzureRM", Required: true},
			{Name: "subscriptionName", Label: "Subscription", Type: "pickList", Required: true, Source: byConnection("AzureSubscriptions", nil)},
			{Name: "resourceGroupName", Label: "Resource group", Type: "pickList", Required: true, Source: byConnection("AzureResourceGroups", sub)},
			{Name: "location", Label: "Location", Type: "pickList", Required: true, Source: byConnection("AzureLocations", sub)},
		},
	}
}

func openAzureEditor(t *testing.T) App {
	t.Helper()
	a := openReleasesHub(t, 160, 45)
	a = feed(t, a, panels.OpenReleaseEditorMsg{Kind: models.DocDefinition, ID: 30})
	doc, err := models.ParseReleaseDoc(models.DocDefinition, []byte(azureDefinitionJSON))
	if err != nil {
		t.Fatal(err)
	}
	def := armDefinition()
	refs := models.ReleaseDocRefs{
		Tasks:     map[string]*models.TaskDefinition{models.TaskRef{ID: def.ID, Major: 3}.Key(): def},
		Endpoints: map[string]string{"00000000-aaaa-0000-0000-000000000001": "DevOps (testing)"},
	}
	a = feed(t, a, releaseDocLoadedMsg{seq: a.rl.editorSeq, doc: doc, refs: refs})
	a, _ = pressAll(t, a, press("2"), press("j"), special(tea.KeyEnter), press("e"))
	return a
}

func TestReleaseEditorListsAzureValuesAndNeverTypesThem(t *testing.T) {
	a := openAzureEditor(t)
	a, msgs := pressAll(t, a, press("j"), press("j"), press("j"), special(tea.KeyEnter))
	rg, ok := find[panels.EditReleaseFieldMsg](msgs)
	if !ok || rg.Field.Kind != models.InputAzure || rg.Request.EndpointID != "00000000-aaaa-0000-0000-000000000001" || rg.Request.Source.Parameters["subscriptionId"] != "sub-1" {
		t.Fatalf("the resource group must be asked of Azure for this connection and subscription, got %+v", rg)
	}
	a = feed(t, a, rg)
	topModal[modals.ReleaseFieldModal](t, a)
	if !strings.Contains(ansi.Strip(a.render()), "Asking Azure DevOps") {
		t.Fatal("the list must show it is loading")
	}
	a = feed(t, a, azureListMsg{key: rg.Request.Key(), err: errors.New("AuthorizationFailed: no read access")})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "couldn't list resource group") || !strings.Contains(out, "AuthorizationFailed") {
		t.Fatalf("Azure's failure must show:\n%s", out)
	}
	a, msgs = pressAll(t, a, press("r"))
	if _, ok := find[modals.RetryAzureListMsg](msgs); !ok {
		t.Fatalf("r must ask again, got %v", msgs)
	}
	a = feed(t, a, azureListMsg{key: rg.Request.Key(), err: errors.New("AuthorizationFailed")})
	a, msgs = pressAll(t, a, press("v"), special(tea.KeyEnter))
	picked, ok := find[modals.ReleaseFieldEditedMsg](msgs)
	if !ok || picked.Value != "$(ResourceGroup)" {
		t.Fatalf("v must offer the pipeline's variables, got %v", msgs)
	}
	a = feed(t, a, modals.ReleaseFieldEditedMsg{Field: rg.Field, Where: rg.Where, Value: "rg-2"})
	a = feed(t, a, rg, azureListMsg{key: rg.Request.Key(), values: []models.AzureValue{{Value: "rg-1", Label: "rg-1"}, {Value: "rg-3", Label: "rg-3"}}})
	if !strings.Contains(ansi.Strip(a.render()), "rg-3") {
		t.Fatal("the loaded list must show")
	}
	a = feed(t, a, rg)
	if strings.Contains(ansi.Strip(a.render()), "Asking Azure DevOps") {
		t.Fatal("a list already loaded is shown at once")
	}
}

func TestReleaseEditorAsksAgainForSettingsThatDependOnAChange(t *testing.T) {
	a := openAzureEditor(t)
	a, msgs := pressAll(t, a, press("j"), special(tea.KeyEnter))
	conn, ok := find[panels.EditReleaseFieldMsg](msgs)
	if !ok || conn.Field.Kind != models.InputEndpoint {
		t.Fatalf("got %v", msgs)
	}
	a = feed(t, a, modals.ReleaseFieldEditedMsg{Field: conn.Field, Where: conn.Where, Value: "00000000-bbbb-0000-0000-000000000002"})
	if n := len(a.rl.editor.Stale()); n != 3 {
		t.Fatalf("subscription, resource group and location depend on the connection, got %d", n)
	}
	out := ansi.Strip(a.render())
	for _, want := range []string{"! 3 need a value from Azure", "listed for the old Azure Resource Manager connection: pick again", "!3"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	a, msgs = pressAll(t, a, ctrl('s'))
	a = feed(t, a, msgs...)
	out = ansi.Strip(a.render())
	if !strings.Contains(out, "Can't save yet") || !strings.Contains(out, "Resource group") {
		t.Fatalf("saving must wait:\n%s", out)
	}
	if _, msgs = pressAll(t, a, ctrl('s')); len(msgs) != 0 {
		t.Fatalf("Ctrl+s must not save while values are missing, got %v", msgs)
	}
	a, msgs = pressAll(t, a, special(tea.KeyEnter))
	a = feed(t, a, msgs...)
	if a.modals.open() || a.rl.editor.Tab() != panels.EditorTasksTab {
		t.Fatal("enter must jump to the first setting to pick")
	}
	sub := editorFieldOf(t, a, "Subscription")
	a = feed(t, a, modals.ReleaseFieldEditedMsg{Field: sub, Where: "testing / Deploy ARM", Value: "sub-2"})
	if n := len(a.rl.editor.Stale()); n != 2 {
		t.Fatalf("picking the subscription leaves resource group and location, got %d", n)
	}
	for _, label := range []string{"Resource group", "Location"} {
		f := editorFieldOf(t, a, label)
		a = feed(t, a, modals.ReleaseFieldEditedMsg{Field: f, Where: "testing / Deploy ARM", Value: label + "-2"})
	}
	if n := len(a.rl.editor.Stale()); n != 0 {
		t.Fatalf("everything is picked, got %d left", n)
	}
	a, msgs = pressAll(t, a, ctrl('s'))
	a = feed(t, a, msgs...)
	if strings.Contains(ansi.Strip(a.render()), "Can't save yet") {
		t.Fatal("saving is allowed once every value is picked")
	}
}

func editorFieldOf(t *testing.T, a App, label string) models.TaskField {
	t.Helper()
	d := a.rl.editor.Doc()
	s := d.Stages[0]
	p := s.Phases[0]
	fields, _ := d.TaskFields(s, p, p.Tasks[0], armDefinition(), a.rl.editor.Edits())
	for _, f := range fields {
		if f.Label == label {
			return f
		}
	}
	t.Fatalf("no field %q", label)
	return models.TaskField{}
}

func TestReleaseEditorAzureDialogsFitTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {120, 32}, {90, 26}, {70, 22}, {55, 18}, {50, 14}} {
		a := openAzureEditor(t)
		a = feed(t, a, tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		rg := editorFieldOf(t, a, "Resource group")
		req, _ := models.AzureRequestFor([]models.TaskField{editorFieldOf(t, a, "Azure Resource Manager connection"), editorFieldOf(t, a, "Subscription")}, rg)
		b := feed(t, a, panels.EditReleaseFieldMsg{Field: rg, Where: "testing / Deploy ARM", Request: req, Variables: []string{"ResourceGroup"}})
		b = feed(t, b, azureListMsg{key: req.Key(), err: errors.New("AuthorizationFailed: The client does not have authorization to perform action 'Microsoft.Resources/subscriptions/resourcegroups/read' over the scope of this subscription")})
		assertFits(t, b.render(), sz[0], sz[1])
		conn := editorFieldOf(t, a, "Azure Resource Manager connection")
		c := feed(t, a, modals.ReleaseFieldEditedMsg{Field: conn, Where: "testing / Deploy ARM", Value: "other"})
		assertFits(t, c.render(), sz[0], sz[1])
		c, msgs := pressAll(t, c, ctrl('s'))
		c = feed(t, c, msgs...)
		assertFits(t, c.render(), sz[0], sz[1])
	}
}
