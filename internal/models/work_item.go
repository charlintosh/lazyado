package models

import (
	"slices"
	"time"
)

// WorkItemType represents the type of work item
type WorkItemType string

const (
	WorkItemTypeStory   WorkItemType = "User Story"
	WorkItemTypePBI     WorkItemType = "Product Backlog Item"
	WorkItemTypeTask    WorkItemType = "Task"
	WorkItemTypeBug     WorkItemType = "Bug"
	WorkItemTypeFeature WorkItemType = "Feature"
	WorkItemTypeEpic    WorkItemType = "Epic"
)

var (
	// parentWorkItemTypes are work item types that can have children
	parentWorkItemTypes = []WorkItemType{
		WorkItemTypeStory,
		WorkItemTypePBI,
		WorkItemTypeBug,
		WorkItemTypeFeature,
		WorkItemTypeEpic,
	}

	// advancedWorkItemTypes are work item types with advanced fields like tags and assignee
	advancedWorkItemTypes = []WorkItemType{
		WorkItemTypeStory,
		WorkItemTypePBI,
		WorkItemTypeBug,
		WorkItemTypeFeature,
		WorkItemTypeEpic,
	}
)

// WorkItemState represents the state of a work item
type WorkItemState string

const (
	WorkItemStateNew      WorkItemState = "New"
	WorkItemStateActive   WorkItemState = "Active"
	WorkItemStateResolved WorkItemState = "Resolved"
	WorkItemStateClosed   WorkItemState = "Closed"
)

// WorkItem represents an Azure DevOps work item
type WorkItem struct {
	ID                 int           `json:"id"`
	Rev                int           `json:"rev"`
	Title              string        `json:"title"`
	State              WorkItemState `json:"state"`
	Type               WorkItemType  `json:"type"`
	AssignedTo         string        `json:"assignedTo"`
	IterationPath      string        `json:"iterationPath"`
	AreaPath           string        `json:"areaPath"`
	Description        string        `json:"description"`
	AcceptanceCriteria string        `json:"acceptanceCriteria"`
	Tags               []string      `json:"tags"`
	ParentID           int           `json:"parentId"`
	ParentTitle        string        `json:"parentTitle"`
	Priority           int           `json:"priority"`
	Effort             float64       `json:"effort"`
	CreatedDate        time.Time     `json:"createdDate"`
	ChangedDate        time.Time     `json:"changedDate"`
	URL                string        `json:"url"`
	WebURL             string        `json:"webUrl"`
	Children           []*WorkItem   `json:"-"` // Child work items (not from API)
	IsExpanded         bool          `json:"-"` // For UI state
}

// ShortType returns a short version of the work item type
func (w *WorkItem) ShortType() string {
	switch w.Type {
	case WorkItemTypeStory:
		return "Story"
	case WorkItemTypePBI:
		return "PBI"
	case WorkItemTypeTask:
		return "Task"
	case WorkItemTypeBug:
		return "Bug"
	case WorkItemTypeFeature:
		return "Feature"
	case WorkItemTypeEpic:
		return "Epic"
	default:
		return string(w.Type)
	}
}

// SprintName extracts the sprint name from the iteration path
func (w *WorkItem) SprintName() string {
	// IterationPath is like "MyProject\\Sprint 42"
	// Return just "Sprint 42"
	for i := len(w.IterationPath) - 1; i >= 0; i-- {
		if w.IterationPath[i] == '\\' {
			return w.IterationPath[i+1:]
		}
	}
	return w.IterationPath
}

// AreaName extracts the area name from the area path
func (w *WorkItem) AreaName() string {
	for i := len(w.AreaPath) - 1; i >= 0; i-- {
		if w.AreaPath[i] == '\\' {
			return w.AreaPath[i+1:]
		}
	}
	return w.AreaPath
}

// HasChildren returns true if the work item has child items
func (w *WorkItem) HasChildren() bool {
	return len(w.Children) > 0
}

// IsParentType returns true if this work item type can have children
func (w *WorkItem) IsParentType() bool {
	return slices.Contains(parentWorkItemTypes, w.Type)
}

// HasAdvancedFields returns true if this work item type has advanced fields like tags and assignee
// PBI, Bug, Story, Feature, Epic all have these fields
func (w *WorkItem) HasAdvancedFields() bool {
	return slices.Contains(advancedWorkItemTypes, w.Type)
}

// WorkItemStateInfo represents state metadata from Azure DevOps
type WorkItemStateInfo struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Category string `json:"category"` // Proposed, InProgress, Resolved, Completed, Removed
}
