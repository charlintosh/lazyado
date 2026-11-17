package api

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/charlintosh/lazyado/internal/models"
)

type annotatedTagResponse struct {
	Name     string `json:"name"`
	Message  string `json:"message"`
	TaggedBy struct {
		Name string  `json:"name"`
		Date gitTime `json:"date"`
	} `json:"taggedBy"`
	TaggedObject struct {
		ObjectID string `json:"objectId"`
	} `json:"taggedObject"`
}

func (r annotatedTagResponse) detail() models.TagDetail {
	return models.TagDetail{Name: r.Name, Message: r.Message, Tagger: r.TaggedBy.Name, Date: r.TaggedBy.Date.Time, CommitID: r.TaggedObject.ObjectID}
}

func (c *Client) ListTags(repositoryID string) ([]models.Tag, error) {
	resp, err := c.get(fmt.Sprintf("%s/refs?filter=tags/&peelTags=true&$top=%d", repositoryPath(repositoryID), branchPageSize))
	if err != nil {
		return nil, fmt.Errorf("loading tags: %w", err)
	}
	var data struct {
		Value []struct {
			Name           string `json:"name"`
			ObjectID       string `json:"objectId"`
			PeeledObjectID string `json:"peeledObjectId"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading tags: %w", err)
	}
	tags := make([]models.Tag, 0, len(data.Value))
	for _, r := range data.Value {
		commit := r.PeeledObjectID
		if commit == "" {
			commit = r.ObjectID
		}
		tags = append(tags, models.Tag{Name: strings.TrimPrefix(r.Name, tagRefPrefix), ObjectID: r.ObjectID, CommitID: commit})
	}
	sort.SliceStable(tags, func(i, j int) bool { return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name) })
	return tags, nil
}

func (c *Client) GetAnnotatedTag(repositoryID, objectID string) (models.TagDetail, error) {
	resp, err := c.get(repositoryPath(repositoryID) + "/annotatedtags/" + url.PathEscape(objectID))
	if err != nil {
		return models.TagDetail{}, fmt.Errorf("loading tag: %w", err)
	}
	var data annotatedTagResponse
	if err := decode(resp, &data); err != nil {
		return models.TagDetail{}, fmt.Errorf("loading tag: %w", err)
	}
	return data.detail(), nil
}

func (c *Client) CreateTag(repositoryID, name, commitID, message string) (models.TagDetail, error) {
	payload := map[string]any{
		"name":         name,
		"taggedObject": map[string]string{"objectId": commitID},
		"message":      message,
	}
	var reply annotatedTagResponse
	if err := c.exchangeJSON("POST", repositoryPath(repositoryID)+"/annotatedtags", payload, &reply); err != nil {
		return models.TagDetail{}, fmt.Errorf("creating tag %s: %w", name, err)
	}
	return reply.detail(), nil
}
