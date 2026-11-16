package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	ltable "charm.land/lipgloss/v2/table"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

var defaultTaskStates = []string{"To Do", "In Progress", "Done"}

const (
	dropMarker  = "▸ "
	dropPrompt  = "drop on a state column"
	gripGlyph   = "⠿"
	countArrow  = " → "
	ghostNote   = "moving…"
	phantomNote = "↳ drop to set "
	savingNote  = "◌ Saving to "
	savedNote   = "✓ Moved to "
	failedNote  = "✕ Not saved · "
)

var (
	ghostBorder   = lipgloss.Border{Left: "┆"}
	phantomBorder = lipgloss.Border{Left: "┇"}
)

type cardLook int

const (
	lookNormal cardLook = iota
	lookHover
	lookGhost
	lookPhantom
	lookSaving
	lookSaved
	lookFailed
)

type taskLane struct {
	parent *models.WorkItem
	cols   [][]*models.WorkItem
}

func (l taskLane) cells() []*models.WorkItem {
	out := []*models.WorkItem{l.parent}
	for _, col := range l.cols {
		out = append(out, col...)
	}
	return out
}

func (l taskLane) size() int {
	n := 1
	for _, col := range l.cols {
		n += len(col)
	}
	return n
}

func (l taskLane) cell(i int) *models.WorkItem {
	if i == 0 {
		return l.parent
	}
	i--
	for _, col := range l.cols {
		if i < len(col) {
			return col[i]
		}
		i -= len(col)
	}
	return nil
}

func (l taskLane) columnOf(cell int) int {
	if cell <= 0 {
		return -1
	}
	n := 1
	for ci, col := range l.cols {
		if cell < n+len(col) {
			return ci
		}
		n += len(col)
	}
	return -1
}

func (l taskLane) firstCellIn(col int) int {
	if col < 0 || col >= len(l.cols) {
		return 0
	}
	n := 1
	for ci, c := range l.cols {
		if ci == col {
			if len(c) == 0 {
				return 0
			}
			return n
		}
		n += len(c)
	}
	return 0
}

type TaskboardPanel struct {
	items        []models.WorkItem
	statesByType map[string][]models.WorkItemStateInfo
	lanes        []taskLane
	states       []string
	styles       *styles.Styles
	keys         keys.KeyMap
	viewport     viewport.Model
	lane         int
	cell         int
	width        int
	height       int
	focused      bool
	originX      int
	originY      int
	drag         taskDrag
	hover        int
	moves        map[int]taskMove
	flashSeq     int
	collapsed    map[int]bool
}

func NewTaskboardPanel(st *styles.Styles, k keys.KeyMap) TaskboardPanel {
	vp := viewport.New()
	vp.KeyMap = viewport.KeyMap{}
	return TaskboardPanel{styles: st, keys: k, viewport: vp}
}

func (t *TaskboardPanel) SetItems(items []models.WorkItem, statesByType map[string][]models.WorkItemStateInfo) {
	t.items = items
	t.statesByType = statesByType
	t.settleMoves()
	t.rebuild()
}

func (t TaskboardPanel) displayState(task *models.WorkItem) string {
	if m, ok := t.moves[task.ID]; ok && m.to != "" {
		return m.to
	}
	return string(task.State)
}

func (t *TaskboardPanel) rebuild() {
	selectedID := 0
	if sel := t.SelectedItem(); sel != nil {
		selectedID = sel.ID
	}
	parents := backlogItems(t.items)
	var present []string
	seen := map[string]bool{}
	for _, p := range parents {
		for _, c := range p.Children {
			if state := t.displayState(c); c.Type == models.WorkItemTypeTask && !seen[state] {
				seen[state] = true
				present = append(present, state)
			}
		}
	}
	states := models.BoardColumns(t.statesByType, models.WorkItemTypeTask, present)
	if len(states) == 0 {
		states = defaultTaskStates
	}
	index := map[string]int{}
	for i, s := range states {
		index[s] = i
	}
	lanes := make([]taskLane, 0, len(parents))
	for _, p := range parents {
		lane := taskLane{parent: p, cols: make([][]*models.WorkItem, len(states))}
		for _, c := range p.Children {
			if i, ok := index[t.displayState(c)]; ok && c.Type == models.WorkItemTypeTask {
				lane.cols[i] = append(lane.cols[i], c)
			}
		}
		lanes = append(lanes, lane)
	}
	t.states = states
	t.lanes = lanes
	t.seedCollapsed()
	t.lane = max(min(t.lane, len(lanes)-1), 0)
	t.cell = 0
	if selectedID > 0 {
		t.selectByID(selectedID)
	}
	t.reconcileDrag()
	t.render()
}

func (t *TaskboardPanel) selectByID(id int) {
	for li, l := range t.lanes {
		for ci, c := range l.cells() {
			if c.ID == id {
				t.lane = li
				t.cell = ci
				t.keepSelectionVisible()
				return
			}
		}
	}
}

func (t *TaskboardPanel) SelectedItem() *models.WorkItem {
	if t.lane < 0 || t.lane >= len(t.lanes) || t.cell < 0 {
		return nil
	}
	return t.lanes[t.lane].cell(t.cell)
}

func (t TaskboardPanel) Progress() (int, int) {
	done, total := 0, 0
	for _, l := range t.lanes {
		for ci, col := range l.cols {
			total += len(col)
			if models.IsCompletedState(t.states[ci]) {
				done += len(col)
			}
		}
	}
	return done, total
}

func (t TaskboardPanel) Title() string {
	switch t.dragMode() {
	case dragValid:
		target, _ := t.dragTarget()
		return fmt.Sprintf("Taskboard  moving %d to %s", t.drag.taskID, target)
	case dragSame:
		return fmt.Sprintf("Taskboard  moving %d · already %s", t.drag.taskID, t.drag.from)
	case dragInvalid:
		return fmt.Sprintf("Taskboard  moving %d · %s", t.drag.taskID, dropPrompt)
	}
	if start, count := t.stateWindow(); count < len(t.states) {
		return fmt.Sprintf("Taskboard  states %d-%d of %d  %s/%s to scroll", start+1, start+count, len(t.states),
			ShortKey(t.keys.Left), ShortKey(t.keys.Right))
	}
	done, total := t.Progress()
	title := fmt.Sprintf("Taskboard  %d backlog items  %d/%d tasks done", len(t.lanes), done, total)
	if n := t.collapsedCount(); n > 0 {
		title += fmt.Sprintf("  · %d collapsed", n)
	}
	return title
}

func (t *TaskboardPanel) SetSize(width, height int) {
	if t.width == width && t.height == height {
		return
	}
	t.width = width
	t.height = height
	t.viewport.SetWidth(max(width, 1))
	t.viewport.SetHeight(max(height, 1))
	t.render()
}

func (t *TaskboardPanel) SetFocused(focused bool) {
	if t.focused == focused {
		return
	}
	t.focused = focused
	if !focused {
		t.hover = 0
	}
	t.render()
}

func (t TaskboardPanel) Update(msg tea.Msg) (TaskboardPanel, tea.Cmd) {
	if !t.focused || len(t.lanes) == 0 {
		return t, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return t.handleKey(msg)
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			t.moveLane(-1)
		case tea.MouseWheelDown:
			t.moveLane(1)
		}
		t.render()
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			t.press(msg.X, msg.Y)
		}
	case tea.MouseMotionMsg:
		t.pointerAt(msg.X, msg.Y)
	case tea.MouseReleaseMsg:
		return t, t.drop(msg.X, msg.Y)
	}
	return t, nil
}

func (t TaskboardPanel) handleKey(msg tea.KeyPressMsg) (TaskboardPanel, tea.Cmd) {
	if t.drag.active {
		if key.Matches(msg, t.keys.Back) {
			t.cancelDrag()
		}
		return t, nil
	}
	switch {
	case key.Matches(msg, t.keys.Up):
		t.moveLane(-1)
	case key.Matches(msg, t.keys.Down):
		t.moveLane(1)
	case key.Matches(msg, t.keys.Left) && t.cell == 0:
		t.setCollapsed(t.lane, true)
	case key.Matches(msg, t.keys.Left):
		t.cell = max(t.cell-1, 0)
	case key.Matches(msg, t.keys.Right) && t.isCollapsed(t.lane):
		t.setCollapsed(t.lane, false)
	case key.Matches(msg, t.keys.Right):
		t.cell = min(t.cell+1, t.lanes[t.lane].size()-1)
	case key.Matches(msg, t.keys.ToggleRows):
		t.toggleAllRows()
	case key.Matches(msg, t.keys.Top):
		t.lane, t.cell = 0, 0
	case key.Matches(msg, t.keys.Bottom):
		t.lane, t.cell = len(t.lanes)-1, 0
	case key.Matches(msg, t.keys.MoveItemLeft):
		return t, t.moveTask(-1)
	case key.Matches(msg, t.keys.MoveItemRight):
		return t, t.moveTask(1)
	case key.Matches(msg, t.keys.Select):
		if item := t.SelectedItem(); item != nil {
			opened := *item
			return t, func() tea.Msg { return OpenWorkItemMsg{Item: opened} }
		}
	case key.Matches(msg, t.keys.View):
		if item := t.SelectedItem(); item != nil {
			viewed := *item
			return t, func() tea.Msg { return ViewWorkItemMsg{Item: viewed} }
		}
	}
	t.render()
	return t, nil
}

func (t *TaskboardPanel) moveLane(delta int) {
	target := max(min(t.lane+delta, len(t.lanes)-1), 0)
	if target == t.lane {
		return
	}
	col := t.lanes[t.lane].columnOf(t.cell)
	t.lane = target
	t.cell = t.lanes[target].firstCellIn(col)
	t.keepSelectionVisible()
}

func (t *TaskboardPanel) moveTask(delta int) tea.Cmd {
	col := t.lanes[t.lane].columnOf(t.cell)
	target := col + delta
	if col < 0 || target < 0 || target >= len(t.states) {
		return nil
	}
	return t.beginMove(t.SelectedItem(), t.states[target])
}

func (t TaskboardPanel) stateWindow() (int, int) {
	n := max(len(t.states), 1)
	usable := t.width - styles.TaskCellPadding - 2
	laneWidth := max(min(int(float64(usable)*styles.TaskLaneWidthRatio), styles.TaskLaneMaxWidth), styles.TaskLaneMinWidth)
	fit := max((usable-laneWidth)/(styles.TaskStateMinWidth+styles.TaskCellPadding+1), 1)
	count := min(n, fit)
	col := 0
	if t.lane < len(t.lanes) {
		col = max(t.lanes[t.lane].columnOf(t.cell), 0)
	}
	start := 0
	if col >= count {
		start = col - count + 1
	}
	return start, count
}

func (t TaskboardPanel) columnWidths(count int) (int, int) {
	usable := t.width - (count + 2) - (count+1)*styles.TaskCellPadding
	laneWidth := max(min(int(float64(usable)*styles.TaskLaneWidthRatio), styles.TaskLaneMaxWidth), styles.TaskLaneMinWidth)
	colWidth := max((usable-laneWidth)/max(count, 1), 1)
	return laneWidth, colWidth
}

func (t TaskboardPanel) phantomColumn(li int) int {
	target, ok := t.dragTarget()
	if !ok || li != t.drag.lane {
		return noColumn
	}
	for ci, s := range t.states {
		if s == target {
			return ci
		}
	}
	return noColumn
}

func (t TaskboardPanel) laneHeight(li int) int {
	if t.isCollapsed(li) {
		return styles.TaskCollapsedHeight
	}
	phantom := t.phantomColumn(li)
	tallest := 1
	for ci, col := range t.lanes[li].cols {
		n := len(col)
		if ci == phantom {
			n++
		}
		tallest = max(tallest, n)
	}
	return max(tallest*styles.TaskCardHeight+(tallest-1)*styles.TaskCardGap, styles.TaskParentHeight)
}

func (t TaskboardPanel) headerLabel(ci int) string {
	s := t.states[ci]
	count := 0
	for _, l := range t.lanes {
		count += len(l.cols[ci])
	}
	label := fmt.Sprintf("%s  %d", s, count)
	target, valid := t.dragTarget()
	switch {
	case valid && s == target:
		return dropMarker + label + countArrow + fmt.Sprint(count+1)
	case valid && s == t.drag.from:
		return label + countArrow + fmt.Sprint(count-1)
	case t.dragMode() == dragSame && s == t.drag.from:
		return label + "  · no change"
	}
	return label
}

func (t *TaskboardPanel) render() {
	if t.width <= 0 || t.height <= 0 {
		return
	}
	if len(t.lanes) == 0 {
		k := t.keys
		t.viewport.SetContent(Muted(fmt.Sprintf("No backlog items match. Pick another sprint with %s, or change the filters with %s-%s or %s.",
			k.Panel1.Help().Key, k.Panel2.Help().Key, k.Panel4.Help().Key, k.ClearFilters.Help().Key)))
		return
	}
	start, count := t.stateWindow()
	laneWidth, colWidth := t.columnWidths(count)
	headers := []string{t.rowsLabel()}
	target, valid := t.dragTarget()
	targetCol := noColumn
	for ci := start; ci < start+count && ci < len(t.states); ci++ {
		if valid && t.states[ci] == target {
			targetCol = len(headers)
		}
		headers = append(headers, t.headerLabel(ci))
	}
	rows := make([][]string, 0, len(t.lanes))
	for li := range t.lanes {
		rows = append(rows, t.laneRow(li, laneWidth, colWidth, start, count))
	}
	tbl := ltable.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.ColorBorder)).
		BorderRow(true).
		BorderColumn(true).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			st := lipgloss.NewStyle().Padding(0, 1)
			if col == targetCol {
				st = st.Background(styles.ColorDropTint)
			}
			switch {
			case row == ltable.HeaderRow && col == targetCol:
				return st.Bold(true).Foreground(styles.ColorPrimary)
			case row == ltable.HeaderRow:
				return st.Bold(true).Foreground(styles.ColorTextMuted)
			}
			return st
		})
	t.viewport.SetContent(tbl.String())
	t.scrollToLane()
}

func (t *TaskboardPanel) laneTop(i int) int {
	y := styles.TaskTableHeaderLines
	for j := 0; j < i; j++ {
		y += t.laneHeight(j) + 1
	}
	return y
}

func (t *TaskboardPanel) scrollToLane() {
	top := t.laneTop(t.lane)
	bottom := top + t.laneHeight(t.lane)
	if t.lane == 0 {
		top = 0
	}
	widgets.Reveal(&t.viewport, top, bottom)
}

func (t TaskboardPanel) laneRow(li, laneWidth, colWidth, start, count int) []string {
	l := t.lanes[li]
	height := t.laneHeight(li)
	selectedLane := li == t.lane
	phantom := t.phantomColumn(li)
	target, valid := t.dragTarget()
	if t.isCollapsed(li) {
		row := []string{t.renderCollapsedParent(l, laneWidth, selectedLane)}
		for ci := start; ci < start+count && ci < len(t.states); ci++ {
			row = append(row, t.renderTaskCount(l, ci, colWidth, valid && t.states[ci] == target))
		}
		return row
	}
	row := []string{Block(t.renderParent(l, laneWidth, selectedLane && t.cell == 0), laneWidth, height)}
	cell := 1
	for ci, col := range l.cols {
		visible := ci >= start && ci < start+count
		cards := make([]string, 0, len(col)+1)
		for _, task := range col {
			if visible {
				cards = append(cards, t.renderTaskCard(task, colWidth, t.lookFor(task), selectedLane && t.cell == cell))
			}
			cell++
		}
		if !visible {
			continue
		}
		if ci == phantom {
			if task, ok := t.taskByID(t.drag.taskID); ok {
				cards = append(cards, t.renderTaskCard(task, colWidth, lookPhantom, false))
			}
		}
		row = append(row, stackCards(cards, colWidth, height, valid && t.states[ci] == target))
	}
	return row
}

func stackCards(cards []string, width, height int, tinted bool) string {
	blank := strings.Repeat(" ", width)
	if tinted {
		blank = lipgloss.NewStyle().Background(styles.ColorDropTint).Render(blank)
	}
	lines := make([]string, 0, height)
	for i, card := range cards {
		if i > 0 {
			for range styles.TaskCardGap {
				lines = append(lines, blank)
			}
		}
		lines = append(lines, strings.Split(card, "\n")...)
	}
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return strings.Join(lines[:height], "\n")
}

func (t TaskboardPanel) renderParent(l taskLane, width int, selected bool) string {
	p := l.parent
	done, total := 0, 0
	for ci, col := range l.cols {
		total += len(col)
		if models.IsCompletedState(t.states[ci]) {
			done += len(col)
		}
	}
	title := Muted(chevronOpen) + " " + lipgloss.NewStyle().Foreground(styles.ColorLight).Render(fmt.Sprintf("%d ", p.ID)) +
		lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(p.Title)
	meta := "  " + StateText(string(p.State)) + Muted(fmt.Sprintf("  %d/%d done", done, total))
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(styles.TypeColor(string(p.Type))).
		PaddingLeft(1).
		Width(width)
	if selected && t.focused {
		style = style.BorderForeground(styles.ColorPrimary)
		title = Muted(chevronOpen) + " " + Badge(fmt.Sprintf("%d", p.ID), styles.ColorPrimary, styles.ColorOnAccent) + " " +
			lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(p.Title)
	}
	return style.Render(Line(title, width-styles.CardChromeWidth) + "\n" + Line(meta, width-styles.CardChromeWidth))
}

func (t TaskboardPanel) lookFor(task *models.WorkItem) cardLook {
	if t.drag.active && task.ID == t.drag.taskID {
		return lookGhost
	}
	if m, ok := t.moves[task.ID]; ok {
		switch m.flash {
		case flashSaving:
			return lookSaving
		case flashSaved:
			return lookSaved
		case flashFailed:
			return lookFailed
		}
	}
	if !t.drag.active && task.ID == t.hover {
		return lookHover
	}
	return lookNormal
}

type cardPaint struct {
	bg       *lipgloss.Style
	border   lipgloss.Border
	edge     lipgloss.Style
	id       lipgloss.Style
	title    lipgloss.Style
	note     string
	noteText lipgloss.Style
	grip     bool
}

func (t TaskboardPanel) cardPaint(task *models.WorkItem, look cardLook, selected bool) cardPaint {
	bg := styles.ColorSurface
	if selected {
		bg = styles.ColorSurfaceAlt
		if t.focused {
			bg = styles.ColorHighlight
		}
	}
	if look == lookHover && !selected {
		bg = styles.ColorChip
	}
	if look == lookPhantom {
		bg = styles.ColorPhantom
	}
	base := lipgloss.NewStyle().Background(bg)
	p := cardPaint{
		bg:       &base,
		border:   lipgloss.ThickBorder(),
		edge:     lipgloss.NewStyle().Foreground(styles.TypeColor(string(task.Type))),
		id:       base.Foreground(styles.ColorLight),
		title:    base.Foreground(styles.ColorText),
		note:     task.AssignedTo,
		noteText: base.Foreground(styles.ColorOffWhite),
		grip:     look == lookHover,
	}
	if task.AssignedTo == "" {
		p.note, p.noteText = "Unassigned", base.Foreground(styles.ColorTextMuted)
	}
	m := t.moves[task.ID]
	switch look {
	case lookGhost:
		plain := lipgloss.NewStyle()
		p.bg, p.border = nil, ghostBorder
		p.edge = plain.Foreground(styles.ColorTextMuted)
		p.id, p.title = plain.Foreground(styles.ColorMuted), plain.Foreground(styles.ColorMuted)
		p.note, p.noteText = ghostNote, plain.Foreground(styles.ColorMuted)
	case lookPhantom:
		target, _ := t.dragTarget()
		p.border, p.edge = phantomBorder, lipgloss.NewStyle().Foreground(styles.ColorPrimary)
		p.id, p.title = base.Foreground(styles.ColorPrimaryLight), base.Foreground(styles.ColorPrimaryLight)
		p.note, p.noteText = phantomNote+target, base.Foreground(styles.ColorPrimary).Bold(true)
	case lookSaving:
		p.edge = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
		p.note, p.noteText = savingNote+m.target+"…", base.Foreground(styles.ColorPrimary).Bold(true)
	case lookSaved:
		p.edge = lipgloss.NewStyle().Foreground(styles.ColorSuccess)
		p.note, p.noteText = savedNote+m.target, base.Foreground(styles.ColorSuccess).Bold(true)
	case lookFailed:
		p.edge = lipgloss.NewStyle().Foreground(styles.ColorError)
		p.note, p.noteText = failedNote+m.from, base.Foreground(styles.ColorError).Bold(true)
	}
	return p
}

func (t TaskboardPanel) renderTaskCard(task *models.WorkItem, width int, look cardLook, selected bool) string {
	p := t.cardPaint(task, look, selected)
	inner := max(width-styles.CardChromeWidth, 1)
	fill := lipgloss.NewStyle()
	if p.bg != nil {
		fill = *p.bg
	}
	titleWidth := inner
	grip := ""
	if p.grip {
		titleWidth = max(inner-2, 1)
		grip = fill.Render(" ") + fill.Foreground(styles.ColorLight).Render(gripGlyph)
	}
	title := p.id.Render(fmt.Sprintf("%d ", task.ID)) + p.title.Render(task.Title)
	body := fill.Width(titleWidth).MaxWidth(titleWidth).Inline(true).Render(title) + grip + "\n" +
		fill.Width(inner).MaxWidth(inner).Inline(true).Render(p.noteText.Render(p.note))
	card := fill.
		BorderStyle(p.border).
		BorderLeft(true).
		BorderForeground(p.edge.GetForeground()).
		PaddingLeft(1).
		Width(width)
	if p.bg != nil {
		card = card.BorderBackground(p.bg.GetBackground())
	}
	return card.Render(body)
}

func (t TaskboardPanel) View() string {
	if t.width <= 0 || t.height <= 0 {
		return ""
	}
	return Block(t.viewport.View(), t.width, t.height)
}

func (t *TaskboardPanel) KeyBindings() []key.Binding {
	k := t.keys
	if t.drag.active {
		drop := widgets.Relabel(k.DropTask, "cancel")
		switch t.dragMode() {
		case dragValid:
			target, _ := t.dragTarget()
			drop = widgets.Relabel(k.DropTask, "move to "+target)
		case dragSame:
			drop = widgets.Relabel(k.DropTask, "no change")
		}
		return []key.Binding{drop, widgets.Relabel(k.Back, "cancel")}
	}
	rows := widgets.Relabel(k.ToggleRows, "collapse all")
	if !t.anyExpanded() {
		rows = widgets.Relabel(k.ToggleRows, "expand all")
	}
	bindings := []key.Binding{
		k.View, k.MoveItemLeft, k.MoveItemRight, k.ChangeState, k.Assign,
		k.CreateTask, k.EditTask, k.AddComment, rows,
	}
	switch {
	case t.cell == 0 && t.isCollapsed(t.lane):
		bindings = append([]key.Binding{widgets.Relabel(k.Right, "expand row")}, bindings...)
	case t.cell == 0 && len(t.lanes) > 0:
		bindings = append([]key.Binding{widgets.Relabel(k.Left, "collapse row")}, bindings...)
	}
	if t.hover != 0 {
		return append([]key.Binding{widgets.Relabel(k.DragTask, fmt.Sprintf("move %d", t.hover))}, bindings...)
	}
	return append(bindings, k.DragTask)
}
