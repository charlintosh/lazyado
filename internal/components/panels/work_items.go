package panels

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	colID gridColumn = iota
	colTitle
	colState
	colAssigned
	colEffort
	colIteration
	colTags
)

type WorkItemsPanel struct {
	items         []models.WorkItem
	flattenedView []workItemViewEntry
	styles        styles.Styles
	keys          keys.KeyMap
	width         int
	height        int
	focused       bool
	cursor        int
	offset        int
	sortField     SortField
	sortDir       SortDirection
}

type workItemViewEntry struct {
	item   *models.WorkItem
	level  int
	isLast bool
}

func NewWorkItemsPanel(s styles.Styles, k keys.KeyMap) WorkItemsPanel {
	return WorkItemsPanel{
		items:  []models.WorkItem{},
		styles: s,
		keys:   k,
	}
}

func (w *WorkItemsPanel) GetAvailableActions() []string {
	if len(w.flattenedView) == 0 {
		return Hints(styles.ColorSurface, w.keys.CreateParent, w.keys.Refresh)
	}
	return Hints(styles.ColorSurface,
		w.keys.View, w.keys.ChangeState, w.keys.Assign, w.keys.CreateTask, w.keys.EditTask,
		w.keys.AddComment, w.keys.CreateBranch, w.keys.Select, w.keys.DeleteTask, w.keys.Info,
	)
}

func (w WorkItemsPanel) Init() tea.Cmd {
	return nil
}

func (w WorkItemsPanel) Update(msg tea.Msg) (WorkItemsPanel, tea.Cmd) {
	if !w.focused {
		return w, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return w.handleKeyMsg(msg)
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			w.moveCursor(-1)
		case tea.MouseWheelDown:
			w.moveCursor(1)
		}
	}
	return w, nil
}

func (w WorkItemsPanel) handleKeyMsg(msg tea.KeyPressMsg) (WorkItemsPanel, tea.Cmd) {
	switch {
	case key.Matches(msg, w.keys.Up):
		w.moveCursor(-1)
	case key.Matches(msg, w.keys.Down):
		w.moveCursor(1)
	case key.Matches(msg, w.keys.Top):
		w.setCursor(0)
	case key.Matches(msg, w.keys.Bottom):
		w.setCursor(len(w.flattenedView) - 1)
	case key.Matches(msg, w.keys.HalfPageUp):
		w.moveCursor(-max(w.visibleRows()/2, 1))
	case key.Matches(msg, w.keys.HalfPageDown):
		w.moveCursor(max(w.visibleRows()/2, 1))
	case key.Matches(msg, w.keys.Left):
		w.collapseCurrentItem()
	case key.Matches(msg, w.keys.Right):
		w.expandCurrentItem()
	case key.Matches(msg, w.keys.Select):
		if item := w.SelectedItem(); item != nil {
			return w, func() tea.Msg { return OpenWorkItemMsg{Item: *item} }
		}
	case key.Matches(msg, w.keys.View):
		if item := w.SelectedItem(); item != nil {
			return w, func() tea.Msg { return ViewWorkItemMsg{Item: *item} }
		}
	}
	return w, nil
}

func (w *WorkItemsPanel) moveCursor(delta int) {
	w.setCursor(w.cursor + delta)
}

func (w *WorkItemsPanel) setCursor(c int) {
	w.cursor = w.clampCursor(c)
	w.ensureVisible()
}

func (w *WorkItemsPanel) visibleRows() int {
	return max(w.height-styles.GridHeaderLines, 1)
}

func (w *WorkItemsPanel) ensureVisible() {
	rows := w.visibleRows()
	if w.cursor < w.offset {
		w.offset = w.cursor
	}
	if w.cursor >= w.offset+rows {
		w.offset = w.cursor - rows + 1
	}
	w.offset = max(min(w.offset, len(w.flattenedView)-rows), 0)
}

func (w *WorkItemsPanel) collapseCurrentItem() {
	entry := w.currentEntry()
	if entry == nil {
		return
	}
	if entry.level > 0 {
		for i := w.cursor - 1; i >= 0; i-- {
			if w.flattenedView[i].level < entry.level {
				w.setCursor(i)
				return
			}
		}
		return
	}
	if entry.item.IsExpanded {
		entry.item.IsExpanded = false
		w.rebuildFlattenedView()
		w.setCursor(w.cursor)
	}
}

func (w *WorkItemsPanel) expandCurrentItem() {
	if entry := w.currentEntry(); entry != nil && entry.item.HasChildren() && !entry.item.IsExpanded {
		entry.item.IsExpanded = true
		w.rebuildFlattenedView()
		w.setCursor(w.cursor)
	}
}

func (w WorkItemsPanel) Count() int {
	return len(w.flattenedView)
}

func (w WorkItemsPanel) Title() string {
	return fmt.Sprintf("Backlog items · %d", len(w.flattenedView))
}

func (w WorkItemsPanel) View() string {
	if w.width <= 0 || w.height <= 0 {
		return ""
	}
	cols := w.columns()
	var lines []string
	lines = append(lines, w.renderHeader(cols))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", w.width)))

	if len(w.flattenedView) == 0 {
		lines = append(lines, "", w.styles.TextMuted.Render("  No work items match the current filters."),
			w.styles.TextMuted.Render("  Press "+w.keys.CreateParent.Help().Key+" to create one, or change the filters with 1–4."))
		return FitBlock(strings.Join(lines, "\n"), w.width, w.height)
	}

	end := min(w.offset+w.visibleRows(), len(w.flattenedView))
	for i := w.offset; i < end; i++ {
		lines = append(lines, w.renderRow(w.flattenedView[i], i == w.cursor, cols))
	}
	return FitBlock(strings.Join(lines, "\n"), w.width, w.height)
}

type columnLayout struct {
	kind  gridColumn
	title string
	width int
}

func (w WorkItemsPanel) columns() []columnLayout {
	optional := []columnLayout{
		{colState, w.sortTitle("State", SortByState), styles.GridColState},
		{colAssigned, "Assigned To", styles.GridColAssigned},
		{colEffort, "Effort", styles.GridColEffort},
		{colIteration, "Iteration", styles.GridColIteration},
		{colTags, "Tags", styles.GridColTags},
	}
	fixed := 1 + styles.GridColID + styles.GridColGap + styles.GridColTitleMin
	used := fixed
	var chosen []columnLayout
	for _, c := range optional {
		if used+styles.GridColGap+c.width > w.width {
			break
		}
		used += styles.GridColGap + c.width
		chosen = append(chosen, c)
	}
	titleWidth := styles.GridColTitleMin + (w.width - used)
	cols := []columnLayout{
		{colID, w.sortTitle("ID", SortByID), styles.GridColID},
		{colTitle, "Title", max(titleWidth, 1)},
	}
	return append(cols, chosen...)
}

func (w WorkItemsPanel) sortTitle(title string, field SortField) string {
	if w.sortField != field {
		return title
	}
	if w.sortDir == SortDesc {
		return title + " ▼"
	}
	return title + " ▲"
}

func (w WorkItemsPanel) renderHeader(cols []columnLayout) string {
	head := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true)
	parts := []string{" "}
	for i, c := range cols {
		if i > 0 {
			parts = append(parts, strings.Repeat(" ", styles.GridColGap))
		}
		parts = append(parts, FitWidth(head.Render(c.title), c.width))
	}
	return strings.Join(parts, "")
}

func (w WorkItemsPanel) renderRow(entry workItemViewEntry, selected bool, cols []columnLayout) string {
	var bg color.Color
	cursor := " "
	if selected {
		bg = styles.ColorSurface
		if w.focused {
			bg = styles.ColorHighlight
		}
		cursor = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Background(bg).Render("▌")
	}
	paint := func(fg color.Color) lipgloss.Style {
		st := lipgloss.NewStyle().Foreground(fg)
		if bg != nil {
			st = st.Background(bg)
		}
		return st
	}
	cell := func(s string, width int) string {
		if bg == nil {
			return FitWidth(s, width)
		}
		return FillBlock(s, width, 1, bg)
	}

	item := entry.item
	parts := []string{cursor}
	for i, c := range cols {
		if i > 0 {
			parts = append(parts, paint(styles.ColorText).Render(strings.Repeat(" ", styles.GridColGap)))
		}
		var content string
		switch c.kind {
		case colID:
			content = paint(styles.ColorLight).Render(fmt.Sprintf("%d", item.ID))
		case colTitle:
			content = w.renderTitleCell(entry, paint)
		case colState:
			content = StateDot(string(item.State), bg)
		case colAssigned:
			if item.AssignedTo == "" {
				content = paint(styles.ColorTextMuted).Render("Unassigned")
			} else {
				content = paint(styles.ColorText).Render(item.AssignedTo)
			}
		case colEffort:
			if item.Effort > 0 {
				content = paint(styles.ColorText).Render(formatEffort(item.Effort))
			}
		case colIteration:
			content = paint(styles.ColorLight).Render(item.SprintName())
		case colTags:
			if len(item.Tags) > 0 {
				content = Chip(item.Tags[0])
			}
		}
		parts = append(parts, cell(content, c.width))
	}
	return strings.Join(parts, "")
}

func (w WorkItemsPanel) renderTitleCell(entry workItemViewEntry, paint func(color.Color) lipgloss.Style) string {
	item := entry.item
	caret := " "
	if entry.level == 0 && item.HasChildren() {
		caret = "▸"
		if item.IsExpanded {
			caret = "▾"
		}
	}
	indent := strings.Repeat(" ", entry.level*styles.GridTreeIndent)
	titleStyle := paint(styles.ColorText)
	if item.Type == models.WorkItemTypeEpic || item.Type == models.WorkItemTypeFeature {
		titleStyle = titleStyle.Bold(true)
	}
	glyph := paint(styles.TypeColor(string(item.Type))).Render("■")
	return paint(styles.ColorTextMuted).Render(indent+caret+" ") + glyph + paint(styles.ColorText).Render(" ") +
		titleStyle.Render(item.Title)
}

func formatEffort(e float64) string {
	if e == float64(int(e)) {
		return fmt.Sprintf("%d", int(e))
	}
	return fmt.Sprintf("%.1f", e)
}

func (w *WorkItemsPanel) SetSize(width, height int) {
	w.width = width
	w.height = height
	w.ensureVisible()
}

func (w *WorkItemsPanel) SetFocused(focused bool) {
	w.focused = focused
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

	cursor := w.cursor
	switch {
	case oldLen == 0 && len(w.flattenedView) > 0:
		cursor = 0
	case selectedID > 0:
		cursor = w.findByIDOrFallback(selectedID, selectedParentID)
	}
	w.setCursor(cursor)
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
	return w.cursor
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
	if w.cursor >= 0 && w.cursor < len(w.flattenedView) {
		return w.flattenedView[w.cursor].item
	}
	return nil
}

func (w *WorkItemsPanel) currentEntry() *workItemViewEntry {
	if w.cursor >= 0 && w.cursor < len(w.flattenedView) {
		return &w.flattenedView[w.cursor]
	}
	return nil
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
	w.setCursor(w.cursor)
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
		for i, child := range item.Children {
			w.flattenedView = append(w.flattenedView, workItemViewEntry{
				item:   child,
				level:  1,
				isLast: i == len(item.Children)-1,
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
