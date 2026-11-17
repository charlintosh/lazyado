package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/charlintosh/lazyado/internal/models"
)

const (
	branchRefPrefix = "refs/heads/"
	branchPageSize  = 1000
	commitPreview   = 2
)

func templatePaths(target string) []string {
	return []string{
		"/.azuredevops/pull_request_template/branches/" + target + ".md",
		"/.azuredevops/pull_request_template.md",
		"/.vsts/pull_request_template.md",
		"/docs/pull_request_template.md",
		"/pull_request_template.md",
	}
}

func repositoryPath(repositoryID string) string {
	return "/git/repositories/" + url.PathEscape(repositoryID)
}

func (c *Client) ListBranches(repositoryID string) ([]string, error) {
	resp, err := c.get(fmt.Sprintf("%s/refs?filter=heads/&$top=%d", repositoryPath(repositoryID), branchPageSize))
	if err != nil {
		return nil, fmt.Errorf("loading branches: %w", err)
	}
	var data struct {
		Value []struct {
			Name string `json:"name"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading branches: %w", err)
	}
	branches := make([]string, 0, len(data.Value))
	for _, r := range data.Value {
		branches = append(branches, strings.TrimPrefix(r.Name, branchRefPrefix))
	}
	sort.Strings(branches)
	return branches, nil
}

func (c *Client) DefaultBranch(repositoryID string) (string, error) {
	resp, err := c.get(repositoryPath(repositoryID))
	if err != nil {
		return "", fmt.Errorf("loading repository: %w", err)
	}
	var data struct {
		DefaultBranch string `json:"defaultBranch"`
	}
	if err := decode(resp, &data); err != nil {
		return "", fmt.Errorf("loading repository: %w", err)
	}
	return strings.TrimPrefix(data.DefaultBranch, branchRefPrefix), nil
}

func (c *Client) CompareBranches(repositoryID, source, target string) (models.BranchComparison, error) {
	q := url.Values{}
	q.Set("baseVersion", target)
	q.Set("baseVersionType", "branch")
	q.Set("targetVersion", source)
	q.Set("targetVersionType", "branch")
	q.Set("$top", "0")
	resp, err := c.get(repositoryPath(repositoryID) + "/diffs/commits?" + q.Encode())
	if err != nil {
		return models.BranchComparison{}, fmt.Errorf("comparing branches: %w", err)
	}
	var data struct {
		AheadCount   int            `json:"aheadCount"`
		BehindCount  int            `json:"behindCount"`
		ChangeCounts map[string]int `json:"changeCounts"`
	}
	if err := decode(resp, &data); err != nil {
		return models.BranchComparison{}, fmt.Errorf("comparing branches: %w", err)
	}
	cmp := models.BranchComparison{Ahead: data.AheadCount, Behind: data.BehindCount}
	for _, n := range data.ChangeCounts {
		cmp.Files += n
	}
	if cmp.Ahead > 0 {
		cmp.Commits, err = c.branchCommits(repositoryID, source, target)
	}
	return cmp, err
}

func (c *Client) branchCommits(repositoryID, source, target string) ([]string, error) {
	q := url.Values{}
	q.Set("searchCriteria.itemVersion.version", source)
	q.Set("searchCriteria.compareVersion.version", target)
	q.Set("searchCriteria.$top", strconv.Itoa(commitPreview))
	resp, err := c.get(repositoryPath(repositoryID) + "/commits?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("loading commits: %w", err)
	}
	var data struct {
		Value []struct {
			Comment string `json:"comment"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading commits: %w", err)
	}
	commits := make([]string, len(data.Value))
	for i, v := range data.Value {
		commits[i] = v.Comment
	}
	return commits, nil
}

func (c *Client) GetBranchFile(repositoryID, path, branch string) (string, error) {
	q := url.Values{}
	q.Set("path", path)
	q.Set("versionDescriptor.versionType", "branch")
	q.Set("versionDescriptor.version", branch)
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

func (c *Client) PullRequestTemplate(repositoryID, target string) string {
	for _, path := range templatePaths(target) {
		if text, err := c.GetBranchFile(repositoryID, path, target); err == nil && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

type newPullRequestPayload struct {
	SourceRefName string `json:"sourceRefName"`
	TargetRefName string `json:"targetRefName"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	IsDraft       bool   `json:"isDraft"`
	Reviewers     []struct {
		ID         string `json:"id"`
		IsRequired bool   `json:"isRequired"`
	} `json:"reviewers,omitempty"`
	WorkItemRefs []struct {
		ID string `json:"id"`
	} `json:"workItemRefs,omitempty"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels,omitempty"`
}

func newPullRequestBody(pr models.NewPullRequest) newPullRequestPayload {
	p := newPullRequestPayload{
		SourceRefName: branchRefPrefix + pr.Source,
		TargetRefName: branchRefPrefix + pr.Target,
		Title:         pr.Title,
		Description:   pr.Description,
		IsDraft:       pr.Draft,
	}
	for _, r := range pr.Reviewers {
		p.Reviewers = append(p.Reviewers, struct {
			ID         string `json:"id"`
			IsRequired bool   `json:"isRequired"`
		}{r.ID, r.Required})
	}
	for _, id := range pr.WorkItems {
		p.WorkItemRefs = append(p.WorkItemRefs, struct {
			ID string `json:"id"`
		}{strconv.Itoa(id)})
	}
	for _, l := range pr.Labels {
		p.Labels = append(p.Labels, struct {
			Name string `json:"name"`
		}{l})
	}
	return p
}

func (c *Client) CreatePullRequest(repositoryID string, pr models.NewPullRequest) (models.PullRequest, error) {
	body, err := json.Marshal(newPullRequestBody(pr))
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("encoding pull request: %w", err)
	}
	target := fmt.Sprintf("%s%s/pullrequests?api-version=%s", c.baseURL, repositoryPath(repositoryID), apiVersion)
	resp, err := c.doRequest("POST", target, bytes.NewReader(body))
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("creating pull request: %w", err)
	}
	var data pullRequestResponse
	if err := decode(resp, &data); err != nil {
		return models.PullRequest{}, fmt.Errorf("creating pull request: %w", err)
	}
	return c.pullRequest(data), nil
}
