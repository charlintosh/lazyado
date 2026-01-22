package api

import (
	"fmt"

	"github.com/charlintosh/lazyado/internal/models"
)

// workItemTypesResponse represents the API response for work item types
type workItemTypesResponse struct {
	Count int                   `json:"count"`
	Value []workItemTypeAPIItem `json:"value"`
}

type workItemTypeAPIItem struct {
	Name          string `json:"name"`
	ReferenceName string `json:"referenceName"`
	Description   string `json:"description"`
	Color         string `json:"color"`
	Icon          struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"icon"`
}

// statesResponse represents the API response for work item states
type statesResponse struct {
	Count int            `json:"count"`
	Value []stateAPIItem `json:"value"`
}

type stateAPIItem struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Category string `json:"stateCategory"`
}

// GetWorkItemTypes fetches all work item types for the project
func (c *Client) GetWorkItemTypes() ([]string, error) {
	resp, err := c.get("/wit/workitemtypes")
	if err != nil {
		return nil, err
	}

	var apiResp workItemTypesResponse
	if err := decode(resp, &apiResp); err != nil {
		return nil, err
	}

	types := make([]string, 0, apiResp.Count)
	for _, item := range apiResp.Value {
		types = append(types, item.Name)
	}

	return types, nil
}

// GetWorkItemTypeStates fetches all states for a specific work item type
func (c *Client) GetWorkItemTypeStates(workItemType string) ([]models.WorkItemStateInfo, error) {
	endpoint := fmt.Sprintf("/wit/workitemtypes/%s/states", workItemType)
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}

	var apiResp statesResponse
	if err := decode(resp, &apiResp); err != nil {
		return nil, err
	}

	states := make([]models.WorkItemStateInfo, 0, apiResp.Count)
	for _, item := range apiResp.Value {
		states = append(states, models.WorkItemStateInfo{
			Name:     item.Name,
			Color:    item.Color,
			Category: item.Category,
		})
	}

	return states, nil
}

// GetAllWorkItemTypeStates fetches states for all work item types
func (c *Client) GetAllWorkItemTypeStates() (map[string][]models.WorkItemStateInfo, error) {
	types, err := c.GetWorkItemTypes()
	if err != nil {
		return nil, err
	}

	statesByType := make(map[string][]models.WorkItemStateInfo)
	for _, t := range types {
		states, err := c.GetWorkItemTypeStates(t)
		if err != nil {
			// Skip types that fail (some system types may not have states)
			continue
		}
		statesByType[t] = states
	}

	return statesByType, nil
}
