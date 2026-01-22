package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SortField represents the field to sort by
type SortField int

const (
	SortByID SortField = iota
	SortByState
	SortByType
)

// SortDirection represents the sort direction
type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

// Column definitions
type column struct {
	title    string
	width    int
	minWidth int
	flex     bool // If true, this column takes remaining space
}

// WorkItemsPanel is the work items list component
type WorkItemsPanel struct {
	items         []models.WorkItem
	flattenedView []workItemViewEntry // Flattened view with hierarchy info
	cursor        int
	styles        styles.Styles
	keys          keys.KeyMap
	width         int
	height        int
	focused       bool
	offset        int // For scrolling
	columns       []column
	sortField     SortField
	sortDir       SortDirection
}

// GetAvailableActions returns a list of available actions for the work items panel
func (w *WorkItemsPanel) GetAvailableActions() []string {
	if len(w.flattenedView) == 0 {
		return []string{}
	}

	keyStyle := w.styles.AccentBold
	descStyle := w.styles.TextMuted

	sel := w.keys.Select.Help()
	view := w.keys.View.Help()
	change := w.keys.ChangeState.Help()
	branch := w.keys.CreateBranch.Help()
	assign := w.keys.Assign.Help()
	task := w.keys.CreateTask.Help()
	parent := w.keys.CreateParent.Help()
	edit := w.keys.EditTask.Help()
	del := w.keys.DeleteTask.Help()
	comment := w.keys.AddComment.Help()

	actions := []string{
		keyStyle.Render(sel.Key) + descStyle.Render(" - "+sel.Desc),
		keyStyle.Render(view.Key) + descStyle.Render(" - "+view.Desc),
		keyStyle.Render(change.Key) + descStyle.Render(" - "+change.Desc),
		keyStyle.Render(assign.Key) + descStyle.Render(" - "+assign.Desc),
		keyStyle.Render(task.Key) + descStyle.Render(" - "+task.Desc),
		keyStyle.Render(parent.Key) + descStyle.Render(" - "+parent.Desc),
		keyStyle.Render(edit.Key) + descStyle.Render(" - "+edit.Desc),
		keyStyle.Render(del.Key) + descStyle.Render(" - "+del.Desc),
		keyStyle.Render(comment.Key) + descStyle.Render(" - "+comment.Desc),
		keyStyle.Render(branch.Key) + descStyle.Render(" - "+branch.Desc),
	}

	return actions
}

// workItemViewEntry represents a work item in the flattened display view
type workItemViewEntry struct {
	item   *models.WorkItem
	level  int  // Indentation level (0 for parents, 1 for children)
	isLast bool // Is this the last child of its parent?
}

// NewWorkItemsPanel creates a new work items panel
func NewWorkItemsPanel(styles styles.Styles, keys keys.KeyMap) WorkItemsPanel {
	return WorkItemsPanel{
		items:  []models.WorkItem{},
		styles: styles,
		keys:   keys,
		columns: []column{
			{title: "ID", width: 7, minWidth: 6},
			{title: "TYPE", width: 8, minWidth: 6},
			{title: "STATE", width: 12, minWidth: 8},
			{title: "ASSIGNED", width: 14, minWidth: 10},
			{title: "TITLE", flex: true, minWidth: 20},
		},
	}
}

// Init initializes the work items panel
func (w WorkItemsPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the work items panel
func (w WorkItemsPanel) Update(msg tea.Msg) (WorkItemsPanel, tea.Cmd) {
	if !w.focused {
		return w, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, w.keys.Up):
			w.moveUp()
		case key.Matches(msg, w.keys.Down):
			w.moveDown()
		case key.Matches(msg, w.keys.Top):
			w.moveToTop()
		case key.Matches(msg, w.keys.Bottom):
			w.moveToBottom()
		case key.Matches(msg, w.keys.Left):
			// Collapse current item if it's expanded
			if entry := w.currentEntry(); entry != nil && entry.item.IsExpanded {
				entry.item.IsExpanded = false
				w.rebuildFlattenedView()
			}
		case key.Matches(msg, w.keys.Right):
			// Expand current item if it has children
			if entry := w.currentEntry(); entry != nil && entry.item.HasChildren() && !entry.item.IsExpanded {
				entry.item.IsExpanded = true
				w.rebuildFlattenedView()
			}
		case key.Matches(msg, w.keys.Select):
			if w.SelectedItem() != nil {
				return w, func() tea.Msg { return OpenWorkItemMsg{Item: *w.SelectedItem()} }
			}
		case key.Matches(msg, w.keys.View):
			if w.SelectedItem() != nil {
				return w, func() tea.Msg { return ViewWorkItemMsg{Item: *w.SelectedItem()} }
			}
		case key.Matches(msg, w.keys.SortByID):
			w.toggleSort(SortByID)
		case key.Matches(msg, w.keys.SortByState):
			w.toggleSort(SortByState)
		case key.Matches(msg, w.keys.SortByType):
			w.toggleSort(SortByType)
		}
	}

	return w, nil
}

// View renders the work items panel
func (w WorkItemsPanel) View() string {
	var b strings.Builder

	// Panel title with number (consistent with filter panel style)
	numberStyle := w.styles.TextMuted
	if w.focused {
		numberStyle = w.styles.AccentBold
	}
	titleStyle := w.styles.FilterGroupTitle
	if w.focused {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
	}
	titlePrefix := numberStyle.Render("5. ")
	b.WriteString("  " + titlePrefix + titleStyle.Render("Work Items"))
	b.WriteString("\n\n")

	// Calculate column widths
	colWidths := w.calculateColumnWidths()

	// Header
	header := w.renderHeader(colWidths)
	b.WriteString(header)
	b.WriteString("\n")

	// Separator line
	separator := w.renderSeparator(colWidths)
	b.WriteString(separator)
	b.WriteString("\n")

	// Scroll up indicator
	if w.offset > 0 {
		scrollStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
		b.WriteString(scrollStyle.Render("  ▲ more"))
		b.WriteString("\n")
	}

	// Items
	if len(w.flattenedView) == 0 {
		emptyMsg := w.styles.Subtitle.Render("  No work items found")
		b.WriteString(emptyMsg)
	} else {
		visibleItems := w.visibleItemCount()
		for i := w.offset; i < len(w.flattenedView) && i < w.offset+visibleItems; i++ {
			entry := w.flattenedView[i]
			isCursor := i == w.cursor
			line := w.renderItemEntry(entry, isCursor, colWidths)
			b.WriteString(line)
			if i < len(w.flattenedView)-1 && i < w.offset+visibleItems-1 {
				b.WriteString("\n")
			}
		}

		// Scroll down indicator
		if w.offset+visibleItems < len(w.flattenedView) {
			b.WriteString("\n")
			scrollStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
			b.WriteString(scrollStyle.Render("  ▼ more"))
		}
	}

	// Apply panel styling
	content := b.String()
	if w.focused {
		return w.styles.PanelActive.
			Width(w.width).
			Height(w.height).
			Render(content)
	}
	return w.styles.PanelInactive.
		Width(w.width).
		Height(w.height).
		Render(content)
}

func (w *WorkItemsPanel) calculateColumnWidths() []int {
	availableWidth := w.width - 6 // Account for borders and padding

	// Calculate fixed columns total width
	fixedWidth := 0
	flexCount := 0
	for _, col := range w.columns {
		if col.flex {
			flexCount++
		} else {
			fixedWidth += col.width + 1 // +1 for separator
		}
	}

	// Calculate flex column width
	flexWidth := availableWidth - fixedWidth
	if flexWidth < 20 {
		flexWidth = 20
	}

	// Build widths array
	widths := make([]int, len(w.columns))
	for i, col := range w.columns {
		if col.flex {
			widths[i] = flexWidth / flexCount
		} else {
			widths[i] = col.width
		}
	}

	return widths
}

func (w *WorkItemsPanel) renderHeader(colWidths []int) string {
	headerStyle := w.styles.TextMuted.Bold(true)

	sortedStyle := w.styles.AccentBold

	var parts []string

	for i, col := range w.columns {
		width := colWidths[i]
		title := col.title

		// Add sort indicator
		isSorted := (col.title == "ID" && w.sortField == SortByID) ||
			(col.title == "STATE" && w.sortField == SortByState) ||
			(col.title == "TYPE" && w.sortField == SortByType)

		if isSorted {
			arrow := "▲"
			if w.sortDir == SortDesc {
				arrow = "▼"
			}
			title = title + arrow
		}

		if len(title) > width {
			title = title[:width]
		}

		if isSorted {
			parts = append(parts, sortedStyle.Width(width).Render(title))
		} else {
			parts = append(parts, headerStyle.Width(width).Render(title))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

func (w *WorkItemsPanel) renderSeparator(colWidths []int) string {
	sepStyle := lipgloss.NewStyle().Foreground(styles.ColorBorder)

	// Use content width (accounting for panel padding)
	contentWidth := w.width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	return sepStyle.Render(strings.Repeat("─", contentWidth))
}

func (w *WorkItemsPanel) renderItem(item models.WorkItem, isCursor bool, colWidths []int) string {
	// Cursor indicator
	cursor := "  "
	if isCursor {
		cursor = "▸ "
	}

	// Format values
	id := fmt.Sprintf("#%d", item.ID)
	typeStr := item.ShortType()
	stateStr := string(item.State)
	assigned := components.TruncateStr(item.AssignedTo, colWidths[3])
	if assigned == "" {
		assigned = "-"
	}
	title := components.TruncateStr(item.Title, colWidths[4])

	// For cursor row, use plain text with unified background
	if isCursor {
		rowStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorText).
			Background(styles.ColorPrimary). // Purple highlight
			Width(w.width - 4)

		// Build plain text cells (no individual colors)
		cells := []string{
			padRight(components.TruncateStr(id, colWidths[0]), colWidths[0]),
			padRight(components.TruncateStr(typeStr, colWidths[1]), colWidths[1]),
			padRight(components.TruncateStr(stateStr, colWidths[2]), colWidths[2]),
			padRight(assigned, colWidths[3]),
			padRight(title, colWidths[4]),
		}
		row := cursor + strings.Join(cells, "  ")
		return rowStyle.Render(row)
	}

	// Non-cursor rows with individual cell colors
	idStyle := w.styles.Accent
	typeStyle := w.styles.TypeBadge(string(item.Type))
	stateStyle := w.styles.StateBadge(string(item.State))
	assignedStyle := w.styles.TextMuted
	titleStyle := lipgloss.NewStyle().Foreground(styles.ColorText)

	// Build cells with proper width and colors
	cells := []string{
		idStyle.Width(colWidths[0]).Render(components.TruncateStr(id, colWidths[0])),
		typeStyle.Width(colWidths[1]).Render(components.TruncateStr(typeStr, colWidths[1])),
		stateStyle.Width(colWidths[2]).Render(components.TruncateStr(stateStr, colWidths[2])),
		assignedStyle.Width(colWidths[3]).Render(assigned),
		titleStyle.Width(colWidths[4]).Render(title),
	}

	row := cursor + strings.Join(cells, "  ")
	return row
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func (w *WorkItemsPanel) visibleItemCount() int {
	visible := w.height - 7 // title, blank line, header, separator, borders
	if visible < 1 {
		visible = 1
	}
	return visible
}

func (w *WorkItemsPanel) moveUp() {
	if w.cursor > 0 {
		w.cursor--
	}
}

func (w *WorkItemsPanel) moveDown() {
	if w.cursor < len(w.flattenedView)-1 {
		w.cursor++
	}
}

func (w *WorkItemsPanel) moveToTop() {
	w.cursor = 0
	w.offset = 0
}

func (w *WorkItemsPanel) moveToBottom() {
	if len(w.flattenedView) > 0 {
		w.cursor = len(w.flattenedView) - 1
	}
}

func (w *WorkItemsPanel) toggleSort(field SortField) {
	if w.sortField == field {
		// Toggle direction if same field
		if w.sortDir == SortAsc {
			w.sortDir = SortDesc
		} else {
			w.sortDir = SortAsc
		}
	} else {
		w.sortField = field
		w.sortDir = SortAsc
	}
	w.sortItems()
	w.rebuildFlattenedView()
}

func (w *WorkItemsPanel) sortItems() {
	if len(w.items) == 0 {
		return
	}

	// Only sort top-level items (items without a parent in the list)
	// Children relationships are already established by the API
	var topLevel []models.WorkItem

	for i := range w.items {
		item := &w.items[i]
		// Only include items that don't have a parent in the list
		if item.ParentID == 0 {
			topLevel = append(topLevel, *item)
		} else {
			// Check if parent exists in our list
			hasParent := false
			for j := range w.items {
				if w.items[j].ID == item.ParentID {
					hasParent = true
					break
				}
			}
			// If no parent in list, treat as top-level
			if !hasParent {
				topLevel = append(topLevel, *item)
			}
		}
	}

	// Sort top-level items according to current sort field
	sort.SliceStable(topLevel, func(i, j int) bool {
		return w.compareItems(&topLevel[i], &topLevel[j])
	})

	// Replace items with sorted top-level items
	// (Children are already attached by the API)
	w.items = topLevel
}

// compareItems compares two work items based on current sort field and direction
func (w *WorkItemsPanel) compareItems(a, b *models.WorkItem) bool {
	var less bool

	switch w.sortField {
	case SortByID:
		less = a.ID < b.ID
	case SortByState:
		stateA := string(a.State)
		stateB := string(b.State)
		if stateA != stateB {
			less = stateA < stateB
		} else {
			less = a.ID < b.ID
		}
	case SortByType:
		typeA := string(a.Type)
		typeB := string(b.Type)
		if typeA != typeB {
			less = typeA < typeB
		} else {
			less = a.ID < b.ID
		}
	default:
		less = a.ID < b.ID
	}

	if w.sortDir == SortDesc {
		return !less
	}
	return less
}

// SetSize sets the size of the work items panel
func (w *WorkItemsPanel) SetSize(width, height int) {
	w.width = width
	w.height = height

	// Adjust offset to keep cursor visible (now that we have correct height)
	visible := w.visibleItemCount()
	if w.cursor < w.offset {
		w.offset = w.cursor
	}
	if w.cursor >= w.offset+visible {
		w.offset = w.cursor - visible + 1
	}
	if w.offset < 0 {
		w.offset = 0
	}
}

// SetFocused sets whether the panel is focused
func (w *WorkItemsPanel) SetFocused(focused bool) {
	w.focused = focused
}

// SetItems sets the work items
func (w *WorkItemsPanel) SetItems(items []models.WorkItem) {
	// Remember currently selected item ID and parent ID
	var selectedID int
	var selectedParentID int
	if current := w.SelectedItem(); current != nil {
		selectedID = current.ID
		selectedParentID = current.ParentID
	}

	oldLen := len(w.items)
	w.items = items

	// Re-apply current sort
	w.sortItems()

	// Rebuild flattened view
	w.rebuildFlattenedView()

	// Only reset position if this is new data (not just a refresh)
	if oldLen == 0 && len(w.flattenedView) > 0 {
		w.cursor = 0
		w.offset = 0
	} else if selectedID > 0 {
		// Try to restore cursor to previously selected item
		found := false
		for i, entry := range w.flattenedView {
			if entry.item.ID == selectedID {
				w.cursor = i
				found = true
				break
			}
		}
		// If not found (item was deleted)
		if !found {
			if selectedParentID > 0 {
				// Was a child - try to find the parent
				for i, entry := range w.flattenedView {
					if entry.item.ID == selectedParentID {
						w.cursor = i
						found = true
						break
					}
				}
			} else {
				// Was a parent - go to first item
				if len(w.flattenedView) > 0 {
					w.cursor = 0
					found = true
				}
			}
			// Still not found - clamp to valid range intelligently
			if !found && w.cursor >= len(w.flattenedView) {
				// Keep cursor near where it was, but within bounds
				if len(w.flattenedView) > 0 {
					w.cursor = len(w.flattenedView) - 1
					// If there's space to stay in middle of viewport, do so
					if w.cursor > 5 {
						w.cursor = min(w.cursor, len(w.flattenedView)-1)
					}
				} else {
					w.cursor = 0
				}
			}
		}
	} else {
		// No previous selection, clamp to valid range
		if w.cursor >= len(w.flattenedView) && len(w.flattenedView) > 0 {
			w.cursor = len(w.flattenedView) - 1
		}
	}

	// Final safety check
	if w.cursor >= len(w.flattenedView) {
		w.cursor = max(0, len(w.flattenedView)-1)
	}
	if w.cursor < 0 {
		w.cursor = 0
	}

	// Adjust offset to keep cursor visible
	visible := w.visibleItemCount()
	if w.cursor < w.offset {
		w.offset = w.cursor
	}
	if w.cursor >= w.offset+visible {
		w.offset = w.cursor - visible + 1
	}
}

// SelectedItem returns the currently selected work item
func (w *WorkItemsPanel) SelectedItem() *models.WorkItem {
	if w.cursor >= 0 && w.cursor < len(w.flattenedView) {
		return w.flattenedView[w.cursor].item
	}
	return nil
}

// currentEntry returns the currently selected view entry
func (w *WorkItemsPanel) currentEntry() *workItemViewEntry {
	if w.cursor >= 0 && w.cursor < len(w.flattenedView) {
		return &w.flattenedView[w.cursor]
	}
	return nil
}

// rebuildFlattenedView rebuilds the flattened view from items
func (w *WorkItemsPanel) rebuildFlattenedView() {
	w.flattenedView = nil
	for i := range w.items {
		w.addToFlattenedView(&w.items[i], 0)
	}
	// Adjust cursor if needed
	if w.cursor >= len(w.flattenedView) {
		w.cursor = len(w.flattenedView) - 1
	}
	if w.cursor < 0 {
		w.cursor = 0
	}
}

// addToFlattenedView recursively adds items to the flattened view
func (w *WorkItemsPanel) addToFlattenedView(item *models.WorkItem, level int) {
	// Skip child items that are already shown under their parent
	if level == 0 && item.ParentID > 0 {
		// Check if this item's parent is in the list
		for i := range w.items {
			if w.items[i].ID == item.ParentID {
				return // Skip, will be shown under parent
			}
		}
	}

	w.flattenedView = append(w.flattenedView, workItemViewEntry{
		item:  item,
		level: level,
	})

	// Add children if expanded
	if item.IsExpanded && item.HasChildren() {
		for i, child := range item.Children {
			entry := workItemViewEntry{
				item:   child,
				level:  level + 1,
				isLast: i == len(item.Children)-1,
			}
			w.flattenedView = append(w.flattenedView, entry)
		}
	}
}

// renderItemEntry renders a work item entry with hierarchy visualization
func (w *WorkItemsPanel) renderItemEntry(entry workItemViewEntry, isCursor bool, colWidths []int) string {
	item := entry.item

	// Cursor indicator and indentation
	cursor := "  "
	indent := ""

	if entry.level > 0 {
		// Child item - add tree characters
		if entry.isLast {
			indent = "  └─ "
		} else {
			indent = "  ├─ "
		}
	} else if item.HasChildren() {
		// Parent item with children
		if item.IsExpanded {
			cursor = "▼ "
		} else {
			cursor = "▶ "
		}
	} else if isCursor && entry.level == 0 {
		cursor = "▸ "
	}

	prefix := cursor + indent

	// Format values
	id := fmt.Sprintf("#%d", item.ID)
	typeStr := item.ShortType()
	stateStr := string(item.State)
	assigned := components.TruncateStr(item.AssignedTo, colWidths[3])
	if assigned == "" {
		assigned = "-"
	}

	// Adjust title width for indentation
	titleWidth := colWidths[4] - len(indent)
	if titleWidth < 10 {
		titleWidth = 10
	}
	title := components.TruncateStr(item.Title, titleWidth)

	// For cursor row, use plain text with unified background
	if isCursor {
		rowStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorText).
			Background(styles.ColorPrimary). // Purple highlight
			Width(w.width - 4)

		// Build plain text cells (no individual colors)
		cells := []string{
			padRight(components.TruncateStr(id, colWidths[0]), colWidths[0]),
			padRight(components.TruncateStr(typeStr, colWidths[1]), colWidths[1]),
			padRight(components.TruncateStr(stateStr, colWidths[2]), colWidths[2]),
			padRight(assigned, colWidths[3]),
			padRight(title, titleWidth),
		}
		row := prefix + strings.Join(cells, "  ")
		return rowStyle.Render(row)
	}

	// Non-cursor rows with individual cell colors
	idStyle := w.styles.Accent
	typeStyle := w.styles.TypeBadge(string(item.Type))
	stateStyle := w.styles.StateBadge(string(item.State))
	assignedStyle := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)

	// Dim child items slightly
	titleColor := styles.ColorText
	if entry.level > 0 {
		titleColor = styles.ColorTextMuted
	}
	titleStyle := lipgloss.NewStyle().Foreground(titleColor)

	// Build cells with proper width and colors
	cells := []string{
		idStyle.Width(colWidths[0]).Render(components.TruncateStr(id, colWidths[0])),
		typeStyle.Width(colWidths[1]).Render(components.TruncateStr(typeStr, colWidths[1])),
		stateStyle.Width(colWidths[2]).Render(components.TruncateStr(stateStr, colWidths[2])),
		assignedStyle.Width(colWidths[3]).Render(assigned),
		titleStyle.Render(title),
	}

	return prefix + strings.Join(cells, "  ")
}

// OpenWorkItemMsg is sent when a work item should be opened in browser
type OpenWorkItemMsg struct {
	Item models.WorkItem
}

// ViewWorkItemMsg is sent when a work item should be viewed in detail
type ViewWorkItemMsg struct {
	Item models.WorkItem
}
