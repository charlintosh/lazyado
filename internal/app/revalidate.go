package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/charlintosh/lazyado/internal/models"
)

const (
	revalidateAfter   = 30 * time.Second
	commentsKept      = 200
	checksKept        = 200
	pullRequestsLists = 16
)

func (a *App) storeComments(msg commentsLoadedMsg) []models.Comment {
	if msg.err == nil {
		comments := msg.comments
		if comments == nil {
			comments = []models.Comment{}
		}
		a.comments.Put(msg.workItemID, comments)
		return comments
	}
	if cached, ok := a.comments.Get(msg.workItemID); ok {
		return cached
	}
	return []models.Comment{}
}

type pullRequestListKey struct {
	repo     string
	all      bool
	status   models.PullRequestStatus
	criteria models.PullRequestCriteria
}

func (s *pullRequestState) listKey(status models.PullRequestStatus) pullRequestListKey {
	return pullRequestListKey{repo: s.repo.ID, all: s.allRepos, status: status, criteria: s.criteria}
}

func (a *App) showPullRequests() tea.Cmd {
	status := pullRequestStatusFor(a.pr.pivot)
	cached, age, ok := a.pr.lists.Lookup(a.pr.listKey(status))
	if !ok {
		a.pr.items, a.pr.loaded = nil, false
		a.applyPullRequests()
		return a.fetchPullRequests(false)
	}
	a.pr.items, a.pr.status, a.pr.loaded = cached, status, true
	a.pr.panel.SetLoading(false)
	a.applyPullRequests()
	if age < revalidateAfter {
		a.pr.seq++
		a.pr.loading = false
		a.pr.load.stop()
		return nil
	}
	return a.fetchPullRequests(true)
}

func (a *App) storeChecks(msg checksLoadedMsg) []models.PullRequestCheck {
	if msg.err == nil {
		a.pr.checksCache.Put(msg.id, msg.checks)
		return msg.checks
	}
	cached, _ := a.pr.checksCache.Get(msg.id)
	return cached
}
