package api

import (
	"fmt"
	"io"
	"net/url"

	"github.com/charlintosh/lazyado/internal/models"
)

const itemsRoot = "/"

type commitRefResponse struct {
	CommitID string `json:"commitId"`
	Comment  string `json:"comment"`
	Author   struct {
		Name string  `json:"name"`
		Date gitTime `json:"date"`
	} `json:"author"`
}

func (r commitRefResponse) ref() models.CommitRef {
	return models.CommitRef{ID: r.CommitID, Comment: r.Comment, Author: r.Author.Name, Date: r.Author.Date.Time}
}

func versionQuery(q url.Values, v models.GitVersion) {
	if v.IsZero() {
		return
	}
	q.Set("versionDescriptor.versionType", v.Type())
	q.Set("versionDescriptor.version", v.Name)
}

func (c *Client) ListItems(repositoryID, folder string, v models.GitVersion) ([]models.GitItem, error) {
	if folder == "" {
		folder = itemsRoot
	}
	q := url.Values{}
	q.Set("scopePath", folder)
	q.Set("recursionLevel", "OneLevel")
	q.Set("latestProcessedChange", "true")
	versionQuery(q, v)
	resp, err := c.get(repositoryPath(repositoryID) + "/items?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", folder, err)
	}
	var data struct {
		Value []struct {
			Path                  string             `json:"path"`
			IsFolder              bool               `json:"isFolder"`
			GitObjectType         string             `json:"gitObjectType"`
			LatestProcessedChange *commitRefResponse `json:"latestProcessedChange"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading %s: %w", folder, err)
	}
	items := make([]models.GitItem, 0, len(data.Value))
	for _, v := range data.Value {
		if v.Path == folder || v.GitObjectType == "commit" {
			continue
		}
		it := models.GitItem{Path: v.Path, Folder: v.IsFolder}
		if v.LatestProcessedChange != nil {
			it.Last = v.LatestProcessedChange.ref()
		}
		items = append(items, it)
	}
	return models.SortGitItems(items), nil
}

func (c *Client) GetItemText(repositoryID, path string, v models.GitVersion) (string, error) {
	q := url.Values{}
	q.Set("path", path)
	versionQuery(q, v)
	q.Set("$format", "text")
	resp, err := c.get(repositoryPath(repositoryID) + "/items?" + q.Encode())
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

func (c *Client) ListTagNames(repositoryID string) ([]string, error) {
	tags, err := c.ListTags(repositoryID)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names, nil
}
