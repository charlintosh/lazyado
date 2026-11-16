package ui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/pkg/browser"
	"github.com/charlintosh/lazyado/pkg/clipboard"
)

const commentsDebounce = 150 * time.Millisecond

func loadDataCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		iterations, err := client.GetIterations()
		if err != nil {
			return errMsg{err: err}
		}
		areas, err := client.GetAreas()
		if err != nil {
			return errMsg{err: err}
		}
		statesByType, err := client.GetAllWorkItemTypeStates()
		if err != nil {
			statesByType = map[string][]models.WorkItemStateInfo{}
		}
		members, err := client.GetTeamMembers()
		if err != nil {
			members = []models.TeamMember{}
		}
		me, err := client.GetCurrentUser()
		if err != nil {
			me = models.Identity{}
		}
		saved, err := config.LoadFilterState()
		if err != nil {
			saved = nil
		}
		return dataLoadedMsg{
			iterations:   iterations,
			areas:        areas,
			statesByType: statesByType,
			teamMembers:  members,
			me:           me,
			savedFilters: saved,
		}
	}
}

func loadIdentityCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		me, err := client.GetCurrentUser()
		if err != nil {
			return nil
		}
		return identityLoadedMsg{me: me}
	}
}

func loadWorkItemsCmd(client *api.Client, seq int, q workItemQuery) tea.Cmd {
	return func() tea.Msg {
		items, err := client.QueryWorkItems(q.sprint, q.state, q.assigned, q.area)
		if err != nil {
			return errMsg{err: err}
		}
		return workItemsLoadedMsg{seq: seq, items: items}
	}
}

func saveFiltersCmd(state config.FilterState) tea.Cmd {
	return func() tea.Msg {
		_ = config.SaveFilterState(&state)
		return nil
	}
}

func scheduleCommentsCmd(workItemID int) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg {
		return commentsDueMsg{workItemID: workItemID}
	})
}

func loadCommentsCmd(client *api.Client, workItemID int) tea.Cmd {
	return func() tea.Msg {
		comments, err := client.GetComments(workItemID)
		if err != nil {
			comments = []models.Comment{}
		}
		return commentsLoadedMsg{workItemID: workItemID, comments: comments}
	}
}

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := browser.Open(url); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func copyCmd(text, notice string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.Copy(text); err != nil {
			return errMsg{err: err}
		}
		return copiedMsg{notice: notice}
	}
}

func workItemCmd(notice string, run func() error) tea.Cmd {
	return func() tea.Msg {
		if err := run(); err != nil {
			return errMsg{err: err}
		}
		return workItemChangedMsg{notice: notice}
	}
}

func commentCmd(workItemID int, notice string, run func() error) tea.Cmd {
	return func() tea.Msg {
		if err := run(); err != nil {
			return errMsg{err: err}
		}
		return commentChangedMsg{workItemID: workItemID, notice: notice}
	}
}

func moveWorkItemCmd(client *api.Client, id int, state string) tea.Cmd {
	return func() tea.Msg {
		if err := client.UpdateWorkItemState(id, state); err != nil {
			return workItemMoveFailedMsg{id: id, state: state, err: err}
		}
		return workItemMovedMsg{id: id, state: state}
	}
}

func updateWorkItemStateCmd(client *api.Client, id int, state string) tea.Cmd {
	return workItemCmd("State changed to "+state, func() error {
		return client.UpdateWorkItemState(id, state)
	})
}

func assignWorkItemCmd(client *api.Client, id int, email, name string) tea.Cmd {
	return workItemCmd("Assigned to "+name, func() error {
		return client.AssignWorkItem(id, email)
	})
}

func updateTaskCmd(client *api.Client, id int, title, description string) tea.Cmd {
	return workItemCmd("Task updated", func() error {
		return client.UpdateWorkItem(id, title, description)
	})
}

func updateWorkItemCmd(client *api.Client, req modals.WorkItemUpdateRequestMsg) tea.Cmd {
	return workItemCmd("Work item updated", func() error {
		return client.UpdatePBI(req.ItemID, req.Title, req.Description, req.Priority, req.Effort, req.Tags, req.AssignedTo)
	})
}

func deleteWorkItemCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		if err := client.DeleteWorkItem(id); err != nil {
			return errMsg{err: err}
		}
		return workItemChangedMsg{notice: "Work item deleted", deletedID: id}
	}
}

func fetchDetailCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		item, err := client.GetWorkItem(id)
		if err != nil {
			return nil
		}
		return detailFetchedMsg{item: *item}
	}
}

func createTaskCmd(client *api.Client, parentID int, title, description string) tea.Cmd {
	return func() tea.Msg {
		parent, err := client.GetWorkItem(parentID)
		if err != nil {
			return errMsg{err: err}
		}
		id, err := client.CreateWorkItem("Task", title, description, parentID, parent.AreaPath, parent.IterationPath, parent.AssignedTo)
		if err != nil {
			return errMsg{err: err}
		}
		return workItemChangedMsg{notice: fmt.Sprintf("Task created: #%d", id)}
	}
}

func createParentWorkItemCmd(client *api.Client, req modals.ParentCreateRequestMsg) tea.Cmd {
	return func() tea.Msg {
		id, err := client.CreateParentWorkItem(req.WorkItemType, req.Title, req.Description, req.AcceptanceCriteria,
			"New", req.IterationPath, req.AreaPath, req.AssignedTo, req.Effort, req.Priority, req.Tags, req.DefectCause)
		if err != nil {
			return errMsg{err: err}
		}
		return workItemChangedMsg{notice: fmt.Sprintf("%s created: #%d", req.WorkItemType, id)}
	}
}

func createCommentCmd(client *api.Client, workItemID int, text string) tea.Cmd {
	return commentCmd(workItemID, "Comment added", func() error {
		return client.CreateComment(workItemID, text)
	})
}

func updateCommentCmd(client *api.Client, workItemID, commentID int, text string) tea.Cmd {
	return commentCmd(workItemID, "Comment updated", func() error {
		return client.UpdateComment(workItemID, commentID, text)
	})
}

func deleteCommentCmd(client *api.Client, workItemID, commentID int) tea.Cmd {
	return commentCmd(workItemID, "Comment deleted", func() error {
		return client.DeleteComment(workItemID, commentID)
	})
}

func quickSearchWorkItemCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		item, err := client.GetWorkItem(id)
		if err != nil {
			return modals.QuickSearchErrorMsg{Err: err}
		}
		return modals.QuickSearchSuccessMsg{Item: item}
	}
}
