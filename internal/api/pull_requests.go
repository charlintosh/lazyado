package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/models"
)

const (
	pullRequestPageSize = 200
	changesPageSize     = 2000
	codeReviewArtifact  = "vstfs:///CodeReview/CodeReviewId/%s/%d"
)

type pullRequestResponse struct {
	PullRequestID int         `json:"pullRequestId"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Status        string      `json:"status"`
	IsDraft       bool        `json:"isDraft"`
	CreatedBy     identityRef `json:"createdBy"`
	CreationDate  time.Time   `json:"creationDate"`
	ClosedDate    time.Time   `json:"closedDate"`
	SourceRefName string      `json:"sourceRefName"`
	TargetRefName string      `json:"targetRefName"`
	MergeStatus   string      `json:"mergeStatus"`
	Repository    struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	} `json:"repository"`
	Reviewers []struct {
		identityRef
		Vote        int  `json:"vote"`
		IsRequired  bool `json:"isRequired"`
		IsContainer bool `json:"isContainer"`
	} `json:"reviewers"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (c *Client) pullRequest(r pullRequestResponse) models.PullRequest {
	pr := models.PullRequest{
		ID:             r.PullRequestID,
		Title:          r.Title,
		Description:    r.Description,
		Status:         models.PullRequestStatus(r.Status),
		IsDraft:        r.IsDraft,
		Author:         models.Identity{ID: r.CreatedBy.ID, DisplayName: r.CreatedBy.DisplayName, UniqueName: r.CreatedBy.UniqueName},
		SourceBranch:   models.BranchName(r.SourceRefName),
		TargetBranch:   models.BranchName(r.TargetRefName),
		RepositoryID:   r.Repository.ID,
		RepositoryName: r.Repository.Name,
		ProjectID:      r.Repository.Project.ID,
		MergeStatus:    r.MergeStatus,
		CreatedDate:    r.CreationDate,
		ClosedDate:     r.ClosedDate,
		WebURL:         fmt.Sprintf("%s/_git/%s/pullrequest/%d", c.webURL, url.PathEscape(r.Repository.Name), r.PullRequestID),
	}
	for _, rv := range r.Reviewers {
		pr.Reviewers = append(pr.Reviewers, models.Reviewer{
			ID:          rv.ID,
			DisplayName: rv.DisplayName,
			UniqueName:  rv.UniqueName,
			Vote:        models.Vote(rv.Vote),
			Required:    rv.IsRequired,
			Group:       rv.IsContainer,
		})
	}
	for _, l := range r.Labels {
		pr.Labels = append(pr.Labels, l.Name)
	}
	return pr
}

func (c *Client) sendJSON(method, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}
	target := fmt.Sprintf("%s%s%sapi-version=%s", c.baseURL, endpoint, separator, apiVersion)
	resp, err := c.doRequest(method, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func pullRequestPath(repositoryID string, id int) string {
	return fmt.Sprintf("/git/repositories/%s/pullRequests/%d", url.PathEscape(repositoryID), id)
}

func (c *Client) ListRepositories() ([]models.Repository, error) {
	resp, err := c.get("/git/repositories")
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}
	var data struct {
		Value []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			IsDisabled bool   `json:"isDisabled"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}
	repos := make([]models.Repository, 0, len(data.Value))
	for _, r := range data.Value {
		if !r.IsDisabled {
			repos = append(repos, models.Repository{ID: r.ID, Name: r.Name})
		}
	}
	sort.SliceStable(repos, func(i, j int) bool { return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name) })
	return repos, nil
}

func (c *Client) GetFavoriteArtifactIDs() ([]string, error) {
	target := fmt.Sprintf("https://dev.azure.com/%s/_apis/favorite/favorites?api-version=%s-preview", url.PathEscape(c.organization), apiVersion)
	resp, err := c.doRequest("GET", target, nil)
	if err != nil {
		return nil, fmt.Errorf("loading favorites: %w", err)
	}
	var data struct {
		Value []struct {
			ArtifactID        string `json:"artifactId"`
			ArtifactIsDeleted bool   `json:"artifactIsDeleted"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading favorites: %w", err)
	}
	ids := make([]string, 0, len(data.Value))
	for _, f := range data.Value {
		if f.ArtifactID != "" && !f.ArtifactIsDeleted {
			ids = append(ids, strings.ToLower(f.ArtifactID))
		}
	}
	return ids, nil
}

func criteriaQuery(status models.PullRequestStatus, criteria models.PullRequestCriteria) string {
	q := url.Values{}
	q.Set("searchCriteria.status", string(status))
	for name, value := range map[string]string{
		"searchCriteria.creatorId":     criteria.CreatorID,
		"searchCriteria.reviewerId":    criteria.ReviewerID,
		"searchCriteria.sourceRefName": criteria.SourceRef,
		"searchCriteria.targetRefName": criteria.TargetRef,
	} {
		if value != "" {
			q.Set(name, value)
		}
	}
	q.Set("$top", fmt.Sprint(pullRequestPageSize))
	return q.Encode()
}

func (c *Client) ListPullRequests(status models.PullRequestStatus, repositoryID string, criteria models.PullRequestCriteria) ([]models.PullRequest, error) {
	scope := "/git/pullrequests"
	if repositoryID != "" {
		scope = fmt.Sprintf("/git/repositories/%s/pullrequests", url.PathEscape(repositoryID))
	}
	resp, err := c.get(scope + "?" + criteriaQuery(status, criteria))
	if err != nil {
		return nil, fmt.Errorf("listing pull requests: %w", err)
	}
	var data struct {
		Value []pullRequestResponse `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("listing pull requests: %w", err)
	}
	prs := make([]models.PullRequest, len(data.Value))
	for i, r := range data.Value {
		prs[i] = c.pullRequest(r)
	}
	return prs, nil
}

func (c *Client) GetPullRequest(id int) (models.PullRequest, error) {
	resp, err := c.get(fmt.Sprintf("/git/pullrequests/%d", id))
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("loading pull request %d: %w", id, err)
	}
	var data pullRequestResponse
	if err := decode(resp, &data); err != nil {
		return models.PullRequest{}, fmt.Errorf("loading pull request %d: %w", id, err)
	}
	return c.pullRequest(data), nil
}

type threadPosition struct {
	Line   int `json:"line"`
	Offset int `json:"offset"`
}

type threadContext struct {
	FilePath       string          `json:"filePath"`
	LeftFileStart  *threadPosition `json:"leftFileStart,omitempty"`
	LeftFileEnd    *threadPosition `json:"leftFileEnd,omitempty"`
	RightFileStart *threadPosition `json:"rightFileStart,omitempty"`
	RightFileEnd   *threadPosition `json:"rightFileEnd,omitempty"`
}

type threadResponse struct {
	ID            int            `json:"id"`
	Status        string         `json:"status"`
	IsDeleted     bool           `json:"isDeleted"`
	PublishedDate time.Time      `json:"publishedDate"`
	ThreadContext *threadContext `json:"threadContext"`
	Comments      []struct {
		ID              int           `json:"id"`
		ParentCommentID int           `json:"parentCommentId"`
		Content         string        `json:"content"`
		CommentType     string        `json:"commentType"`
		IsDeleted       bool          `json:"isDeleted"`
		Author          identityRef   `json:"author"`
		PublishedDate   time.Time     `json:"publishedDate"`
		UsersLiked      []identityRef `json:"usersLiked"`
	} `json:"comments"`
}

func (r threadResponse) thread() (models.PullRequestThread, bool) {
	t := models.PullRequestThread{ID: r.ID, Status: r.Status, Date: r.PublishedDate}
	if ctx := r.ThreadContext; ctx != nil {
		t.FilePath = ctx.FilePath
		start, end := ctx.RightFileStart, ctx.RightFileEnd
		if start == nil {
			start, end, t.LeftSide = ctx.LeftFileStart, ctx.LeftFileEnd, ctx.LeftFileStart != nil
		}
		if start != nil {
			t.Line = start.Line
		}
		if end != nil {
			t.EndLine = end.Line
		}
	}
	for _, cm := range r.Comments {
		if cm.IsDeleted || strings.TrimSpace(cm.Content) == "" {
			continue
		}
		t.Comments = append(t.Comments, models.PullRequestComment{
			ID:         cm.ID,
			ParentID:   cm.ParentCommentID,
			AuthorID:   cm.Author.ID,
			AuthorName: cm.Author.DisplayName,
			AuthorUPN:  cm.Author.UniqueName,
			Content:    cm.Content,
			Date:       cm.PublishedDate,
			System:     strings.EqualFold(cm.CommentType, "system"),
			Likes:      likers(cm.UsersLiked),
		})
	}
	return t, !r.IsDeleted && len(t.Comments) > 0
}

func likers(refs []identityRef) []models.Identity {
	out := make([]models.Identity, 0, len(refs))
	for _, r := range refs {
		out = append(out, models.Identity{ID: r.ID, DisplayName: r.DisplayName, UniqueName: r.UniqueName})
	}
	return out
}

func (c *Client) GetPullRequestThreads(repositoryID string, id, iteration int) ([]models.PullRequestThread, error) {
	endpoint := pullRequestPath(repositoryID, id) + "/threads"
	if iteration > 0 {
		endpoint += fmt.Sprintf("?$iteration=%d", iteration)
	}
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("loading threads: %w", err)
	}
	var data struct {
		Value []threadResponse `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading threads: %w", err)
	}
	threads := make([]models.PullRequestThread, 0, len(data.Value))
	for _, r := range data.Value {
		if t, ok := r.thread(); ok {
			threads = append(threads, t)
		}
	}
	sort.SliceStable(threads, func(i, j int) bool { return threads[i].Date.After(threads[j].Date) })
	return threads, nil
}

type newComment struct {
	ParentCommentID int    `json:"parentCommentId"`
	Content         string `json:"content"`
	CommentType     string `json:"commentType"`
}

func (c *Client) CreatePullRequestThread(repositoryID string, id int, content string, anchor *models.ThreadAnchor) error {
	payload := struct {
		Comments      []newComment   `json:"comments"`
		Status        string         `json:"status"`
		ThreadContext *threadContext `json:"threadContext,omitempty"`
	}{
		Comments: []newComment{{Content: content, CommentType: "text"}},
		Status:   models.ThreadActive,
	}
	if anchor != nil {
		start, end := &threadPosition{Line: anchor.Line, Offset: 1}, &threadPosition{Line: anchor.Line, Offset: 1}
		if anchor.EndLine > anchor.Line {
			end = &threadPosition{Line: anchor.EndLine, Offset: max(anchor.EndOffset, 1)}
		}
		payload.ThreadContext = &threadContext{FilePath: anchor.FilePath, RightFileStart: start, RightFileEnd: end}
		if anchor.LeftSide {
			payload.ThreadContext = &threadContext{FilePath: anchor.FilePath, LeftFileStart: start, LeftFileEnd: end}
		}
	}
	if err := c.sendJSON("POST", pullRequestPath(repositoryID, id)+"/threads", payload); err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	return nil
}

func (c *Client) ReplyToPullRequestThread(repositoryID string, id, threadID, parentCommentID int, content string) error {
	endpoint := fmt.Sprintf("%s/threads/%d/comments", pullRequestPath(repositoryID, id), threadID)
	payload := newComment{ParentCommentID: parentCommentID, Content: content, CommentType: "text"}
	if err := c.sendJSON("POST", endpoint, payload); err != nil {
		return fmt.Errorf("replying to thread: %w", err)
	}
	return nil
}

func (c *Client) UpdatePullRequestComment(repositoryID string, id, threadID, commentID int, content string) error {
	endpoint := fmt.Sprintf("%s/threads/%d/comments/%d", pullRequestPath(repositoryID, id), threadID, commentID)
	payload := struct {
		Content string `json:"content"`
	}{Content: content}
	if err := c.sendJSON("PATCH", endpoint, payload); err != nil {
		return fmt.Errorf("updating comment: %w", err)
	}
	return nil
}

func (c *Client) DeletePullRequestComment(repositoryID string, id, threadID, commentID int) error {
	target := fmt.Sprintf("%s%s/threads/%d/comments/%d?api-version=%s", c.baseURL, pullRequestPath(repositoryID, id), threadID, commentID, apiVersion)
	resp, err := c.doRequest("DELETE", target, nil)
	if err != nil {
		return fmt.Errorf("deleting comment: %w", err)
	}
	return resp.Body.Close()
}

func (c *Client) LikePullRequestComment(repositoryID string, id, threadID, commentID int, like bool) error {
	method := "POST"
	if !like {
		method = "DELETE"
	}
	target := fmt.Sprintf("%s%s/threads/%d/comments/%d/likes?api-version=%s-preview.1", c.baseURL, pullRequestPath(repositoryID, id), threadID, commentID, apiVersion)
	resp, err := c.doRequest(method, target, nil)
	if err != nil {
		return fmt.Errorf("liking comment: %w", err)
	}
	return resp.Body.Close()
}

func (c *Client) SetPullRequestThreadStatus(repositoryID string, id, threadID int, status string) error {
	endpoint := fmt.Sprintf("%s/threads/%d", pullRequestPath(repositoryID, id), threadID)
	payload := struct {
		Status string `json:"status"`
	}{Status: status}
	if err := c.sendJSON("PATCH", endpoint, payload); err != nil {
		return fmt.Errorf("updating thread: %w", err)
	}
	return nil
}

func (c *Client) VotePullRequest(repositoryID string, id int, reviewerID string, vote models.Vote) error {
	endpoint := fmt.Sprintf("%s/reviewers/%s", pullRequestPath(repositoryID, id), url.PathEscape(reviewerID))
	payload := struct {
		Vote int `json:"vote"`
	}{Vote: int(vote)}
	if err := c.sendJSON("PUT", endpoint, payload); err != nil {
		return fmt.Errorf("voting: %w", err)
	}
	return nil
}

type policyEvaluation struct {
	Status        string `json:"status"`
	Configuration struct {
		IsBlocking bool `json:"isBlocking"`
		IsEnabled  bool `json:"isEnabled"`
		Type       struct {
			DisplayName string `json:"displayName"`
		} `json:"type"`
		Settings struct {
			DisplayName          string `json:"displayName"`
			MinimumApproverCount int    `json:"minimumApproverCount"`
			StatusName           string `json:"statusName"`
			StatusGenre          string `json:"statusGenre"`
		} `json:"settings"`
	} `json:"configuration"`
	Context struct {
		IsExpired bool `json:"isExpired"`
	} `json:"context"`
}

func statusContextName(genre, name string) string {
	if genre == "" {
		return name
	}
	return genre + "/" + name
}

func (e policyEvaluation) check() (models.PullRequestCheck, bool) {
	cfg := e.Configuration
	if !cfg.IsEnabled || strings.EqualFold(e.Status, "notApplicable") {
		return models.PullRequestCheck{}, false
	}
	check := models.PullRequestCheck{Name: cfg.Type.DisplayName, Required: cfg.IsBlocking}
	switch {
	case cfg.Settings.StatusName != "":
		check.Name = statusContextName(cfg.Settings.StatusGenre, cfg.Settings.StatusName)
	case cfg.Settings.DisplayName != "":
		check.Name += ": " + cfg.Settings.DisplayName
	case cfg.Settings.MinimumApproverCount > 0:
		check.Detail = fmt.Sprintf("%d required", cfg.Settings.MinimumApproverCount)
	}
	switch strings.ToLower(e.Status) {
	case "approved":
		check.State = models.CheckPassed
	case "rejected", "broken":
		check.State = models.CheckFailed
	case "running":
		check.State = models.CheckRunning
	default:
		check.State = models.CheckPending
	}
	if e.Context.IsExpired && check.State == models.CheckPassed {
		check.State, check.Detail = models.CheckPending, "expired, needs a new run"
	}
	return check, true
}

type pullRequestStatus struct {
	ID          int    `json:"id"`
	State       string `json:"state"`
	Description string `json:"description"`
	Context     struct {
		Name  string `json:"name"`
		Genre string `json:"genre"`
	} `json:"context"`
}

func (s pullRequestStatus) check() models.PullRequestCheck {
	check := models.PullRequestCheck{Name: statusContextName(s.Context.Genre, s.Context.Name), Detail: s.Description}
	switch strings.ToLower(s.State) {
	case "succeeded":
		check.State = models.CheckPassed
	case "failed", "error":
		check.State = models.CheckFailed
	default:
		check.State = models.CheckPending
	}
	return check
}

func (c *Client) GetPullRequestChecks(pr models.PullRequest) ([]models.PullRequestCheck, error) {
	artifact := fmt.Sprintf(codeReviewArtifact, pr.ProjectID, pr.ID)
	resp, err := c.getPreview("/policy/evaluations?artifactId=" + url.QueryEscape(artifact))
	if err != nil {
		return nil, fmt.Errorf("loading policies: %w", err)
	}
	var evaluations struct {
		Value []policyEvaluation `json:"value"`
	}
	if err := decode(resp, &evaluations); err != nil {
		return nil, fmt.Errorf("loading policies: %w", err)
	}
	var checks []models.PullRequestCheck
	seen := map[string]bool{}
	for _, e := range evaluations.Value {
		if check, ok := e.check(); ok {
			checks = append(checks, check)
			seen[check.Name] = true
		}
	}
	for _, check := range c.pullRequestStatuses(pr) {
		if !seen[check.Name] {
			checks = append(checks, check)
		}
	}
	sort.SliceStable(checks, func(i, j int) bool { return checks[i].Required && !checks[j].Required })
	return checks, nil
}

func (c *Client) pullRequestStatuses(pr models.PullRequest) []models.PullRequestCheck {
	resp, err := c.get(pullRequestPath(pr.RepositoryID, pr.ID) + "/statuses")
	if err != nil {
		return nil
	}
	var data struct {
		Value []pullRequestStatus `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil
	}
	sort.SliceStable(data.Value, func(i, j int) bool { return data.Value[i].ID < data.Value[j].ID })
	latest := map[string]models.PullRequestCheck{}
	var order []string
	for _, s := range data.Value {
		check := s.check()
		if _, ok := latest[check.Name]; !ok {
			order = append(order, check.Name)
		}
		latest[check.Name] = check
	}
	checks := make([]models.PullRequestCheck, len(order))
	for i, name := range order {
		checks[i] = latest[name]
	}
	return checks
}

type commitRef struct {
	CommitID string `json:"commitId"`
}

func changeKind(changeType string) models.ChangeKind {
	t := strings.ToLower(changeType)
	switch {
	case strings.Contains(t, "delete"):
		return models.ChangeDelete
	case strings.Contains(t, "rename"):
		return models.ChangeRename
	case strings.Contains(t, "add"):
		return models.ChangeAdd
	default:
		return models.ChangeEdit
	}
}

func (c *Client) GetPullRequestChanges(repositoryID string, id int) (models.PullRequestChanges, error) {
	base := pullRequestPath(repositoryID, id)
	resp, err := c.get(base + "/iterations")
	if err != nil {
		return models.PullRequestChanges{}, fmt.Errorf("loading updates: %w", err)
	}
	var iterations struct {
		Value []struct {
			ID              int       `json:"id"`
			SourceRefCommit commitRef `json:"sourceRefCommit"`
			CommonRefCommit commitRef `json:"commonRefCommit"`
			TargetRefCommit commitRef `json:"targetRefCommit"`
		} `json:"value"`
	}
	if err := decode(resp, &iterations); err != nil {
		return models.PullRequestChanges{}, fmt.Errorf("loading updates: %w", err)
	}
	if len(iterations.Value) == 0 {
		return models.PullRequestChanges{}, nil
	}
	last := iterations.Value[len(iterations.Value)-1]
	changes := models.PullRequestChanges{Iteration: last.ID, HeadCommit: last.SourceRefCommit.CommitID, BaseCommit: last.CommonRefCommit.CommitID}
	if changes.BaseCommit == "" {
		changes.BaseCommit = last.TargetRefCommit.CommitID
	}
	resp, err = c.get(fmt.Sprintf("%s/iterations/%d/changes?$top=%d&$compareTo=0", base, last.ID, changesPageSize))
	if err != nil {
		return models.PullRequestChanges{}, fmt.Errorf("loading changed files: %w", err)
	}
	var data struct {
		ChangeEntries []struct {
			ChangeType   string `json:"changeType"`
			OriginalPath string `json:"originalPath"`
			Item         struct {
				Path     string `json:"path"`
				IsFolder bool   `json:"isFolder"`
			} `json:"item"`
		} `json:"changeEntries"`
	}
	if err := decode(resp, &data); err != nil {
		return models.PullRequestChanges{}, fmt.Errorf("loading changed files: %w", err)
	}
	for _, e := range data.ChangeEntries {
		path := e.Item.Path
		if path == "" {
			path = e.OriginalPath
		}
		if e.Item.IsFolder || path == "" {
			continue
		}
		changes.Files = append(changes.Files, models.PullRequestFile{Path: path, OriginalPath: e.OriginalPath, Kind: changeKind(e.ChangeType)})
	}
	sort.SliceStable(changes.Files, func(i, j int) bool { return changes.Files[i].Path < changes.Files[j].Path })
	return changes, nil
}

func (c *Client) GetFileContent(repositoryID, path, commit string) (string, error) {
	endpoint := fmt.Sprintf("/git/repositories/%s/items?path=%s&versionDescriptor.versionType=commit&versionDescriptor.version=%s&$format=text",
		url.PathEscape(repositoryID), url.QueryEscape(path), url.QueryEscape(commit))
	resp, err := c.get(endpoint)
	if err != nil {
		return "", fmt.Errorf("loading %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxFileBytes+1))
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	if int64(len(body)) > c.maxFileBytes {
		return "", fmt.Errorf("loading %s: %w", path, models.ErrFileTooLarge)
	}
	return string(body), nil
}

func (c *Client) GetPullRequestWorkItemIDs(repositoryID string, id int) ([]int, error) {
	resp, err := c.get(pullRequestPath(repositoryID, id) + "/workitems")
	if err != nil {
		return nil, fmt.Errorf("loading linked work items: %w", err)
	}
	var data struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading linked work items: %w", err)
	}
	ids := make([]int, 0, len(data.Value))
	for _, ref := range data.Value {
		var n int
		if _, err := fmt.Sscanf(ref.ID, "%d", &n); err == nil {
			ids = append(ids, n)
		}
	}
	return ids, nil
}
