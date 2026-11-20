package ui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/cache"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/components/screens"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type Panel int

const (
	PanelWorkItems Panel = iota
	PanelDetails
	PanelComments
	panelCount
)

type mainView int

const (
	viewList mainView = iota
	viewBoard
	viewTaskboard
	viewPullRequests
	viewPipelines
	viewReleases
	viewFiles
	viewCommits
	viewBranches
	viewTags
)

const (
	sprintTaskboard = iota
	sprintBacklog
)

var sprintPivots = []string{"Taskboard", "Backlog"}

type ViewMode int

const (
	ViewMain ViewMode = iota
	ViewDetail
	ViewPullRequest
	ViewRun
	ViewRelease
	ViewReleaseEditor
)

type App struct {
	workItemsPanel panels.WorkItemsPanel
	boardPanel     panels.BoardPanel
	taskboardPanel panels.TaskboardPanel
	detailsPanel   panels.DetailsPanel
	commentsPanel  panels.CommentsPanel
	detailView     panels.DetailView
	pr             pullRequestState
	pl             pipelineState
	rl             releaseState
	code           codeState
	commits        commitsState
	branches       branchesState
	tags           tagsState
	splashScreen   screens.SplashScreen
	notification   notification.Notification
	statusHelp     help.Model
	spinner        spinner.Model
	modals         modalStack
	keyword        textinput.Model
	filters        *models.FilterState
	loadedQuery    workItemQuery
	workItemsLoad  inflight
	shell          shellLayout
	header         panels.PageHeader

	activePanel     Panel
	rightTab        Panel
	hub             panels.Hub
	sprintView      int
	filterBar       bool
	fullscreen      bool
	viewMode        ViewMode
	loading         bool
	spinning        bool
	loadSeq         int
	commentsFor     int
	detailFromPR    bool
	initialLoadDone bool
	throttleSeen    int
	truncatedFor    workItemQuery

	iterations   []models.Iteration
	areas        []models.Area
	workItems    []models.WorkItem
	statesByType map[string][]models.WorkItemStateInfo
	teamMembers  []models.TeamMember
	me           models.Identity
	comments     *cache.LRU[int, []models.Comment]

	client *api.Client
	styles *styles.Styles
	keys   keys.KeyMap
	width  int
	height int
}

func NewApp(client *api.Client, limits models.DiffLimits) App {
	st := styles.DefaultStyles()
	k := keys.DefaultKeyMap()
	keyword := widgets.SearchInput(keywordPrompt, keywordPlaceholder)
	keyword.SetWidth(styles.KeywordInputWidth - len([]rune(keywordPrompt)) - 1)
	return App{
		keyword:        keyword,
		filters:        models.NewFilterState(nil, nil, nil),
		workItemsPanel: panels.NewWorkItemsPanel(st, k),
		boardPanel:     panels.NewBoardPanel(st, k),
		taskboardPanel: panels.NewTaskboardPanel(st, k),
		detailsPanel:   panels.NewDetailsPanel(st, k),
		commentsPanel:  panels.NewCommentsPanel(st, k),
		detailView:     panels.NewDetailView(st, k),
		pr: pullRequestState{
			panel: panels.NewPullRequestsPanel(k), view: panels.NewPullRequestView(k, limits),
			checksCache: cache.Count[int, []models.PullRequestCheck](checksKept),
			lists:       cache.Count[pullRequestListKey, []models.PullRequest](pullRequestsLists),
		},
		pl: pipelineState{panel: panels.NewPipelinesPanel(k), view: panels.NewPipelineRunView(k)},
		rl: releaseState{
			defs: panels.NewReleaseDefinitionsPanel(k), list: panels.NewReleasesPanel(k),
			approvals: panels.NewApprovalsPanel(k), view: panels.NewReleaseView(k), editor: panels.NewReleaseEditor(k),
		},
		commits:      commitsState{panel: panels.NewCommitsPanel(k)},
		branches:     branchesState{panel: panels.NewBranchesPanel(k)},
		tags:         tagsState{panel: panels.NewTagsPanel(k)},
		code:         codeState{files: newFilesState(panels.NewRepoTreePanel(k), panels.NewRepoContentPanel(k), limits.MaxFileBytes())},
		splashScreen: screens.NewSplashScreen(st),
		statusHelp:   widgets.NewHelp(),
		spinner: spinner.New(
			spinner.WithSpinner(spinner.MiniDot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary)),
		),
		comments:    cache.Count[int, []models.Comment](commentsKept),
		activePanel: PanelWorkItems,
		rightTab:    PanelDetails,
		hub:         panels.HubBacklog,
		viewMode:    ViewMain,
		loading:     true,
		spinning:    true,
		client:      client,
		styles:      st,
		keys:        k,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		loadDataCmd(a.client),
		screens.AnimateCmd(),
		a.splashScreen.Tick(),
		a.spinner.Tick,
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := a.update(msg)
	return next, tea.Batch(cmd, next.reconcile())
}

func (a App) update(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.resize(msg)
	case tea.KeyPressMsg:
		return a.handleKey(msg)
	case tea.MouseMsg:
		return a.handleMouse(msg)
	case tea.PasteMsg:
		if a.keyword.Focused() && !a.modals.open() {
			return a.updateKeyword(msg)
		}
		if a.searchingPullRequests() {
			var cmd tea.Cmd
			a.pr.panel, cmd = a.pr.panel.Update(msg)
			return a, cmd
		}
		if a.searchingPipelines() {
			var cmd tea.Cmd
			a.pl.panel, cmd = a.pl.panel.Update(msg)
			return a, cmd
		}
		if a.searchingReleases() {
			var cmd tea.Cmd
			a.rl.defs, cmd = a.rl.defs.Update(msg)
			return a, cmd
		}
		if a.searchingCommits() {
			var cmd tea.Cmd
			a.commits.panel, cmd = a.commits.panel.Update(msg)
			return a, cmd
		}
		if a.searchingBranches() {
			var cmd tea.Cmd
			a.branches.panel, cmd = a.branches.panel.Update(msg)
			return a, cmd
		}
		if a.searchingTags() {
			var cmd tea.Cmd
			a.tags.panel, cmd = a.tags.panel.Update(msg)
			return a, cmd
		}
	case spinner.TickMsg:
		return a.tick(msg)
	case screens.FrameMsg:
		return a.animateSplash(msg)
	}
	if next, cmd, ok := a.handleResult(msg); ok {
		return next, cmd
	}
	return a.updateModal(msg)
}

func (a App) resize(msg tea.WindowSizeMsg) (App, tea.Cmd) {
	a.width, a.height = msg.Width, msg.Height
	a.splashScreen.SetSize(msg.Width, msg.Height)
	var cmd tea.Cmd
	a.modals, cmd = a.modals.broadcast(msg)
	return a, cmd
}

func (a App) handleMouse(msg tea.MouseMsg) (App, tea.Cmd) {
	if a.modals.open() {
		return a.updateModal(msg)
	}
	var cmd tea.Cmd
	switch {
	case a.reviewing():
		a.pr.view, cmd = a.pr.view.Update(msg)
	case a.viewingRun():
		a.pl.view, cmd = a.pl.view.Update(msg)
	case a.viewingRelease():
		a.rl.view, cmd = a.rl.view.Update(msg)
	case a.viewingReleaseEditor():
		a.rl.editor, cmd = a.rl.editor.Update(msg)
	case a.viewMode == ViewDetail:
		a.detailView, cmd = a.detailView.Update(msg)
	case a.activePanel == PanelWorkItems:
		cmd = a.updateMainPanel(msg)
	case a.activePanel == PanelDetails:
		a.detailsPanel, cmd = a.detailsPanel.Update(msg)
	case a.activePanel == PanelComments:
		a.commentsPanel, cmd = a.commentsPanel.Update(msg)
	}
	return a, cmd
}

func (a App) tick(msg spinner.TickMsg) (App, tea.Cmd) {
	var cmds []tea.Cmd
	if a.showSplash() {
		var cmd tea.Cmd
		a.splashScreen, cmd = a.splashScreen.Update(msg)
		cmds = append(cmds, cmd)
	}
	if msg.ID != a.spinner.ID() {
		next, cmd := a.updateModal(msg)
		return next, tea.Batch(append(cmds, cmd)...)
	}
	if !a.busy() {
		a.spinning = false
		return a, tea.Batch(cmds...)
	}
	var cmd tea.Cmd
	a.spinner, cmd = a.spinner.Update(msg)
	return a, tea.Batch(append(cmds, cmd)...)
}

func (a App) animateSplash(msg screens.FrameMsg) (App, tea.Cmd) {
	if !a.showSplash() {
		return a, nil
	}
	var cmd tea.Cmd
	a.splashScreen, cmd = a.splashScreen.Update(msg)
	return a, cmd
}

func (a *App) reconcile() tea.Cmd {
	a.syncFocus()
	a.syncLayout()
	selection := a.syncSelection()
	checks := a.syncPullRequestSelection()
	preview := a.syncPipelinePreview()
	listStages := a.syncListStages()
	releases := a.syncReleaseSelection()
	a.syncSpinner()
	files := a.syncFiles()
	commits := a.syncCommits()
	branches := a.syncBranches()
	tags := a.syncTags()
	return tea.Batch(selection, checks, preview, listStages, releases, files, commits, branches, tags, a.syncReviewFile(), a.syncRunLog(), a.syncReleaseLog(), a.ensureSpinner(), a.syncThrottle())
}

func (a *App) syncSpinner() {
	frame := a.spinner.View()
	a.pr.panel.SetSpinner(frame)
	a.pr.view.SetSpinner(frame)
	a.pl.panel.SetSpinner(frame)
	a.pl.view.SetSpinner(frame)
	a.rl.defs.SetSpinner(frame)
	a.rl.list.SetSpinner(frame)
	a.rl.approvals.SetSpinner(frame)
	a.rl.view.SetSpinner(frame)
	a.rl.editor.SetSpinner(frame)
	a.code.files.tree.SetSpinner(frame)
	a.code.files.content.SetSpinner(frame)
	a.commits.panel.SetSpinner(frame)
	a.branches.panel.SetSpinner(frame)
	a.tags.panel.SetSpinner(frame)
}

func (a *App) syncSelection() tea.Cmd {
	item := a.selectedItem()
	a.detailsPanel.SetItem(item)
	id := 0
	if item != nil {
		id = item.ID
	}
	if id == a.commentsFor {
		return nil
	}
	a.commentsFor = id
	if id == 0 {
		a.commentsPanel.Clear()
		return nil
	}
	a.commentsPanel.StartLoading(id)
	cached, age, ok := a.comments.Lookup(id)
	if !ok {
		return scheduleCommentsCmd(id)
	}
	a.commentsPanel.SetComments(id, cached)
	if age < revalidateAfter {
		return nil
	}
	return scheduleCommentsCmd(id)
}

func (a *App) ensureSpinner() tea.Cmd {
	if a.spinning || !a.busy() {
		return nil
	}
	a.spinning = true
	return a.spinner.Tick
}

func (a *App) busy() bool {
	return a.loading || a.pr.loading || a.commentsPanel.Loading() || (a.viewMode == ViewDetail && a.detailView.Loading()) ||
		(a.reviewing() && a.pr.view.Loading()) || (a.inPullRequests() && a.pr.checksFor != 0 && !a.pr.checksLoaded) ||
		(a.inPipelines() && (a.pl.loading || a.pl.panel.Loading() || a.previewLoading())) || (a.viewingRun() && a.pl.view.Loading()) ||
		(a.inReleases() && a.releaseLoading()) || (a.viewingRelease() && a.rl.view.Loading()) || (a.viewingReleaseEditor() && a.rl.editor.Loading()) ||
		(a.inFiles() && a.filesLoading()) || (a.inCommits() && a.commitsLoading()) ||
		(a.inBranches() && (a.pr.reposLoading || a.branches.loading)) || (a.inTags() && a.tagsLoading())
}

func (a *App) showSplash() bool {
	return a.loading && len(a.workItems) == 0 && !a.initialLoadDone
}

func (a *App) env() modals.Env {
	return modals.Env{Keys: a.keys, Width: a.width, Height: a.height}
}

func (a *App) openModal(m modals.Modal) tea.Cmd {
	a.modals = a.modals.push(m)
	return m.Init()
}

func (a *App) closeModal() {
	a.modals = a.modals.pop()
}

func (a App) updateModal(msg tea.Msg) (App, tea.Cmd) {
	var cmd tea.Cmd
	a.modals, cmd = a.modals.update(msg)
	return a, cmd
}

func (a *App) notify(text string) tea.Cmd {
	var cmd tea.Cmd
	a.notification, cmd = a.notification.Show(notification.NotificationSuccess, text)
	return cmd
}

func (a *App) reload() tea.Cmd {
	a.loadSeq++
	a.loading = true
	a.loadedQuery = queryFor(a.filters, a.me)
	return loadWorkItemsCmd(a.workItemsLoad.restart(a.client), a.loadSeq, a.loadedQuery)
}

func (a *App) applyView() {
	items := a.visibleItems()
	a.workItemsPanel.SetItems(items)
	a.boardPanel.SetItems(items, a.statesByType)
	a.taskboardPanel.SetItems(items, a.statesByType)
}

func (a *App) visibleItems() []models.WorkItem {
	person := a.filters.GetSelectedAssigned()
	keyword := a.keyword.Value()
	everyone := person == models.FilterAll || (person == models.AssignedMe && !a.me.Known())
	if everyone && strings.TrimSpace(keyword) == "" {
		return a.workItems
	}
	return models.FilterTree(a.workItems, func(it *models.WorkItem) bool {
		return models.MatchesAssignee(it, person, a.me) && models.MatchesKeyword(it, keyword)
	})
}

func (a *App) mainView() mainView {
	switch {
	case a.hub == panels.HubPullRequests:
		return viewPullRequests
	case a.hub == panels.HubBuilds:
		return viewPipelines
	case a.hub == panels.HubReleases:
		return viewReleases
	case a.hub == panels.HubFiles:
		return viewFiles
	case a.hub == panels.HubCommits:
		return viewCommits
	case a.hub == panels.HubBranches:
		return viewBranches
	case a.hub == panels.HubTags:
		return viewTags
	case a.hub == panels.HubBoard:
		return viewBoard
	case a.hub == panels.HubSprint && a.sprintView == sprintTaskboard:
		return viewTaskboard
	}
	return viewList
}

func (a *App) mutate(cmd tea.Cmd) tea.Cmd {
	a.loading = true
	return cmd
}

func (a *App) selectedItem() *models.WorkItem {
	switch a.mainView() {
	case viewBoard:
		return a.boardPanel.SelectedItem()
	case viewTaskboard:
		return a.taskboardPanel.SelectedItem()
	case viewPullRequests, viewPipelines, viewReleases, viewFiles, viewCommits, viewBranches, viewTags:
		return nil
	default:
		return a.workItemsPanel.SelectedItem()
	}
}

func (a *App) updateMainPanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch a.mainView() {
	case viewBoard:
		a.boardPanel, cmd = a.boardPanel.Update(msg)
	case viewTaskboard:
		a.taskboardPanel, cmd = a.taskboardPanel.Update(msg)
	case viewPullRequests:
		a.pr.panel, cmd = a.pr.panel.Update(msg)
	case viewPipelines:
		a.pl.panel, cmd = a.pl.panel.Update(msg)
	case viewReleases:
		cmd = a.updateReleasePanel(msg)
	case viewFiles:
		cmd = a.updateFilesPanel(msg)
	case viewCommits:
		a.commits.panel, cmd = a.commits.panel.Update(msg)
	case viewBranches:
		a.branches.panel, cmd = a.branches.panel.Update(msg)
	case viewTags:
		a.tags.panel, cmd = a.tags.panel.Update(msg)
	default:
		a.workItemsPanel, cmd = a.workItemsPanel.Update(msg)
	}
	return cmd
}

func registerWorkflowColors(statesByType map[string][]models.WorkItemStateInfo) {
	models.RegisterStateCategories(statesByType)
	for _, states := range statesByType {
		for _, st := range states {
			styles.RegisterStateColor(st.Name, st.Color)
		}
	}
}
