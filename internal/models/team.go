package models

import "strings"

// TeamMember represents a team member
type TeamMember struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
}

type Identity struct {
	ID          string
	DisplayName string
	UniqueName  string
}

func (i Identity) Known() bool {
	return i.ID != "" || i.DisplayName != "" || i.UniqueName != ""
}

func (i Identity) Owns(c Comment) bool {
	switch {
	case i.ID != "" && c.CreatedByID != "":
		return strings.EqualFold(i.ID, c.CreatedByID)
	case i.UniqueName != "" && c.CreatedByUniqueName != "":
		return strings.EqualFold(i.UniqueName, c.CreatedByUniqueName)
	}
	return false
}

func (i Identity) Matches(displayName, uniqueName string) bool {
	if i.UniqueName != "" && uniqueName != "" {
		return strings.EqualFold(i.UniqueName, uniqueName)
	}
	return i.DisplayName != "" && i.DisplayName == displayName
}
