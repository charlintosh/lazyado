package models

import "time"

// Comment represents a work item comment
type Comment struct {
	ID           int              `json:"id"`
	Text         string           `json:"text"`
	RenderedText string           `json:"renderedText,omitempty"`
	Format       string           `json:"format,omitempty"` // "markdown" or "html"
	Mentions     []CommentMention `json:"mentions,omitempty"`
	CreatedBy    string           `json:"createdBy"`
	CreatedDate  time.Time        `json:"createdDate"`
	ModifiedBy   string           `json:"modifiedBy,omitempty"`
	ModifiedDate time.Time        `json:"modifiedDate,omitempty"`
}

// CommentMention represents a mention in a comment
type CommentMention struct {
	ArtifactID   string `json:"artifactId"`   // The artifact portion (e.g., work item ID)
	ArtifactType string `json:"artifactType"` // Type: "person", "workItem", etc.
	TargetID     string `json:"targetId"`     // Resolved target ID
}
