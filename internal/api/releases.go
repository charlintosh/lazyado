package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/models"
)

const (
	releaseDefinitionsPage = 200
	releaseStageVersion    = apiVersion + "-preview.7"
	releaseArtifactType    = "releasedefinition"
	releaseExpand          = "environments,artifacts,approvals"
	releasableResults      = "succeeded,partiallySucceeded"
	artifactVersionsScan   = 100
)

var trailingNumber = regexp.MustCompile(`(\d+)\D*$`)

type releaseIdentity struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	IsContainer bool   `json:"isContainer"`
}

func (i releaseIdentity) identity() models.Identity {
	return models.Identity{ID: i.ID, DisplayName: i.DisplayName, UniqueName: i.UniqueName}
}

type flexID string

func (f *flexID) UnmarshalJSON(b []byte) error {
	*f = flexID(strings.Trim(string(b), `"`))
	return nil
}

func (f flexID) Int64() (int64, error) { return strconv.ParseInt(string(f), 10, 64) }

func (f flexID) String() string { return string(f) }

type namedRef struct {
	ID   flexID `json:"id"`
	Name string `json:"name"`
}

type approvalResponse struct {
	ID          int             `json:"id"`
	Type        string          `json:"approvalType"`
	Status      string          `json:"status"`
	Approver    releaseIdentity `json:"approver"`
	Rank        int             `json:"rank"`
	IsAutomated bool            `json:"isAutomated"`
	CreatedOn   time.Time       `json:"createdOn"`
	Release     namedRef        `json:"release"`
	Definition  namedRef        `json:"releaseDefinition"`
	Environment namedRef        `json:"releaseEnvironment"`
}

func (a approvalResponse) approval() models.ReleaseApproval {
	releaseID, _ := a.Release.ID.Int64()
	stageID, _ := a.Environment.ID.Int64()
	definitionID, _ := a.Definition.ID.Int64()
	return models.ReleaseApproval{
		ID: a.ID, Kind: a.Type, Status: a.Status, Approver: a.Approver.identity(), Group: a.Approver.IsContainer,
		Order: a.Rank, Automated: a.IsAutomated, Created: a.CreatedOn,
		ReleaseID: int(releaseID), ReleaseName: a.Release.Name, DefinitionID: int(definitionID), Definition: a.Definition.Name, StageID: int(stageID), StageName: a.Environment.Name,
	}
}

type releaseTask struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	StartTime  *time.Time `json:"startTime"`
	FinishTime *time.Time `json:"finishTime"`
	AgentName  string     `json:"agentName"`
	LogURL     string     `json:"logUrl"`
	Issues     []struct {
		IssueType string `json:"issueType"`
		Message   string `json:"message"`
	} `json:"issues"`
}

func logIDOf(url string) int {
	h := fnv.New32a()
	h.Write([]byte(url))
	return int(h.Sum32()&0x7fffffff) + 1
}

func (t releaseTask) record(kind string) models.TimelineRecord {
	r := models.TimelineRecord{ID: strconv.Itoa(t.ID), Type: kind, Name: t.Name, State: models.RunStateOf(t.Status, t.Status), Worker: t.AgentName, LogURL: t.LogURL}
	if t.StartTime != nil {
		r.Started = *t.StartTime
	}
	if t.FinishTime != nil {
		r.Finished = *t.FinishTime
	}
	if t.LogURL != "" {
		r.LogID = logIDOf(t.LogURL)
	}
	for _, is := range t.Issues {
		r.Issues = append(r.Issues, models.RunIssue{Error: strings.EqualFold(is.IssueType, "error"), Message: is.Message})
	}
	return r
}

type approvalSnapshot struct {
	Approvals []struct {
		IsAutomated bool            `json:"isAutomated"`
		Approver    releaseIdentity `json:"approver"`
	} `json:"approvals"`
}

func (s approvalSnapshot) names() []string {
	var out []string
	for _, a := range s.Approvals {
		if !a.IsAutomated && a.Approver.DisplayName != "" {
			out = append(out, a.Approver.DisplayName)
		}
	}
	return out
}

type condition struct {
	ConditionType string `json:"conditionType"`
	Name          string `json:"name"`
}

func triggerOf(conditions []condition) (string, bool) {
	for _, c := range conditions {
		switch strings.ToLower(c.ConditionType) {
		case "event":
			return "after release", true
		case "environmentstate":
			return "after " + c.Name, true
		}
	}
	return "manual", false
}

type environmentResponse struct {
	ID                  int                `json:"id"`
	Name                string             `json:"name"`
	Rank                int                `json:"rank"`
	Status              string             `json:"status"`
	PreDeployApprovals  []approvalResponse `json:"preDeployApprovals"`
	PostDeployApprovals []approvalResponse `json:"postDeployApprovals"`
	PreSnapshot         approvalSnapshot   `json:"preApprovalsSnapshot"`
	PostSnapshot        approvalSnapshot   `json:"postApprovalsSnapshot"`
	Conditions          []condition        `json:"conditions"`
	DeploySteps         []struct {
		Attempt      int             `json:"attempt"`
		RequestedFor releaseIdentity `json:"requestedFor"`
		QueuedOn     time.Time       `json:"queuedOn"`
		Phases       []struct {
			Name   string `json:"name"`
			Rank   int    `json:"rank"`
			Status string `json:"status"`
			Jobs   []struct {
				Job   releaseTask   `json:"job"`
				Tasks []releaseTask `json:"tasks"`
			} `json:"deploymentJobs"`
		} `json:"releaseDeployPhases"`
	} `json:"deploySteps"`
}

func (e environmentResponse) stage(release models.Release) models.ReleaseStage {
	var approvals []models.ReleaseApproval
	for _, a := range append(slices.Clone(e.PreDeployApprovals), e.PostDeployApprovals...) {
		ap := a.approval()
		ap.ReleaseID, ap.ReleaseName, ap.Definition, ap.StageID, ap.StageName = release.ID, release.Name, release.Definition, e.ID, e.Name
		approvals = append(approvals, ap)
	}
	trigger, _ := triggerOf(e.Conditions)
	s := models.ReleaseStage{
		ID: e.ID, Name: e.Name, Rank: e.Rank, Status: e.Status, Approvals: approvals,
		Approvers: e.PreSnapshot.names(), PostApproval: e.PostSnapshot.names(), Trigger: trigger,
		State: models.StageStateOf(e.Status, approvals),
	}
	if len(e.DeploySteps) == 0 {
		return s
	}
	step := e.DeploySteps[len(e.DeploySteps)-1]
	for _, d := range e.DeploySteps {
		if d.Attempt >= step.Attempt {
			step = d
		}
	}
	s.Attempt, s.Requested, s.Started = step.Attempt, step.RequestedFor.identity(), step.QueuedOn
	for _, p := range step.Phases {
		for _, j := range p.Jobs {
			job := models.RunJob{TimelineRecord: j.Job.record("Job"), Stage: e.Name}
			if job.Name == "" {
				job.Name = p.Name
			}
			for _, t := range j.Tasks {
				job.Steps = append(job.Steps, t.record("Task"))
			}
			if job.Finished.After(s.Finished) {
				s.Finished = job.Finished
			}
			s.Jobs = append(s.Jobs, job)
		}
	}
	return s
}

type artifactResponse struct {
	Alias     string `json:"alias"`
	Type      string `json:"type"`
	IsPrimary bool   `json:"isPrimary"`
	Reference struct {
		Version    namedRef `json:"version"`
		Branch     namedRef `json:"branch"`
		Definition namedRef `json:"definition"`
	} `json:"definitionReference"`
}

func (a artifactResponse) artifact() models.ReleaseArtifact {
	defID, _ := a.Reference.Definition.ID.Int64()
	return models.ReleaseArtifact{
		Alias: a.Alias, Type: a.Type, Primary: a.IsPrimary,
		Version: a.Reference.Version.Name, VersionID: a.Reference.Version.ID.String(), Branch: a.Reference.Branch.Name,
		DefinitionID: int(defID), Definition: a.Reference.Definition.Name, RepositoryID: a.Reference.Definition.ID.String(),
	}
}

type releaseResponse struct {
	ID           int                   `json:"id"`
	Name         string                `json:"name"`
	Status       string                `json:"status"`
	Description  string                `json:"description"`
	CreatedOn    time.Time             `json:"createdOn"`
	CreatedBy    releaseIdentity       `json:"createdBy"`
	Definition   namedRef              `json:"releaseDefinition"`
	Environments []environmentResponse `json:"environments"`
	Artifacts    []artifactResponse    `json:"artifacts"`
	Links        webLinks              `json:"_links"`
}

func (r releaseResponse) release() models.Release {
	defID, _ := r.Definition.ID.Int64()
	rel := models.Release{
		ID: r.ID, Name: r.Name, Status: r.Status, Description: r.Description, Created: r.CreatedOn, CreatedBy: r.CreatedBy.identity(),
		DefinitionID: int(defID), Definition: r.Definition.Name, WebURL: r.Links.Web.Href,
	}
	for _, a := range r.Artifacts {
		rel.Artifacts = append(rel.Artifacts, a.artifact())
	}
	envs := slices.Clone(r.Environments)
	slices.SortStableFunc(envs, func(a, b environmentResponse) int { return a.Rank - b.Rank })
	for _, e := range envs {
		rel.Stages = append(rel.Stages, e.stage(rel))
	}
	return rel
}

func (c *Client) releaseEndpoint(endpoint, version string) string {
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s%sapi-version=%s", c.releaseURL, endpoint, separator, version)
}

func (c *Client) releaseGet(endpoint string, target any) (string, error) {
	resp, err := c.doRequest("GET", c.releaseEndpoint(endpoint, apiVersion), nil)
	if err != nil {
		return "", err
	}
	token := resp.Header.Get(continuationHeader)
	return token, decode(resp, target)
}

func (c *Client) releaseSend(method, endpoint, version string, payload, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	resp, err := c.doRequest(method, c.releaseEndpoint(endpoint, version), bytes.NewReader(body))
	if err != nil {
		return err
	}
	if target == nil {
		return resp.Body.Close()
	}
	return decode(resp, target)
}

func (c *Client) ListReleaseDefinitions(search, continuation string) ([]models.ReleaseDefinition, string, error) {
	q := url.Values{}
	if search != "" {
		q.Set("searchText", search)
	}
	if continuation != "" {
		q.Set("continuationToken", continuation)
	}
	return c.releaseDefinitions(q)
}

func (c *Client) ReleaseDefinitionsByID(ids []int) ([]models.ReleaseDefinition, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := url.Values{}
	q.Set("definitionIdFilter", joinInts(ids))
	defs, _, err := c.releaseDefinitions(q)
	return defs, err
}

func (c *Client) releaseDefinitions(q url.Values) ([]models.ReleaseDefinition, string, error) {
	q.Set("$expand", "lastRelease")
	q.Set("queryOrder", "nameAscending")
	q.Set("$top", strconv.Itoa(releaseDefinitionsPage))
	var data struct {
		Value []struct {
			ID          int      `json:"id"`
			Name        string   `json:"name"`
			Path        string   `json:"path"`
			Links       webLinks `json:"_links"`
			LastRelease *struct {
				Name      string    `json:"name"`
				CreatedOn time.Time `json:"createdOn"`
			} `json:"lastRelease"`
		} `json:"value"`
	}
	token, err := c.releaseGet("/release/definitions?"+q.Encode(), &data)
	if err != nil {
		return nil, "", fmt.Errorf("listing release pipelines: %w", err)
	}
	defs := make([]models.ReleaseDefinition, len(data.Value))
	for i, d := range data.Value {
		defs[i] = models.ReleaseDefinition{ID: d.ID, Name: d.Name, Folder: d.Path, WebURL: d.Links.Web.Href}
		if d.LastRelease != nil {
			defs[i].LastRelease, defs[i].LastCreated = d.LastRelease.Name, d.LastRelease.CreatedOn
		}
	}
	return defs, token, nil
}

func (c *Client) ListReleases(definitionID, top int, continuation string) ([]models.Release, string, error) {
	q := url.Values{}
	q.Set("definitionId", strconv.Itoa(definitionID))
	q.Set("$expand", releaseExpand)
	q.Set("$top", strconv.Itoa(top))
	q.Set("queryOrder", "descending")
	if continuation != "" {
		q.Set("continuationToken", continuation)
	}
	var data struct {
		Value []releaseResponse `json:"value"`
	}
	token, err := c.releaseGet("/release/releases?"+q.Encode(), &data)
	if err != nil {
		return nil, "", fmt.Errorf("listing releases: %w", err)
	}
	releases := make([]models.Release, len(data.Value))
	for i, r := range data.Value {
		releases[i] = r.release()
	}
	return releases, token, nil
}

func (c *Client) GetRelease(id int) (models.Release, error) {
	var data releaseResponse
	if _, err := c.releaseGet(fmt.Sprintf("/release/releases/%d", id), &data); err != nil {
		return models.Release{}, fmt.Errorf("loading release %d: %w", id, err)
	}
	return data.release(), nil
}

func (c *Client) ListReleaseApprovals(assignedTo string) ([]models.ReleaseApproval, error) {
	q := url.Values{}
	q.Set("statusFilter", "pending")
	q.Set("typeFilter", "preDeploy,postDeploy")
	if assignedTo != "" {
		q.Set("assignedToFilter", assignedTo)
		q.Set("includeMyGroupApprovals", "true")
	}
	var data struct {
		Value []approvalResponse `json:"value"`
	}
	if _, err := c.releaseGet("/release/approvals?"+q.Encode(), &data); err != nil {
		return nil, fmt.Errorf("loading approvals: %w", err)
	}
	out := make([]models.ReleaseApproval, 0, len(data.Value))
	for _, a := range data.Value {
		if !a.IsAutomated {
			out = append(out, a.approval())
		}
	}
	return out, nil
}

func (c *Client) DecideApproval(approvalID int, approve bool, comment string) error {
	status := "rejected"
	if approve {
		status = "approved"
	}
	payload := map[string]string{"status": status, "comments": comment}
	if err := c.releaseSend("PATCH", fmt.Sprintf("/release/approvals/%d", approvalID), apiVersion, payload, nil); err != nil {
		return fmt.Errorf("sending the approval: %w", err)
	}
	return nil
}

func (c *Client) DeployReleaseStage(releaseID, stageID int, comment string) error {
	payload := map[string]any{"status": "inProgress", "comment": comment, "variables": map[string]any{}}
	if err := c.releaseSend("PATCH", fmt.Sprintf("/release/releases/%d/environments/%d", releaseID, stageID), releaseStageVersion, payload, nil); err != nil {
		return fmt.Errorf("deploying the stage: %w", err)
	}
	return nil
}

func (c *Client) CancelReleaseStage(releaseID, stageID int, comment string) error {
	payload := map[string]any{"status": "canceled", "comment": comment}
	if err := c.releaseSend("PATCH", fmt.Sprintf("/release/releases/%d/environments/%d", releaseID, stageID), releaseStageVersion, payload, nil); err != nil {
		return fmt.Errorf("canceling the deployment: %w", err)
	}
	return nil
}

func (c *Client) GetReleaseDefinition(id int) (models.ReleaseDefinitionSpec, error) {
	var data struct {
		ID        int                `json:"id"`
		Name      string             `json:"name"`
		Path      string             `json:"path"`
		Links     webLinks           `json:"_links"`
		Artifacts []artifactResponse `json:"artifacts"`
		Stages    []struct {
			Name       string      `json:"name"`
			Rank       int         `json:"rank"`
			Conditions []condition `json:"conditions"`
		} `json:"environments"`
		Variables map[string]struct {
			Value         string `json:"value"`
			AllowOverride bool   `json:"allowOverride"`
			IsSecret      bool   `json:"isSecret"`
		} `json:"variables"`
	}
	if _, err := c.releaseGet(fmt.Sprintf("/release/definitions/%d", id), &data); err != nil {
		return models.ReleaseDefinitionSpec{}, fmt.Errorf("loading release pipeline %d: %w", id, err)
	}
	spec := models.ReleaseDefinitionSpec{ReleaseDefinition: models.ReleaseDefinition{ID: data.ID, Name: data.Name, Folder: data.Path, WebURL: data.Links.Web.Href}}
	for _, a := range data.Artifacts {
		spec.Artifacts = append(spec.Artifacts, a.artifact())
	}
	stages := slices.Clone(data.Stages)
	slices.SortStableFunc(stages, func(a, b struct {
		Name       string      `json:"name"`
		Rank       int         `json:"rank"`
		Conditions []condition `json:"conditions"`
	}) int {
		return a.Rank - b.Rank
	})
	for _, s := range stages {
		trigger, automated := triggerOf(s.Conditions)
		spec.Stages = append(spec.Stages, models.ReleaseStageSpec{Name: s.Name, Automated: automated, Trigger: trigger})
	}
	for name, v := range data.Variables {
		if v.AllowOverride {
			spec.Variables = append(spec.Variables, models.ReleaseVariable{Name: name, Value: v.Value, Secret: v.IsSecret})
		}
	}
	slices.SortFunc(spec.Variables, func(a, b models.ReleaseVariable) int { return strings.Compare(a.Name, b.Name) })
	return spec, nil
}

func (c *Client) ListArtifactBuilds(buildDefinitionID int, ref string) ([]models.PipelineRun, error) {
	criteria := models.RunCriteria{PipelineID: buildDefinitionID, Ref: ref, Result: releasableResults}
	runs, _, err := c.ListPipelineRuns(criteria, artifactVersionsScan, "")
	return runs, err
}

func (c *Client) CreateRelease(rel models.NewRelease) (models.Release, error) {
	artifacts := make([]map[string]any, 0, len(rel.Artifacts))
	for _, a := range rel.Artifacts {
		instance := map[string]string{"id": a.VersionID, "name": a.Version}
		if a.Branch != "" {
			instance["sourceBranch"] = a.Branch
		}
		artifacts = append(artifacts, map[string]any{"alias": a.Alias, "instanceReference": instance})
	}
	payload := map[string]any{"definitionId": rel.DefinitionID, "description": rel.Description, "artifacts": artifacts, "isDraft": false}
	if len(rel.ManualStages) > 0 {
		payload["manualEnvironments"] = rel.ManualStages
	}
	if len(rel.Variables) > 0 {
		vars := make(map[string]map[string]string, len(rel.Variables))
		for k, v := range rel.Variables {
			vars[k] = map[string]string{"value": v}
		}
		payload["variables"] = vars
	}
	var data releaseResponse
	if err := c.releaseSend("POST", "/release/releases", apiVersion, payload, &data); err != nil {
		return models.Release{}, fmt.Errorf("creating the release: %w", err)
	}
	return data.release(), nil
}

func (c *Client) GetReleaseLog(logURL string, startLine int) ([]string, bool, error) {
	target := logURL
	if !strings.Contains(target, "api-version=") {
		separator := "?"
		if strings.Contains(target, "?") {
			separator = "&"
		}
		target += separator + "api-version=" + apiVersion + "-preview"
	}
	resp, err := c.doRequest("GET", target, nil)
	if err != nil {
		return nil, false, fmt.Errorf("loading log: %w", err)
	}
	defer resp.Body.Close()
	limited := &io.LimitedReader{R: resp.Body, N: c.maxFileBytes}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for n := 1; scanner.Scan(); n++ {
		if n >= startLine {
			lines = append(lines, CleanLogLine(scanner.Text()))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("reading log: %w", err)
	}
	return lines, limited.N <= 0, nil
}

func (c *Client) GetFavoriteReleaseDefinitionIDs() ([]int, error) {
	target := fmt.Sprintf("https://dev.azure.com/%s/_apis/favorite/favorites?api-version=%s-preview", url.PathEscape(c.organization), apiVersion)
	resp, err := c.doRequest("GET", target, nil)
	if err != nil {
		return nil, fmt.Errorf("loading favorites: %w", err)
	}
	var data struct {
		Value []struct {
			ArtifactID        string `json:"artifactId"`
			ArtifactType      string `json:"artifactType"`
			ArtifactIsDeleted bool   `json:"artifactIsDeleted"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading favorites: %w", err)
	}
	var ids []int
	for _, f := range data.Value {
		if f.ArtifactIsDeleted || !strings.Contains(strings.ToLower(f.ArtifactType), releaseArtifactType) {
			continue
		}
		if m := trailingNumber.FindStringSubmatch(f.ArtifactID); m != nil {
			if id, err := strconv.Atoi(m[1]); err == nil {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

func (c *Client) ReleaseDefinitionWebURL(id int) string {
	return fmt.Sprintf("%s/_release?definitionId=%d", c.webURL, id)
}

func (c *Client) ReleaseWebURL(id int) string {
	return fmt.Sprintf("%s/_releaseProgress?releaseId=%d", c.webURL, id)
}

const latestDeploymentsScan = 100

func (c *Client) LatestDeployments() (map[int]models.DefinitionDeployment, error) {
	q := url.Values{}
	q.Set("$top", strconv.Itoa(latestDeploymentsScan))
	q.Set("queryOrder", "descending")
	q.Set("latestAttemptsOnly", "true")
	var data struct {
		Value []struct {
			Release            namedRef  `json:"release"`
			ReleaseDefinition  namedRef  `json:"releaseDefinition"`
			ReleaseEnvironment namedRef  `json:"releaseEnvironment"`
			DeploymentStatus   string    `json:"deploymentStatus"`
			OperationStatus    string    `json:"operationStatus"`
			QueuedOn           time.Time `json:"queuedOn"`
			CompletedOn        time.Time `json:"completedOn"`
		} `json:"value"`
	}
	if _, err := c.releaseGet("/release/deployments?"+q.Encode(), &data); err != nil {
		return nil, fmt.Errorf("loading recent deployments: %w", err)
	}
	latest := map[int]models.DefinitionDeployment{}
	for _, d := range data.Value {
		id, err := d.ReleaseDefinition.ID.Int64()
		if err != nil || latest[int(id)].Release != "" {
			continue
		}
		when := d.CompletedOn
		if when.IsZero() {
			when = d.QueuedOn
		}
		latest[int(id)] = models.DefinitionDeployment{
			DefinitionID: int(id), Release: d.Release.Name, Stage: d.ReleaseEnvironment.Name,
			State: models.DeploymentState(d.DeploymentStatus, d.OperationStatus), When: when,
		}
	}
	return latest, nil
}
