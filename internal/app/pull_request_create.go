package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/models"
)

type pullRequestCreatedMsg struct {
	pr models.PullRequest
}

func loadBranchesCmd(client *api.Client, repositoryID string) tea.Cmd {
	return func() tea.Msg {
		msg := modals.PullRequestBranchesMsg{RepositoryID: repositoryID}
		msg.Branches, msg.Err = client.ListBranches(repositoryID)
		if msg.Err == nil {
			msg.Default, _ = client.DefaultBranch(repositoryID)
		}
		return msg
	}
}

func compareBranchesCmd(client *api.Client, req modals.PullRequestCompareRequestMsg) tea.Cmd {
	return func() tea.Msg {
		msg := modals.PullRequestComparedMsg{RepositoryID: req.RepositoryID, Source: req.Source, Target: req.Target}
		criteria := models.PullRequestCriteria{SourceRef: "refs/heads/" + req.Source, TargetRef: "refs/heads/" + req.Target}
		if open, err := client.ListPullRequests(models.PullRequestActive, req.RepositoryID, criteria); err == nil && len(open) > 0 {
			msg.Comparison.ExistingID, msg.Comparison.ExistingTitle = open[0].ID, open[0].Title
			return msg
		}
		msg.Comparison, msg.Err = client.CompareBranches(req.RepositoryID, req.Source, req.Target)
		if msg.Err == nil && msg.Comparison.Ahead > 0 {
			msg.Comparison.Template = client.PullRequestTemplate(req.RepositoryID, req.Target)
		}
		return msg
	}
}

func createPullRequestCmd(client *api.Client, req modals.PullRequestCreateRequestMsg) tea.Cmd {
	return func() tea.Msg {
		pr, err := client.CreatePullRequest(req.Repository.ID, req.PullRequest)
		if err != nil {
			return errMsg{err: err}
		}
		return pullRequestCreatedMsg{pr: pr}
	}
}

func (a *App) openPullRequestForm() tea.Cmd {
	if !a.pr.reposLoaded {
		return a.info("Repositories are still loading. Try again in a moment.")
	}
	data := modals.PullRequestFormData{Members: a.teamMembers, Open: a.pr.items}
	if a.pr.repo.ID != "" {
		data.Repository = a.pr.repo
	} else {
		data.Repositories = a.pr.repos
	}
	return a.openModal(modals.NewPullRequestForm(a.env(), data))
}

func (a *App) openPullRequestFormFrom(repo models.Repository, source string) tea.Cmd {
	data := modals.PullRequestFormData{Members: a.teamMembers, Open: a.pr.items, Repository: repo, Source: source}
	return a.openModal(modals.NewPullRequestForm(a.env(), data))
}

func (a App) handlePullRequestFormResult(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case modals.PullRequestBranchesRequestMsg:
		return a, loadBranchesCmd(a.client, msg.RepositoryID), true
	case modals.PullRequestCompareRequestMsg:
		return a, compareBranchesCmd(a.client, msg), true
	case modals.PullRequestBranchesMsg, modals.PullRequestComparedMsg:
		if !topIs[modals.PullRequestForm](a.modals) {
			return a, nil, true
		}
		next, cmd := a.updateModal(msg)
		return next, cmd, true
	case modals.PullRequestCreateRequestMsg:
		return a, a.mutate(createPullRequestCmd(a.client, msg)), true
	case modals.OpenPullRequestRequestMsg:
		a.closeModal()
		return a, a.mutate(loadPullRequestByIDCmd(a.client, msg.ID)), true
	case pullRequestCreatedMsg:
		a.pr.lists.Clear()
		a.loading = false
		if topIs[modals.PullRequestForm](a.modals) {
			a.closeModal()
		}
		notice := a.notify(fmt.Sprintf("Pull request !%d created", msg.pr.ID))
		return a, tea.Batch(notice, a.openReview(msg.pr), a.loadPullRequests()), true
	}
	return a, nil, false
}
