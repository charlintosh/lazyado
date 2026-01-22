package models

import "time"

// Iteration represents an Azure DevOps iteration (sprint)
type Iteration struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	StartDate  time.Time `json:"startDate"`
	FinishDate time.Time `json:"finishDate"`
	TimeFrame  string    `json:"timeFrame"` // "past", "current", "future"
	URL        string    `json:"url"`
}

// IsCurrent returns true if this is the current iteration
func (i *Iteration) IsCurrent() bool {
	return i.TimeFrame == "current"
}

// IsPast returns true if this iteration is in the past
func (i *Iteration) IsPast() bool {
	return i.TimeFrame == "past"
}

// IsFuture returns true if this iteration is in the future
func (i *Iteration) IsFuture() bool {
	return i.TimeFrame == "future"
}

// DisplayName returns a formatted name for display
func (i *Iteration) DisplayName() string {
	if i.IsCurrent() {
		return i.Name + " (current)"
	}
	return i.Name
}
