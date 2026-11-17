package ui

import (
	"slices"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

const (
	pullRequestsMine = iota
	pullRequestsActive
	pullRequestsCompleted
	pullRequestsAbandoned
)

var pullRequestPivots = []string{"Mine", "Active", "Completed", "Abandoned"}

type pullRequestState struct {
	panel        panels.PullRequestsPanel
	view         panels.PullRequestView
	pivot        int
	items        []models.PullRequest
	status       models.PullRequestStatus
	seq          int
	loading      bool
	loaded       bool
	checksFor    int
	checks       []models.PullRequestCheck
	checksLoaded bool
	criteria     models.PullRequestCriteria

	repos        []models.Repository
	reposLoaded  bool
	reposLoading bool
	repo         models.Repository
	allRepos     bool
	savedRepo    string
	starred      map[string]bool
	favorites    []string
}

const repositoryGroup = 100

type repositoriesLoadedMsg struct {
	repos   []models.Repository
	starred []string
	err     error
}

func loadRepositoriesCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		repos, err := client.ListRepositories()
		if err != nil {
			return repositoriesLoadedMsg{err: err}
		}
		starred, err := client.GetFavoriteArtifactIDs()
		if err != nil {
			starred = nil
		}
		return repositoriesLoadedMsg{repos: repos, starred: starred}
	}
}

func (s *pullRequestState) scoped() bool {
	return s.allRepos || s.repo.ID != ""
}

func (s *pullRequestState) savedScope() string {
	switch {
	case s.allRepos:
		return models.AllRepositories
	case s.repo.ID != "":
		return s.repo.Name
	}
	return s.savedRepo
}

func (s *pullRequestState) scopeLabel() string {
	switch {
	case s.allRepos:
		return "all"
	case s.repo.ID != "":
		return s.repo.Name
	}
	return "choose"
}

func (a *App) openRepositoryPicker() tea.Cmd {
	if !a.pr.reposLoaded {
		return nil
	}
	current := a.pr.repo.ID
	if a.pr.allRepos {
		current = models.AllRepositories
	}
	local := make(map[string]bool, len(a.pr.favorites))
	for _, id := range a.pr.favorites {
		local[id] = true
	}
	favorites := modals.RepositoryFavorites{Starred: a.pr.starred, Local: local}
	return a.openModal(modals.NewRepositoryPickerModal(a.env(), a.pr.repos, favorites, current))
}

func (a *App) setRepositoryScope(repo models.Repository, all bool) tea.Cmd {
	a.pr.repo, a.pr.allRepos = repo, all
	a.pr.panel.SetScope(a.pr.scoped(), !all)
	a.pr.items, a.pr.loaded = nil, false
	a.applyPullRequests()
	return a.loadPullRequests()
}

func (a *App) restoreRepositoryScope() tea.Cmd {
	if a.pr.scoped() {
		return nil
	}
	if a.pr.savedRepo == models.AllRepositories {
		return a.setRepositoryScope(models.Repository{}, true)
	}
	for _, r := range a.pr.repos {
		if r.Name == a.pr.savedRepo || len(a.pr.repos) == 1 {
			return a.setRepositoryScope(r, false)
		}
	}
	if a.inPullRequests() && !a.modals.open() {
		return a.openRepositoryPicker()
	}
	return nil
}

type pullRequestsLoadedMsg struct {
	seq    int
	status models.PullRequestStatus
	items  []models.PullRequest
}

type pullRequestsFailedMsg struct {
	seq int
	err error
}

type checksDueMsg struct {
	id int
}

type checksLoadedMsg struct {
	id     int
	checks []models.PullRequestCheck
}

func pullRequestStatusFor(pivot int) models.PullRequestStatus {
	switch pivot {
	case pullRequestsCompleted:
		return models.PullRequestCompleted
	case pullRequestsAbandoned:
		return models.PullRequestAbandoned
	}
	return models.PullRequestActive
}

type pullRequestFoundMsg struct {
	pr models.PullRequest
}

func loadPullRequestByIDCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		pr, err := client.GetPullRequest(id)
		if err != nil {
			return errMsg{err: err}
		}
		return pullRequestFoundMsg{pr: pr}
	}
}

func loadPullRequestsCmd(client *api.Client, seq int, status models.PullRequestStatus, repositoryID string, criteria models.PullRequestCriteria) tea.Cmd {
	return func() tea.Msg {
		items, err := client.ListPullRequests(status, repositoryID, criteria)
		if err != nil {
			return pullRequestsFailedMsg{seq: seq, err: err}
		}
		return pullRequestsLoadedMsg{seq: seq, status: status, items: items}
	}
}

func scheduleChecksCmd(id int) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg {
		return checksDueMsg{id: id}
	})
}

func loadChecksCmd(client *api.Client, pr models.PullRequest) tea.Cmd {
	return func() tea.Msg {
		checks, err := client.GetPullRequestChecks(pr)
		if err != nil {
			checks = nil
		}
		return checksLoadedMsg{id: pr.ID, checks: checks}
	}
}

func (a *App) inPullRequests() bool {
	return a.hub == panels.HubPullRequests
}

func (a *App) openPullRequests() tea.Cmd {
	a.switchHub(panels.HubPullRequests)
	switch {
	case !a.pr.reposLoaded && !a.pr.reposLoading:
		a.pr.reposLoading = true
		return loadRepositoriesCmd(a.client)
	case !a.pr.scoped():
		return a.openRepositoryPicker()
	case a.pr.loaded && a.pr.status == pullRequestStatusFor(a.pr.pivot):
		return nil
	}
	return a.loadPullRequests()
}

func (a *App) loadPullRequests() tea.Cmd {
	if !a.pr.scoped() {
		return nil
	}
	a.pr.seq++
	a.pr.loading = true
	a.pr.panel.SetLoading(true)
	return loadPullRequestsCmd(a.client, a.pr.seq, pullRequestStatusFor(a.pr.pivot), a.pr.repo.ID, a.pr.criteria)
}

func (a *App) refreshPullRequests() tea.Cmd {
	cmds := []tea.Cmd{a.loadPullRequests()}
	if !a.me.Known() {
		cmds = append(cmds, loadIdentityCmd(a.client))
	}
	if pr := a.pr.panel.Selected(); pr != nil {
		cmds = append(cmds, loadChecksCmd(a.client, *pr))
	}
	return tea.Batch(cmds...)
}

func (a *App) pullRequestPivotLabels() []string {
	labels := make([]string, len(pullRequestPivots))
	for i, name := range pullRequestPivots {
		labels[i] = a.filterKeys()[i].Help().Key + " " + name
	}
	return labels
}

func (a *App) cyclePullRequestPivot(delta int) tea.Cmd {
	return a.selectPullRequestPivot((a.pr.pivot + delta + len(pullRequestPivots)) % len(pullRequestPivots))
}

func (a *App) selectPullRequestPivot(pivot int) tea.Cmd {
	if pivot == a.pr.pivot {
		return nil
	}
	a.pr.pivot = pivot
	if a.pr.loaded && a.pr.status == pullRequestStatusFor(a.pr.pivot) {
		a.applyPullRequests()
		return nil
	}
	a.pr.items = nil
	a.applyPullRequests()
	return a.loadPullRequests()
}

func (a *App) applyPullRequests() {
	items := a.pr.items
	if a.pr.pivot == pullRequestsMine && a.me.Known() {
		items = nil
		for _, pr := range a.pr.items {
			if pr.Involves(a.me) {
				items = append(items, pr)
			}
		}
	}
	a.pr.panel.SetItems(items)
}

func (a *App) syncPullRequestSelection() tea.Cmd {
	if !a.inPullRequests() || a.reviewing() {
		return nil
	}
	id := 0
	if pr := a.pr.panel.Selected(); pr != nil {
		id = pr.ID
	}
	if id == a.pr.checksFor {
		return nil
	}
	a.pr.checksFor, a.pr.checks, a.pr.checksLoaded = id, nil, false
	if id == 0 {
		return nil
	}
	return scheduleChecksCmd(id)
}

func (a App) handlePullRequestKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	k := a.keys
	if a.pr.panel.Searching() {
		var cmd tea.Cmd
		a.pr.panel, cmd = a.pr.panel.Update(msg)
		return a, cmd
	}
	for pivot, b := range a.filterKeys() {
		if key.Matches(msg, b) {
			return a, a.selectPullRequestPivot(pivot)
		}
	}
	switch {
	case key.Matches(msg, k.HubBacklog):
		a.switchHub(panels.HubBacklog)
	case key.Matches(msg, k.HubBoard):
		a.switchHub(panels.HubBoard)
	case key.Matches(msg, k.HubSprint):
		a.switchHub(panels.HubSprint)
	case key.Matches(msg, k.HubPullRequests):
	case key.Matches(msg, k.HubBuilds):
		return a, a.openPipelines()
	case key.Matches(msg, k.PrevView), key.Matches(msg, k.PrevPanel), key.Matches(msg, k.Left):
		return a, a.cyclePullRequestPivot(-1)
	case key.Matches(msg, k.NextView), key.Matches(msg, k.NextPanel), key.Matches(msg, k.Right):
		return a, a.cyclePullRequestPivot(1)
	case key.Matches(msg, k.Fullscreen):
		a.fullscreen = !a.fullscreen
	case key.Matches(msg, k.QuickSearch):
		return a, a.openModal(modals.NewQuickSearchModal(a.env()))
	case key.Matches(msg, k.Repository):
		return a, a.openRepositoryPicker()
	case key.Matches(msg, k.Search):
		return a, a.pr.panel.FocusSearch()
	case key.Matches(msg, k.NewPullRequest):
		return a, a.openPullRequestForm()
	case key.Matches(msg, k.Vote):
		if pr := a.pr.panel.Selected(); pr != nil {
			return a, a.openVote(*pr)
		}
	case key.Matches(msg, k.AddComment):
		if pr := a.pr.panel.Selected(); pr != nil {
			return a, a.openModal(modals.NewPullRequestCommentModal(a.env(), *pr, nil, a.teamMembers))
		}
	default:
		var cmd tea.Cmd
		a.pr.panel, cmd = a.pr.panel.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a App) handlePullRequestResult(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case repositoriesLoadedMsg:
		a.pr.reposLoading = false
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		a.pr.repos, a.pr.reposLoaded = msg.repos, true
		a.pr.starred = make(map[string]bool, len(msg.starred))
		for _, id := range msg.starred {
			a.pr.starred[id] = true
		}
		return a, a.restoreRepositoryScope(), true
	case modals.RepositoryFavoriteMsg:
		a.pr.favorites = slices.DeleteFunc(slices.Clone(a.pr.favorites), func(id string) bool { return id == msg.Repository.ID })
		if msg.Favorite {
			a.pr.favorites = append(a.pr.favorites, msg.Repository.ID)
		}
		return a, saveFiltersCmd(a.savedState()), true
	case modals.RepositoryChosenMsg:
		a.closeModal()
		load := a.setRepositoryScope(msg.Repository, msg.All)
		return a, tea.Batch(load, saveFiltersCmd(a.savedState())), true
	case pullRequestsLoadedMsg:
		if msg.seq != a.pr.seq {
			return a, nil, true
		}
		a.pr.loading, a.pr.loaded = false, true
		a.pr.status, a.pr.items = msg.status, msg.items
		a.pr.panel.SetLoading(false)
		a.applyPullRequests()
		return a, nil, true
	case pullRequestsFailedMsg:
		if msg.seq != a.pr.seq {
			return a, nil, true
		}
		a.pr.loading = false
		a.pr.panel.SetLoading(false)
		return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
	case checksDueMsg:
		pr := a.pr.panel.Selected()
		if pr == nil || pr.ID != msg.id || msg.id != a.pr.checksFor {
			return a, nil, true
		}
		return a, loadChecksCmd(a.client, *pr), true
	case checksLoadedMsg:
		if msg.id == a.pr.checksFor {
			a.pr.checks, a.pr.checksLoaded = msg.checks, true
		}
		a.pr.view.SetChecks(msg.id, msg.checks)
		return a, nil, true
	case panels.BrowsePullRequestMsg:
		return a, openBrowserCmd(msg.PullRequest.WebURL), true
	case panels.SearchPullRequestsMsg:
		if msg.Criteria == a.pr.criteria {
			return a, nil, true
		}
		a.pr.criteria, a.pr.items, a.pr.loaded = msg.Criteria, nil, false
		a.applyPullRequests()
		return a, a.loadPullRequests(), true
	case panels.OpenPullRequestByIDMsg:
		return a, a.mutate(loadPullRequestByIDCmd(a.client, msg.ID)), true
	case pullRequestFoundMsg:
		a.loading = false
		return a, a.openReview(msg.pr), true
	}
	if next, cmd, ok := a.handlePullRequestFormResult(msg); ok {
		return next, cmd, true
	}
	return a.handleReviewResult(msg)
}

func (a *App) renderPullRequestPane(width, height int) string {
	cw, ch := panels.PaneContentSize(width, height)
	content := panels.RenderPullRequestPreview(panels.PullRequestPreviewProps{
		PullRequest:  a.pr.panel.Selected(),
		Checks:       a.pr.checks,
		ChecksLoaded: a.pr.checksLoaded,
		Spinner:      a.spinner.View(),
		Me:           a.me,
		Width:        cw,
		Height:       ch,
	})
	return panels.Pane(panels.PaneTitle("", "Preview", false), content, width, height, false)
}
