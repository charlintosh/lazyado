package ui

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

const (
	commitPreviewWidth = 46
	commitPreviewShow  = 110
)

type commitsKey struct {
	repo     string
	version  models.GitVersion
	criteria models.CommitCriteria
}

type commitsState struct {
	panel     panels.CommitsPanel
	criteria  models.CommitCriteria
	loadedFor commitsKey
	seq       int
	loading   bool
	commits   []models.Commit
	detailFor string
	details   map[string]models.CommitDetail
	failed    map[string]error
}

type commitsLoadedMsg struct {
	seq     int
	commits []models.Commit
	more    bool
	err     error
}

type commitDetailDueMsg struct {
	id string
}

type commitDetailMsg struct {
	id     string
	detail models.CommitDetail
	err    error
}

func loadCommitsCmd(client *api.Client, seq int, k commitsKey, skip int) tea.Cmd {
	return func() tea.Msg {
		commits, err := client.ListCommits(k.repo, k.version, k.criteria, skip)
		return commitsLoadedMsg{seq: seq, commits: commits, more: len(commits) == api.CommitsPage, err: err}
	}
}

func scheduleCommitDetailCmd(id string) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg { return commitDetailDueMsg{id: id} })
}

func loadCommitDetailCmd(client *api.Client, repositoryID, id string) tea.Cmd {
	return func() tea.Msg {
		detail, err := client.GetCommitDetail(repositoryID, id)
		return commitDetailMsg{id: id, detail: detail, err: err}
	}
}

func (a *App) inCommits() bool {
	return a.hub == panels.HubCommits
}

func (a *App) openCommits() tea.Cmd {
	return a.openCodeHub(panels.HubCommits)
}

func (a *App) openCommitsFor(criteria models.CommitCriteria) tea.Cmd {
	a.commits.criteria = criteria
	a.commits.panel.SetText("")
	if a.inCommits() {
		return nil
	}
	return a.openCommits()
}

func (a *App) wantedCommits() commitsKey {
	return commitsKey{repo: a.code.repo.ID, version: a.code.version, criteria: a.commits.criteria}
}

func (a *App) syncCommits() tea.Cmd {
	if !a.inCommits() || a.viewMode != ViewMain || a.code.repo.ID == "" {
		return nil
	}
	s := &a.commits
	if want := a.wantedCommits(); want != s.loadedFor {
		s.loadedFor = want
		s.seq++
		s.loading = true
		s.commits = nil
		s.panel.ShowLoading(want.criteria)
		return loadCommitsCmd(a.client, s.seq, want, 0)
	}
	return a.syncCommitDetail()
}

func (a *App) syncCommitDetail() tea.Cmd {
	s := &a.commits
	id := ""
	if c := s.panel.Selected(); c != nil {
		id = c.ID
	}
	if id == s.detailFor {
		return nil
	}
	s.detailFor = id
	if _, ok := s.details[id]; ok || s.failed[id] != nil || id == "" || a.shell.mainWidth < commitPreviewShow {
		return nil
	}
	return scheduleCommitDetailCmd(id)
}

func (a *App) refreshCommits() tea.Cmd {
	if a.code.repo.ID == "" {
		return a.restoreCodeRepository()
	}
	a.commits.loadedFor = commitsKey{}
	a.commits.details, a.commits.failed = nil, nil
	return nil
}

func (a *App) commitsLoading() bool {
	return a.pr.reposLoading || a.commits.loading || a.commits.panel.Loading()
}

func (a *App) commitWebURL(id string) string {
	return a.code.repo.WebURL + "/commit/" + id
}

func (a App) handleCommitsKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	k := a.keys
	s := &a.commits
	if s.panel.Searching() {
		var cmd tea.Cmd
		s.panel, cmd = s.panel.Update(msg)
		return a, cmd
	}
	if hub, ok := a.hubFor(msg); ok {
		return a, a.goToHub(hub)
	}
	sel := s.panel.Selected()
	switch {
	case key.Matches(msg, k.Fullscreen):
		a.fullscreen = !a.fullscreen
	case key.Matches(msg, k.QuickSearch):
		return a, a.openModal(modals.NewQuickSearchModal(a.env()))
	case key.Matches(msg, k.Repository):
		return a, a.openCodeRepositoryPicker()
	case a.code.repo.ID == "":
	case key.Matches(msg, k.Version):
		return a, a.openVersionPicker()
	case key.Matches(msg, k.Search):
		return a, s.panel.FocusSearch()
	case key.Matches(msg, k.Back) && s.panel.Filtered():
		s.criteria = models.CommitCriteria{}
		s.panel.SetText("")
	case key.Matches(msg, k.Select) && sel != nil:
		return a, a.browseFilesAt(models.GitVersion{Kind: models.VersionCommit, Name: sel.ID})
	case key.Matches(msg, k.Copy) && sel != nil:
		return a, copyCmd(sel.ID, "Commit ID copied to clipboard")
	case key.Matches(msg, k.OpenBrowser) && sel != nil:
		return a, openBrowserCmd(a.commitWebURL(sel.ID))
	default:
		var cmd tea.Cmd
		s.panel, cmd = s.panel.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a *App) browseFilesAt(v models.GitVersion) tea.Cmd {
	a.setCodeVersion(v)
	return a.goToHub(panels.HubFiles)
}

func (a App) handleCommitsResult(msg tea.Msg) (App, tea.Cmd, bool) {
	s := &a.commits
	switch msg := msg.(type) {
	case commitsLoadedMsg:
		if msg.seq != s.seq {
			return a, nil, true
		}
		s.loading = false
		if msg.err != nil {
			s.panel.SetCommits(s.commits, false)
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		s.commits = append(s.commits, msg.commits...)
		s.panel.SetCommits(s.commits, msg.more)
		return a, nil, true
	case panels.MoreCommitsMsg:
		if s.loading || !s.panel.More() {
			return a, nil, true
		}
		s.loading = true
		s.panel.SetLoadingMore(true)
		return a, loadCommitsCmd(a.client, s.seq, s.loadedFor, len(s.commits)), true
	case panels.SearchCommitsMsg:
		criteria, text := models.ParseCommitQuery(msg.Query)
		s.criteria = criteria
		s.panel.SetText(text)
		return a, nil, true
	case commitDetailDueMsg:
		if msg.id != s.detailFor {
			return a, nil, true
		}
		return a, loadCommitDetailCmd(a.client, a.code.repo.ID, msg.id), true
	case commitDetailMsg:
		if msg.err != nil {
			if s.failed == nil {
				s.failed = map[string]error{}
			}
			s.failed[msg.id] = msg.err
			return a, nil, true
		}
		if s.details == nil {
			s.details = map[string]models.CommitDetail{}
		}
		s.details[msg.id] = msg.detail
		return a, nil, true
	}
	return a, nil, false
}

func (a *App) renderCommitsBody(width, height int) string {
	s := &a.commits
	if a.code.repo.ID == "" {
		return a.renderNoRepository("Commits", width, height)
	}
	listWidth, previewWidth := width, 0
	if width >= commitPreviewShow {
		listWidth, previewWidth = width-commitPreviewWidth, commitPreviewWidth
	}
	title := s.panel.Title()
	if filter := s.panel.Filter(); filter != "" && !s.panel.Searching() {
		title += " · /" + filter
	}
	list := panels.Pane(panels.PaneTitle("", title, true), s.panel.View(), listWidth, height, true)
	if previewWidth == 0 {
		return list
	}
	cw, ch := panels.PaneContentSize(previewWidth, height)
	props := panels.CommitPreviewProps{Commit: s.panel.Selected(), Spinner: a.spinner.View(), Width: cw, Height: ch}
	if sel := props.Commit; sel != nil {
		if d, ok := s.details[sel.ID]; ok {
			props.Detail = &d
		}
		props.Err = s.failed[sel.ID]
	}
	preview := panels.Pane(panels.PaneTitle("", "Preview", false), panels.RenderCommitPreview(props), previewWidth, height, false)
	return lipgloss.JoinHorizontal(lipgloss.Top, list, preview)
}

func (a *App) syncCommitsLayout(width, height int) {
	listWidth := width
	if width >= commitPreviewShow {
		listWidth = width - commitPreviewWidth
	}
	cw, ch := panels.PaneContentSize(listWidth, height)
	a.commits.panel.SetSize(cw, ch)
}

func (a *App) commitsBindings() []key.Binding {
	if a.code.repo.ID == "" {
		return []key.Binding{a.keys.Repository}
	}
	return a.commits.panel.KeyBindings()
}

func (a *App) searchingCommits() bool {
	return a.viewMode == ViewMain && a.inCommits() && !a.modals.open() && a.commits.panel.Searching()
}
