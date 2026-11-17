package api

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/charlintosh/lazyado/internal/models"
)

const (
	CommitsPage = 100
	changesPage = 1000
)

type commitResponse struct {
	commitRefResponse
	Committer struct {
		Name string  `json:"name"`
		Date gitTime `json:"date"`
	} `json:"committer"`
	Parents      []string       `json:"parents"`
	ChangeCounts map[string]int `json:"changeCounts"`
}

func (r commitResponse) commit() models.Commit {
	return models.Commit{
		CommitRef: r.ref(),
		Committer: r.Committer.Name,
		Parents:   r.Parents,
		Adds:      r.ChangeCounts["Add"],
		Edits:     r.ChangeCounts["Edit"],
		Deletes:   r.ChangeCounts["Delete"],
	}
}

func (c *Client) ListCommits(repositoryID string, v models.GitVersion, criteria models.CommitCriteria, skip int) ([]models.Commit, error) {
	q := url.Values{}
	if !v.IsZero() {
		q.Set("searchCriteria.itemVersion.versionType", v.Type())
		q.Set("searchCriteria.itemVersion.version", v.Name)
	}
	if criteria.Author != "" {
		q.Set("searchCriteria.author", criteria.Author)
	}
	if criteria.Path != "" {
		q.Set("searchCriteria.itemPath", criteria.Path)
	}
	q.Set("searchCriteria.$top", strconv.Itoa(CommitsPage))
	q.Set("searchCriteria.$skip", strconv.Itoa(skip))
	resp, err := c.get(repositoryPath(repositoryID) + "/commits?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("loading commits: %w", err)
	}
	var data struct {
		Value []commitResponse `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading commits: %w", err)
	}
	commits := make([]models.Commit, len(data.Value))
	for i, v := range data.Value {
		commits[i] = v.commit()
	}
	return commits, nil
}

func (c *Client) GetCommitDetail(repositoryID, id string) (models.CommitDetail, error) {
	base := repositoryPath(repositoryID) + "/commits/" + url.PathEscape(id)
	resp, err := c.get(base)
	if err != nil {
		return models.CommitDetail{}, fmt.Errorf("loading commit %s: %w", models.ShortCommit(id), err)
	}
	var commit commitResponse
	if err := decode(resp, &commit); err != nil {
		return models.CommitDetail{}, fmt.Errorf("loading commit %s: %w", models.ShortCommit(id), err)
	}
	resp, err = c.get(fmt.Sprintf("%s/changes?top=%d", base, changesPage))
	if err != nil {
		return models.CommitDetail{}, fmt.Errorf("loading the changes of %s: %w", models.ShortCommit(id), err)
	}
	var changes struct {
		Changes []struct {
			ChangeType string `json:"changeType"`
			Item       struct {
				Path     string `json:"path"`
				IsFolder bool   `json:"isFolder"`
			} `json:"item"`
		} `json:"changes"`
	}
	if err := decode(resp, &changes); err != nil {
		return models.CommitDetail{}, fmt.Errorf("loading the changes of %s: %w", models.ShortCommit(id), err)
	}
	detail := models.CommitDetail{Commit: commit.commit()}
	for _, ch := range changes.Changes {
		if !ch.Item.IsFolder && ch.Item.Path != "" {
			detail.Files = append(detail.Files, models.CommitFile{Path: ch.Item.Path, Kind: changeKind(ch.ChangeType)})
		}
	}
	return detail, nil
}
