package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SortField int

const (
	SortByID SortField = iota
	SortByState
	SortByType
)

type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

const (
	colTreeWidth     = 3
	colIDWidth       = 7
	colTypeWidth     = 8
	colStateWidth    = 12
	colAssignedWidth = 14
)

type WorkItemsPanel struct {
	items         []models.WorkItem
	flattenedView []workItemViewEntry
	table         table.Model
	styles        styles.Styles
	keys          keys.KeyMap
	width         int
	height        int
	focused       bool
	sortField     SortField
	sortDir       SortDirection
}

type workItemViewEntry struct {
	item   *models.WorkItem
	level  int
	isLast bool
}

func NewWorkItemsPanel(s styles.Styles, k keys.KeyMap) WorkItemsPanel {
	tableStyles := table.DefaultStyles()
	tableStyles.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorTextMuted).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(0, 1)
	tableStyles.Cell = lipgloss.NewStyle().
		Foreground(styles.ColorText).
		Padding(0, 1)
	tableStyles.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorText).
		Background(styles.ColorPrimary).
		Padding(0, 1)

	km := table.DefaultKeyMap()
	km.PageUp = key.NewBinding(key.WithKeys("pgup"))
	km.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"))

	cols := []table.Column{
		{Title: "", Width: colTreeWidth},
		{Title: "ID▲", Width: colIDWidth},
		{Title: "TYPE", Width: colTypeWidth},
		{Title: "STATE", Width: colStateWidth},
		{Title: "ASSIGNED", Width: colAssignedWidth},
		{Title: "TITLE", Width: 30},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
		table.WithStyles(tableStyles),
		table.WithKeyMap(km),
		table.WithHeight(10),
	)

	return WorkItemsPanel{
		items:  []models.WorkItem{},
		table:  t,
		styles: s,
		keys:   k,
	}
}

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
	info := w.keys.Info.Help()

	return []string{
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
		keyStyle.Render(info.Key) + descStyle.Render(" - "+info.Desc),
	}
}

func (w WorkItemsPanel) Init() tea.Cmd {
	return nil
}

func (w WorkItemsPanel) Update(msg tea.Msg) (WorkItemsPanel, tea.Cmd) {
	if !w.focused {
		return w, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		if updated, cmd, handled := w.handleKeyMsg(msg); handled {
			return updated, cmd
		}
	}

	var cmd tea.Cmd
	w.table, cmd = w.table.Update(msg)

	return w, cmd
}

func (w WorkItemsPanel) handleKeyMsg(msg tea.KeyMsg) (WorkItemsPanel, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, w.keys.Left):
		w.collapseCurrentItem()
		return w, nil, true
	case key.Matches(msg, w.keys.Right):
		w.expandCurrentItem()
		return w, nil, true
	case key.Matches(msg, w.keys.Select):
		if item := w.SelectedItem(); item != nil {
			return w, func() tea.Msg { return OpenWorkItemMsg{Item: *item} }, true
		}
		return w, nil, true
	case key.Matches(msg, w.keys.View):
		if item := w.SelectedItem(); item != nil {
			return w, func() tea.Msg { return ViewWorkItemMsg{Item: *item} }, true
		}
		return w, nil, true
	case key.Matches(msg, w.keys.SortByID):
		w.toggleSort(SortByID)
		return w, nil, true
	case key.Matches(msg, w.keys.SortByState):
		w.toggleSort(SortByState)
		return w, nil, true
	case key.Matches(msg, w.keys.SortByType):
		w.toggleSort(SortByType)
		return w, nil, true
	}
	return w, nil, false
}

func (w *WorkItemsPanel) collapseCurrentItem() {
	if entry := w.currentEntry(); entry != nil && entry.item.IsExpanded {
		entry.item.IsExpanded = false
		w.rebuildFlattenedView()
		w.syncTableRows()
	}
}

func (w *WorkItemsPanel) expandCurrentItem() {
	if entry := w.currentEntry(); entry != nil && entry.item.HasChildren() && !entry.item.IsExpanded {
		entry.item.IsExpanded = true
		w.rebuildFlattenedView()
		w.syncTableRows()
	}
}

func (w WorkItemsPanel) View() string {
	var b strings.Builder

	numberStyle := w.styles.TextMuted
	if w.focused {
		numberStyle = w.styles.AccentBold
	}
	titleStyle := w.styles.FilterGroupTitle
	if w.focused {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
	}
	b.WriteString("  " + numberStyle.Render("5. ") + titleStyle.Render("Work Items"))
	b.WriteString("\n\n")

	if len(w.flattenedView) == 0 {
		emptyMsg := w.styles.Subtitle.Render("  No work items found")
		b.WriteString(emptyMsg)
	} else {
		b.WriteString(w.table.View())
	}

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

func (w *WorkItemsPanel) SetSize(width, height int) {
	w.width = width
	w.height = height

	tableHeight := height - 2
	if tableHeight < 3 {
		tableHeight = 3
	}

	innerWidth := width - styles.PanelPaddedOffset
	w.table.SetWidth(innerWidth)
	w.table.SetHeight(tableHeight)
	w.table.SetColumns(w.buildColumns())
	w.syncTableRows()
}

func (w *WorkItemsPanel) SetFocused(focused bool) {
	w.focused = focused
	if focused {
		w.table.Focus()
	} else {
		w.table.Blur()
	}
}

func (w *WorkItemsPanel) SetItems(items []models.WorkItem) {
	var selectedID int
	var selectedParentID int
	if current := w.SelectedItem(); current != nil {
		selectedID = current.ID
		selectedParentID = current.ParentID
	}

	oldLen := len(w.items)
	w.items = items
	w.sortItems()
	w.rebuildFlattenedView()

	desiredCursor := w.findCursorAfterRefresh(oldLen, selectedID, selectedParentID)
	desiredCursor = w.clampCursor(desiredCursor)

	w.syncTableRows()
	w.table.SetCursor(desiredCursor)
}

func (w *WorkItemsPanel) findCursorAfterRefresh(oldLen, selectedID, selectedParentID int) int {
	if oldLen == 0 && len(w.flattenedView) > 0 {
		return 0
	}

	if selectedID > 0 {
		return w.findByIDOrFallback(selectedID, selectedParentID)
	}

	return w.table.Cursor()
}

func (w *WorkItemsPanel) findByIDOrFallback(selectedID, selectedParentID int) int {
	for i, entry := range w.flattenedView {
		if entry.item.ID == selectedID {
			return i
		}
	}

	if selectedParentID > 0 {
		for i, entry := range w.flattenedView {
			if entry.item.ID == selectedParentID {
				return i
			}
		}
	}

	return w.table.Cursor()
}

func (w *WorkItemsPanel) clampCursor(cursor int) int {
	if cursor < 0 {
		return 0
	}
	if len(w.flattenedView) > 0 && cursor >= len(w.flattenedView) {
		return len(w.flattenedView) - 1
	}
	return cursor
}

func (w *WorkItemsPanel) SelectedItem() *models.WorkItem {
	cursor := w.table.Cursor()
	if cursor >= 0 && cursor < len(w.flattenedView) {
		return w.flattenedView[cursor].item
	}
	return nil
}

func (w *WorkItemsPanel) currentEntry() *workItemViewEntry {
	cursor := w.table.Cursor()
	if cursor >= 0 && cursor < len(w.flattenedView) {
		return &w.flattenedView[cursor]
	}
	return nil
}

func (w *WorkItemsPanel) buildColumns() []table.Column {
	idTitle := "ID"
	typeTitle := "TYPE"
	stateTitle := "STATE"

	arrow := "▲"
	if w.sortDir == SortDesc {
		arrow = "▼"
	}

	switch w.sortField {
	case SortByID:
		idTitle += arrow
	case SortByType:
		typeTitle += arrow
	case SortByState:
		stateTitle += arrow
	}

	titleWidth := w.titleColumnWidth()

	return []table.Column{
		{Title: "", Width: colTreeWidth},
		{Title: idTitle, Width: colIDWidth},
		{Title: typeTitle, Width: colTypeWidth},
		{Title: stateTitle, Width: colStateWidth},
		{Title: "ASSIGNED", Width: colAssignedWidth},
		{Title: "TITLE", Width: titleWidth},
	}
}

func (w *WorkItemsPanel) titleColumnWidth() int {
	cellPadding := 2
	fixedCols := colTreeWidth + colIDWidth + colTypeWidth + colStateWidth + colAssignedWidth
	totalPadding := 6 * cellPadding
	innerWidth := w.width - styles.PanelPaddedOffset
	titleWidth := innerWidth - fixedCols - totalPadding
	if titleWidth < styles.WorkItemsMinFlexWidth {
		titleWidth = styles.WorkItemsMinFlexWidth
	}
	return titleWidth
}

func (w *WorkItemsPanel) syncTableRows() {
	rows := make([]table.Row, len(w.flattenedView))
	for i, entry := range w.flattenedView {
		rows[i] = w.buildRow(entry)
	}
	w.table.SetRows(rows)
}

func (w *WorkItemsPanel) buildRow(entry workItemViewEntry) table.Row {
	item := entry.item

	indicator := w.treeIndicator(entry)
	id := fmt.Sprintf("#%d", item.ID)
	typeStr := item.ShortType()
	stateStr := string(item.State)
	assigned := item.AssignedTo
	if assigned == "" {
		assigned = "-"
	}
	title := item.Title

	return table.Row{indicator, id, typeStr, stateStr, assigned, title}
}

func (w *WorkItemsPanel) treeIndicator(entry workItemViewEntry) string {
	if entry.level > 0 {
		if entry.isLast {
			return " └─"
		}
		return " ├─"
	}
	if entry.item.HasChildren() {
		if entry.item.IsExpanded {
			return "▼"
		}
		return "▶"
	}
	return ""
}

func (w *WorkItemsPanel) toggleSort(field SortField) {
	if w.sortField == field {
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
	w.table.SetColumns(w.buildColumns())
	w.syncTableRows()
}

func (w *WorkItemsPanel) sortItems() {
	if len(w.items) == 0 {
		return
	}

	var topLevel []models.WorkItem

	for i := range w.items {
		item := &w.items[i]
		if item.ParentID == 0 {
			topLevel = append(topLevel, *item)
		} else {
			hasParent := false
			for j := range w.items {
				if w.items[j].ID == item.ParentID {
					hasParent = true
					break
				}
			}
			if !hasParent {
				topLevel = append(topLevel, *item)
			}
		}
	}

	sort.SliceStable(topLevel, func(i, j int) bool {
		return w.compareItems(&topLevel[i], &topLevel[j])
	})

	w.items = topLevel
}

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

func (w *WorkItemsPanel) rebuildFlattenedView() {
	w.flattenedView = nil
	for i := range w.items {
		w.addToFlattenedView(&w.items[i], 0)
	}
}

func (w *WorkItemsPanel) addToFlattenedView(item *models.WorkItem, level int) {
	if level == 0 && item.ParentID > 0 {
		for i := range w.items {
			if w.items[i].ID == item.ParentID {
				return
			}
		}
	}

	w.flattenedView = append(w.flattenedView, workItemViewEntry{
		item:  item,
		level: level,
	})

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

type OpenWorkItemMsg struct {
	Item models.WorkItem
}

type ViewWorkItemMsg struct {
	Item models.WorkItem
}
