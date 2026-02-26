package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/debug"
	"github.com/charlintosh/lazyado/internal/models"
)

var prLogger = debug.Scope("pull_requests")

type repositoriesResponse struct {
	Count int `json:"count"`
	Value []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"value"`
}

type pullRequestsResponse struct {
	Count int              `json:"count"`
	Value []pullRequestAPI `json:"value"`
}

type pullRequestAPI struct {
	PullRequestID int    `json:"pullRequestId"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	CreatedBy     struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
	} `json:"createdBy"`
	CreationDate  time.Time `json:"creationDate"`
	ClosedDate    time.Time `json:"closedDate"`
	SourceRefName string    `json:"sourceRefName"`
	TargetRefName string    `json:"targetRefName"`
	MergeStatus   string    `json:"mergeStatus"`
	IsDraft       bool      `json:"isDraft"`
	Repository    struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"repository"`
	Reviewers []struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
		Vote        int    `json:"vote"`
		IsRequired  bool   `json:"isRequired"`
		HasDeclined bool   `json:"hasDeclined"`
		IsFlagged   bool   `json:"isFlagged"`
		IsContainer bool   `json:"isContainer"`
	} `json:"reviewers"`
	URL string `json:"url"`
}

func (c *Client) GetRepositories() ([]models.Repository, error) {
	resp, err := c.get("/git/repositories")
	if err != nil {
		return nil, fmt.Errorf("fetching repositories: %w", err)
	}

	var result repositoriesResponse
	if err := decode(resp, &result); err != nil {
		return nil, err
	}

	repos := make([]models.Repository, 0, len(result.Value))
	for _, r := range result.Value {
		repos = append(repos, models.Repository{
			ID:   r.ID,
			Name: r.Name,
		})
	}

	prLogger.Debugf("fetched %d repositories", len(repos))
	return repos, nil
}

func (c *Client) GetPullRequests(repoID string, status models.PRStatus) ([]models.PullRequest, error) {
	endpoint := fmt.Sprintf("/git/repositories/%s/pullrequests?searchCriteria.status=%s&$top=50", repoID, string(status))
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetching pull requests: %w", err)
	}

	var result pullRequestsResponse
	if err := decode(resp, &result); err != nil {
		return nil, err
	}

	prs := make([]models.PullRequest, 0, len(result.Value))
	for _, p := range result.Value {
		prs = append(prs, toPullRequest(p))
	}

	prLogger.Debugf("fetched %d pull requests for repo %s", len(prs), repoID)
	return prs, nil
}

func (c *Client) GetAllPullRequests(status models.PRStatus) ([]models.PullRequest, error) {
	repos, err := c.GetRepositories()
	if err != nil {
		return nil, err
	}

	var allPRs []models.PullRequest
	for _, repo := range repos {
		prs, err := c.GetPullRequests(repo.ID, status)
		if err != nil {
			prLogger.Debugf("error fetching PRs for repo %s: %v", repo.Name, err)
			continue
		}
		allPRs = append(allPRs, prs...)
	}

	prLogger.Debugf("fetched %d total pull requests across %d repos", len(allPRs), len(repos))
	return allPRs, nil
}

func (c *Client) VotePullRequest(repoID string, prID int, reviewerID string, vote models.PRVote) error {
	endpoint := fmt.Sprintf("/git/repositories/%s/pullRequests/%d/reviewers/%s", repoID, prID, reviewerID)

	body := fmt.Sprintf(`{"vote":%d,"id":"%s"}`, int(vote), reviewerID)
	resp, err := c.put(endpoint, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("voting on pull request: %w", err)
	}
	resp.Body.Close()

	prLogger.Debugf("voted %d on PR #%d in repo %s", int(vote), prID, repoID)
	return nil
}

func (c *Client) PullRequestWebURL(repoName string, prID int) string {
	return fmt.Sprintf("%s/_git/%s/pullrequest/%d", c.webURL, repoName, prID)
}

func (c *Client) GetConnectionProfile() (*models.ConnectionProfile, error) {
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/connectionData", c.organization)
	resp, err := c.doRequest("GET", url+"?api-version=7.1", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching connection data: %w", err)
	}

	var result struct {
		AuthenticatedUser struct {
			ID          string `json:"id"`
			DisplayName string `json:"customDisplayName"`
		} `json:"authenticatedUser"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("decoding connection data: %w", err)
	}
	resp.Body.Close()

	return &models.ConnectionProfile{
		ID:          result.AuthenticatedUser.ID,
		DisplayName: result.AuthenticatedUser.DisplayName,
	}, nil
}

func toPullRequest(p pullRequestAPI) models.PullRequest {
	reviewers := make([]models.PRReviewer, 0, len(p.Reviewers))
	for _, r := range p.Reviewers {
		reviewers = append(reviewers, models.PRReviewer{
			ID:          r.ID,
			DisplayName: r.DisplayName,
			UniqueName:  r.UniqueName,
			Vote:        models.PRVote(r.Vote),
			IsRequired:  r.IsRequired,
			HasDeclined: r.HasDeclined,
			IsFlagged:   r.IsFlagged,
			IsContainer: r.IsContainer,
		})
	}

	return models.PullRequest{
		ID:           p.PullRequestID,
		Title:        p.Title,
		Description:  p.Description,
		Status:       models.PRStatus(p.Status),
		CreatedBy:    p.CreatedBy.DisplayName,
		CreatedByID:  p.CreatedBy.ID,
		CreationDate: p.CreationDate,
		ClosedDate:   p.ClosedDate,
		SourceBranch: p.SourceRefName,
		TargetBranch: p.TargetRefName,
		MergeStatus:  models.PRMergeStatus(p.MergeStatus),
		IsDraft:      p.IsDraft,
		Repository: models.Repository{
			ID:   p.Repository.ID,
			Name: p.Repository.Name,
		},
		Reviewers: reviewers,
		URL:       p.URL,
	}
}
