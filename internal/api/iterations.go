package api

import (
	"time"

	"github.com/charlintosh/lazyado/internal/models"
)

// iterationsResponse represents the API response for iterations
type iterationsResponse struct {
	Count int                `json:"count"`
	Value []iterationAPIItem `json:"value"`
}

type iterationAPIItem struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	Attributes iterationAttrs `json:"attributes"`
	URL        string         `json:"url"`
}

type iterationAttrs struct {
	StartDate  *time.Time `json:"startDate"`
	FinishDate *time.Time `json:"finishDate"`
	TimeFrame  string     `json:"timeFrame"`
}

// GetIterations fetches all iterations (sprints) for the team
func (c *Client) GetIterations() ([]models.Iteration, error) {
	resp, err := c.getTeam("/work/teamsettings/iterations")
	if err != nil {
		return nil, err
	}

	var apiResp iterationsResponse
	if err := decode(resp, &apiResp); err != nil {
		return nil, err
	}

	iterations := make([]models.Iteration, 0, apiResp.Count)
	for _, item := range apiResp.Value {
		iter := models.Iteration{
			ID:        item.ID,
			Name:      item.Name,
			Path:      item.Path,
			TimeFrame: item.Attributes.TimeFrame,
			URL:       item.URL,
		}

		if item.Attributes.StartDate != nil {
			iter.StartDate = *item.Attributes.StartDate
		}
		if item.Attributes.FinishDate != nil {
			iter.FinishDate = *item.Attributes.FinishDate
		}

		iterations = append(iterations, iter)
	}

	return iterations, nil
}

// GetCurrentIteration returns the current iteration
func (c *Client) GetCurrentIteration() (*models.Iteration, error) {
	iterations, err := c.GetIterations()
	if err != nil {
		return nil, err
	}

	for _, iter := range iterations {
		if iter.IsCurrent() {
			return &iter, nil
		}
	}

	// Return the first one if no current found
	if len(iterations) > 0 {
		return &iterations[0], nil
	}

	return nil, nil
}
