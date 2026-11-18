package ui

import (
	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/models"
)

type dataLoadedMsg struct {
	iterations   []models.Iteration
	areas        []models.Area
	statesByType map[string][]models.WorkItemStateInfo
	teamMembers  []models.TeamMember
	me           models.Identity
	savedFilters *config.FilterState
}

type workItemsLoadedMsg struct {
	seq       int
	items     []models.WorkItem
	truncated bool
	limit     int
}

type errMsg struct {
	err error
}

type commentsDueMsg struct {
	workItemID int
}

type commentsLoadedMsg struct {
	workItemID int
	comments   []models.Comment
	err        error
}

type workItemChangedMsg struct {
	notice    string
	deletedID int
	item      *models.WorkItem
}

type workItemMovedMsg struct {
	id    int
	state string
	item  *models.WorkItem
}

type workItemMoveFailedMsg struct {
	id    int
	state string
	err   error
}

type identityLoadedMsg struct {
	me models.Identity
}

type detailFetchedMsg struct {
	item models.WorkItem
}

type commentChangedMsg struct {
	workItemID int
	notice     string
}

type copiedMsg struct {
	notice string
}

type workItemQuery struct {
	sprint   string
	state    string
	assigned string
	area     string
}

func queryFor(fs *models.FilterState, me models.Identity) workItemQuery {
	if fs == nil {
		return workItemQuery{}
	}
	assigned := models.FilterAll
	if fs.GetSelectedAssigned() == models.AssignedMe && !me.Known() {
		assigned = models.AssignedMe
	}
	return workItemQuery{
		sprint:   fs.GetSelectedSprint(),
		state:    fs.GetSelectedState(),
		assigned: assigned,
		area:     fs.GetSelectedArea(),
	}
}
