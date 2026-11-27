package ui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
)

type itemAction int

const (
	actionNone itemAction = iota
	actionView
	actionState
	actionAssign
	actionComment
	actionTask
	actionEdit
	actionDelete
	actionShare
	actionShareLink
)

var detailActions = []itemAction{actionState, actionAssign, actionTask, actionEdit, actionDelete, actionShare, actionShareLink}

var panelActions = map[Panel][]itemAction{
	PanelWorkItems: {actionState, actionAssign, actionComment, actionTask, actionEdit, actionDelete, actionShare, actionShareLink},
	PanelDetails:   {actionView, actionState, actionAssign, actionComment, actionEdit, actionShare, actionShareLink},
	PanelComments:  {actionComment, actionShare, actionShareLink},
}

func (a App) handleKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	if a.showSplash() && !key.Matches(msg, a.keys.Quit) {
		a.splashSkipped = true
		return a, nil
	}
	if a.modals.open() {
		return a.updateModal(msg)
	}
	if a.keyword.Focused() {
		return a.handleKeywordKey(msg)
	}
	if a.searchingPullRequests() {
		return a.handlePullRequestKey(msg)
	}
	if a.searchingPipelines() {
		return a.handlePipelineKey(msg)
	}
	if a.searchingReleases() {
		return a.handleReleaseKey(msg)
	}
	if a.searchingCommits() {
		return a.handleCommitsKey(msg)
	}
	if a.searchingBranches() {
		return a.handleBranchesKey(msg)
	}
	if a.searchingTags() {
		return a.handleTagsKey(msg)
	}
	if a.notification.IsVisible() && (key.Matches(msg, a.keys.Back) || key.Matches(msg, a.keys.DismissNotification)) {
		a.notification = a.notification.Dismiss()
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Quit) && a.viewMode == ViewMain:
		return a, tea.Quit
	case key.Matches(msg, a.keys.Help):
		return a, a.openModal(modals.NewHelpModal(a.env()))
	case key.Matches(msg, a.keys.Refresh):
		return a, a.refresh()
	}
	if a.reviewing() {
		var cmd tea.Cmd
		a.pr.view, cmd = a.pr.view.Update(msg)
		return a, cmd
	}
	if a.viewingRun() {
		var cmd tea.Cmd
		a.pl.view, cmd = a.pl.view.Update(msg)
		return a, cmd
	}
	if a.viewingRelease() {
		var cmd tea.Cmd
		a.rl.view, cmd = a.rl.view.Update(msg)
		return a, cmd
	}
	if a.viewingReleaseEditor() {
		var cmd tea.Cmd
		a.rl.editor, cmd = a.rl.editor.Update(msg)
		return a, cmd
	}
	if a.viewMode == ViewDetail {
		if cmd, ok := a.handleDetailAction(msg); ok {
			return a, cmd
		}
		var cmd tea.Cmd
		a.detailView, cmd = a.detailView.Update(msg)
		return a, cmd
	}
	if a.inPullRequests() {
		return a.handlePullRequestKey(msg)
	}
	if a.inPipelines() {
		return a.handlePipelineKey(msg)
	}
	if a.inReleases() {
		return a.handleReleaseKey(msg)
	}
	if a.inFiles() {
		return a.handleFilesKey(msg)
	}
	if a.inCommits() {
		return a.handleCommitsKey(msg)
	}
	if a.inBranches() {
		return a.handleBranchesKey(msg)
	}
	if a.inTags() {
		return a.handleTagsKey(msg)
	}
	if next, cmd, ok := a.handleNavigationKey(msg); ok {
		return next, cmd
	}
	if cmd, ok := a.handleActionKey(msg); ok {
		return a, cmd
	}
	return a.updateActivePanel(msg)
}

func (a App) handleKeywordKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	if key.Matches(msg, a.keys.Back) || key.Matches(msg, a.keys.Select) {
		a.keyword.Blur()
		return a, nil
	}
	return a.updateKeyword(msg)
}

func (a App) updateKeyword(msg tea.Msg) (App, tea.Cmd) {
	before := a.keyword.Value()
	var cmd tea.Cmd
	a.keyword, cmd = a.keyword.Update(msg)
	if a.keyword.Value() != before {
		a.applyView()
	}
	return a, cmd
}

func (a App) handleNavigationKey(msg tea.KeyPressMsg) (App, tea.Cmd, bool) {
	k := a.keys
	for group, b := range a.filterKeys() {
		if key.Matches(msg, b) {
			return a, a.openFilterPicker(group), true
		}
	}
	if hub, ok := a.hubFor(msg); ok {
		return a, a.goToHub(hub), true
	}
	switch {
	case key.Matches(msg, k.PrevView):
		a.cycleView(-1)
	case key.Matches(msg, k.NextView):
		a.cycleView(1)
	case key.Matches(msg, k.FilterBar):
		a.filterBar = !a.filterBar
	case key.Matches(msg, k.Search):
		a.filterBar = true
		return a, a.keyword.Focus(), true
	case key.Matches(msg, k.ClearFilters):
		return a, a.clearFilters(), true
	case key.Matches(msg, k.Fullscreen):
		a.fullscreen = !a.fullscreen
	case key.Matches(msg, k.Panel5):
		a.activePanel = PanelWorkItems
	case key.Matches(msg, k.Panel6):
		a.activePanel = PanelDetails
	case key.Matches(msg, k.Panel7):
		a.activePanel = PanelComments
	case key.Matches(msg, k.NextPanel):
		a.activePanel = (a.activePanel + 1) % panelCount
	case key.Matches(msg, k.PrevPanel):
		a.activePanel = (a.activePanel + panelCount - 1) % panelCount
	case key.Matches(msg, k.QuickSearch):
		return a, a.openModal(modals.NewQuickSearchModal(a.env())), true
	default:
		return a, nil, false
	}
	return a, nil, true
}

func (a *App) refresh() tea.Cmd {
	if a.reviewing() {
		return a.refreshReview()
	}
	if a.viewingRun() {
		return a.refreshRun()
	}
	if a.viewingRelease() {
		return a.refreshRelease()
	}
	if a.viewingReleaseEditor() {
		return a.refreshReleaseEditor()
	}
	if a.inPullRequests() {
		return a.refreshPullRequests()
	}
	if a.inPipelines() {
		return a.refreshPipelines()
	}
	if a.inReleases() {
		return a.refreshReleases()
	}
	if a.inFiles() {
		return a.refreshFiles()
	}
	if a.inCommits() {
		return a.refreshCommits()
	}
	if a.inBranches() {
		return a.refreshBranches()
	}
	if a.inTags() {
		return a.refreshTags()
	}
	if !a.me.Known() {
		return tea.Batch(a.reload(), loadIdentityCmd(a.client))
	}
	return a.reload()
}

func (a *App) switchHub(hub panels.Hub) {
	a.hub = hub
	a.activePanel = PanelWorkItems
}

func (a *App) cycleView(delta int) {
	if a.hub != panels.HubSprint {
		return
	}
	a.sprintView = (a.sprintView + delta + len(sprintPivots)) % len(sprintPivots)
	a.activePanel = PanelWorkItems
}

func (a *App) handleActionKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if key.Matches(msg, a.keys.CreateParent) && a.activePanel == PanelWorkItems {
		return a.openModal(modals.NewParentTypeModal(a.env())), true
	}
	action := a.actionFor(msg)
	item := a.selectedItem()
	if action == actionNone || item == nil || !a.allows(action) {
		return nil, false
	}
	return a.runAction(action, *item), true
}

func (a *App) handleDetailAction(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	item := a.detailView.Item()
	action := a.actionFor(msg)
	if item == nil || !slices.Contains(detailActions, action) {
		return nil, false
	}
	if a.detailView.CommentFocused() && (action == actionEdit || action == actionDelete) {
		return nil, false
	}
	return a.runAction(action, *item), true
}

func (a *App) runAction(action itemAction, item models.WorkItem) tea.Cmd {
	switch action {
	case actionView:
		return a.openDetail(item)
	case actionState:
		return a.openModal(modals.NewStateModal(a.env(), item, a.statesByType))
	case actionAssign:
		return a.openModal(modals.NewAssignModal(a.env(), item, a.teamMembers))
	case actionComment:
		return a.openModal(modals.NewCommentModal(a.env(), item, a.teamMembers))
	case actionTask:
		if !item.IsParentType() {
			return nil
		}
		return a.openModal(modals.NewCreateTaskForm(a.env(), item))
	case actionEdit:
		return a.openModal(a.editForm(item))
	case actionDelete:
		return a.openModal(modals.NewDeleteWorkItemModal(a.env(), item))
	case actionShare:
		return a.share(share.WorkItem(item), false)
	case actionShareLink:
		return a.share(share.WorkItem(item), true)
	}
	return nil
}

func (a *App) actionFor(msg tea.KeyPressMsg) itemAction {
	k := a.keys
	bindings := []struct {
		binding key.Binding
		action  itemAction
	}{
		{k.View, actionView},
		{k.ChangeState, actionState},
		{k.Assign, actionAssign},
		{k.AddComment, actionComment},
		{k.CreateTask, actionTask},
		{k.EditTask, actionEdit},
		{k.DeleteTask, actionDelete},
		{k.Share, actionShare},
		{k.ShareLink, actionShareLink},
	}
	for _, b := range bindings {
		if key.Matches(msg, b.binding) {
			return b.action
		}
	}
	return actionNone
}

func (a *App) allows(action itemAction) bool {
	for _, allowed := range panelActions[a.activePanel] {
		if allowed == action {
			return true
		}
	}
	return false
}

func (a *App) editForm(item models.WorkItem) modals.Modal {
	if item.HasAdvancedFields() {
		return modals.NewEditWorkItemForm(a.env(), item, a.teamMembers)
	}
	return modals.NewEditTaskForm(a.env(), item)
}

func (a App) updateActivePanel(msg tea.KeyPressMsg) (App, tea.Cmd) {
	var cmd tea.Cmd
	switch a.activePanel {
	case PanelWorkItems:
		cmd = a.updateMainPanel(msg)
	case PanelDetails:
		a.detailsPanel, cmd = a.detailsPanel.Update(msg)
	case PanelComments:
		a.commentsPanel, cmd = a.commentsPanel.Update(msg)
	}
	return a, cmd
}
