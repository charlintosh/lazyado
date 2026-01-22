package api

import (
	"fmt"

	"github.com/charlintosh/lazyado/internal/models"
)

// teamMembersResponse represents the response from team members API
type teamMembersResponse struct {
	Value []teamMemberItem `json:"value"`
	Count int              `json:"count"`
}

type teamMemberItem struct {
	Identity identityRef `json:"identity"`
}

type identityRef struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	ID          string `json:"id"`
}

// GetTeamMembers fetches all members of the configured team
func (c *Client) GetTeamMembers() ([]models.TeamMember, error) {
	// Azure DevOps API: GET https://dev.azure.com/{org}/_apis/projects/{project}/teams/{team}/members
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/projects/%s/teams/%s/members?api-version=%s",
		c.organization, c.project, c.team, apiVersion)

	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var apiResp teamMembersResponse
	if err := decode(resp, &apiResp); err != nil {
		return nil, err
	}

	members := make([]models.TeamMember, 0, len(apiResp.Value))
	for _, item := range apiResp.Value {
		members = append(members, models.TeamMember{
			ID:          item.Identity.ID,
			DisplayName: item.Identity.DisplayName,
			UniqueName:  item.Identity.UniqueName,
		})
	}

	return members, nil
}
