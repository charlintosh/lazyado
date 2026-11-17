package api

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/charlintosh/lazyado/internal/models"
)

const zeroObjectID = "0000000000000000000000000000000000000000"

type refUpdate struct {
	Name        string `json:"name"`
	OldObjectID string `json:"oldObjectId"`
	NewObjectID string `json:"newObjectId"`
}

func (c *Client) ListBranchDetails(repositoryID, base string) ([]models.Branch, error) {
	q := url.Values{}
	q.Set("baseVersionDescriptor.version", base)
	q.Set("baseVersionDescriptor.versionType", "branch")
	resp, err := c.get(repositoryPath(repositoryID) + "/stats/branches?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("loading branches: %w", err)
	}
	var stats struct {
		Value []struct {
			Name          string            `json:"name"`
			AheadCount    int               `json:"aheadCount"`
			BehindCount   int               `json:"behindCount"`
			IsBaseVersion bool              `json:"isBaseVersion"`
			Commit        commitRefResponse `json:"commit"`
		} `json:"value"`
	}
	if err := decode(resp, &stats); err != nil {
		return nil, fmt.Errorf("loading branches: %w", err)
	}
	refs, err := c.branchRefs(repositoryID, fmt.Sprintf("filter=heads/&$top=%d", branchPageSize))
	if err != nil {
		return nil, err
	}
	mine, err := c.branchRefs(repositoryID, "includeMyBranches=true")
	if err != nil {
		mine = nil
	}
	branches := make([]models.Branch, 0, len(stats.Value))
	for _, s := range stats.Value {
		ref := refs[s.Name]
		_, owned := mine[s.Name]
		branches = append(branches, models.Branch{
			Name: s.Name, ObjectID: ref.objectID, Last: s.Commit.ref(), Ahead: s.AheadCount, Behind: s.BehindCount,
			Default: s.IsBaseVersion, Locked: ref.locked, Creator: ref.creator, Mine: owned,
		})
	}
	sort.SliceStable(branches, func(i, j int) bool {
		if branches[i].Default != branches[j].Default {
			return branches[i].Default
		}
		return strings.ToLower(branches[i].Name) < strings.ToLower(branches[j].Name)
	})
	return branches, nil
}

type branchRef struct {
	objectID string
	creator  string
	locked   bool
}

func (c *Client) branchRefs(repositoryID, query string) (map[string]branchRef, error) {
	resp, err := c.get(repositoryPath(repositoryID) + "/refs?" + query)
	if err != nil {
		return nil, fmt.Errorf("loading branches: %w", err)
	}
	var data struct {
		Value []struct {
			Name     string `json:"name"`
			ObjectID string `json:"objectId"`
			IsLocked bool   `json:"isLocked"`
			Creator  struct {
				DisplayName string `json:"displayName"`
			} `json:"creator"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return nil, fmt.Errorf("loading branches: %w", err)
	}
	out := make(map[string]branchRef, len(data.Value))
	for _, r := range data.Value {
		if name, ok := strings.CutPrefix(r.Name, branchRefPrefix); ok {
			out[name] = branchRef{objectID: r.ObjectID, creator: r.Creator.DisplayName, locked: r.IsLocked}
		}
	}
	return out, nil
}

func (c *Client) ResolveCommit(repositoryID string, v models.GitVersion) (string, error) {
	if v.Kind == models.VersionCommit {
		return v.Name, nil
	}
	q := url.Values{}
	q.Set("searchCriteria.itemVersion.versionType", v.Type())
	q.Set("searchCriteria.itemVersion.version", v.Name)
	q.Set("searchCriteria.$top", "1")
	resp, err := c.get(repositoryPath(repositoryID) + "/commits?" + q.Encode())
	if err != nil {
		return "", fmt.Errorf("finding the commit of %s: %w", v.Name, err)
	}
	var data struct {
		Value []struct {
			CommitID string `json:"commitId"`
		} `json:"value"`
	}
	if err := decode(resp, &data); err != nil {
		return "", fmt.Errorf("finding the commit of %s: %w", v.Name, err)
	}
	if len(data.Value) == 0 {
		return "", fmt.Errorf("%s %s has no commits", v.Type(), v.Name)
	}
	return data.Value[0].CommitID, nil
}

func (c *Client) CreateBranch(repositoryID, name, commitID string) error {
	return c.updateRef(repositoryID, refUpdate{Name: branchRefPrefix + name, OldObjectID: zeroObjectID, NewObjectID: commitID}, "creating "+name)
}

func (c *Client) DeleteBranch(repositoryID, name, objectID string) error {
	return c.updateRef(repositoryID, refUpdate{Name: branchRefPrefix + name, OldObjectID: objectID, NewObjectID: zeroObjectID}, "deleting "+name)
}

func (c *Client) updateRef(repositoryID string, update refUpdate, action string) error {
	var reply struct {
		Value []struct {
			Success       bool   `json:"success"`
			UpdateStatus  string `json:"updateStatus"`
			CustomMessage string `json:"customMessage"`
		} `json:"value"`
	}
	if err := c.exchangeJSON("POST", repositoryPath(repositoryID)+"/refs", []refUpdate{update}, &reply); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	if len(reply.Value) == 0 || reply.Value[0].Success {
		return nil
	}
	r := reply.Value[0]
	reason := r.CustomMessage
	if reason == "" {
		reason = r.UpdateStatus
	}
	return fmt.Errorf("%s: %s", action, reason)
}
