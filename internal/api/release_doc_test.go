package api

import (
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
)

const definitionDocJSON = `{"id":7,"name":"Update infra","revision":42,"unknownSetting":{"nested":[1,2]},
"variables":{"LogDays":{"value":"30"}},
"environments":[{"id":1,"name":"test","rank":1,"variables":{},"deployPhases":[{"name":"Agent job","workflowTasks":[
{"taskId":"46e4be58-0000-0000-0000-000000000001","version":"1.*","name":"Deploy","enabled":true,"inputs":{"scriptPath":"a.sh"}}]}]}]}`

func TestGetAndSaveReleaseDefinitionRoundTripsUnknownFields(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/release/definitions": definitionDocJSON}, &seen)
	doc, err := c.GetReleaseDoc(models.DocDefinition, 7)
	if err != nil {
		t.Fatal(err)
	}
	if seen[0].method != "GET" || !strings.HasPrefix(seen[0].url, "https://vsrm.dev.azure.com/contoso/web/_apis/release/definitions/7?api-version=7.1") {
		t.Fatalf("asked %s %s", seen[0].method, seen[0].url)
	}
	s := doc.Stages[0]
	task := s.Phases[0].Tasks[0]
	edit := models.DocEdit{Path: append(doc.TaskPath(s, s.Phases[0], task), "inputs", "scriptPath"), Before: "a.sh", After: "b.sh"}
	saved, err := c.SaveReleaseDoc(doc, models.DocEdits{edit}, "Point at b")
	if err != nil || saved.Revision != 42 {
		t.Fatalf("saved %+v, %v", saved, err)
	}
	put := seen[1]
	if put.method != "PUT" || !strings.HasPrefix(put.url, "https://vsrm.dev.azure.com/contoso/web/_apis/release/definitions?api-version=7.1") {
		t.Fatalf("saved with %s %s", put.method, put.url)
	}
	for _, want := range []string{`"scriptPath":"b.sh"`, `"comment":"Point at b"`, `"unknownSetting":{"nested":[1,2]}`, `"revision":42`} {
		if !strings.Contains(put.body, want) {
			t.Fatalf("body misses %s: %s", want, put.body)
		}
	}
}

func TestSaveReleaseUsesTheReleaseEndpoint(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/release/releases/118": `{"id":118,"name":"Release-118","environments":[]}`}, &seen)
	doc, err := c.GetReleaseDoc(models.DocRelease, 118)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.SaveReleaseDoc(doc, nil, ""); err != nil {
		t.Fatal(err)
	}
	if seen[1].method != "PUT" || !strings.Contains(seen[1].url, "/_apis/release/releases/118?api-version=7.1") {
		t.Fatalf("saved with %s %s", seen[1].method, seen[1].url)
	}
}

func TestGetTaskDefinitionPicksTheMajorVersionAndKeepsOptionOrder(t *testing.T) {
	body := `{"count":3,"value":[
{"id":"t1","name":"AzureCLI","friendlyName":"Azure CLI","version":{"major":2,"minor":9,"patch":0},"inputs":[]},
{"id":"t1","name":"AzureCLI","friendlyName":"Azure CLI","version":{"major":1,"minor":200,"patch":0},
 "groups":[{"name":"advanced","displayName":"Advanced"}],
 "inputs":[{"name":"scriptLocation","label":"Script Location","type":"pickList","defaultValue":"scriptPath",
  "options":{"zeta":"Last alphabetically","alpha":"First alphabetically"},"properties":{"EditableOptions":"True"},
  "helpMarkDown":"Pick one. [Learn more](https://example.com/docs)","visibleRule":"a = b","groupName":"advanced"},
  {"name":"timeout","type":"int","defaultValue":5}]},
{"id":"t1","name":"AzureCLI","version":{"major":1,"minor":100,"patch":0},"inputs":[]}]}`
	var seen []recorded
	c := clientRouting(t, map[string]string{"/distributedtask/tasks/": body}, &seen)
	def, err := c.GetTaskDefinition(models.TaskRef{ID: "t1", Major: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(seen[0].url, "https://dev.azure.com/contoso/_apis/distributedtask/tasks/t1?api-version=7.1-preview") {
		t.Fatalf("asked %s", seen[0].url)
	}
	in := def.Inputs[0]
	if def.Version != "1.200.0" || def.Groups[0].Label != "Advanced" || in.Options[0].Value != "zeta" || in.Options[1].Label != "First alphabetically" {
		t.Fatalf("definition %+v", def)
	}
	if !in.Editable || in.Kind() != models.InputText || in.Help != "Pick one. Learn more" || in.Group != "advanced" || def.Inputs[1].Default != "5" {
		t.Fatalf("input %+v %+v", in, def.Inputs[1])
	}
}

func TestLookupsAskForTheIDsTheyNeed(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/serviceendpoint/endpoints":      `{"value":[{"id":"AAAA-1","name":"Prod subscription","type":"azurerm"}]}`,
		"/distributedtask/queues":         `{"value":[{"id":11,"name":"Azure Pipelines","pool":{"isHosted":true}}]}`,
		"/distributedtask/variablegroups": `{"value":[{"id":3,"name":"shared","variables":{"a":{},"b":{}}}]}`,
	}, &seen)
	names, err := c.ServiceEndpointNames([]string{"AAAA-1", "bbbb-2"})
	if err != nil || names["aaaa-1"] != "Prod subscription" {
		t.Fatalf("endpoints %v %v", names, err)
	}
	queues, _ := c.AgentQueues([]int{11, 12})
	groups, _ := c.VariableGroupInfo([]int{3})
	if queues[11].Name != "Azure Pipelines" || !queues[11].Hosted || groups[3].Variables != 2 {
		t.Fatalf("queues %v groups %v", queues, groups)
	}
	if !strings.Contains(seen[0].url, "endpointIds=AAAA-1%2Cbbbb-2") || !strings.Contains(seen[1].url, "queueIds=11,12") || !strings.Contains(seen[2].url, "groupIds=3") {
		t.Fatalf("asked %v", seen)
	}
	if none, err := c.AgentQueues(nil); err != nil || len(none) != 0 || len(seen) != 3 {
		t.Fatal("no IDs must not send a request")
	}
}

func TestListDefinitionRevisionsNewestFirst(t *testing.T) {
	var seen []recorded
	body := `{"value":[{"revision":1,"changeType":"add","changedBy":{"displayName":"Ana Example"},"changedDate":"2026-11-01T10:00:00Z"},` +
		`{"revision":2,"changeType":"update","comment":"Use v2","changedDate":"2026-11-02T10:00:00Z"}]}`
	c := clientRouting(t, map[string]string{"/release/definitions/7/revisions": body}, &seen)
	revs, err := c.ListDefinitionRevisions(7)
	if err != nil || len(revs) != 2 || revs[0].Revision != 2 || revs[0].Comment != "Use v2" || revs[1].ChangedBy != "Ana Example" {
		t.Fatalf("revisions %+v %v", revs, err)
	}
}

func TestTaskDefinitionReadsDataSourceBindings(t *testing.T) {
	body := `{"value":[{"id":"t2","name":"AzureResourceManagerTemplateDeployment","version":{"major":3,"minor":1,"patch":0},
"inputs":[{"name":"ConnectedServiceName","label":"Azure Resource Manager connection","type":"connectedService:AzureRM"},
 {"name":"subscriptionName","label":"Subscription","type":"pickList","properties":{"EditableOptions":"True"}},
 {"name":"resourceGroupName","label":"Resource group","type":"pickList","properties":{"EditableOptions":"True"}},
 {"name":"location","label":"Location","type":"pickList"}],
"dataSourceBindings":[
 {"target":"subscriptionName","endpointId":"$(ConnectedServiceName)","dataSourceName":"AzureSubscriptions","resultTemplate":"{\"Value\":\"{{{subscriptionId}}}\"}"},
 {"target":"resourceGroupName","endpointId":"$(ConnectedServiceName)","dataSourceName":"AzureResourceGroupsByExplicitSubscription","parameters":{"subscriptionId":"$(subscriptionName)"}},
 {"target":"location","endpointId":"$(ConnectedServiceName)","dataSourceName":"AzureLocationsByExplicitSubscription","parameters":{"subscriptionId":"$(subscriptionName)"}}]}]}`
	var seen []recorded
	def, err := clientRouting(t, map[string]string{"/distributedtask/tasks/": body}, &seen).GetTaskDefinition(models.TaskRef{ID: "t2", Major: 3})
	if err != nil {
		t.Fatal(err)
	}
	rg := def.Inputs[2]
	if rg.Kind() != models.InputAzure || rg.Source.Name != "AzureResourceGroupsByExplicitSubscription" || rg.Source.EndpointInput != "ConnectedServiceName" {
		t.Fatalf("resource group %+v %+v", rg, rg.Source)
	}
	if got := rg.Source.Inputs(); len(got) != 2 || got[0] != "connectedservicename" || got[1] != "subscriptionname" {
		t.Fatalf("depends on %v", got)
	}
	if def.Inputs[0].Kind() != models.InputEndpoint || def.Inputs[1].Kind() != models.InputAzure || def.Inputs[3].Kind() != models.InputAzure {
		t.Fatal("bound inputs must be listed by Azure, never typed")
	}
}

func TestAzureListAsksTheProxyAndReadsValues(t *testing.T) {
	var seen []recorded
	reply := `{"result":["{\"Value\":\"sub-2\",\"DisplayValue\":\"Prod (sub-2)\"}","{\"Value\":\"sub-1\",\"DisplayValue\":\"Dev (sub-1)\"}","plain"],"statusCode":"ok"}`
	c := clientRouting(t, map[string]string{"/serviceendpoint/endpointproxy": reply}, &seen)
	req := models.AzureListRequest{EndpointID: "conn-1", Source: models.DataSource{Name: "AzureSubscriptions", Parameters: map[string]string{"a": "b"}, ResultTemplate: "tpl"}}
	values, err := c.AzureList(req)
	if err != nil {
		t.Fatal(err)
	}
	if seen[0].method != "POST" || !strings.Contains(seen[0].url, "/_apis/serviceendpoint/endpointproxy?endpointId=conn-1&api-version=7.1-preview.1") {
		t.Fatalf("asked %s %s", seen[0].method, seen[0].url)
	}
	for _, want := range []string{`"dataSourceName":"AzureSubscriptions"`, `"parameters":{"a":"b"}`, `"resultTemplate":"tpl"`} {
		if !strings.Contains(seen[0].body, want) {
			t.Fatalf("body misses %s: %s", want, seen[0].body)
		}
	}
	if len(values) != 3 || values[0].Label != "Dev (sub-1)" || values[0].Value != "sub-1" || values[1].Value != "plain" {
		t.Fatalf("values %+v", values)
	}
	failing := clientRouting(t, map[string]string{"/serviceendpoint/endpointproxy": `{"result":[],"statusCode":"forbidden","errorMessage":"AuthorizationFailed"}`}, &seen)
	if _, err := failing.AzureList(req); err == nil || err.Error() != "AuthorizationFailed" {
		t.Fatalf("Azure's error must come through as it is, got %v", err)
	}
}
