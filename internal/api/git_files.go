package api

import (
	"encoding/json"
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

func (c *Client) ListAllItems(repositoryID string, v models.GitVersion, limit int) (*models.FileIndex, error) {
	q := url.Values{}
	q.Set("scopePath", itemsRoot)
	q.Set("recursionLevel", "Full")
	versionQuery(q, v)
	resp, err := c.get(repositoryPath(repositoryID) + "/items?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("indexing files: %w", err)
	}
	defer resp.Body.Close()
	items, commit, truncated, err := decodeItemStream(resp.Body, limit)
	if err != nil {
		return nil, fmt.Errorf("indexing files: %w", err)
	}
	return models.NewFileIndex(items, commit, truncated), nil
}

func decodeItemStream(r io.Reader, limit int) ([]models.GitItem, string, bool, error) {
	dec := json.NewDecoder(r)
	if err := seekArray(dec, "value"); err != nil {
		return nil, "", false, err
	}
	var items []models.GitItem
	commit := ""
	for dec.More() {
		if len(items) == limit {
			return items, commit, true, nil
		}
		var v struct {
			Path          string `json:"path"`
			IsFolder      bool   `json:"isFolder"`
			GitObjectType string `json:"gitObjectType"`
			CommitID      string `json:"commitId"`
		}
		if err := dec.Decode(&v); err != nil {
			return nil, "", false, err
		}
		if commit == "" {
			commit = v.CommitID
		}
		if v.Path == itemsRoot || v.GitObjectType == "commit" {
			continue
		}
		items = append(items, models.GitItem{Path: v.Path, Folder: v.IsFolder})
	}
	return items, commit, false, nil
}

func seekArray(dec *json.Decoder, field string) error {
	if _, err := dec.Token(); err != nil {
		return err
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if name, ok := tok.(string); ok && name == field {
			_, err := dec.Token()
			return err
		}
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return err
		}
	}
	return fmt.Errorf("no %s in the response", field)
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
