package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/models"
)

const (
	maxReleaseDocBytes = 16 << 20
	maxEndpointLookups = 100
)

var markdownLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)

type flexText string

func (f *flexText) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		*f = flexText(v)
	case nil:
		*f = ""
	default:
		*f = flexText(fmt.Sprint(v))
	}
	return nil
}

type orderedOptions []models.TaskOption

func (o *orderedOptions) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil
	}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return err
		}
		var label flexText
		if err := dec.Decode(&label); err != nil {
			return err
		}
		key, _ := k.(string)
		*o = append(*o, models.TaskOption{Value: key, Label: string(label)})
	}
	return nil
}

type taskDefinitionResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	FriendlyName string `json:"friendlyName"`
	Version      struct {
		Major  int  `json:"major"`
		Minor  int  `json:"minor"`
		Patch  int  `json:"patch"`
		IsTest bool `json:"isTest"`
	} `json:"version"`
	Groups []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"groups"`
	Inputs []struct {
		Name         string              `json:"name"`
		Label        string              `json:"label"`
		Type         string              `json:"type"`
		DefaultValue flexText            `json:"defaultValue"`
		Required     bool                `json:"required"`
		HelpMarkDown string              `json:"helpMarkDown"`
		VisibleRule  string              `json:"visibleRule"`
		GroupName    string              `json:"groupName"`
		Options      orderedOptions      `json:"options"`
		Properties   map[string]flexText `json:"properties"`
	} `json:"inputs"`
	DataSourceBindings []struct {
		Target         string            `json:"target"`
		EndpointID     string            `json:"endpointId"`
		EndpointURL    string            `json:"endpointUrl"`
		DataSourceName string            `json:"dataSourceName"`
		Parameters     map[string]string `json:"parameters"`
		ResultSelector string            `json:"resultSelector"`
		ResultTemplate string            `json:"resultTemplate"`
	} `json:"dataSourceBindings"`
}

var endpointRef = regexp.MustCompile(`^\$\(([A-Za-z0-9_.\-]+)\)$`)

func (t taskDefinitionResponse) definition() *models.TaskDefinition {
	def := &models.TaskDefinition{
		ID: t.ID, Name: t.Name, FriendlyName: t.FriendlyName, Major: t.Version.Major,
		Version: fmt.Sprintf("%d.%d.%d", t.Version.Major, t.Version.Minor, t.Version.Patch),
	}
	for _, g := range t.Groups {
		def.Groups = append(def.Groups, models.TaskInputGroup{Name: g.Name, Label: g.DisplayName})
	}
	sources := map[string]*models.DataSource{}
	for _, b := range t.DataSourceBindings {
		src := &models.DataSource{
			Name: b.DataSourceName, EndpointURL: b.EndpointURL, Parameters: b.Parameters,
			ResultSelector: b.ResultSelector, ResultTemplate: b.ResultTemplate,
		}
		if m := endpointRef.FindStringSubmatch(strings.TrimSpace(b.EndpointID)); m != nil {
			src.EndpointInput = m[1]
		}
		for target := range strings.SplitSeq(b.Target, ",") {
			sources[strings.ToLower(strings.TrimSpace(target))] = src
		}
	}
	for _, in := range t.Inputs {
		def.Inputs = append(def.Inputs, models.TaskInput{
			Source: sources[strings.ToLower(in.Name)],
			Name:   in.Name, Label: in.Label, Type: in.Type, Default: string(in.DefaultValue), Required: in.Required,
			Help: strings.TrimSpace(markdownLink.ReplaceAllString(in.HelpMarkDown, "$1")), VisibleRule: in.VisibleRule,
			Group: in.GroupName, Options: in.Options, Editable: strings.EqualFold(string(in.Properties["EditableOptions"]), "true"),
		})
	}
	return def
}

func newerTask(a, b taskDefinitionResponse) bool {
	if a.Version.IsTest != b.Version.IsTest {
		return !a.Version.IsTest
	}
	if a.Version.Major != b.Version.Major {
		return a.Version.Major > b.Version.Major
	}
	if a.Version.Minor != b.Version.Minor {
		return a.Version.Minor > b.Version.Minor
	}
	return a.Version.Patch > b.Version.Patch
}

func pickTaskVersion(versions []taskDefinitionResponse, major int) (taskDefinitionResponse, bool) {
	var best taskDefinitionResponse
	found := false
	for _, v := range versions {
		if v.Version.Major == major && (!found || newerTask(v, best)) {
			best, found = v, true
		}
	}
	if found {
		return best, true
	}
	for _, v := range versions {
		if !found || newerTask(v, best) {
			best, found = v, true
		}
	}
	return best, found
}

func (c *Client) readAll(method, target string, body io.Reader) ([]byte, error) {
	resp, err := c.doRequest(method, target, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseDocBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if len(data) > maxReleaseDocBytes {
		return nil, fmt.Errorf("the response is larger than %d MB", maxReleaseDocBytes>>20)
	}
	return data, nil
}

func releaseDocEndpoint(kind models.ReleaseDocKind, id int) string {
	if kind == models.DocRelease {
		return fmt.Sprintf("/release/releases/%d", id)
	}
	return fmt.Sprintf("/release/definitions/%d", id)
}

func releaseDocNoun(kind models.ReleaseDocKind) string {
	if kind == models.DocRelease {
		return "release"
	}
	return "release pipeline"
}

func (c *Client) GetReleaseDoc(kind models.ReleaseDocKind, id int) (*models.ReleaseDoc, error) {
	body, err := c.readAll(http.MethodGet, c.releaseEndpoint(releaseDocEndpoint(kind, id), apiVersion), nil)
	if err != nil {
		return nil, fmt.Errorf("loading %s %d: %w", releaseDocNoun(kind), id, err)
	}
	return models.ParseReleaseDoc(kind, body)
}

func (c *Client) SaveReleaseDoc(doc *models.ReleaseDoc, edits models.DocEdits, comment string) (*models.ReleaseDoc, error) {
	payload, err := doc.Payload(edits, comment)
	if err != nil {
		return nil, err
	}
	endpoint := "/release/definitions"
	if doc.Kind == models.DocRelease {
		endpoint = fmt.Sprintf("/release/releases/%d", doc.ID)
	}
	body, err := c.readAll(http.MethodPut, c.releaseEndpoint(endpoint, apiVersion), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("saving the %s: %w", releaseDocNoun(doc.Kind), err)
	}
	return models.ParseReleaseDoc(doc.Kind, body)
}

func (c *Client) GetTaskDefinition(ref models.TaskRef) (*models.TaskDefinition, error) {
	target := fmt.Sprintf("https://dev.azure.com/%s/_apis/distributedtask/tasks/%s?api-version=%s-preview",
		url.PathEscape(c.organization), url.PathEscape(ref.ID), apiVersion)
	resp, err := c.doRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("loading task %s: %w", ref.ID, err)
	}
	var data struct {
		Value []taskDefinitionResponse `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading task %s: %w", ref.ID, err)
	}
	v, ok := pickTaskVersion(data.Value, ref.Major)
	if !ok {
		return nil, fmt.Errorf("task %s is not installed", ref.ID)
	}
	return v.definition(), nil
}

func joinInts(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

type endpointResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (c *Client) ServiceEndpointNames(ids []string) (map[string]string, error) {
	if len(ids) > maxEndpointLookups {
		ids = ids[:maxEndpointLookups]
	}
	names := map[string]string{}
	if len(ids) == 0 {
		return names, nil
	}
	resp, err := c.getPreview("/serviceendpoint/endpoints?endpointIds=" + url.QueryEscape(strings.Join(ids, ",")))
	if err != nil {
		return names, fmt.Errorf("loading service connections: %w", err)
	}
	var data struct {
		Value []endpointResponse `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return names, fmt.Errorf("loading service connections: %w", err)
	}
	for _, e := range data.Value {
		names[strings.ToLower(e.ID)] = e.Name
	}
	return names, nil
}

func (c *Client) ListServiceEndpoints(kind string) ([]models.ServiceEndpoint, error) {
	endpoint := "/serviceendpoint/endpoints"
	if kind != "" {
		endpoint += "?type=" + url.QueryEscape(kind)
	}
	resp, err := c.getPreview(endpoint)
	if err != nil {
		return nil, fmt.Errorf("listing service connections: %w", err)
	}
	var data struct {
		Value []endpointResponse `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("listing service connections: %w", err)
	}
	out := make([]models.ServiceEndpoint, len(data.Value))
	for i, e := range data.Value {
		out[i] = models.ServiceEndpoint{ID: e.ID, Name: e.Name, Type: e.Type}
	}
	slices.SortFunc(out, func(a, b models.ServiceEndpoint) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

type queueResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Pool struct {
		IsHosted bool `json:"isHosted"`
	} `json:"pool"`
}

func (c *Client) agentQueues(endpoint string) ([]models.AgentQueue, error) {
	resp, err := c.getPreview(endpoint)
	if err != nil {
		return nil, fmt.Errorf("loading agent pools: %w", err)
	}
	var data struct {
		Value []queueResponse `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading agent pools: %w", err)
	}
	out := make([]models.AgentQueue, len(data.Value))
	for i, q := range data.Value {
		out[i] = models.AgentQueue{ID: q.ID, Name: q.Name, Hosted: q.Pool.IsHosted}
	}
	slices.SortFunc(out, func(a, b models.AgentQueue) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

func (c *Client) AgentQueues(ids []int) (map[int]models.AgentQueue, error) {
	queues := map[int]models.AgentQueue{}
	if len(ids) == 0 {
		return queues, nil
	}
	list, err := c.agentQueues("/distributedtask/queues?queueIds=" + joinInts(ids))
	for _, q := range list {
		queues[q.ID] = q
	}
	return queues, err
}

func (c *Client) ListAgentQueues() ([]models.AgentQueue, error) {
	return c.agentQueues("/distributedtask/queues")
}

func (c *Client) AzureList(req models.AzureListRequest) ([]models.AzureValue, error) {
	src := req.Source
	payload := map[string]any{
		"dataSourceDetails": map[string]any{
			"dataSourceName": src.Name, "dataSourceUrl": src.EndpointURL, "resultSelector": src.ResultSelector,
			"parameters": src.Parameters,
		},
		"resultTransformationDetails": map[string]any{"resultTemplate": src.ResultTemplate},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	target := versioned(c.baseURL, "/serviceendpoint/endpointproxy?endpointId="+url.QueryEscape(req.EndpointID), apiVersion+"-preview.1")
	resp, err := c.doRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("asking Azure through the service connection: %w", err)
	}
	var data struct {
		Result       []json.RawMessage `json:"result"`
		StatusCode   string            `json:"statusCode"`
		ErrorMessage string            `json:"errorMessage"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("asking Azure through the service connection: %w", err)
	}
	if data.ErrorMessage != "" || (data.StatusCode != "" && !strings.EqualFold(data.StatusCode, "ok")) {
		msg := data.ErrorMessage
		if msg == "" {
			msg = "Azure answered " + data.StatusCode
		}
		return nil, errors.New(msg)
	}
	return azureValues(data.Result), nil
}

func azureValues(raw []json.RawMessage) []models.AzureValue {
	var out []models.AzureValue
	seen := map[string]bool{}
	for _, item := range raw {
		v := azureValue(item)
		if v.Value == "" || seen[strings.ToLower(v.Value)] {
			continue
		}
		seen[strings.ToLower(v.Value)] = true
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b models.AzureValue) int {
		return strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})
	return out
}

func azureValue(item json.RawMessage) models.AzureValue {
	var text string
	if err := json.Unmarshal(item, &text); err == nil {
		item = json.RawMessage(text)
	}
	var obj map[string]any
	if err := json.Unmarshal(item, &obj); err != nil {
		return models.AzureValue{Value: text, Label: text}
	}
	v := models.AzureValue{}
	for k, val := range obj {
		switch strings.ToLower(k) {
		case "value":
			v.Value = fmt.Sprint(val)
		case "displayvalue":
			v.Label = fmt.Sprint(val)
		}
	}
	if v.Label == "" {
		v.Label = v.Value
	}
	return v
}

func (c *Client) VariableGroupInfo(ids []int) (map[int]models.VariableGroupInfo, error) {
	groups := map[int]models.VariableGroupInfo{}
	if len(ids) == 0 {
		return groups, nil
	}
	resp, err := c.getPreview("/distributedtask/variablegroups?groupIds=" + joinInts(ids))
	if err != nil {
		return groups, fmt.Errorf("loading variable groups: %w", err)
	}
	var data struct {
		Value []struct {
			ID        int                        `json:"id"`
			Name      string                     `json:"name"`
			Variables map[string]json.RawMessage `json:"variables"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return groups, fmt.Errorf("loading variable groups: %w", err)
	}
	for _, g := range data.Value {
		groups[g.ID] = models.VariableGroupInfo{Name: g.Name, Variables: len(g.Variables)}
	}
	return groups, nil
}

func (c *Client) ListDefinitionRevisions(id int) ([]models.DefinitionRevision, error) {
	var data struct {
		Value []struct {
			Revision    int             `json:"revision"`
			ChangedDate time.Time       `json:"changedDate"`
			ChangedBy   releaseIdentity `json:"changedBy"`
			ChangeType  string          `json:"changeType"`
			Comment     string          `json:"comment"`
		} `json:"value"`
	}
	if _, err := c.releaseGet(fmt.Sprintf("/release/definitions/%d/revisions", id), &data); err != nil {
		return nil, fmt.Errorf("loading the history of release pipeline %d: %w", id, err)
	}
	out := make([]models.DefinitionRevision, len(data.Value))
	for i, r := range data.Value {
		out[i] = models.DefinitionRevision{Revision: r.Revision, Changed: r.ChangedDate, ChangedBy: r.ChangedBy.DisplayName, ChangeType: r.ChangeType, Comment: r.Comment}
	}
	slices.SortFunc(out, func(a, b models.DefinitionRevision) int { return b.Revision - a.Revision })
	return out, nil
}
