package models

import "time"

// Comment represents a work item comment
type Comment struct {
	ID           int       `json:"id"`
	Text         string    `json:"text"`
	CreatedBy    string    `json:"createdBy"`
	CreatedDate  time.Time `json:"createdDate"`
	ModifiedBy   string    `json:"modifiedBy,omitempty"`
	ModifiedDate time.Time `json:"modifiedDate,omitempty"`
}
