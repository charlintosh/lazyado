package panels

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	ltable "charm.land/lipgloss/v2/table"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

var defaultTaskStates = []string{"To Do", "In Progress", "Done"}

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

func (l taskLane) height() int {
	tallest := 1
	for _, col := range l.cols {
		tallest = max(tallest, len(col))
	}
	return max(tallest*styles.TaskCardHeight, styles.TaskParentHeight)
}

type TaskboardPanel struct {
	lanes    []taskLane
	states   []string
	styles   styles.Styles
	keys     keys.KeyMap
	viewport viewport.Model
	lane     int
	cell     int
	width    int
	height   int
	focused  bool
}

func NewTaskboardPanel(st styles.Styles, k keys.KeyMap) TaskboardPanel {
	vp := viewport.New()
	vp.KeyMap = viewport.KeyMap{}
	return TaskboardPanel{styles: st, keys: k, viewport: vp}
}

func (t *TaskboardPanel) SetItems(items []models.WorkItem, statesByType map[string][]models.WorkItemStateInfo) {
	selectedID := 0
	if sel := t.SelectedItem(); sel != nil {
		selectedID = sel.ID
	}
	parents := backlogItems(items)
	var present []string
	seen := map[string]bool{}
	for _, p := range parents {
		for _, c := range p.Children {
			if c.Type == models.WorkItemTypeTask && !seen[string(c.State)] {
				seen[string(c.State)] = true
				present = append(present, string(c.State))
			}
		}
	}
	states := models.BoardColumns(statesByType, models.WorkItemTypeTask, present)
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
			if i, ok := index[string(c.State)]; ok && c.Type == models.WorkItemTypeTask {
				lane.cols[i] = append(lane.cols[i], c)
			}
		}
		lanes = append(lanes, lane)
	}
	t.states = states
	t.lanes = lanes
	t.lane = max(min(t.lane, len(lanes)-1), 0)
	t.cell = 0
	if selectedID > 0 {
		t.selectByID(selectedID)
	}
	t.render()
}

func (t *TaskboardPanel) selectByID(id int) {
	for li, l := range t.lanes {
		for ci, c := range l.cells() {
			if c.ID == id {
				t.lane = li
				t.cell = ci
				return
			}
		}
	}
}

func (t *TaskboardPanel) SelectedItem() *models.WorkItem {
	if t.lane < 0 || t.lane >= len(t.lanes) {
		return nil
	}
	cells := t.lanes[t.lane].cells()
	if t.cell < 0 || t.cell >= len(cells) {
		return nil
	}
	return cells[t.cell]
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
	done, total := t.Progress()
	return fmt.Sprintf("Taskboard  %d backlog items  %d/%d tasks done", len(t.lanes), done, total)
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
	t.focused = focused
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
	}
	return t, nil
}

func (t TaskboardPanel) handleKey(msg tea.KeyPressMsg) (TaskboardPanel, tea.Cmd) {
	switch {
	case key.Matches(msg, t.keys.Up):
		t.moveLane(-1)
	case key.Matches(msg, t.keys.Down):
		t.moveLane(1)
	case key.Matches(msg, t.keys.Left):
		t.cell = max(t.cell-1, 0)
	case key.Matches(msg, t.keys.Right):
		t.cell = min(t.cell+1, len(t.lanes[t.lane].cells())-1)
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
}

func (t TaskboardPanel) moveTask(delta int) tea.Cmd {
	l := t.lanes[t.lane]
	col := l.columnOf(t.cell)
	target := col + delta
	if col < 0 || target < 0 || target >= len(t.states) {
		return nil
	}
	moved := *t.SelectedItem()
	state := t.states[target]
	return func() tea.Msg { return MoveWorkItemMsg{Item: moved, NewState: state} }
}

func (t TaskboardPanel) columnWidths() (int, int) {
	n := max(len(t.states), 1)
	usable := t.width - (n + 2) - (n+1)*styles.TaskCellPadding
	laneWidth := int(float64(usable) * styles.TaskLaneWidthRatio)
	laneWidth = max(min(laneWidth, styles.TaskLaneMaxWidth), styles.TaskLaneMinWidth)
	colWidth := max((usable-laneWidth)/n, 1)
	return laneWidth, colWidth
}

func (t *TaskboardPanel) render() {
	if t.width <= 0 || t.height <= 0 {
		return
	}
	if len(t.lanes) == 0 {
		t.viewport.SetContent(Muted("No backlog items in this iteration. Pick another one with 1."))
		return
	}
	laneWidth, colWidth := t.columnWidths()
	headers := []string{"Backlog item"}
	for ci, s := range t.states {
		count := 0
		for _, l := range t.lanes {
			count += len(l.cols[ci])
		}
		headers = append(headers, fmt.Sprintf("%s  %d", s, count))
	}
	rows := make([][]string, 0, len(t.lanes))
	for li, l := range t.lanes {
		rows = append(rows, t.laneRow(li, l, laneWidth, colWidth))
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
			if row == ltable.HeaderRow {
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
		y += t.lanes[j].height() + 1
	}
	return y
}

func (t *TaskboardPanel) scrollToLane() {
	top := t.laneTop(t.lane)
	bottom := top + t.lanes[t.lane].height()
	if t.lane == 0 {
		top = 0
	}
	if top < t.viewport.YOffset() {
		t.viewport.SetYOffset(top)
	} else if bottom > t.viewport.YOffset()+t.viewport.Height() {
		t.viewport.SetYOffset(bottom - t.viewport.Height() + 1)
	}
}

func (t TaskboardPanel) laneRow(li int, l taskLane, laneWidth, colWidth int) []string {
	height := l.height()
	selectedLane := li == t.lane
	row := []string{Block(t.renderParent(l, laneWidth, selectedLane && t.cell == 0), laneWidth, height)}
	cell := 1
	for _, col := range l.cols {
		cards := make([]string, 0, len(col))
		for _, task := range col {
			cards = append(cards, renderTaskCard(task, colWidth, selectedLane && t.cell == cell, t.focused))
			cell++
		}
		row = append(row, Block(lipgloss.JoinVertical(lipgloss.Left, cards...), colWidth, height))
	}
	return row
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
	title := lipgloss.NewStyle().Foreground(styles.ColorLight).Render(fmt.Sprintf("%d ", p.ID)) +
		lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(p.Title)
	meta := StateText(string(p.State)) + Muted(fmt.Sprintf("  %d/%d done", done, total))
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(styles.TypeColor(string(p.Type))).
		PaddingLeft(1).
		Width(width)
	if selected && t.focused {
		style = style.BorderForeground(styles.ColorPrimary)
		title = Badge(fmt.Sprintf("%d", p.ID), styles.ColorPrimary, styles.ColorOnAccent) + " " +
			lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(p.Title)
	}
	return style.Render(Line(title, width-styles.CardChromeWidth) + "\n" + Line(meta, width-styles.CardChromeWidth))
}

func renderTaskCard(task *models.WorkItem, width int, selected, focused bool) string {
	bg := styles.ColorSurface
	if selected {
		bg = styles.ColorSurfaceAlt
		if focused {
			bg = styles.ColorHighlight
		}
	}
	inner := max(width-styles.CardChromeWidth, 1)
	text := lipgloss.NewStyle().Background(bg)
	assignee := text.Foreground(styles.ColorOffWhite).Render(task.AssignedTo)
	if task.AssignedTo == "" {
		assignee = text.Foreground(styles.ColorTextMuted).Render("Unassigned")
	}
	title := text.Foreground(styles.ColorLight).Render(fmt.Sprintf("%d ", task.ID)) + text.Foreground(styles.ColorText).Render(task.Title)
	body := text.Width(inner).MaxWidth(inner).Inline(true).Render(title) + "\n" +
		text.Width(inner).MaxWidth(inner).Inline(true).Render(assignee)
	return lipgloss.NewStyle().
		Background(bg).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(styles.TypeColor(string(task.Type))).
		BorderBackground(bg).
		PaddingLeft(1).
		Width(width).
		Render(body)
}

func (t TaskboardPanel) View() string {
	if t.width <= 0 || t.height <= 0 {
		return ""
	}
	return Block(t.viewport.View(), t.width, t.height)
}

func (t *TaskboardPanel) KeyBindings() []key.Binding {
	return []key.Binding{
		t.keys.View, t.keys.MoveItemLeft, t.keys.MoveItemRight, t.keys.ChangeState, t.keys.Assign,
		t.keys.CreateTask, t.keys.EditTask, t.keys.AddComment,
	}
}
