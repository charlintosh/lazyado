package panels

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
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

type gridColumn int

const (
	colTree gridColumn = iota
	colID
	colType
	colTitle
	colState
	colAssigned
	colEffort
	colIteration
	colTags
)

type columnSpec struct {
	kind  gridColumn
	title string
	width int
}

type WorkItemsPanel struct {
	items         []models.WorkItem
	flattenedView []workItemViewEntry
	table         table.Model
	columns       []columnSpec
	styles        *styles.Styles
	keys          keys.KeyMap
	width         int
	height        int
	focused       bool
	sortField     SortField
	sortDir       SortDirection
}

type workItemViewEntry struct {
	item     *models.WorkItem
	level    int
	siblings tree.Children
	index    int
}

func NewWorkItemsPanel(s *styles.Styles, k keys.KeyMap) WorkItemsPanel {
	t := table.New(
		table.WithStyles(tableStyles()),
		table.WithKeyMap(tableKeyMap(k)),
		table.WithFocused(false),
	)
	return WorkItemsPanel{
		items:  []models.WorkItem{},
		table:  t,
		styles: s,
		keys:   k,
	}
}

func (w *WorkItemsPanel) KeyBindings() []key.Binding {
	if len(w.flattenedView) == 0 {
		return []key.Binding{w.keys.CreateParent, w.keys.Refresh}
	}
	return []key.Binding{
		w.keys.View, w.keys.ChangeState, w.keys.Assign, w.keys.CreateTask, w.keys.EditTask,
		w.keys.AddComment, w.keys.Select, w.keys.DeleteTask, w.keys.Info,
	}
}

func (w WorkItemsPanel) Init() tea.Cmd {
	return nil
}

func (w WorkItemsPanel) Update(msg tea.Msg) (WorkItemsPanel, tea.Cmd) {
	if !w.focused {
		return w, nil
	}
	before := w.table.Cursor()
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if handled, c := w.handleKeyMsg(msg); handled {
			return w, c
		}
		w.table, cmd = w.table.Update(msg)
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			w.table.MoveUp(1)
		case tea.MouseWheelDown:
			w.table.MoveDown(1)
		}
	}
	if w.table.Cursor() != before {
		w.syncRows()
	}
	return w, cmd
}

func (w *WorkItemsPanel) handleKeyMsg(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, w.keys.Left):
		w.collapseCurrentItem()
		return true, nil
	case key.Matches(msg, w.keys.Right):
		w.expandCurrentItem()
		return true, nil
	case key.Matches(msg, w.keys.Select):
		if item := w.SelectedItem(); item != nil {
			opened := *item
			return true, func() tea.Msg { return OpenWorkItemMsg{Item: opened} }
		}
		return true, nil
	case key.Matches(msg, w.keys.View):
		if item := w.SelectedItem(); item != nil {
			viewed := *item
			return true, func() tea.Msg { return ViewWorkItemMsg{Item: viewed} }
		}
		return true, nil
	}
	return false, nil
}

func (w *WorkItemsPanel) collapseCurrentItem() {
	entry := w.currentEntry()
	if entry == nil {
		return
	}
	if entry.level > 0 {
		for i := w.table.Cursor() - 1; i >= 0; i-- {
			if w.flattenedView[i].level < entry.level {
				w.table.SetCursor(i)
				w.syncRows()
				return
			}
		}
		return
	}
	if entry.item.IsExpanded {
		entry.item.IsExpanded = false
		w.rebuildFlattenedView()
		w.syncRows()
	}
}

func (w *WorkItemsPanel) expandCurrentItem() {
	if entry := w.currentEntry(); entry != nil && entry.item.HasChildren() && !entry.item.IsExpanded {
		entry.item.IsExpanded = true
		w.rebuildFlattenedView()
		w.syncRows()
	}
}

func (w WorkItemsPanel) Count() int {
	return len(w.flattenedView)
}

func (w WorkItemsPanel) Title() string {
	return fmt.Sprintf("Backlog items  %d", len(w.flattenedView))
}

func (w WorkItemsPanel) View() string {
	if w.width <= 0 || w.height <= 0 {
		return ""
	}
	if len(w.flattenedView) == 0 {
		msg := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render("No work items match the current filters"),
			Muted("Press "+w.keys.CreateParent.Help().Key+" to create one, or change the filters with 1-4."),
		)
		return Block(lipgloss.NewStyle().Padding(1, 1).Render(msg), w.width, w.height)
	}
	return Block(w.table.View(), w.width, w.height)
}

func (w WorkItemsPanel) layoutColumns() []columnSpec {
	optional := []columnSpec{
		{colState, w.sortTitle("State", SortByState), styles.GridColState},
		{colType, w.sortTitle("Type", SortByType), styles.GridColType},
		{colAssigned, "Assigned To", styles.GridColAssigned},
		{colEffort, "Effort", styles.GridColEffort},
		{colIteration, "Iteration", styles.GridColIteration},
		{colTags, "Tags", styles.GridColTags},
	}
	cell := styles.GridCellPadding
	titleMin := max(min(int(float64(w.width)*styles.GridTitleRatio), styles.GridColTitleMin), styles.GridColTitleFloor)
	used := (styles.GridColTree + cell) + (styles.GridColID + cell) + (titleMin + cell)
	chosen := map[gridColumn]columnSpec{}
	for _, c := range optional {
		if used+c.width+cell > w.width {
			continue
		}
		used += c.width + cell
		chosen[c.kind] = c
	}
	titleWidth := titleMin + max(w.width-used, 0)
	cols := []columnSpec{
		{colTree, "", styles.GridColTree},
		{colID, w.sortTitle("ID", SortByID), styles.GridColID},
	}
	if c, ok := chosen[colType]; ok {
		cols = append(cols, c)
	}
	cols = append(cols, columnSpec{colTitle, "Title", titleWidth})
	for _, k := range []gridColumn{colState, colAssigned, colEffort, colIteration, colTags} {
		if c, ok := chosen[k]; ok {
			cols = append(cols, c)
		}
	}
	return cols
}

func (w WorkItemsPanel) sortTitle(title string, field SortField) string {
	if w.sortField != field {
		return title
	}
	if w.sortDir == SortDesc {
		return title + " desc"
	}
	return title
}

func (w *WorkItemsPanel) syncRows() {
	cursor := w.table.Cursor()
	rows := make([]table.Row, len(w.flattenedView))
	for i, entry := range w.flattenedView {
		rows[i] = w.buildRow(entry, i == cursor)
	}
	w.table.SetRows(rows)
}

func (w *WorkItemsPanel) buildRow(entry workItemViewEntry, selected bool) table.Row {
	item := entry.item
	paint := func(st lipgloss.Style, s string) string {
		if selected {
			return s
		}
		return st.Render(s)
	}
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	row := make(table.Row, 0, len(w.columns))
	for _, c := range w.columns {
		switch c.kind {
		case colTree:
			marker := ""
			if entry.level == 0 && item.HasChildren() {
				marker = "+"
				if item.IsExpanded {
					marker = "-"
				}
			}
			row = append(row, paint(muted, marker))
		case colID:
			row = append(row, paint(lipgloss.NewStyle().Foreground(styles.ColorLight), fmt.Sprintf("%d", item.ID)))
		case colType:
			if selected {
				row = append(row, item.ShortType())
			} else {
				row = append(row, TypeText(item))
			}
		case colTitle:
			prefix := ""
			if entry.level > 0 {
				prefix = tree.RoundedEnumerator(entry.siblings, entry.index) + " "
			}
			titleStyle := lipgloss.NewStyle().Foreground(styles.ColorText)
			if item.Type == models.WorkItemTypeEpic || item.Type == models.WorkItemTypeFeature {
				titleStyle = titleStyle.Bold(true)
			}
			row = append(row, paint(muted, prefix)+paint(titleStyle, item.Title))
		case colState:
			if selected {
				row = append(row, string(item.State))
			} else {
				row = append(row, StateText(string(item.State)))
			}
		case colAssigned:
			if item.AssignedTo == "" {
				row = append(row, paint(muted, "Unassigned"))
			} else {
				row = append(row, item.AssignedTo)
			}
		case colEffort:
			effort := ""
			if item.Effort > 0 {
				effort = formatEffort(item.Effort)
			}
			row = append(row, effort)
		case colIteration:
			row = append(row, paint(lipgloss.NewStyle().Foreground(styles.ColorLight), item.SprintName()))
		case colTags:
			row = append(row, paint(lipgloss.NewStyle().Foreground(styles.ColorLight), strings.Join(item.Tags, ", ")))
		}
	}
	return row
}

func formatEffort(e float64) string {
	if e == float64(int(e)) {
		return fmt.Sprintf("%d", int(e))
	}
	return fmt.Sprintf("%.1f", e)
}

func (w *WorkItemsPanel) SetSize(width, height int) {
	if w.width == width && w.height == height {
		return
	}
	w.width = width
	w.height = height
	w.applyColumns()
}

func (w *WorkItemsPanel) applyColumns() {
	cursor := w.table.Cursor()
	w.columns = w.layoutColumns()
	cols := make([]table.Column, len(w.columns))
	for i, c := range w.columns {
		cols[i] = table.Column{Title: c.title, Width: c.width}
	}
	w.table.SetRows(nil)
	w.table.SetColumns(cols)
	w.table.SetWidth(w.width)
	w.table.SetHeight(w.height)
	w.syncRows()
	w.table.SetCursor(w.clampCursor(cursor))
	w.syncRows()
}

func (w *WorkItemsPanel) SetFocused(focused bool) {
	if w.focused == focused {
		return
	}
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
	expanded := map[int]bool{}
	for _, e := range w.flattenedView {
		if e.item.IsExpanded {
			expanded[e.item.ID] = true
		}
	}

	oldLen := len(w.items)
	w.items = items
	w.sortItems()
	for i := range w.items {
		if expanded[w.items[i].ID] {
			w.items[i].IsExpanded = true
		}
	}
	w.rebuildFlattenedView()

	cursor := w.table.Cursor()
	switch {
	case oldLen == 0 && len(w.flattenedView) > 0:
		cursor = 0
	case selectedID > 0:
		cursor = w.findByIDOrFallback(selectedID, selectedParentID)
	}
	w.syncRows()
	w.table.SetCursor(w.clampCursor(cursor))
	w.syncRows()
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
	if len(w.flattenedView) == 0 || cursor < 0 {
		return 0
	}
	if cursor >= len(w.flattenedView) {
		return len(w.flattenedView) - 1
	}
	return cursor
}

func (w *WorkItemsPanel) SelectedItem() *models.WorkItem {
	if c := w.table.Cursor(); c >= 0 && c < len(w.flattenedView) {
		return w.flattenedView[c].item
	}
	return nil
}

func (w *WorkItemsPanel) currentEntry() *workItemViewEntry {
	if c := w.table.Cursor(); c >= 0 && c < len(w.flattenedView) {
		return &w.flattenedView[c]
	}
	return nil
}

func (w *WorkItemsPanel) sortItems() {
	if len(w.items) == 0 {
		return
	}
	ids := make(map[int]bool, len(w.items))
	for i := range w.items {
		ids[w.items[i].ID] = true
	}
	var topLevel []models.WorkItem
	for i := range w.items {
		item := &w.items[i]
		if item.ParentID == 0 || !ids[item.ParentID] {
			topLevel = append(topLevel, *item)
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
	case SortByState:
		if a.State != b.State {
			less = a.State < b.State
		} else {
			less = a.ID < b.ID
		}
	case SortByType:
		if a.Type != b.Type {
			less = a.Type < b.Type
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
		w.addToFlattenedView(&w.items[i])
	}
}

func (w *WorkItemsPanel) addToFlattenedView(item *models.WorkItem) {
	w.flattenedView = append(w.flattenedView, workItemViewEntry{item: item})
	if item.IsExpanded && item.HasChildren() {
		siblings := tree.NewStringData(make([]string, len(item.Children))...)
		for i, child := range item.Children {
			w.flattenedView = append(w.flattenedView, workItemViewEntry{
				item:     child,
				level:    1,
				siblings: siblings,
				index:    i,
			})
		}
	}
}

type OpenWorkItemMsg struct {
	Item models.WorkItem
}

type ViewWorkItemMsg struct {
	Item models.WorkItem
}
