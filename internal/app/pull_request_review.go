package ui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

const gitattributesPath = "/.gitattributes"

type threadsLoadedMsg struct {
	id      int
	threads []models.PullRequestThread
	err     error
}

type pullRequestLoadedMsg struct {
	pr models.PullRequest
}

type linkedWorkItemsLoadedMsg struct {
	id    int
	items []models.WorkItem
}

type changesLoadedMsg struct {
	id      int
	changes models.PullRequestChanges
	err     error
}

type fileLoadedMsg struct {
	id      int
	path    string
	oldText string
	newText string
	err     error
}

type reviewChangedMsg struct {
	pr     models.PullRequest
	notice string
}

func loadThreadsCmd(client *api.Client, pr models.PullRequest, iteration int) tea.Cmd {
	return func() tea.Msg {
		threads, err := client.GetPullRequestThreads(pr.RepositoryID, pr.ID, iteration)
		return threadsLoadedMsg{id: pr.ID, threads: threads, err: err}
	}
}

func loadPullRequestCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		pr, err := client.GetPullRequest(id)
		if err != nil {
			return nil
		}
		return pullRequestLoadedMsg{pr: pr}
	}
}

func loadLinkedWorkItemsCmd(client *api.Client, pr models.PullRequest) tea.Cmd {
	return func() tea.Msg {
		ids, err := client.GetPullRequestWorkItemIDs(pr.RepositoryID, pr.ID)
		if err != nil {
			return nil
		}
		items := make([]models.WorkItem, 0, len(ids))
		for _, id := range ids {
			if item, err := client.GetWorkItem(id); err == nil {
				items = append(items, *item)
			}
		}
		return linkedWorkItemsLoadedMsg{id: pr.ID, items: items}
	}
}

func loadChangesCmd(client *api.Client, pr models.PullRequest) tea.Cmd {
	return func() tea.Msg {
		changes, err := client.GetPullRequestChanges(pr.RepositoryID, pr.ID)
		if err == nil {
			changes.Attributes, _ = client.GetFileContent(pr.RepositoryID, gitattributesPath, changes.HeadCommit)
		}
		return changesLoadedMsg{id: pr.ID, changes: changes, err: err}
	}
}

func loadFileCmd(client *api.Client, pr models.PullRequest, file models.PullRequestFile, changes models.PullRequestChanges) tea.Cmd {
	return func() tea.Msg {
		msg := fileLoadedMsg{id: pr.ID, path: file.Path}
		oldPath := file.Path
		if file.OriginalPath != "" {
			oldPath = file.OriginalPath
		}
		if file.Kind != models.ChangeAdd {
			msg.oldText, msg.err = client.GetFileContent(pr.RepositoryID, oldPath, changes.BaseCommit)
		}
		if file.Kind != models.ChangeDelete && msg.err == nil {
			msg.newText, msg.err = client.GetFileContent(pr.RepositoryID, file.Path, changes.HeadCommit)
		}
		return msg
	}
}

func reviewCmd(pr models.PullRequest, notice string, run func() error) tea.Cmd {
	return func() tea.Msg {
		if err := run(); err != nil {
			return errMsg{err: err}
		}
		return reviewChangedMsg{pr: pr, notice: notice}
	}
}

func (a *App) reviewing() bool {
	return a.viewMode == ViewPullRequest
}

func (a *App) loadReview(pr models.PullRequest) tea.Cmd {
	return tea.Batch(
		loadThreadsCmd(a.client, pr, a.pr.view.Iteration()),
		loadChecksCmd(a.client, pr),
		loadPullRequestCmd(a.client, pr.ID),
		loadLinkedWorkItemsCmd(a.client, pr),
		loadChangesCmd(a.client, pr),
	)
}

func (a *App) openReview(pr models.PullRequest) tea.Cmd {
	a.viewMode = ViewPullRequest
	a.pr.view.Open(pr)
	return a.loadReview(pr)
}

func (a *App) refreshReview() tea.Cmd {
	pr := a.pr.view.PullRequest()
	if pr == nil {
		return nil
	}
	return a.loadReview(*pr)
}

func (a *App) syncReviewFile() tea.Cmd {
	if !a.reviewing() {
		return nil
	}
	pr := a.pr.view.PullRequest()
	file, changes, ok := a.pr.view.WantedFile()
	if pr == nil || !ok {
		return nil
	}
	return loadFileCmd(a.client, *pr, file, changes)
}

func (a *App) info(text string) tea.Cmd {
	var cmd tea.Cmd
	a.notification, cmd = a.notification.Show(notification.NotificationInfo, text)
	return cmd
}

func (a *App) openVote(pr models.PullRequest) tea.Cmd {
	if a.me.ID == "" {
		return a.info("Couldn't confirm your Azure DevOps identity from the PAT, so voting is unavailable. Press " +
			a.keys.Refresh.Help().Key + " to retry.")
	}
	if pr.Status != models.PullRequestActive {
		return a.info("Only active pull requests can be voted on.")
	}
	current := models.VoteNone
	if r, ok := pr.ReviewerFor(a.me); ok {
		current = r.Vote
	}
	blocking := 0
	if pr.ID == a.pr.checksFor || a.reviewing() {
		blocking = models.BlockingChecks(a.pr.checks)
	}
	return a.openModal(modals.NewVoteModal(a.env(), pr, current, blocking))
}

func (a *App) replacePullRequest(pr models.PullRequest) {
	for i := range a.pr.items {
		if a.pr.items[i].ID == pr.ID {
			a.pr.items[i] = pr
		}
	}
	a.applyPullRequests()
	a.pr.view.Refresh(pr)
}

func (a App) handleReviewResult(msg tea.Msg) (App, tea.Cmd, bool) {
	c := a.client
	switch msg := msg.(type) {
	case panels.ReviewPullRequestMsg:
		return a, a.openReview(msg.PullRequest), true
	case panels.ClosePullRequestViewMsg:
		a.viewMode = ViewMain
		return a, nil, true
	case threadsLoadedMsg:
		a.pr.view.SetThreads(msg.id, msg.threads)
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		return a, nil, true
	case pullRequestLoadedMsg:
		a.replacePullRequest(msg.pr)
		return a, nil, true
	case linkedWorkItemsLoadedMsg:
		a.pr.view.SetWorkItems(msg.id, msg.items)
		return a, nil, true
	case changesLoadedMsg:
		tracked := a.pr.view.Iteration()
		a.pr.view.SetChanges(msg.id, msg.changes)
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		if pr := a.pr.view.PullRequest(); pr != nil && pr.ID == msg.id && msg.changes.Iteration != tracked {
			return a, loadThreadsCmd(c, *pr, msg.changes.Iteration), true
		}
		return a, nil, true
	case fileLoadedMsg:
		a.pr.view.SetFile(msg.id, msg.path, msg.oldText, msg.newText, msg.err)
		return a, nil, true
	case reviewChangedMsg:
		a.loading = false
		pr := msg.pr
		return a, tea.Batch(a.notify(msg.notice), loadThreadsCmd(c, pr, a.pr.view.Iteration()), loadPullRequestCmd(c, pr.ID), loadChecksCmd(c, pr)), true
	case panels.VotePullRequestMsg:
		return a, a.openVote(msg.PullRequest), true
	case panels.CommentPullRequestMsg:
		return a, a.openModal(modals.NewPullRequestCommentModal(a.env(), msg.PullRequest, msg.Anchor, a.teamMembers)), true
	case panels.ReplyPullRequestMsg:
		return a, a.openModal(modals.NewPullRequestReplyModal(a.env(), msg.PullRequest, msg.Thread, msg.Comment, a.teamMembers)), true
	case panels.LikePullRequestCommentMsg:
		pr, thread, comment, like := msg.PullRequest, msg.Thread, msg.Comment, msg.Like
		notice := "Liked " + comment.AuthorName + "'s comment"
		if !like {
			notice = "Like removed"
		}
		return a, a.mutate(reviewCmd(pr, notice, func() error {
			return c.LikePullRequestComment(pr.RepositoryID, pr.ID, thread.ID, comment.ID, like)
		})), true
	case panels.EditPullRequestCommentMsg:
		return a, a.openModal(modals.NewEditPullRequestCommentModal(a.env(), msg.PullRequest, msg.Thread, msg.Comment, a.teamMembers)), true
	case panels.DeletePullRequestCommentMsg:
		return a, a.openModal(modals.NewDeletePullRequestCommentModal(a.env(), msg.PullRequest, msg.Thread, msg.Comment)), true
	case modals.PullRequestCommentDeleteRequestMsg:
		a.closeModal()
		pr, req := msg.PullRequest, msg
		return a, a.mutate(reviewCmd(pr, "Comment deleted", func() error {
			return c.DeletePullRequestComment(pr.RepositoryID, pr.ID, req.ThreadID, req.CommentID)
		})), true
	case panels.LinkedWorkItemsMsg:
		switch len(msg.Items) {
		case 0:
			return a, a.info("This pull request has no linked work items."), true
		case 1:
			return a, a.openDetail(msg.Items[0]), true
		}
		return a, a.openModal(modals.NewLinkedWorkItemsModal(a.env(), msg.PullRequest, msg.Items)), true
	case modals.WorkItemChosenMsg:
		a.closeModal()
		return a, a.openDetail(msg.Item), true
	case panels.ResolvePullRequestThreadMsg:
		pr, thread, status := msg.PullRequest, msg.Thread, msg.Status
		notice := "Thread resolved"
		if status == models.ThreadActive {
			notice = "Thread reopened"
		}
		return a, a.mutate(reviewCmd(pr, notice, func() error {
			return c.SetPullRequestThreadStatus(pr.RepositoryID, pr.ID, thread.ID, status)
		})), true
	case modals.PullRequestVoteRequestMsg:
		a.closeModal()
		pr, vote, me := msg.PullRequest, msg.Vote, a.me.ID
		return a, a.mutate(reviewCmd(pr, "Vote: "+vote.Label(), func() error {
			return c.VotePullRequest(pr.RepositoryID, pr.ID, me, vote)
		})), true
	case modals.PullRequestCommentRequestMsg:
		a.closeModal()
		pr, req := msg.PullRequest, msg
		if req.CommentID != 0 {
			return a, a.mutate(reviewCmd(pr, "Comment updated", func() error {
				return c.UpdatePullRequestComment(pr.RepositoryID, pr.ID, req.ThreadID, req.CommentID, req.Text)
			})), true
		}
		if req.ThreadID != 0 {
			return a, a.mutate(reviewCmd(pr, "Reply added", func() error {
				return c.ReplyToPullRequestThread(pr.RepositoryID, pr.ID, req.ThreadID, req.ParentID, req.Text)
			})), true
		}
		return a, a.mutate(reviewCmd(pr, "Comment added", func() error {
			return c.CreatePullRequestThread(pr.RepositoryID, pr.ID, req.Text, req.Anchor)
		})), true
	}
	return a, nil, false
}

func (a *App) renderReview() string {
	l := a.shell
	crumbs := []string{panels.HubPullRequests.Section(), panels.HubPullRequests.Name()}
	if pr := a.pr.view.PullRequest(); pr != nil {
		crumbs = append(crumbs, pr.RepositoryName, strconv.Itoa(pr.ID))
	}
	body := panels.Block(a.pr.view.View(), a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, a.renderHeader(crumbs...), body, a.renderStatusBar())
}
