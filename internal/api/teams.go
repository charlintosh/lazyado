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

type connectionUser struct {
	ID                  string `json:"id"`
	ProviderDisplayName string `json:"providerDisplayName"`
	Properties          struct {
		Account struct {
			Value string `json:"$value"`
		} `json:"Account"`
	} `json:"properties"`
}

type connectionDataResponse struct {
	AuthenticatedUser connectionUser `json:"authenticatedUser"`
	AuthorizedUser    connectionUser `json:"authorizedUser"`
}

func (c *Client) GetCurrentUser() (models.Identity, error) {
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/connectionData", c.organization)
	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return models.Identity{}, err
	}
	var data connectionDataResponse
	if err := decode(resp, &data); err != nil {
		return models.Identity{}, err
	}
	user := data.AuthenticatedUser
	if user.ID == "" {
		user = data.AuthorizedUser
	}
	if user.ID == "" {
		return models.Identity{}, fmt.Errorf("connection data did not name the signed-in user")
	}
	return models.Identity{ID: user.ID, DisplayName: user.ProviderDisplayName, UniqueName: user.Properties.Account.Value}, nil
}
