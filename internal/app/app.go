package ui

import (
	"fmt"
	"strings"

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
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
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
		headerBar:         panels.NewHeaderBar(styles),
		activePanel:       PanelWorkItems,
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
		tickCmd(),
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

	case tea.KeyMsg:
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
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.stateModal.SetItem(item)
				a.stateModal.SetSize(a.width, a.height)
				a.stateModal.SetVisible(true)
				return a, nil
			}
		}

		// Open branch modal (only when work items panel is active)
		if key.Matches(msg, a.keys.CreateBranch) && a.activePanel == PanelWorkItems {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.branchModal.SetItem(item)
				a.branchModal.SetSize(a.width, a.height)
				a.branchModal.SetVisible(true)
				return a, nil
			}
		}

		// Open assign modal (only when work items panel is active)
		if key.Matches(msg, a.keys.Assign) && a.activePanel == PanelWorkItems {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.assignModal.SetItem(item)
				a.assignModal.SetMembers(a.teamMembers)
				a.assignModal.SetSize(a.width, a.height)
				a.assignModal.SetVisible(true)
				return a, nil
			}
		}

		// Open comment modal (when work items or comments panel is active)
		if key.Matches(msg, a.keys.AddComment) && (a.activePanel == PanelWorkItems || a.activePanel == PanelComments) {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.commentModal.SetItem(item)
				a.commentModal.SetMembers(a.teamMembers)
				a.commentModal.SetSize(a.width, a.height)
				a.commentModal.SetVisible(true)
				return a, nil
			}
		}

		// Create new task (only when work items panel is active)
		if key.Matches(msg, a.keys.CreateTask) && a.activePanel == PanelWorkItems {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
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
			if item := a.workItemsPanel.SelectedItem(); item != nil {
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
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.deleteModal.SetItem(item)
				a.deleteModal.SetSize(a.width, a.height)
				a.deleteModal.SetVisible(true)
				return a, nil
			}
		}

		// Open info modal (only when work items panel is active)
		if key.Matches(msg, a.keys.Info) && a.activePanel == PanelWorkItems {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
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
			newWorkItems, cmd := a.workItemsPanel.Update(msg)
			a.workItemsPanel = newWorkItems
			if cmd != nil {
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
		if item := a.workItemsPanel.SelectedItem(); item != nil {
			cmds = append(cmds, loadCommentsCmd(a.client, item.ID))
		}
		return a, tea.Batch(cmds...)

	case panels.EditCommentMsg:
		if item := a.workItemsPanel.SelectedItem(); item != nil {
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
		if item := a.workItemsPanel.SelectedItem(); item != nil {
			cmds = append(cmds, loadCommentsCmd(a.client, item.ID))
		}
		return a, tea.Batch(cmds...)

	case panels.DeleteCommentMsg:
		if item := a.workItemsPanel.SelectedItem(); item != nil {
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
		if item := a.workItemsPanel.SelectedItem(); item != nil {
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
func (a App) View() string {
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

	// Render help overlay if visible
	if a.helpPanel.IsVisible() {
		_ = a.renderMainView()
		help := a.helpPanel.View()
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, help)
	}

	// Render detail view if in detail mode
	if a.viewMode == ViewDetail {
		return a.detailView.View()
	}

	return a.renderMainView()
}

func (a *App) renderMainView() string {
	if a.width < styles.MinTerminalWidth || a.height < styles.MinTerminalHeight {
		return a.renderMinimalView()
	}

	// Lipgloss border rule: Width(W) → outer = W+2; Height(H) → outer = H+2.
	// All SetSize calls receive content dimensions; borders are added externally.

	availableHeight := a.height - styles.AppHeaderFooterSize // header(1) + statusBar(1)

	// ── Filter panel (left column) ───────────────────────────────────
	// Hide the filter panel on narrow terminals to reclaim space.
	showFilter := a.width >= styles.FilterShowWidth
	filterWidth := 0
	contentWidth := a.width - styles.PanelBorderOffset // single-panel default: outer = a.width
	if showFilter {
		filterWidth = int(float64(a.width) * styles.FilterPanelWidthRatio)
		if filterWidth < styles.FilterMinWidth {
			filterWidth = styles.FilterMinWidth
		}
		if filterWidth > styles.FilterMaxWidth {
			filterWidth = styles.FilterMaxWidth
		}
		// Two side-by-side panels: (filterWidth+2) + (contentWidth+2) = a.width
		contentWidth = a.width - filterWidth - styles.PanelBorderOffset*2
	}
	// Filter outer height = availableHeight  →  Height(availableHeight-2)
	filterContentHeight := availableHeight - styles.PanelBorderOffset

	// ── Bottom row (details / comments) ─────────────────────────────
	// Right side may have 1 or 2 stacked rows; each row adds +2 to total outer height.
	// 2-row case: (workItemsHeight+2) + (bottomRowHeight+2) = availableHeight
	//              workItemsHeight + bottomRowHeight = availableHeight - 4
	rightContentHeight := availableHeight - styles.PanelBorderOffset*2

	rawWorkItemsH := int(float64(rightContentHeight) * styles.WorkItemsHeightRatio)
	if rawWorkItemsH < styles.MinWorkItemsHeight {
		rawWorkItemsH = styles.MinWorkItemsHeight
	}
	rawBottomH := rightContentHeight - rawWorkItemsH

	// Only show the bottom row when it would have a usable inner area (≥7 lines)
	// and the content area is wide enough.
	showBottomRow := rawBottomH >= styles.MinBottomRowHeight && contentWidth >= styles.MinBottomRowWidth

	workItemsHeight := availableHeight - styles.PanelBorderOffset // single-row: outer = availableHeight
	bottomRowHeight := 0
	if showBottomRow {
		workItemsHeight = rawWorkItemsH
		bottomRowHeight = rawBottomH
	}

	// Show comments alongside details when there is enough horizontal space.
	// At contentWidth=55 each panel gets ~26 inner chars — tight but readable.
	showComments := showBottomRow && contentWidth >= styles.CommentsShowWidth
	detailsWidth := 0
	commentsWidth := 0
	if showComments {
		// Two panels: detailsWidth + commentsWidth = contentWidth - PanelBorderOffset
		detailsWidth = (contentWidth - styles.PanelBorderOffset) / 2
		commentsWidth = contentWidth - styles.PanelBorderOffset - detailsWidth
	} else if showBottomRow {
		// Details takes the full bottom row width
		detailsWidth = contentWidth - styles.PanelBorderOffset
	}

	// ── Header bar ───────────────────────────────────────────────────
	a.headerBar.SetWidth(a.width)
	a.headerBar.SetOrganization(a.client.Organization())
	a.headerBar.SetProject(a.client.Project())
	a.headerBar.SetLoading(a.loading)
	a.headerBar.SetNotification(&a.notification)
	headerBar := a.headerBar.View()

	// ── Assemble and return ───────────────────────────────────────────
	a.workItemsPanel.SetSize(contentWidth, workItemsHeight)
	rightSide := a.renderRightSide(showBottomRow, showComments, detailsWidth, commentsWidth, bottomRowHeight)

	var mainContent string
	if showFilter {
		a.filterPanel.SetSize(filterWidth, filterContentHeight)
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, a.filterPanel.View(), rightSide)
	} else {
		mainContent = rightSide
	}

	statusBar := a.renderStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, headerBar, mainContent, statusBar)
}

// renderMinimalView is shown when the terminal is too small to render the full layout.
// It shows one panel at a time; the user switches with Tab/Shift+Tab or 1-6.
func (a *App) renderMinimalView() string {
	availableHeight := a.height - styles.AppHeaderFooterSize // header(1) + statusBar(1)

	a.headerBar.SetWidth(a.width)
	a.headerBar.SetOrganization(a.client.Organization())
	a.headerBar.SetProject(a.client.Project())
	a.headerBar.SetLoading(a.loading)
	a.headerBar.SetNotification(&a.notification)
	headerBar := a.headerBar.View()

	// A single panel fills the available area.
	// Border adds +2 outer; pass content dimensions.
	panelWidth := a.width - styles.PanelBorderOffset
	panelHeight := availableHeight - styles.PanelBorderOffset

	var panelView string
	switch a.activePanel {
	case PanelFilterSprint, PanelFilterState, PanelFilterAssigned, PanelFilterArea:
		a.filterPanel.SetSize(panelWidth, panelHeight)
		panelView = a.filterPanel.View()
	case PanelWorkItems:
		a.workItemsPanel.SetSize(panelWidth, panelHeight)
		panelView = a.workItemsPanel.View()
	case PanelDetails:
		a.detailsPanel.SetSize(panelWidth, panelHeight)
		panelView = a.detailsPanel.View()
	case PanelComments:
		a.commentsPanel.SetSize(panelWidth, panelHeight)
		panelView = a.commentsPanel.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerBar, panelView, a.renderMinimalStatusBar())
}

// renderMinimalStatusBar shows the active panel name and keyboard navigation hint.
func (a *App) renderMinimalStatusBar() string {
	var panelName string
	switch a.activePanel {
	case PanelFilterSprint:
		panelName = "Sprint Filter"
	case PanelFilterState:
		panelName = "State Filter"
	case PanelFilterAssigned:
		panelName = "Assigned Filter"
	case PanelFilterArea:
		panelName = "Area Filter"
	case PanelWorkItems:
		panelName = "Work Items"
	case PanelDetails:
		panelName = "Details"
	case PanelComments:
		panelName = "Comments"
	}

	label := lipgloss.NewStyle().
		Foreground(styles.ColorText).
		Background(styles.ColorPrimary).
		Bold(true).
		Padding(0, 1).
		Render(panelName)

	hint := a.styles.TextMuted.Render("Tab/Shift+Tab · 1-7: switch panel")
	text := label + "  " + hint
	// StatusBar has Padding(0,1) → truncate to width-2 so it never wraps.
	text = truncate.StringWithTail(text, uint(a.width-styles.PanelBorderOffset), "…")
	return a.styles.StatusBar.Width(a.width).Render(text)
}

// renderRightSide builds the right-column view (work items + optional bottom row).
func (a *App) renderRightSide(showBottomRow, showComments bool, detailsWidth, commentsWidth, bottomRowHeight int) string {
	workItemsView := a.workItemsPanel.View()
	if !showBottomRow {
		return workItemsView
	}

	a.detailsPanel.SetSize(detailsWidth, bottomRowHeight)
	detailsView := a.detailsPanel.View()

	if showComments {
		a.commentsPanel.SetSize(commentsWidth, bottomRowHeight)
		bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, detailsView, a.commentsPanel.View())
		return lipgloss.JoinVertical(lipgloss.Left, workItemsView, bottomRow)
	}
	return lipgloss.JoinVertical(lipgloss.Left, workItemsView, detailsView)
}

func (a *App) renderStatusBar() string {
	parts := a.statusBarParts()
	text := strings.Join(parts, "  ")
	// Truncate to terminal width (ANSI-aware) so the bar is always exactly 1 line.
	// lipgloss Height(1) only sets a minimum; content still wraps beyond the width,
	// which pushes the header bar off the top of the alt-screen.
	// StatusBar has Padding(0,1) which adds 2 outer chars; truncate to width-2
	// so the rendered box is exactly a.width wide and never wraps.
	text = truncate.StringWithTail(text, uint(a.width-styles.PanelBorderOffset), "…")
	return a.styles.StatusBar.Width(a.width).Render(text)
}

func (a *App) statusBarParts() []string {
	var parts []string

	// Panel indicator
	panelName := "Filter"
	switch a.activePanel {
	case PanelWorkItems:
		panelName = "Work Items"
	case PanelDetails:
		panelName = "Details"
	case PanelComments:
		panelName = "Comments"
	}

	panelLabel := lipgloss.NewStyle().
		Foreground(styles.ColorText).
		Background(styles.ColorPrimary).
		Bold(true).
		Padding(0, 1).
		Render("Panel")
	parts = append(parts, panelLabel+": "+panelName)

	// Panel-specific actions/help
	switch a.activePanel {
	case PanelWorkItems:
		parts = append(parts, a.workItemsPanel.GetAvailableActions()...)
	case PanelDetails:
		parts = append(parts, a.detailsPanel.GetAvailableActions()...)
	case PanelComments:
		parts = append(parts, a.commentsPanel.GetAvailableActions()...)
	default:
		// Generic short help when not on work items/comments
		parts = append(parts, panels.ShortHelp(a.keys, a.styles))
	}

	return parts
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
	isFilterPanel := a.activePanel == PanelFilterSprint || a.activePanel == PanelFilterState ||
		a.activePanel == PanelFilterAssigned || a.activePanel == PanelFilterArea
	a.filterPanel.SetFocused(isFilterPanel)
	if isFilterPanel {
		a.filterPanel.SetActiveGroup(a.activeFilterGroup)
	}
	a.workItemsPanel.SetFocused(a.activePanel == PanelWorkItems)
	a.detailsPanel.SetFocused(a.activePanel == PanelDetails)
	a.commentsPanel.SetFocused(a.activePanel == PanelComments)
}

func (a *App) updateSizes() {
	a.helpPanel.SetSize(a.width, a.height)
	a.detailView.SetSize(a.width, a.height)
	a.notification.SetSize(a.width, a.height)
	a.headerBar.SetWidth(a.width)
	a.taskModal.SetSize(a.width, a.height)
	a.deleteModal.SetSize(a.width, a.height)
	a.pbiModal.SetSize(a.width, a.height)
	a.commentModal.SetSize(a.width, a.height)
	a.parentSelection.SetSize(a.width, a.height)
	a.bugParentModal.SetSize(a.width, a.height)
	a.commentModal.SetSize(a.width, a.height)
	a.errorModal.SetSize(a.width, a.height)
	a.infoModal.SetSize(a.width, a.height)
	a.quickSearchModal.SetSize(a.width, a.height)
	a.updateContentPanelSizes()
	a.updateFocus()
}

func (a *App) updateContentPanelSizes() {
	if a.width == 0 || a.height == 0 {
		return
	}

	availableHeight := a.height - styles.AppHeaderFooterSize

	if a.width < styles.MinTerminalWidth || a.height < styles.MinTerminalHeight {
		panelWidth := a.width - styles.PanelBorderOffset
		panelHeight := availableHeight - styles.PanelBorderOffset
		a.detailsPanel.SetSize(panelWidth, panelHeight)
		a.commentsPanel.SetSize(panelWidth, panelHeight)
		return
	}

	contentWidth := a.width - styles.PanelBorderOffset
	if a.width >= styles.FilterShowWidth {
		filterWidth := int(float64(a.width) * styles.FilterPanelWidthRatio)
		if filterWidth < styles.FilterMinWidth {
			filterWidth = styles.FilterMinWidth
		}
		if filterWidth > styles.FilterMaxWidth {
			filterWidth = styles.FilterMaxWidth
		}
		contentWidth = a.width - filterWidth - styles.PanelBorderOffset*2
	}

	rightContentHeight := availableHeight - styles.PanelBorderOffset*2
	rawWorkItemsH := int(float64(rightContentHeight) * styles.WorkItemsHeightRatio)
	if rawWorkItemsH < styles.MinWorkItemsHeight {
		rawWorkItemsH = styles.MinWorkItemsHeight
	}
	rawBottomH := rightContentHeight - rawWorkItemsH
	showBottomRow := rawBottomH >= styles.MinBottomRowHeight && contentWidth >= styles.MinBottomRowWidth

	if !showBottomRow {
		return
	}

	showComments := contentWidth >= styles.CommentsShowWidth
	if showComments {
		detailsWidth := (contentWidth - styles.PanelBorderOffset) / 2
		commentsWidth := contentWidth - styles.PanelBorderOffset - detailsWidth
		a.detailsPanel.SetSize(detailsWidth, rawBottomH)
		a.commentsPanel.SetSize(commentsWidth, rawBottomH)
	} else {
		detailsWidth := contentWidth - styles.PanelBorderOffset
		a.detailsPanel.SetSize(detailsWidth, rawBottomH)
	}
}

func (a *App) updateSelectedItem() tea.Cmd {
	item := a.workItemsPanel.SelectedItem()
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

func tickCmd() tea.Cmd {
	return spinner.Tick
}
