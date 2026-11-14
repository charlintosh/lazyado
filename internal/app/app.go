package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/components/screens"
	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charlintosh/lazyado/pkg/browser"
	"github.com/charlintosh/lazyado/pkg/git"
)

// Panel represents the active panel
type Panel int

const (
	PanelFilterSprint Panel = iota
	PanelFilterState
	PanelFilterAssigned
	PanelFilterArea
	PanelWorkItems
	PanelDetails
	PanelComments
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewMain ViewMode = iota
	ViewDetail
)

// App is the main application model
type App struct {
	// Components
	filterPanel      panels.FilterPanel
	workItemsPanel   panels.WorkItemsPanel
	boardPanel       panels.BoardPanel
	taskboardPanel   panels.TaskboardPanel
	detailsPanel     panels.DetailsPanel
	commentsPanel    panels.CommentsPanel
	commentModal     modals.CommentModal
	detailView       panels.DetailView
	helpPanel        panels.HelpPanel
	stateModal       modals.StateModal
	branchModal      modals.BranchModal
	assignModal      modals.AssignModal
	taskModal        modals.TaskModal
	deleteModal      modals.DeleteModal
	pbiModal         modals.PBIModal
	searchModal      modals.SearchModal
	parentSelection  modals.ParentSelectionModal
	bugParentModal   modals.BugParentModal
	errorModal       modals.ErrorModal
	infoModal        modals.InfoModal
	quickSearchModal modals.QuickSearchModal
	splashScreen     screens.SplashScreen
	notification     notification.Notification
	headerBar        panels.HeaderBar

	// State
	activePanel Panel
	rightTab    Panel
	hub         panels.Hub
	viewMode    ViewMode
	loading     bool
	err         error

	// Data
	iterations         []models.Iteration
	areas              []models.Area
	workItems          []models.WorkItem
	statesByType       map[string][]models.WorkItemStateInfo
	teamMembers        []models.TeamMember
	lastCommentsItemID int // Track last item for which comments were loaded

	// Services
	client *api.Client

	// Config
	styles styles.Styles
	keys   keys.KeyMap

	// Active filter group for panel navigation
	activeFilterGroup int

	// Dimensions
	width  int
	height int
	// Track whether the initial data+workitems load completed
	initialLoadDone bool
}

// NewApp creates a new application
func NewApp(client *api.Client) App {
	styles := styles.DefaultStyles()
	keys := keys.DefaultKeyMap()

	// Create empty filter state (will be populated after loading data)
	filterState := models.NewFilterState(nil, nil, nil)

	return App{
		filterPanel:       panels.NewFilterPanel(filterState, styles, keys),
		workItemsPanel:    panels.NewWorkItemsPanel(styles, keys),
		boardPanel:        panels.NewBoardPanel(styles, keys),
		taskboardPanel:    panels.NewTaskboardPanel(styles, keys),
		detailsPanel:      panels.NewDetailsPanel(styles, keys),
		commentsPanel:     panels.NewCommentsPanel(styles, keys),
		commentModal:      modals.NewCommentModal(styles, keys),
		detailView:        panels.NewDetailView(styles, keys),
		helpPanel:         panels.NewHelpPanel(keys, styles),
		stateModal:        modals.NewStateModal(styles, keys),
		branchModal:       modals.NewBranchModal(styles, keys),
		assignModal:       modals.NewAssignModal(styles, keys),
		taskModal:         modals.NewTaskModal(styles, keys),
		deleteModal:       modals.NewDeleteModal(styles, keys),
		pbiModal:          modals.NewPBIModal(styles, keys),
		searchModal:       modals.NewSearchModal(styles, keys),
		parentSelection:   modals.NewParentSelectionModal(styles, keys),
		bugParentModal:    modals.NewBugParentModal(styles, keys),
		errorModal:        modals.NewErrorModal(styles, keys),
		infoModal:         modals.NewInfoModal(styles, keys),
		quickSearchModal:  modals.NewQuickSearchModal(styles, keys),
		splashScreen:      screens.NewSplashScreen(styles),
		notification:      notification.NewNotification(styles, keys),
		headerBar:         panels.NewHeaderBar(styles, keys),
		activePanel:       PanelWorkItems,
		rightTab:          PanelDetails,
		hub:               panels.HubBacklog,
		viewMode:          ViewMain,
		loading:           true,
		client:            client,
		styles:            styles,
		keys:              keys,
		activeFilterGroup: 0,
	}
}

// Init initializes the application
func (a App) Init() tea.Cmd {
	return tea.Batch(
		loadDataCmd(a.client),
		a.headerBar.Init(),
		screens.AnimateCmd(),
		a.splashScreen.Tick(),
	)
}

// Update handles messages
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case screens.FrameMsg:
		if a.loading && !a.initialLoadDone {
			newSplash, cmd := a.splashScreen.Update(msg)
			a.splashScreen = newSplash
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case spinner.TickMsg:
		if a.loading && !a.initialLoadDone {
			newSplash, cmd := a.splashScreen.Update(msg)
			a.splashScreen = newSplash
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Always update headerBar spinner to keep it animated
		newHeaderBar, cmd := a.headerBar.Update(msg)
		a.headerBar = newHeaderBar
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateSizes()
		a.splashScreen.SetSize(a.width, a.height)

	case tea.MouseMsg:
		switch a.activePanel {
		case PanelWorkItems:
			if cmd := a.updateMainPanel(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case PanelDetails:
			newDetails, cmd := a.detailsPanel.Update(msg)
			a.detailsPanel = newDetails
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case PanelComments:
			newComments, cmd := a.commentsPanel.Update(msg)
			a.commentsPanel = newComments
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tea.KeyPressMsg:
		// Update notification first (captures ESC to dismiss)
		if a.notification.IsVisible() {
			newNotif, cmd := a.notification.Update(msg)
			a.notification = newNotif
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Don't process other keys if notification consumed the input
			if !a.notification.IsVisible() {
				return a, tea.Batch(cmds...)
			}
		}

		// Handle modals first (they capture all input when visible)
		if a.errorModal.IsVisible() {
			newModal, cmd := a.errorModal.Update(msg)
			a.errorModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.stateModal.IsVisible() {
			newModal, cmd := a.stateModal.Update(msg)
			a.stateModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.branchModal.IsVisible() {
			newModal, cmd := a.branchModal.Update(msg)
			a.branchModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.assignModal.IsVisible() {
			newModal, cmd := a.assignModal.Update(msg)
			a.assignModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.taskModal.IsVisible() {
			newModal, cmd := a.taskModal.Update(msg)
			a.taskModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.deleteModal.IsVisible() {
			newModal, cmd := a.deleteModal.Update(msg)
			a.deleteModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.pbiModal.IsVisible() {
			newModal, cmd := a.pbiModal.Update(msg)
			a.pbiModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.searchModal.IsVisible() {
			newModal, cmd := a.searchModal.Update(msg)
			a.searchModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.parentSelection.IsVisible() {
			newModal, cmd := a.parentSelection.Update(msg)
			a.parentSelection = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.bugParentModal.IsVisible() {
			newModal, cmd := a.bugParentModal.Update(msg)
			a.bugParentModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.infoModal.IsVisible() {
			newModal, cmd := a.infoModal.Update(msg)
			a.infoModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.commentModal.IsVisible() {
			newModal, cmd := a.commentModal.Update(msg)
			a.commentModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.quickSearchModal.IsVisible() {
			newModal, cmd := a.quickSearchModal.Update(msg)
			a.quickSearchModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Add spinner tick command if loading
			spinnerCmd := a.quickSearchModal.GetSpinnerCmd()
			if spinnerCmd != nil {
				cmds = append(cmds, spinnerCmd)
			}
			return a, tea.Batch(cmds...)
		}

		// Global keys
		if key.Matches(msg, a.keys.Quit) && !a.helpPanel.IsVisible() && a.viewMode == ViewMain {
			return a, tea.Quit
		}

		if key.Matches(msg, a.keys.Help) {
			a.helpPanel.Toggle()
			return a, nil
		}

		// Close help with any key if visible
		if a.helpPanel.IsVisible() {
			if key.Matches(msg, a.keys.Back) || key.Matches(msg, a.keys.Help) {
				a.helpPanel.SetVisible(false)
			}
			return a, nil
		}

		// Handle detail view mode
		if a.viewMode == ViewDetail {
			newDetailView, cmd := a.detailView.Update(msg)
			a.detailView = newDetailView
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if hub, ok := a.hubForKey(msg); ok {
			a.switchHub(hub)
			return a, a.updateSelectedItem()
		}

		// Panel switching with numbers (lazygit style) using centralized KeyMap
		switch {
		case key.Matches(msg, a.keys.Panel1):
			a.activePanel = PanelFilterSprint
			a.activeFilterGroup = 0
			a.updateFocus()
		case key.Matches(msg, a.keys.Panel2):
			a.activePanel = PanelFilterState
			a.activeFilterGroup = 1
			a.updateFocus()
		case key.Matches(msg, a.keys.Panel3):
			a.activePanel = PanelFilterAssigned
			a.activeFilterGroup = 2
			a.updateFocus()
		case key.Matches(msg, a.keys.Panel4):
			a.activePanel = PanelFilterArea
			a.activeFilterGroup = 3
			a.updateFocus()
		case key.Matches(msg, a.keys.Panel5):
			a.activePanel = PanelWorkItems
			a.updateFocus()
		case key.Matches(msg, a.keys.Panel6):
			a.activePanel = PanelDetails
			a.updateFocus()
		case key.Matches(msg, a.keys.Panel7):
			a.activePanel = PanelComments
			a.updateFocus()
		}
		a.applyLayout()

		// Panel switching with Tab
		if key.Matches(msg, a.keys.NextPanel) {
			a.nextPanel()
			a.updateFocus()
		}
		if key.Matches(msg, a.keys.PrevPanel) {
			a.prevPanel()
			a.updateFocus()
		}

		// Refresh
		if key.Matches(msg, a.keys.Refresh) {
			a.loading = true
			return a, loadWorkItemsCmd(a.client, a.filterPanel.FilterState())
		}

		// Quick search (Ctrl+F) - available in main view only
		if key.Matches(msg, a.keys.QuickSearch) && a.viewMode == ViewMain {
			a.quickSearchModal.SetSize(a.width, a.height)
			a.quickSearchModal.SetVisible(true)
			return a, nil
		}

		// Open state change modal (only when work items panel is active)
		if key.Matches(msg, a.keys.ChangeState) && a.activePanel == PanelWorkItems {
			if item := a.selectedItem(); item != nil {
				a.stateModal.SetItem(item)
				a.stateModal.SetSize(a.width, a.height)
				a.stateModal.SetVisible(true)
				return a, nil
			}
		}

		// Open branch modal (only when work items panel is active)
		if key.Matches(msg, a.keys.CreateBranch) && a.activePanel == PanelWorkItems {
			if item := a.selectedItem(); item != nil {
				a.branchModal.SetItem(item)
				a.branchModal.SetSize(a.width, a.height)
				a.branchModal.SetVisible(true)
				return a, nil
			}
		}

		// Open assign modal (only when work items panel is active)
		if key.Matches(msg, a.keys.Assign) && a.activePanel == PanelWorkItems {
			if item := a.selectedItem(); item != nil {
				a.assignModal.SetItem(item)
				a.assignModal.SetMembers(a.teamMembers)
				a.assignModal.SetSize(a.width, a.height)
				a.assignModal.SetVisible(true)
				return a, nil
			}
		}

		// Open comment modal (when work items or comments panel is active)
		if key.Matches(msg, a.keys.AddComment) && (a.activePanel == PanelWorkItems || a.activePanel == PanelComments) {
			if item := a.selectedItem(); item != nil {
				a.commentModal.SetItem(item)
				a.commentModal.SetMembers(a.teamMembers)
				a.commentModal.SetSize(a.width, a.height)
				a.commentModal.SetVisible(true)
				return a, nil
			}
		}

		// Create new task (only when work items panel is active)
		if key.Matches(msg, a.keys.CreateTask) && a.activePanel == PanelWorkItems {
			if item := a.selectedItem(); item != nil {
				// Only allow creating child tasks on parent types (PBI, User Story, Feature, Epic)
				if item.IsParentType() {
					a.taskModal.SetParent(item)
					a.taskModal.SetSize(a.width, a.height)
					a.taskModal.SetVisible(true)
					return a, nil
				}
			}
		}

		// Create new parent item (only when work items panel is active)
		if key.Matches(msg, a.keys.CreateParent) && a.activePanel == PanelWorkItems {
			// Show selection modal to pick type
			a.parentSelection.SetSize(a.width, a.height)
			a.parentSelection.SetVisible(true)

			// Prepare both modals with data so they are ready when selected
			a.pbiModal.SetData(a.iterations, a.areas)
			a.pbiModal.SetMembers(a.teamMembers)
			a.pbiModal.SetSize(a.width, a.height)

			a.bugParentModal.SetData(a.iterations, a.areas)
			a.bugParentModal.SetMembers(a.teamMembers)
			a.bugParentModal.SetSize(a.width, a.height)

			// Set default area from first visible work item (ensures user has permissions)
			if len(a.workItems) > 0 {
				a.pbiModal.SetDefaultArea(a.workItems[0].AreaPath)
				a.bugParentModal.SetDefaultArea(a.workItems[0].AreaPath)
			}
			return a, nil
		}

		// Edit task/PBI (only when work items panel is active)
		if key.Matches(msg, a.keys.EditTask) && a.activePanel == PanelWorkItems {
			if item := a.selectedItem(); item != nil {
				// Use different modals based on item type
				if item.HasAdvancedFields() {
					// For PBIs, Bugs, Stories - use the pbiModal for editing
					a.pbiModal.SetMembers(a.teamMembers)
					a.pbiModal.SetItem(item)
					a.pbiModal.SetSize(a.width, a.height)
					a.pbiModal.SetVisible(true)
				} else {
					// For tasks, use the simple task modal
					a.taskModal.SetTask(item)
					a.taskModal.SetSize(a.width, a.height)
					a.taskModal.SetVisible(true)
				}
				return a, nil
			}
		}

		// Delete task (only when work items panel is active)
		if key.Matches(msg, a.keys.DeleteTask) && a.activePanel == PanelWorkItems {
			if item := a.selectedItem(); item != nil {
				a.deleteModal.SetItem(item)
				a.deleteModal.SetSize(a.width, a.height)
				a.deleteModal.SetVisible(true)
				return a, nil
			}
		}

		// Open info modal (only when work items panel is active)
		if key.Matches(msg, a.keys.Info) && a.activePanel == PanelWorkItems {
			if item := a.selectedItem(); item != nil {
				a.infoModal.SetItem(item)
				a.infoModal.SetSize(a.width, a.height)
				a.infoModal.SetVisible(true)
				return a, nil
			}
		}

		// Update active panel
		switch a.activePanel {
		case PanelFilterSprint, PanelFilterState, PanelFilterAssigned, PanelFilterArea:
			newFilter, cmd := a.filterPanel.Update(msg)
			a.filterPanel = newFilter
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case PanelWorkItems:
			if cmd := a.updateMainPanel(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			if selectedCmd := a.updateSelectedItem(); selectedCmd != nil {
				cmds = append(cmds, selectedCmd)
			}
		case PanelDetails:
			newDetails, cmd := a.detailsPanel.Update(msg)
			a.detailsPanel = newDetails
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case PanelComments:
			newComments, cmd := a.commentsPanel.Update(msg)
			a.commentsPanel = newComments
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case dataLoadedMsg:
		a.iterations = msg.iterations
		a.areas = msg.areas
		a.statesByType = msg.statesByType
		a.teamMembers = msg.teamMembers
		a.stateModal.SetStatesByType(a.statesByType)
		registerWorkflowColors(a.statesByType)
		filterState := models.NewFilterState(a.iterations, a.areas, a.statesByType)

		// Apply saved filter selections
		if savedState, err := config.LoadFilterState(); err == nil {
			filterState.ApplySavedSelections(savedState.Sprint, savedState.State, savedState.Assigned, savedState.Area)
		}

		a.filterPanel.SetFilterState(filterState)
		// Load work items with initial filters
		return a, loadWorkItemsCmd(a.client, filterState)

	case workItemsLoadedMsg:
		a.loading = false
		a.workItems = msg.items
		a.workItemsPanel.SetItems(msg.items)
		a.boardPanel.SetItems(msg.items, a.statesByType)
		a.taskboardPanel.SetItems(msg.items, a.statesByType)
		a.applyLayout()
		a.initialLoadDone = true
		if cmd := a.updateSelectedItem(); cmd != nil {
			return a, cmd
		}

	case panels.FilterChangedMsg:
		a.loading = true
		fs := a.filterPanel.FilterState()

		// Save filter selections for next startup
		_ = config.SaveFilterState(&config.FilterState{
			Sprint:   fs.GetSelectedSprint(),
			State:    fs.GetSelectedState(),
			Assigned: fs.GetSelectedAssigned(),
			Area:     fs.GetSelectedArea(),
		})

		return a, loadWorkItemsCmd(a.client, fs)

	case panels.OpenWorkItemMsg:
		if err := browser.Open(msg.Item.WebURL); err != nil {
			a.err = err
		}

	case panels.OpenSearchMsg:
		// Open search modal for current filter group
		if group := a.filterPanel.FilterState().ActiveFilterGroup(); group != nil {
			a.searchModal.SetFilterGroup(group)
			a.searchModal.SetSize(a.width, a.height)
			a.searchModal.SetVisible(true)
		}

	case modals.SearchSelectedMsg:
		// User selected an option from search
		a.searchModal.SetVisible(false)
		if group := a.filterPanel.FilterState().ActiveFilterGroup(); group != nil {
			// Find and select the option in the original list
			group.ClearSearchFilter()
			if msg.SelectedOption != nil {
				group.SelectByValue(msg.SelectedOption.Value)
			}
		}
		return a, loadWorkItemsCmd(a.client, a.filterPanel.FilterState())

	case modals.SearchCancelledMsg:
		// User cancelled search
		a.searchModal.SetVisible(false)
		if group := a.filterPanel.FilterState().ActiveFilterGroup(); group != nil {
			group.ClearSearchFilter()
		}

	case panels.ViewWorkItemMsg:
		a.viewMode = ViewDetail
		a.detailView.SetItem(&msg.Item)
		a.updateSizes()
		// Load comments for the detail view
		return a, loadCommentsCmd(a.client, msg.Item.ID)

	case panels.CloseDetailViewMsg:
		a.viewMode = ViewMain

	case panels.OpenCommentModalMsg:
		// Open comment modal from detail view
		a.commentModal.SetItem(&msg.Item)
		a.commentModal.SetSize(a.width, a.height)
		a.commentModal.SetMembers(a.teamMembers)
		a.commentModal.SetVisible(true)

	case errMsg:
		a.loading = false
		a.errorModal.SetError(msg.err)
		a.errorModal.SetSize(a.width, a.height)
		a.errorModal.SetVisible(true)

	case modals.ModalClosedMsg:
		// Modal was closed, nothing special to do
		a.stateModal.SetVisible(false)
		a.branchModal.SetVisible(false)
		a.assignModal.SetVisible(false)
		a.taskModal.SetVisible(false)
		a.deleteModal.SetVisible(false)
		a.errorModal.SetVisible(false)
		a.parentSelection.SetVisible(false)
		a.pbiModal.SetVisible(false)
		a.bugParentModal.SetVisible(false)
		a.infoModal.SetVisible(false)
		a.quickSearchModal.SetVisible(false)

	case panels.MoveWorkItemMsg:
		a.loading = true
		return a, updateWorkItemStateCmd(a.client, msg.Item.ID, msg.NewState, a.filterPanel.FilterState())

	case modals.StateChangeRequestMsg:
		a.stateModal.SetVisible(false)
		a.loading = true
		return a, updateWorkItemStateCmd(a.client, msg.Item.ID, msg.NewState, a.filterPanel.FilterState())

	case stateChangedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, fmt.Sprintf("State changed to %s", msg.newState)))
		// Refresh work items to show updated state
		return a, tea.Batch(append(cmds, loadWorkItemsCmd(a.client, a.filterPanel.FilterState()))...)

	case modals.ParentSelectionChosenMsg:
		// User selected PBI or Bug from selection modal
		a.parentSelection.SetVisible(false)
		if msg.Type == "Product Backlog Item" {
			a.pbiModal.SetVisible(true)
		} else if msg.Type == "Bug" {
			a.bugParentModal.SetVisible(true)
		}
		return a, nil

	case modals.BranchCreateRequestMsg:
		a.branchModal.SetVisible(false)
		return a, createBranchCmd(msg.BranchName)

	case modals.BranchCreatedMsg:
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, fmt.Sprintf("Branch created: %s", msg.BranchName)))

	case modals.BranchCreateErrorMsg:
		a.err = msg.Err

	case modals.AssignRequestMsg:
		a.assignModal.SetVisible(false)
		a.loading = true
		return a, assignWorkItemCmd(a.client, msg.Item.ID, msg.UserEmail, msg.UserName, a.filterPanel.FilterState())

	case assignedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, fmt.Sprintf("Assigned to %s", msg.userName)))
		// Refresh work items to show updated assignment
		return a, tea.Batch(append(cmds, loadWorkItemsCmd(a.client, a.filterPanel.FilterState()))...)

	case modals.TaskCreateRequestMsg:
		a.taskModal.SetVisible(false)
		a.loading = true
		return a, createTaskCmd(a.client, msg.ParentID, msg.Title, msg.Description, a.filterPanel.FilterState())

	case taskCreatedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, fmt.Sprintf("Task created: #%d", msg.taskID)))
		return a, tea.Batch(append(cmds, loadWorkItemsCmd(a.client, a.filterPanel.FilterState()))...)

	case modals.ParentCreateRequestMsg:
		a.parentSelection.SetVisible(false)
		a.pbiModal.SetVisible(false)
		a.bugParentModal.SetVisible(false)
		a.loading = true

		// Get area and iteration from first visible work item (where user has permissions)
		// This avoids permission issues with team/project defaults
		var areaPath, iterationPath string
		if len(a.workItems) > 0 {
			areaPath = a.workItems[0].AreaPath
			iterationPath = a.workItems[0].IterationPath
		}

		// Use values from modal with default state "New"
		return a, createParentWorkItemCmd(a.client, msg.WorkItemType, msg.Title, msg.Description, msg.AcceptanceCriteria, "New", iterationPath, areaPath, msg.AssignedTo, msg.Effort, msg.Priority, msg.Tags, msg.DefectCause, a.filterPanel.FilterState())

	case parentCreatedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, fmt.Sprintf("Parent item created: #%d", msg.itemID)))
		return a, tea.Batch(append(cmds, loadWorkItemsCmd(a.client, a.filterPanel.FilterState()))...)

	case modals.TaskUpdateRequestMsg:
		a.taskModal.SetVisible(false)
		a.loading = true
		return a, updateTaskCmd(a.client, msg.TaskID, msg.NewTitle, msg.Description, a.filterPanel.FilterState())

	case taskUpdatedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, "Task updated"))
		return a, tea.Batch(append(cmds, loadWorkItemsCmd(a.client, a.filterPanel.FilterState()))...)

	case modals.TaskDeleteRequestMsg:
		a.deleteModal.SetVisible(false)
		a.loading = true
		return a, deleteTaskCmd(a.client, msg.TaskID, a.filterPanel.FilterState())

	case taskDeletedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, "Task deleted"))
		return a, tea.Batch(append(cmds, loadWorkItemsCmd(a.client, a.filterPanel.FilterState()))...)

	case modals.PBIUpdateRequestMsg:
		a.pbiModal.SetVisible(false)
		a.loading = true
		return a, updatePBICmd(a.client, msg.ItemID, msg.Title, msg.Description, msg.Priority, msg.Effort, msg.Tags, msg.AssignedTo, a.filterPanel.FilterState())

	case pbiUpdatedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, "PBI updated"))
		return a, tea.Batch(append(cmds, loadWorkItemsCmd(a.client, a.filterPanel.FilterState()))...)

	case panels.CommentsLoadedMsg:
		a.commentsPanel.SetComments(msg.Comments)
		a.detailView.SetComments(msg.Comments)

	case modals.CommentCreateRequestMsg:
		a.commentModal.SetVisible(false)
		a.loading = true
		return a, createCommentCmd(a.client, msg.WorkItemID, msg.Text, a.filterPanel.FilterState())

	case commentCreatedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, "Comment added"))
		// Reload comments to show the new one
		if item := a.selectedItem(); item != nil {
			cmds = append(cmds, loadCommentsCmd(a.client, item.ID))
		}
		return a, tea.Batch(cmds...)

	case panels.EditCommentMsg:
		if item := a.selectedItem(); item != nil {
			a.commentModal.SetComment(&msg.Comment, item.ID)
			a.commentModal.SetMembers(a.teamMembers)
			a.commentModal.SetSize(a.width, a.height)
			a.commentModal.SetVisible(true)
		}

	case modals.CommentUpdatedMsg:
		a.commentModal.SetVisible(false)
		a.loading = true
		return a, updateCommentCmd(a.client, msg.WorkItemID, msg.CommentID, msg.Text)

	case commentUpdatedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, "Comment updated"))
		if item := a.selectedItem(); item != nil {
			cmds = append(cmds, loadCommentsCmd(a.client, item.ID))
		}
		return a, tea.Batch(cmds...)

	case panels.DeleteCommentMsg:
		if item := a.selectedItem(); item != nil {
			a.deleteModal.SetCommentItem(&msg.Comment, item.ID)
			a.deleteModal.SetSize(a.width, a.height)
			a.deleteModal.SetVisible(true)
		}

	case modals.CommentDeleteRequestMsg:
		a.deleteModal.SetVisible(false)
		a.loading = true
		return a, deleteCommentCmd(a.client, msg.WorkItemID, msg.CommentID)

	case commentDeletedMsg:
		a.loading = false
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, "Comment deleted"))
		if item := a.selectedItem(); item != nil {
			cmds = append(cmds, loadCommentsCmd(a.client, item.ID))
		}
		return a, tea.Batch(cmds...)

	case notification.AutoDismissMsg:
		// Update notification to handle auto-dismiss
		newNotif, cmd := a.notification.Update(msg)
		a.notification = newNotif
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case modals.InfoCopiedMsg:
		a.infoModal.SetVisible(false)
		cmds = append(cmds, a.notification.Show(notification.NotificationSuccess, msg.Text))

	case modals.InfoCopyErrorMsg:
		a.infoModal.SetVisible(false)
		a.errorModal.SetError(msg.Err)
		a.errorModal.SetSize(a.width, a.height)
		a.errorModal.SetVisible(true)

	case modals.QuickSearchRequestMsg:
		a.quickSearchModal.SetLoading(true)
		return a, quickSearchWorkItemCmd(a.client, msg.ID)

	case modals.QuickSearchSuccessMsg:
		a.quickSearchModal.SetVisible(false)
		a.viewMode = ViewDetail
		a.detailView.SetItem(msg.Item)
		a.updateSizes()
		return a, loadCommentsCmd(a.client, msg.Item.ID)

	case modals.QuickSearchErrorMsg:
		a.quickSearchModal.SetLoading(false)
		a.quickSearchModal.SetError(msg.Err)
	}

	return a, tea.Batch(cmds...)
}

// View renders the application
func (a App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (a App) render() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Show splash screen only for the first (initial) data load
	if a.loading && len(a.workItems) == 0 && !a.initialLoadDone {
		return a.splashScreen.View()
	}

	// Render error modal if visible (highest priority)
	if a.errorModal.IsVisible() {
		return a.errorModal.View()
	}

	// Render state modal if visible
	if a.stateModal.IsVisible() {
		return a.stateModal.View()
	}

	// Render branch modal if visible
	if a.branchModal.IsVisible() {
		return a.branchModal.View()
	}

	// Render assign modal if visible
	if a.assignModal.IsVisible() {
		return a.assignModal.View()
	}

	// Render task modal if visible
	if a.taskModal.IsVisible() {
		return a.taskModal.View()
	}

	// Render delete modal if visible
	if a.deleteModal.IsVisible() {
		return a.deleteModal.View()
	}

	// Render PBI modal if visible
	if a.pbiModal.IsVisible() {
		return a.pbiModal.View()
	}

	// Render search modal if visible
	if a.searchModal.IsVisible() {
		return a.searchModal.View()
	}

	// Render parent selection/modal if visible
	if a.parentSelection.IsVisible() {
		return a.parentSelection.View()
	}
	if a.bugParentModal.IsVisible() {
		return a.bugParentModal.View()
	}

	// Render info modal if visible
	if a.infoModal.IsVisible() {
		return a.infoModal.View()
	}

	// Render comment modal if visible
	if a.commentModal.IsVisible() {
		return a.commentModal.View()
	}

	if a.quickSearchModal.IsVisible() {
		return a.quickSearchModal.View()
	}

	if a.width < styles.ShellMinWidth || a.height < styles.ShellMinHeight {
		return a.renderTooSmall()
	}

	if a.helpPanel.IsVisible() {
		return a.helpPanel.View()
	}

	if a.viewMode == ViewDetail {
		return a.renderDetailView()
	}

	return a.renderMainView()
}

type shellLayout struct {
	bodyHeight int
	showRail   bool
	railWidth  int
	mainWidth  int
	showPane   bool
	paneWidth  int
}

func (a *App) layout() shellLayout {
	l := shellLayout{bodyHeight: max(a.height-styles.TopBarHeight-styles.StatusBarHeight, 0)}
	l.showRail = a.width >= styles.RailShowWidth
	if l.showRail {
		l.railWidth = styles.RailWidth
	}
	rest := a.width - l.railWidth
	l.showPane = a.hub == panels.HubBacklog && a.width >= styles.RightPaneShowWidth
	if l.showPane {
		pw := int(float64(a.width) * styles.RightPaneWidthRatio)
		l.paneWidth = max(min(pw, styles.RightPaneMaxWidth), styles.RightPaneMinWidth)
		rest -= l.paneWidth
	}
	l.mainWidth = max(rest, 0)
	return l
}

func (a *App) isFilterPanel() bool {
	return a.activePanel == PanelFilterSprint || a.activePanel == PanelFilterState ||
		a.activePanel == PanelFilterAssigned || a.activePanel == PanelFilterArea
}

func (a *App) isPanePanel() bool {
	return a.activePanel == PanelDetails || a.activePanel == PanelComments
}

func (a *App) applyLayout() {
	if a.width == 0 || a.height == 0 {
		return
	}
	l := a.layout()
	mainHeight := max(l.bodyHeight-styles.PageHeaderHeight, 0)

	railWidth := l.railWidth
	if !l.showRail {
		railWidth = a.width
	}
	a.filterPanel.SetSize(railWidth, l.bodyHeight)
	a.filterPanel.SetHub(a.hub)

	cw, ch := panels.BoxContentSize(l.mainWidth, mainHeight)
	a.workItemsPanel.SetSize(cw, ch)
	a.boardPanel.SetSize(cw, ch)
	a.taskboardPanel.SetSize(cw, ch)

	pw, ph := panels.BoxContentSize(l.paneWidth, l.bodyHeight)
	if !l.showPane {
		pw, ph = cw, ch
	}
	a.detailsPanel.SetSize(pw, ph)
	a.commentsPanel.SetSize(pw, ph)
	a.detailView.SetSize(a.width, l.bodyHeight)
}

func (a *App) renderTooSmall() string {
	msg := a.styles.TextMuted.Render(fmt.Sprintf("Terminal too small (%d×%d).\nResize to at least %d×%d.",
		a.width, a.height, styles.ShellMinWidth, styles.ShellMinHeight))
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, msg)
}

func (a *App) renderHeaderBar(crumbs ...string) string {
	a.headerBar.SetWidth(a.width)
	a.headerBar.SetOrganization(a.client.Organization())
	a.headerBar.SetProject(a.client.Project())
	a.headerBar.SetCrumbs(crumbs...)
	a.headerBar.SetLoading(a.loading)
	a.headerBar.SetNotification(&a.notification)
	return a.headerBar.View()
}

func (a *App) renderMainView() string {
	a.applyLayout()
	l := a.layout()
	header := a.renderHeaderBar("Boards", a.hub.Name())

	var body string
	switch {
	case a.isFilterPanel() && !l.showRail:
		body = a.filterPanel.View()
	default:
		var cols []string
		if l.showRail {
			cols = append(cols, a.filterPanel.View())
		}
		cols = append(cols, a.renderMainColumn(l))
		if l.showPane {
			cols = append(cols, a.renderPane(l.paneWidth, l.bodyHeight))
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	}
	body = panels.FitBlock(body, a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, a.renderStatusBar())
}

func (a *App) renderMainColumn(l shellLayout) string {
	mainHeight := max(l.bodyHeight-styles.PageHeaderHeight, 0)
	var box string
	if a.isPanePanel() && !l.showPane {
		box = a.renderPane(l.mainWidth, mainHeight)
	} else {
		focused := a.activePanel == PanelWorkItems
		title, content := a.mainPanelView()
		box = panels.RenderTitledBox(panels.BoxTitle("5", title, focused), content, l.mainWidth, mainHeight, focused)
	}
	header := panels.FitBlock(a.renderPageHeader(l.mainWidth), l.mainWidth, styles.PageHeaderHeight)
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (a *App) mainPanelView() (string, string) {
	switch a.hub {
	case panels.HubBoard:
		return a.boardPanel.Title(), a.boardPanel.View()
	case panels.HubSprint:
		return a.taskboardPanel.Title(), a.taskboardPanel.View()
	default:
		return a.workItemsPanel.Title(), a.workItemsPanel.View()
	}
}

func (a *App) renderPageHeader(width int) string {
	team := a.client.Team()
	if team == "" {
		team = a.client.Project()
	}
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(" " + team)
	sep := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  ›  ")
	view := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Underline(true)
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)

	var line1 string
	var tools []string
	switch a.hub {
	case panels.HubBoard:
		line1 = title + sep + view.Render("Board")
		tools = []string{
			panels.ToolbarItem("+", "New item", a.keys.CreateParent),
			panels.ToolbarItem("☰", "View as Backlog", a.keys.HubBacklog),
			panels.ToolbarItem("⇄", "Move card", a.keys.MoveItemRight),
			panels.ToolbarItem("◷", "Taskboard", a.keys.HubSprint),
			panels.ToolbarItem("↻", "Refresh", a.keys.Refresh),
		}
	case panels.HubSprint:
		line1 = title + sep + view.Render(a.sprintName()) + muted.Render(a.sprintDates())
		tools = []string{
			panels.ToolbarItem("+", "New task", a.keys.CreateTask),
			panels.ToolbarItem("⇄", "Move task", a.keys.MoveItemRight),
			panels.ToolbarItem("⊞", "Board", a.keys.HubBoard),
			panels.ToolbarItem("⫶", "Iteration", a.keys.Panel1),
			panels.ToolbarItem("↻", "Refresh", a.keys.Refresh),
		}
	default:
		line1 = title + sep + view.Render("Backlog")
		tools = []string{
			panels.ToolbarItem("+", "New Work Item", a.keys.CreateParent),
			panels.ToolbarItem("⊞", "View as Board", a.keys.HubBoard),
			panels.ToolbarItem("◷", "Taskboard", a.keys.HubSprint),
			panels.ToolbarItem("▸", "Expand", a.keys.Right),
			panels.ToolbarItem("↻", "Refresh", a.keys.Refresh),
		}
	}
	toolbar := " " + strings.Join(tools, "    ")
	return panels.FitWidth(line1, width) + "\n" + panels.FitWidth(toolbar, width)
}

func (a *App) currentIteration() *models.Iteration {
	selected := ""
	if fs := a.filterPanel.FilterState(); fs != nil {
		selected = fs.GetSelectedSprint()
	}
	var current *models.Iteration
	for i := range a.iterations {
		it := &a.iterations[i]
		if it.Path == selected {
			return it
		}
		if it.IsCurrent() {
			current = it
		}
	}
	return current
}

func (a *App) sprintName() string {
	if it := a.currentIteration(); it != nil {
		return it.Name
	}
	return "All iterations"
}

func (a *App) sprintDates() string {
	it := a.currentIteration()
	if it == nil || it.StartDate.IsZero() || it.FinishDate.IsZero() {
		return ""
	}
	out := "   " + it.StartDate.Format("Jan 2") + " – " + it.FinishDate.Format("Jan 2")
	if days := workingDaysLeft(time.Now(), it.FinishDate); days > 0 {
		out += fmt.Sprintf(" · %d working days left", days)
	}
	return out
}

func workingDaysLeft(from, to time.Time) int {
	days := 0
	for d := from.Truncate(24 * time.Hour); !d.After(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}

func (a *App) renderPane(width, height int) string {
	detailsFocused := a.activePanel == PanelDetails
	commentsFocused := a.activePanel == PanelComments
	discussion := "Discussion"
	if n := a.commentsPanel.Count(); n > 0 {
		discussion = fmt.Sprintf("Discussion %d", n)
	}
	detailsTitle := panels.BoxTitle("6", "Details", detailsFocused)
	commentsTitle := panels.BoxTitle("7", discussion, commentsFocused)
	if a.rightTab == PanelDetails && !detailsFocused {
		detailsTitle = panels.BoxTitle("6", "Details", false) + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("•")
	}
	if a.rightTab == PanelComments && !commentsFocused {
		commentsTitle = panels.BoxTitle("7", discussion, false) + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("•")
	}
	title := detailsTitle + lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(" │ ") + commentsTitle
	content := a.detailsPanel.View()
	if a.rightTab == PanelComments {
		content = a.commentsPanel.View()
	}
	return panels.RenderTitledBox(title, content, width, height, detailsFocused || commentsFocused)
}

func (a *App) renderDetailView() string {
	a.applyLayout()
	l := a.layout()
	id := ""
	if item := a.detailItem(); item != nil {
		id = fmt.Sprintf("%d", item.ID)
	}
	header := a.renderHeaderBar("Boards", a.hub.Name(), id)
	body := panels.FitBlock(a.detailView.View(), a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, a.renderStatusBar())
}

func (a *App) detailItem() *models.WorkItem {
	return a.detailView.Item()
}

func (a *App) modeLabel() string {
	switch {
	case a.viewMode == ViewDetail:
		return "WORK ITEM"
	case a.isFilterPanel():
		return "FILTERS"
	case a.activePanel == PanelDetails:
		return "DETAILS"
	case a.activePanel == PanelComments:
		return "DISCUSSION"
	default:
		return a.hub.Mode()
	}
}

func (a *App) renderStatusBar() string {
	pill := lipgloss.NewStyle().Foreground(styles.ColorOnAccent).Background(styles.ColorPrimary).Bold(true).
		Render(" " + a.modeLabel() + " ")
	left := append([]string{pill}, a.statusHints()...)
	right := panels.Hints(styles.ColorSurface, a.keys.Help, a.keys.Quit)
	return panels.JoinBar(left, right, a.width, styles.ColorSurface)
}

func (a *App) statusHints() []string {
	if a.viewMode == ViewDetail {
		return a.detailView.GetAvailableActions()
	}
	switch a.activePanel {
	case PanelWorkItems:
		switch a.hub {
		case panels.HubBoard:
			return a.boardPanel.GetAvailableActions()
		case panels.HubSprint:
			return a.taskboardPanel.GetAvailableActions()
		default:
			return a.workItemsPanel.GetAvailableActions()
		}
	case PanelDetails:
		return a.detailsPanel.GetAvailableActions()
	case PanelComments:
		return a.commentsPanel.GetAvailableActions()
	default:
		return a.filterPanel.GetAvailableActions()
	}
}

func (a *App) selectedItem() *models.WorkItem {
	switch a.hub {
	case panels.HubBoard:
		return a.boardPanel.SelectedItem()
	case panels.HubSprint:
		return a.taskboardPanel.SelectedItem()
	default:
		return a.workItemsPanel.SelectedItem()
	}
}

func (a *App) updateMainPanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch a.hub {
	case panels.HubBoard:
		a.boardPanel, cmd = a.boardPanel.Update(msg)
	case panels.HubSprint:
		a.taskboardPanel, cmd = a.taskboardPanel.Update(msg)
	default:
		a.workItemsPanel, cmd = a.workItemsPanel.Update(msg)
	}
	return cmd
}

func (a *App) hubForKey(msg tea.KeyPressMsg) (panels.Hub, bool) {
	switch {
	case key.Matches(msg, a.keys.HubBacklog):
		return panels.HubBacklog, true
	case key.Matches(msg, a.keys.HubBoard):
		return panels.HubBoard, true
	case key.Matches(msg, a.keys.HubSprint):
		return panels.HubSprint, true
	}
	return a.hub, false
}

func (a *App) switchHub(hub panels.Hub) {
	a.hub = hub
	a.activePanel = PanelWorkItems
	a.updateFocus()
	a.applyLayout()
}

func registerWorkflowColors(statesByType map[string][]models.WorkItemStateInfo) {
	models.RegisterStateCategories(statesByType)
	for _, states := range statesByType {
		for _, st := range states {
			styles.RegisterStateColor(st.Name, st.Color)
		}
	}
}

func (a *App) nextPanel() {
	switch a.activePanel {
	case PanelFilterSprint, PanelFilterState, PanelFilterAssigned, PanelFilterArea:
		a.activePanel = PanelWorkItems
	case PanelWorkItems:
		a.activePanel = PanelDetails
	case PanelDetails:
		a.activePanel = PanelComments
	case PanelComments:
		switch a.activeFilterGroup {
		case 0:
			a.activePanel = PanelFilterSprint
		case 1:
			a.activePanel = PanelFilterState
		case 2:
			a.activePanel = PanelFilterAssigned
		case 3:
			a.activePanel = PanelFilterArea
		default:
			a.activePanel = PanelFilterSprint
			a.activeFilterGroup = 0
		}
	}
}

func (a *App) prevPanel() {
	switch a.activePanel {
	case PanelComments:
		a.activePanel = PanelDetails
	case PanelDetails:
		a.activePanel = PanelWorkItems
	case PanelWorkItems:
		switch a.activeFilterGroup {
		case 0:
			a.activePanel = PanelFilterSprint
		case 1:
			a.activePanel = PanelFilterState
		case 2:
			a.activePanel = PanelFilterAssigned
		case 3:
			a.activePanel = PanelFilterArea
		default:
			a.activePanel = PanelFilterSprint
			a.activeFilterGroup = 0
		}
	case PanelFilterSprint, PanelFilterState, PanelFilterAssigned, PanelFilterArea:
		a.activePanel = PanelComments
	}
}

func (a *App) updateFocus() {
	isFilter := a.isFilterPanel()
	a.filterPanel.SetFocused(isFilter)
	if isFilter {
		a.filterPanel.SetActiveGroup(a.activeFilterGroup)
	}
	main := a.activePanel == PanelWorkItems
	a.workItemsPanel.SetFocused(main && a.hub == panels.HubBacklog)
	a.boardPanel.SetFocused(main && a.hub == panels.HubBoard)
	a.taskboardPanel.SetFocused(main && a.hub == panels.HubSprint)
	a.detailsPanel.SetFocused(a.activePanel == PanelDetails)
	a.commentsPanel.SetFocused(a.activePanel == PanelComments)
	if a.isPanePanel() {
		a.rightTab = a.activePanel
	}
}

func (a *App) updateSizes() {
	a.helpPanel.SetSize(a.width, a.height)
	a.notification.SetSize(a.width, a.height)
	a.headerBar.SetWidth(a.width)
	a.taskModal.SetSize(a.width, a.height)
	a.deleteModal.SetSize(a.width, a.height)
	a.pbiModal.SetSize(a.width, a.height)
	a.commentModal.SetSize(a.width, a.height)
	a.parentSelection.SetSize(a.width, a.height)
	a.bugParentModal.SetSize(a.width, a.height)
	a.errorModal.SetSize(a.width, a.height)
	a.infoModal.SetSize(a.width, a.height)
	a.quickSearchModal.SetSize(a.width, a.height)
	a.updateFocus()
	a.applyLayout()
}

func (a *App) updateSelectedItem() tea.Cmd {
	item := a.selectedItem()
	a.detailsPanel.SetItem(item)

	if item != nil && item.ID != a.lastCommentsItemID {
		// Load comments only if item changed
		a.lastCommentsItemID = item.ID
		return loadCommentsCmd(a.client, item.ID)
	}

	if item == nil && a.lastCommentsItemID != 0 {
		// Clear comments if no item selected
		a.lastCommentsItemID = 0
		a.commentsPanel.ClearComments()
	}

	return nil
}

// Message types

type dataLoadedMsg struct {
	iterations   []models.Iteration
	areas        []models.Area
	statesByType map[string][]models.WorkItemStateInfo
	teamMembers  []models.TeamMember
}

type workItemsLoadedMsg struct {
	items []models.WorkItem
}

type errMsg struct {
	err error
}

type stateChangedMsg struct {
	newState string
}

type assignedMsg struct {
	userName string
}

type taskCreatedMsg struct {
	taskID int
}

type parentCreatedMsg struct {
	itemID int
}

type taskUpdatedMsg struct{}

type taskDeletedMsg struct{}

type pbiUpdatedMsg struct{}

type commentCreatedMsg struct{}

type commentUpdatedMsg struct{}

type commentDeletedMsg struct{}

// Commands

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
			// Non-fatal - we can still work with hardcoded states
			statesByType = make(map[string][]models.WorkItemStateInfo)
		}
		teamMembers, err := client.GetTeamMembers()
		if err != nil {
			// Non-fatal - we can still work without team members
			teamMembers = []models.TeamMember{}
		}
		return dataLoadedMsg{iterations: iterations, areas: areas, statesByType: statesByType, teamMembers: teamMembers}
	}
}

func loadWorkItemsCmd(client *api.Client, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		sprint := filterState.GetSelectedSprint()
		state := filterState.GetSelectedState()
		assigned := filterState.GetSelectedAssigned()
		area := filterState.GetSelectedArea()

		items, err := client.QueryWorkItems(sprint, state, assigned, area)
		if err != nil {
			return errMsg{err: err}
		}
		return workItemsLoadedMsg{items: items}
	}
}

func loadCommentsCmd(client *api.Client, workItemID int) tea.Cmd {
	return func() tea.Msg {
		comments, err := client.GetComments(workItemID)
		if err != nil {
			// Don't fail if we can't load comments, just return empty
			return panels.CommentsLoadedMsg{Comments: []models.Comment{}}
		}
		return panels.CommentsLoadedMsg{Comments: comments}
	}
}

func updateWorkItemStateCmd(client *api.Client, itemID int, newState string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItemState(itemID, newState)
		if err != nil {
			return errMsg{err: err}
		}
		return stateChangedMsg{newState: newState}
	}
}

func createBranchCmd(branchName string) tea.Cmd {
	return func() tea.Msg {
		if !git.IsGitRepo() {
			return modals.BranchCreateErrorMsg{Err: fmt.Errorf("not a git repository")}
		}
		if git.HasUncommittedChanges() {
			return modals.BranchCreateErrorMsg{Err: fmt.Errorf("uncommitted changes exist")}
		}
		err := git.CreateBranch(branchName, true)
		if err != nil {
			return modals.BranchCreateErrorMsg{Err: err}
		}
		return modals.BranchCreatedMsg{BranchName: branchName}
	}
}

func assignWorkItemCmd(client *api.Client, itemID int, userEmail, userName string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.AssignWorkItem(itemID, userEmail)
		if err != nil {
			return errMsg{err: err}
		}
		return assignedMsg{userName: userName}
	}
}

func createTaskCmd(client *api.Client, parentID int, title, description string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		// Get parent item to copy area, iteration paths, and assigned user
		parent, err := client.GetWorkItem(parentID)
		if err != nil {
			return errMsg{err: err}
		}

		taskID, err := client.CreateWorkItem("Task", title, description, parentID, parent.AreaPath, parent.IterationPath, parent.AssignedTo)
		if err != nil {
			return errMsg{err: err}
		}
		return taskCreatedMsg{taskID: taskID}
	}
}

func createParentWorkItemCmd(client *api.Client, workItemType, title, description, acceptanceCriteria, state, iterationPath, areaPath, assignedTo string, effort float64, priority int, tags []string, defectCause string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		itemID, err := client.CreateParentWorkItem(workItemType, title, description, acceptanceCriteria, state, iterationPath, areaPath, assignedTo, effort, priority, tags, defectCause)
		if err != nil {
			return errMsg{err: err}
		}
		return parentCreatedMsg{itemID: itemID}
	}
}

func updateTaskCmd(client *api.Client, taskID int, newTitle, description string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItem(taskID, newTitle, description)
		if err != nil {
			return errMsg{err: err}
		}
		return taskUpdatedMsg{}
	}
}

func deleteTaskCmd(client *api.Client, taskID int, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteWorkItem(taskID)
		if err != nil {
			return errMsg{err: err}
		}
		return taskDeletedMsg{}
	}
}

func updatePBICmd(client *api.Client, itemID int, title, description string, priority int, effort float64, tags []string, assignedTo string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdatePBI(itemID, title, description, priority, effort, tags, assignedTo)
		if err != nil {
			return errMsg{err: err}
		}
		return pbiUpdatedMsg{}
	}
}

func createCommentCmd(client *api.Client, workItemID int, text string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.CreateComment(workItemID, text)
		if err != nil {
			return errMsg{err: err}
		}
		return commentCreatedMsg{}
	}
}

func updateCommentCmd(client *api.Client, workItemID, commentID int, text string) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateComment(workItemID, commentID, text)
		if err != nil {
			return errMsg{err: err}
		}
		return commentUpdatedMsg{}
	}
}

func deleteCommentCmd(client *api.Client, workItemID, commentID int) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteComment(workItemID, commentID)
		if err != nil {
			return errMsg{err: err}
		}
		return commentDeletedMsg{}
	}
}

func quickSearchWorkItemCmd(client *api.Client, itemID int) tea.Cmd {
	return func() tea.Msg {
		item, err := client.GetWorkItem(itemID)
		if err != nil {
			return modals.QuickSearchErrorMsg{Err: err}
		}
		return modals.QuickSearchSuccessMsg{Item: item}
	}
}
