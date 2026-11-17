package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/models"
)

func (a App) handleResult(msg tea.Msg) (App, tea.Cmd, bool) {
	if next, cmd, ok := a.handleData(msg); ok {
		return next, cmd, true
	}
	if next, cmd, ok := a.handlePullRequestResult(msg); ok {
		return next, cmd, true
	}
	if next, cmd, ok := a.handlePipelineResult(msg); ok {
		return next, cmd, true
	}
	if next, cmd, ok := a.handleReleaseHubResult(msg); ok {
		return next, cmd, true
	}
	if next, cmd, ok := a.handleCodeResult(msg); ok {
		return next, cmd, true
	}
	if next, cmd, ok := a.handlePanelIntent(msg); ok {
		return next, cmd, true
	}
	return a.handleModalIntent(msg)
}

func (a App) handleData(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
		a.iterations = msg.iterations
		a.areas = msg.areas
		a.statesByType = msg.statesByType
		a.teamMembers = msg.teamMembers
		a.pr.view.SetMembers(a.teamMembers)
		a.pr.panel.SetMembers(a.teamMembers)
		registerWorkflowColors(a.statesByType)
		filters := models.NewFilterState(a.iterations, a.areas, a.statesByType)
		if s := msg.savedFilters; s != nil {
			filters.ApplySavedSelections(s.Sprint, s.State, s.Assigned, s.Area)
		}
		a.filters = filters
		if s := msg.savedFilters; s != nil {
			a.pr.savedRepo, a.pr.favorites = s.Repository, s.FavoriteRepositories
			a.code.savedRepo = s.CodeRepository
			a.rl.localStars = s.FavoriteReleases
		}
		a.setIdentity(msg.me)
		return a, a.reload(), true
	case identityLoadedMsg:
		a.setIdentity(msg.me)
		a.filters.SetPeople(a.workItems, a.teamMembers, a.me)
		if queryFor(a.filters, a.me) != a.loadedQuery {
			return a, a.reload(), true
		}
		a.applyView()
		return a, nil, true
	case workItemsLoadedMsg:
		if msg.seq != a.loadSeq {
			return a, nil, true
		}
		a.loading = false
		a.initialLoadDone = true
		a.workItems = msg.items
		a.filters.SetPeople(a.workItems, a.teamMembers, a.me)
		a.applyView()
		if item, ok := a.loadedItem(a.detailID()); ok {
			a.detailView.Refresh(item)
		}
		return a, nil, true
	case errMsg:
		a.loading = false
		return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
	case commentsDueMsg:
		if msg.workItemID != a.commentsFor {
			return a, nil, true
		}
		return a, loadCommentsCmd(a.client, msg.workItemID), true
	case commentsLoadedMsg:
		a.commentsPanel.SetComments(msg.workItemID, msg.comments)
		a.detailView.SetComments(msg.workItemID, msg.comments)
		return a, nil, true
	case workItemChangedMsg:
		cmds := []tea.Cmd{a.notify(msg.notice), a.reload()}
		id := a.detailID()
		_, loaded := a.loadedItem(id)
		switch {
		case id != 0 && id == msg.deletedID:
			a.viewMode = ViewMain
		case id != 0 && !loaded:
			cmds = append(cmds, fetchDetailCmd(a.client, id))
		}
		return a, tea.Batch(cmds...), true
	case workItemMovedMsg:
		flash := a.taskboardPanel.MoveSaved(msg.id)
		return a, tea.Batch(a.notify("State changed to "+msg.state), flash, a.reload()), true
	case workItemMoveFailedMsg:
		a.loading = false
		if !a.taskboardPanel.HasPendingMove(msg.id) {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		flash := a.taskboardPanel.MoveFailed(msg.id)
		var notice tea.Cmd
		a.notification, notice = a.notification.Show(notification.NotificationError,
			fmt.Sprintf("Couldn't move #%d to %s: %v", msg.id, msg.state, msg.err))
		return a, tea.Batch(notice, flash), true
	case detailFetchedMsg:
		if _, loaded := a.loadedItem(msg.item.ID); !loaded && a.detailID() == msg.item.ID {
			msg.item.Children = a.detailView.Item().Children
			a.detailView.Refresh(msg.item)
		}
		return a, nil, true
	case commentChangedMsg:
		a.loading = false
		notice := a.notify(msg.notice)
		return a, tea.Batch(notice, loadCommentsCmd(a.client, msg.workItemID)), true
	case copiedMsg:
		return a, a.notify(msg.notice), true
	case notification.AutoDismissMsg:
		a.notification = a.notification.Update(msg)
		return a, nil, true
	}
	return a, nil, false
}

func (a App) handlePanelIntent(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case panels.OpenWorkItemMsg:
		return a, openBrowserCmd(msg.Item.WebURL), true
	case panels.ViewWorkItemMsg:
		return a, a.openDetail(msg.Item), true
	case panels.CloseDetailViewMsg:
		a.viewMode = ViewMain
		if a.detailFromPR {
			a.viewMode = ViewPullRequest
		}
		a.detailFromPR = false
		return a, nil, true
	case panels.OpenCommentModalMsg:
		return a, a.openModal(modals.NewCommentModal(a.env(), msg.Item, a.teamMembers)), true
	case panels.EditCommentMsg:
		if !a.me.Owns(msg.Comment) {
			return a, a.commentReadOnly(), true
		}
		return a, a.openModal(modals.NewEditCommentModal(a.env(), msg.WorkItemID, msg.Comment, a.teamMembers)), true
	case panels.DeleteCommentMsg:
		if !a.me.Owns(msg.Comment) {
			return a, a.commentReadOnly(), true
		}
		return a, a.openModal(modals.NewDeleteCommentModal(a.env(), msg.WorkItemID, msg.Comment)), true
	case panels.CommentReadOnlyMsg:
		return a, a.commentReadOnly(), true
	case panels.MoveWorkItemMsg:
		return a, a.mutate(moveWorkItemCmd(a.client, msg.Item.ID, msg.NewState)), true
	case panels.LikeCommentMsg:
		return a, a.mutate(likeCommentCmd(a.client, msg.WorkItemID, msg.Comment, msg.Like)), true
	case panels.MoveFlashExpiredMsg:
		a.taskboardPanel.ExpireFlash(msg)
		return a, nil, true
	}
	return a, nil, false
}

func (a App) handleModalIntent(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case modals.ModalClosedMsg:
		a.closeModal()
		return a, nil, true
	case modals.QuickSearchRequestMsg:
		return a, quickSearchCmd(a.client, msg.Query), true
	case modals.QuickSearchSuccessMsg:
		switch {
		case !topIs[modals.QuickSearchModal](a.modals):
			return a, nil, true
		case msg.Item != nil && msg.PullRequest != nil:
			next, cmd := a.updateModal(msg)
			return next, cmd, true
		}
		a.closeModal()
		if msg.PullRequest != nil {
			return a, a.openReview(*msg.PullRequest), true
		}
		return a, a.openDetail(*msg.Item), true
	case modals.FilterOptionChosenMsg:
		a.closeModal()
		if msg.Group >= 0 && msg.Group < len(a.filters.Groups) {
			a.filters.Groups[msg.Group].SelectByValue(msg.Value)
		}
		return a, a.applyFilters(), true
	case modals.ParentTypeChosenMsg:
		a.closeModal()
		if msg.Type == models.WorkItemTypeBug {
			return a, a.openModal(modals.NewCreateBugForm(a.env(), a.formData())), true
		}
		return a, a.openModal(modals.NewCreatePBIForm(a.env(), a.formData())), true
	case modals.CopyRequestMsg:
		a.closeModal()
		return a, copyCmd(msg.Text, msg.Notice), true
	}
	if cmd, ok := a.mutationFor(msg); ok {
		a.closeModal()
		return a, a.mutate(cmd), true
	}
	return a, nil, false
}

func (a *App) mutationFor(msg tea.Msg) (tea.Cmd, bool) {
	c := a.client
	switch msg := msg.(type) {
	case modals.StateChangeRequestMsg:
		return updateWorkItemStateCmd(c, msg.Item.ID, msg.NewState), true
	case modals.AssignRequestMsg:
		return assignWorkItemCmd(c, msg.Item.ID, msg.UserEmail, msg.UserName), true
	case modals.ParentCreateRequestMsg:
		return createParentWorkItemCmd(c, a.withCreateDefaults(msg)), true
	case modals.WorkItemUpdateRequestMsg:
		return updateWorkItemCmd(c, msg), true
	case modals.TaskCreateRequestMsg:
		return createTaskCmd(c, msg.ParentID, msg.Title, msg.Description), true
	case modals.TaskUpdateRequestMsg:
		return updateTaskCmd(c, msg.TaskID, msg.Title, msg.Description), true
	case modals.WorkItemDeleteRequestMsg:
		return deleteWorkItemCmd(c, msg.ID), true
	case modals.CommentCreateRequestMsg:
		return createCommentCmd(c, msg.WorkItemID, msg.Text), true
	case modals.CommentUpdateRequestMsg:
		return updateCommentCmd(c, msg.WorkItemID, msg.CommentID, msg.Text), true
	case modals.CommentDeleteRequestMsg:
		return deleteCommentCmd(c, msg.WorkItemID, msg.CommentID), true
	}
	return nil, false
}

func (a *App) setIdentity(me models.Identity) {
	a.me = me
	a.commentsPanel.SetIdentity(me)
	a.detailView.SetIdentity(me)
	a.pr.panel.SetIdentity(me)
	a.pr.view.SetIdentity(me)
	a.rl.view.SetIdentity(me)
	a.applyPullRequests()
}

func (a *App) commentReadOnly() tea.Cmd {
	text := "Only the author can edit or delete this comment."
	if !a.me.Known() {
		text = "Couldn't confirm your Azure DevOps identity from the PAT, so comments are read-only. Press " +
			a.keys.Refresh.Help().Key + " to retry."
	}
	var cmd tea.Cmd
	a.notification, cmd = a.notification.Show(notification.NotificationInfo, text)
	return cmd
}

func (a *App) detailID() int {
	if item := a.detailView.Item(); item != nil && a.viewMode == ViewDetail {
		return item.ID
	}
	return 0
}

func (a *App) loadedItem(id int) (models.WorkItem, bool) {
	for i := range a.workItems {
		if a.workItems[i].ID == id {
			return a.workItems[i], true
		}
	}
	return models.WorkItem{}, false
}

func (a *App) openDetail(item models.WorkItem) tea.Cmd {
	a.detailFromPR = a.reviewing()
	a.viewMode = ViewDetail
	a.detailView.SetItem(&item)
	return loadCommentsCmd(a.client, item.ID)
}

func (a *App) applyFilters() tea.Cmd {
	fs := a.filters
	save := saveFiltersCmd(a.savedState())
	if queryFor(fs, a.me) != a.loadedQuery {
		return tea.Batch(save, a.reload())
	}
	a.applyView()
	return save
}

func (a *App) savedState() config.FilterState {
	fs := a.filters
	return config.FilterState{
		Sprint:     fs.GetSelectedSprint(),
		State:      fs.GetSelectedState(),
		Assigned:   fs.GetSelectedAssigned(),
		Area:       fs.GetSelectedArea(),
		Repository: a.pr.savedScope(),

		CodeRepository: a.code.savedName(),

		FavoriteRepositories: a.pr.favorites,
		FavoriteReleases:     a.rl.localStars,
	}
}

func (a *App) clearFilters() tea.Cmd {
	a.keyword.SetValue("")
	a.keyword.Blur()
	a.filters.Reset(models.FilterTypeAssigned, models.FilterTypeState, models.FilterTypeArea)
	return a.applyFilters()
}

func (a *App) formData() modals.FormData {
	area, iteration := a.createDefaults()
	return modals.FormData{
		Members:          a.teamMembers,
		Iterations:       a.iterations,
		DefaultArea:      area,
		DefaultIteration: iteration,
	}
}

func (a *App) createDefaults() (string, string) {
	var area, iteration string
	if len(a.workItems) > 0 {
		area, iteration = a.workItems[0].AreaPath, a.workItems[0].IterationPath
	}
	if iteration == "" {
		for _, it := range a.iterations {
			if it.IsCurrent() {
				iteration = it.Path
			}
		}
	}
	return area, iteration
}

func (a *App) withCreateDefaults(req modals.ParentCreateRequestMsg) modals.ParentCreateRequestMsg {
	area, iteration := a.createDefaults()
	if req.AreaPath == "" {
		req.AreaPath = area
	}
	if req.IterationPath == "" {
		req.IterationPath = iteration
	}
	return req
}
