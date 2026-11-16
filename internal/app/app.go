package ui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
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
)

type App struct {
	workItemsPanel panels.WorkItemsPanel
	boardPanel     panels.BoardPanel
	taskboardPanel panels.TaskboardPanel
	detailsPanel   panels.DetailsPanel
	commentsPanel  panels.CommentsPanel
	detailView     panels.DetailView
	splashScreen   screens.SplashScreen
	notification   notification.Notification
	statusHelp     help.Model
	spinner        spinner.Model
	modals         modalStack
	keyword        textinput.Model
	filters        *models.FilterState
	loadedQuery    workItemQuery
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
	initialLoadDone bool

	iterations   []models.Iteration
	areas        []models.Area
	workItems    []models.WorkItem
	statesByType map[string][]models.WorkItemStateInfo
	teamMembers  []models.TeamMember
	me           models.Identity

	client *api.Client
	styles *styles.Styles
	keys   keys.KeyMap
	width  int
	height int
}

func NewApp(client *api.Client) App {
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
		splashScreen:   screens.NewSplashScreen(st),
		statusHelp:     widgets.NewHelp(),
		spinner: spinner.New(
			spinner.WithSpinner(spinner.Line),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary)),
		),
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
	return tea.Batch(selection, a.ensureSpinner())
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
	return a.loading || a.commentsPanel.Loading() || (a.viewMode == ViewDetail && a.detailView.Loading())
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
	return loadWorkItemsCmd(a.client, a.loadSeq, a.loadedQuery)
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
