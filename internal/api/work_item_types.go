package api

import (
	"fmt"
	"sync"

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
	States []stateAPIItem `json:"states"`
}

func (t workItemTypeAPIItem) states() ([]models.WorkItemStateInfo, bool) {
	if len(t.States) == 0 {
		return nil, false
	}
	out := make([]models.WorkItemStateInfo, 0, len(t.States))
	for _, s := range t.States {
		info := s.info()
		if info.Category == "" {
			return nil, false
		}
		out = append(out, info)
	}
	return out, true
}

// statesResponse represents the API response for work item states
type statesResponse struct {
	Count int            `json:"count"`
	Value []stateAPIItem `json:"value"`
}

type stateAPIItem struct {
	Name          string `json:"name"`
	Color         string `json:"color"`
	StateCategory string `json:"stateCategory"`
	Category      string `json:"category"`
}

func (s stateAPIItem) info() models.WorkItemStateInfo {
	category := s.StateCategory
	if category == "" {
		category = s.Category
	}
	return models.WorkItemStateInfo{Name: s.Name, Color: s.Color, Category: category}
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
		states = append(states, item.info())
	}

	return states, nil
}

// GetAllWorkItemTypeStates fetches states for all work item types
func (c *Client) GetAllWorkItemTypeStates() (map[string][]models.WorkItemStateInfo, error) {
	resp, err := c.get("/wit/workitemtypes")
	if err != nil {
		return nil, err
	}
	var apiResp workItemTypesResponse
	if err := decode(resp, &apiResp); err != nil {
		return nil, err
	}

	statesByType := make(map[string][]models.WorkItemStateInfo, len(apiResp.Value))
	var missing []string
	for _, t := range apiResp.Value {
		if states, ok := t.states(); ok {
			statesByType[t.Name] = states
		} else {
			missing = append(missing, t.Name)
		}
	}
	if len(missing) > 0 {
		clientLogger.Debugf("loading states separately for %d work item types", len(missing))
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, name := range missing {
		wg.Go(func() {
			states, err := c.GetWorkItemTypeStates(name)
			if err != nil {
				return
			}
			mu.Lock()
			statesByType[name] = states
			mu.Unlock()
		})
	}
	wg.Wait()
	return statesByType, nil
}
