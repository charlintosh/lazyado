package ui

import (
	"slices"
	"strings"

	"github.com/charlintosh/lazyado/internal/models"
)

func (a *App) applyUpdated(updated models.WorkItem) bool {
	i := slices.IndexFunc(a.workItems, func(w models.WorkItem) bool { return w.ID == updated.ID })
	if i < 0 {
		a.refreshDetail(updated)
		return !a.loadedQuery.admits(updated)
	}
	old := a.workItems[i]
	if updated.ParentID != old.ParentID || !a.loadedQuery.admits(updated) {
		return false
	}
	updated.ParentTitle, updated.Children = old.ParentTitle, old.Children
	for _, child := range updated.Children {
		child.ParentTitle = updated.Title
	}
	a.workItems[i] = updated
	a.applyView()
	if a.detailID() == updated.ID {
		a.detailView.Refresh(updated)
	}
	return true
}

func (a *App) refreshDetail(updated models.WorkItem) {
	if a.detailID() != updated.ID {
		return
	}
	if current := a.detailView.Item(); current != nil {
		updated.Children = current.Children
		if updated.ParentID == current.ParentID {
			updated.ParentTitle = current.ParentTitle
		}
	}
	a.detailView.Refresh(updated)
}

func (q workItemQuery) admits(item models.WorkItem) bool {
	all := func(v string) bool { return v == "" || v == models.FilterAll }
	switch {
	case q.assigned == models.AssignedMe:
		return false
	case !all(q.state) && string(item.State) != q.state:
		return false
	case !all(q.sprint) && item.IterationPath != q.sprint:
		return false
	}
	if all(q.area) {
		return true
	}
	area := strings.Trim(q.area, `\`)
	return item.AreaPath == area || strings.HasPrefix(item.AreaPath, area+`\`)
}
