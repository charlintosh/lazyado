package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

const (
	continuationHeader = "x-ms-continuationtoken"
	recentRunsScan     = 100
	pipelinesPageSize  = 200
	refsPageSize       = 1000
	tagRefPrefix       = "refs/tags/"
	systemDebug        = "system.debug"
)

var logTimestamp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z ?`)

type buildResponse struct {
	ID            int               `json:"id"`
	BuildNumber   string            `json:"buildNumber"`
	Status        string            `json:"status"`
	Result        string            `json:"result"`
	Reason        string            `json:"reason"`
	SourceBranch  string            `json:"sourceBranch"`
	SourceVersion string            `json:"sourceVersion"`
	QueueTime     time.Time         `json:"queueTime"`
	StartTime     time.Time         `json:"startTime"`
	FinishTime    time.Time         `json:"finishTime"`
	TriggerInfo   map[string]string `json:"triggerInfo"`
	RequestedFor  identityRef       `json:"requestedFor"`
	Definition    struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"definition"`
	Repository struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"repository"`
	Queue struct {
		Name string `json:"name"`
	} `json:"queue"`
	Links webLinks `json:"_links"`
}

type webLinks struct {
	Web struct {
		Href string `json:"href"`
	} `json:"web"`
}

func (b buildResponse) run() models.PipelineRun {
	message := b.TriggerInfo["ci.message"]
	if message == "" {
		message = b.TriggerInfo["pr.title"]
	}
	return models.PipelineRun{
		ID:             b.ID,
		Number:         b.BuildNumber,
		PipelineID:     b.Definition.ID,
		PipelineName:   b.Definition.Name,
		PipelineFolder: b.Definition.Path,
		Status:         b.Status,
		Result:         b.Result,
		Reason:         b.Reason,
		Ref:            b.SourceBranch,
		Commit:         b.SourceVersion,
		Message:        message,
		RequestedFor:   models.Identity{ID: b.RequestedFor.ID, DisplayName: b.RequestedFor.DisplayName, UniqueName: b.RequestedFor.UniqueName},
		RepositoryID:   b.Repository.ID,
		RepositoryName: b.Repository.Name,
		Pool:           b.Queue.Name,
		Queued:         b.QueueTime,
		Started:        b.StartTime,
		Finished:       b.FinishTime,
		WebURL:         b.Links.Web.Href,
	}
}

func (c *Client) getPage(endpoint string, target any) (string, error) {
	resp, err := c.get(endpoint)
	if err != nil {
		return "", err
	}
	token := resp.Header.Get(continuationHeader)
	return token, decode(resp, target)
}

func runQuery(criteria models.RunCriteria, top int, continuation string) url.Values {
	q := url.Values{}
	q.Set("queryOrder", "queueTimeDescending")
	q.Set("$top", strconv.Itoa(top))
	if criteria.PipelineID != 0 {
		q.Set("definitions", strconv.Itoa(criteria.PipelineID))
	}
	if criteria.Ref != "" {
		q.Set("branchName", criteria.Ref)
	}
	if criteria.Status != "" {
		q.Set("statusFilter", criteria.Status)
	}
	if criteria.Result != "" {
		q.Set("resultFilter", criteria.Result)
	}
	if criteria.RequestedFor != "" {
		q.Set("requestedFor", criteria.RequestedFor)
	}
	if continuation != "" {
		q.Set("continuationToken", continuation)
	}
	return q
}

func (c *Client) ListPipelineRuns(criteria models.RunCriteria, top int, continuation string) ([]models.PipelineRun, string, error) {
	var data struct {
		Value []buildResponse `json:"value"`
	}
	token, err := c.getPage("/build/builds?"+runQuery(criteria, top, continuation).Encode(), &data)
	if err != nil {
		return nil, "", fmt.Errorf("listing pipeline runs: %w", err)
	}
	runs := make([]models.PipelineRun, len(data.Value))
	for i, b := range data.Value {
		runs[i] = b.run()
	}
	return runs, token, nil
}

func (c *Client) RecentPipelines(limit int) ([]models.Pipeline, error) {
	runs, _, err := c.ListPipelineRuns(models.RunCriteria{}, recentRunsScan, "")
	if err != nil {
		return nil, err
	}
	return models.LatestPerPipeline(runs, limit), nil
}

func (c *Client) ListPipelines(namePattern, continuation string) ([]models.Pipeline, string, error) {
	q := url.Values{}
	q.Set("includeLatestBuilds", "true")
	q.Set("queryOrder", "definitionNameAscending")
	q.Set("$top", strconv.Itoa(pipelinesPageSize))
	if namePattern != "" {
		q.Set("name", namePattern)
	}
	if continuation != "" {
		q.Set("continuationToken", continuation)
	}
	var data struct {
		Value []struct {
			ID          int            `json:"id"`
			Name        string         `json:"name"`
			Path        string         `json:"path"`
			Links       webLinks       `json:"_links"`
			LatestBuild *buildResponse `json:"latestBuild"`
		} `json:"value"`
	}
	token, err := c.getPage("/build/definitions?"+q.Encode(), &data)
	if err != nil {
		return nil, "", fmt.Errorf("listing pipelines: %w", err)
	}
	pipelines := make([]models.Pipeline, len(data.Value))
	for i, d := range data.Value {
		pipelines[i] = models.Pipeline{ID: d.ID, Name: d.Name, Folder: d.Path, WebURL: d.Links.Web.Href}
		if d.LatestBuild != nil {
			run := d.LatestBuild.run()
			run.PipelineID, run.PipelineName, run.PipelineFolder = d.ID, d.Name, d.Path
			pipelines[i].Latest = &run
		}
	}
	return pipelines, token, nil
}

func (c *Client) GetPipelineRun(id int) (models.PipelineRun, error) {
	resp, err := c.get(fmt.Sprintf("/build/builds/%d", id))
	if err != nil {
		return models.PipelineRun{}, fmt.Errorf("loading run %d: %w", id, err)
	}
	var b buildResponse
	if err := decode(resp, &b); err != nil {
		return models.PipelineRun{}, fmt.Errorf("loading run %d: %w", id, err)
	}
	return b.run(), nil
}

func (c *Client) GetRunTimeline(id int) ([]models.TimelineRecord, error) {
	resp, err := c.get(fmt.Sprintf("/build/builds/%d/timeline", id))
	if err != nil {
		return nil, fmt.Errorf("loading run %d stages: %w", id, err)
	}
	var data struct {
		Records []struct {
			ID         string     `json:"id"`
			ParentID   string     `json:"parentId"`
			Type       string     `json:"type"`
			Name       string     `json:"name"`
			Identifier string     `json:"identifier"`
			Order      int        `json:"order"`
			State      string     `json:"state"`
			Result     string     `json:"result"`
			StartTime  *time.Time `json:"startTime"`
			FinishTime *time.Time `json:"finishTime"`
			WorkerName string     `json:"workerName"`
			Log        *struct {
				ID int `json:"id"`
			} `json:"log"`
			Issues []struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"issues"`
		} `json:"records"`
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("loading run %d stages: %w", id, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("loading run %d stages: %w", id, err)
	}
	records := make([]models.TimelineRecord, 0, len(data.Records))
	for _, r := range data.Records {
		rec := models.TimelineRecord{
			ID: r.ID, ParentID: r.ParentID, Type: r.Type, Name: r.Name, Identifier: r.Identifier, Order: r.Order,
			State: models.RunStateOf(r.State, r.Result), Worker: r.WorkerName,
		}
		if r.StartTime != nil {
			rec.Started = *r.StartTime
		}
		if r.FinishTime != nil {
			rec.Finished = *r.FinishTime
		}
		if r.Log != nil {
			rec.LogID = r.Log.ID
		}
		for _, is := range r.Issues {
			rec.Issues = append(rec.Issues, models.RunIssue{Error: strings.EqualFold(is.Type, "error"), Message: is.Message})
		}
		records = append(records, rec)
	}
	return records, nil
}

func (c *Client) GetRunLog(id, logID, startLine int) ([]string, bool, error) {
	endpoint := fmt.Sprintf("/build/builds/%d/logs/%d", id, logID)
	if startLine > 1 {
		endpoint += "?startLine=" + strconv.Itoa(startLine)
	}
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, false, fmt.Errorf("loading log: %w", err)
	}
	defer resp.Body.Close()
	limited := &io.LimitedReader{R: resp.Body, N: c.maxFileBytes}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, CleanLogLine(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("reading log: %w", err)
	}
	return lines, limited.N <= 0, nil
}

func CleanLogLine(line string) string {
	line = strings.TrimSuffix(strings.TrimPrefix(line, "\ufeff"), "\r")
	return ansi.Strip(logTimestamp.ReplaceAllString(line, ""))
}

func (c *Client) GetRunChanges(id int) ([]models.RunChange, error) {
	resp, err := c.get(fmt.Sprintf("/build/builds/%d/changes?$top=20", id))
	if err != nil {
		return nil, fmt.Errorf("loading run %d changes: %w", id, err)
	}
	var data struct {
		Value []struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				DisplayName string `json:"displayName"`
			} `json:"author"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading run %d changes: %w", id, err)
	}
	changes := make([]models.RunChange, len(data.Value))
	for i, v := range data.Value {
		changes[i] = models.RunChange{ID: v.ID, Message: v.Message, Author: v.Author.DisplayName}
	}
	return changes, nil
}

func (c *Client) GetPipelineDefinition(id int) (models.PipelineDefinition, error) {
	resp, err := c.get(fmt.Sprintf("/build/definitions/%d", id))
	if err != nil {
		return models.PipelineDefinition{}, fmt.Errorf("loading pipeline %d: %w", id, err)
	}
	var data struct {
		ID         int      `json:"id"`
		Name       string   `json:"name"`
		Path       string   `json:"path"`
		Links      webLinks `json:"_links"`
		Repository struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Type          string `json:"type"`
			DefaultBranch string `json:"defaultBranch"`
		} `json:"repository"`
		Process struct {
			YAMLFilename string `json:"yamlFilename"`
		} `json:"process"`
		Variables map[string]struct {
			Value         string `json:"value"`
			AllowOverride bool   `json:"allowOverride"`
			IsSecret      bool   `json:"isSecret"`
		} `json:"variables"`
	}
	if err := decode(resp, &data); err != nil {
		return models.PipelineDefinition{}, fmt.Errorf("loading pipeline %d: %w", id, err)
	}
	def := models.PipelineDefinition{
		Pipeline:       models.Pipeline{ID: data.ID, Name: data.Name, Folder: data.Path, WebURL: data.Links.Web.Href},
		RepositoryID:   data.Repository.ID,
		RepositoryName: data.Repository.Name,
		RepositoryType: data.Repository.Type,
		DefaultBranch:  data.Repository.DefaultBranch,
		YAMLPath:       data.Process.YAMLFilename,
	}
	for name, v := range data.Variables {
		if v.AllowOverride {
			def.Variables = append(def.Variables, models.PipelineVariable{Name: name, Value: v.Value, Secret: v.IsSecret})
		}
	}
	slices.SortFunc(def.Variables, func(a, b models.PipelineVariable) int { return strings.Compare(a.Name, b.Name) })
	return def, nil
}

func (c *Client) ListRefs(repositoryID string) ([]models.GitRef, error) {
	var refs []models.GitRef
	for _, filter := range []string{"heads/", "tags/"} {
		resp, err := c.get(fmt.Sprintf("%s/refs?filter=%s&$top=%d", repositoryPath(repositoryID), filter, refsPageSize))
		if err != nil {
			return nil, fmt.Errorf("loading branches and tags: %w", err)
		}
		var data struct {
			Value []struct {
				Name     string `json:"name"`
				ObjectID string `json:"objectId"`
			} `json:"value"`
		}
		if err := decode(resp, &data); err != nil {
			return nil, fmt.Errorf("loading branches and tags: %w", err)
		}
		for _, r := range data.Value {
			refs = append(refs, models.GitRef{Name: r.Name, Commit: r.ObjectID})
		}
	}
	return refs, nil
}

func (c *Client) GetFileAtRef(repositoryID, path, ref string) (string, error) {
	q := url.Values{}
	q.Set("path", path)
	q.Set("versionDescriptor.versionType", "branch")
	q.Set("versionDescriptor.version", models.RefLabel(ref))
	if strings.HasPrefix(ref, tagRefPrefix) {
		q.Set("versionDescriptor.versionType", "tag")
	}
	q.Set("$format", "text")
	resp, err := c.get(repositoryPath(repositoryID) + "/items?" + q.Encode())
	if err != nil {
		return "", fmt.Errorf("loading %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxFileBytes))
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(body), nil
}

func (c *Client) postJSON(endpoint string, payload, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	resp, err := c.post(endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return decode(resp, target)
}

func runVariables(opts models.RunOptions) map[string]string {
	vars := make(map[string]string, len(opts.Variables)+1)
	for k, v := range opts.Variables {
		vars[k] = v
	}
	if opts.Diagnostics {
		vars[systemDebug] = "true"
	}
	return vars
}

func (c *Client) RunPipeline(def models.PipelineDefinition, opts models.RunOptions) (int, error) {
	var created struct {
		ID int `json:"id"`
	}
	vars := runVariables(opts)
	if !def.YAML() {
		payload := map[string]any{"definition": map[string]int{"id": def.ID}, "sourceBranch": opts.Ref}
		if opts.Commit != "" {
			payload["sourceVersion"] = opts.Commit
		}
		if len(vars) > 0 {
			encoded, err := json.Marshal(vars)
			if err != nil {
				return 0, fmt.Errorf("encoding variables: %w", err)
			}
			payload["parameters"] = string(encoded)
		}
		if err := c.postJSON("/build/builds", payload, &created); err != nil {
			return 0, fmt.Errorf("queueing %s: %w", def.Name, err)
		}
		return created.ID, nil
	}
	self := map[string]string{"refName": opts.Ref}
	if opts.Commit != "" {
		self["version"] = opts.Commit
	}
	payload := map[string]any{"resources": map[string]any{"repositories": map[string]any{"self": self}}}
	if len(opts.Parameters) > 0 {
		payload["templateParameters"] = opts.Parameters
	}
	if len(vars) > 0 {
		values := make(map[string]map[string]string, len(vars))
		for k, v := range vars {
			values[k] = map[string]string{"value": v}
		}
		payload["variables"] = values
	}
	if len(opts.StagesToSkip) > 0 {
		payload["stagesToSkip"] = opts.StagesToSkip
	}
	if err := c.postJSON(fmt.Sprintf("/pipelines/%d/runs", def.ID), payload, &created); err != nil {
		return 0, fmt.Errorf("running %s: %w", def.Name, err)
	}
	return created.ID, nil
}

func (c *Client) PipelineWebURL(id int) string {
	return fmt.Sprintf("%s/_build?definitionId=%d", c.webURL, id)
}

func (c *Client) RunWebURL(id int) string {
	return fmt.Sprintf("%s/_build/results?buildId=%d", c.webURL, id)
}
