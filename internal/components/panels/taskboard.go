package panels

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	return tallest * styles.TaskCardHeight
}

type TaskboardPanel struct {
	lanes   []taskLane
	states  []string
	styles  styles.Styles
	keys    keys.KeyMap
	lane    int
	cell    int
	offset  int
	width   int
	height  int
	focused bool
}

func NewTaskboardPanel(st styles.Styles, k keys.KeyMap) TaskboardPanel {
	return TaskboardPanel{styles: st, keys: k}
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
	t.ensureVisible()
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
	return fmt.Sprintf("Taskboard · %d backlog items · %d/%d tasks done", len(t.lanes), done, total)
}

func (t *TaskboardPanel) SetSize(width, height int) {
	t.width = width
	t.height = height
	t.ensureVisible()
}

func (t *TaskboardPanel) SetFocused(focused bool) {
	t.focused = focused
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
			return t, func() tea.Msg { return OpenWorkItemMsg{Item: *item} }
		}
	case key.Matches(msg, t.keys.View):
		if item := t.SelectedItem(); item != nil {
			return t, func() tea.Msg { return ViewWorkItemMsg{Item: *item} }
		}
	}
	t.ensureVisible()
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
	t.ensureVisible()
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

func (t *TaskboardPanel) laneTop(i int) int {
	y := 0
	for j := 0; j < i; j++ {
		y += t.lanes[j].height() + 1
	}
	return y
}

func (t *TaskboardPanel) ensureVisible() {
	if len(t.lanes) == 0 {
		t.offset = 0
		return
	}
	view := max(t.height-styles.BoardHeaderLines, 1)
	top := t.laneTop(t.lane)
	bottom := top + t.lanes[t.lane].height()
	if top < t.offset {
		t.offset = top
	}
	if bottom > t.offset+view {
		t.offset = bottom - view
	}
	t.offset = max(t.offset, 0)
}

func (t TaskboardPanel) layout() (int, int) {
	laneWidth := int(float64(t.width) * styles.TaskLaneWidthRatio)
	laneWidth = max(min(laneWidth, styles.TaskLaneMaxWidth), styles.TaskLaneMinWidth)
	n := max(len(t.states), 1)
	colWidth := (t.width - laneWidth - n*styles.BoardColumnGap) / n
	return laneWidth, max(colWidth, 1)
}

func (t TaskboardPanel) View() string {
	if t.width <= 0 || t.height <= 0 {
		return ""
	}
	if len(t.lanes) == 0 {
		return FitBlock(t.styles.TextMuted.Render("No backlog items in this iteration. Pick another one with 1."), t.width, t.height)
	}
	laneWidth, colWidth := t.layout()
	gap := strings.Repeat(" ", styles.BoardColumnGap)
	head := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true)
	header := FitWidth(head.Render("BACKLOG ITEM"), laneWidth)
	for ci, s := range t.states {
		count := 0
		for _, l := range t.lanes {
			count += len(l.cols[ci])
		}
		label := head.Render(strings.ToUpper(s)) + lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(fmt.Sprintf("  %d", count))
		header += gap + FitWidth(label, colWidth)
	}
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", t.width))

	var body []string
	for li, l := range t.lanes {
		body = append(body, t.renderLane(li, l, laneWidth, colWidth)...)
		body = append(body, lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("┄", t.width)))
	}
	view := max(t.height-styles.BoardHeaderLines, 0)
	start := min(t.offset, max(len(body)-1, 0))
	end := min(start+view, len(body))
	lines := append([]string{header, rule}, body[start:end]...)
	return FitBlock(strings.Join(lines, "\n"), t.width, t.height)
}

func (t TaskboardPanel) renderLane(li int, l taskLane, laneWidth, colWidth int) []string {
	height := l.height()
	selectedLane := li == t.lane
	parentSelected := selectedLane && t.cell == 0

	var laneBg color.Color
	if selectedLane {
		laneBg = styles.ColorSurfaceAlt
	}
	parentBg := laneBg
	if parentSelected && t.focused {
		parentBg = styles.ColorHighlight
	}
	parentBlock := t.renderParent(l, laneWidth, height, parentBg)

	blocks := []string{parentBlock}
	cellIndex := 1
	for _, col := range l.cols {
		var cards []string
		for _, task := range col {
			bg := styles.ColorSurface
			if selectedLane && t.cell == cellIndex && t.focused {
				bg = styles.ColorHighlight
			}
			cards = append(cards, renderTaskCard(task, colWidth, bg)...)
			cellIndex++
		}
		blocks = append(blocks, FitBlock("", styles.BoardColumnGap, height), FitBlock(strings.Join(cards, "\n"), colWidth, height))
	}
	return strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, blocks...), "\n")
}

func (t TaskboardPanel) renderParent(l taskLane, width, height int, bg color.Color) string {
	p := l.parent
	paint := func(fg color.Color) lipgloss.Style {
		st := lipgloss.NewStyle().Foreground(fg)
		if bg != nil {
			st = st.Background(bg)
		}
		return st
	}
	bar := paint(styles.TypeColor(string(p.Type))).Render("▌")
	title := paint(styles.ColorLight).Render(fmt.Sprintf("%d ", p.ID)) + paint(styles.ColorText).Bold(true).Render(p.Title)
	done, total := 0, 0
	for ci, col := range l.cols {
		total += len(col)
		if models.IsCompletedState(t.states[ci]) {
			done += len(col)
		}
	}
	meta := StateDot(string(p.State), bg) + paint(styles.ColorTextMuted).Render(fmt.Sprintf(" · %d/%d done", done, total))
	lines := []string{bar + paint(styles.ColorText).Render(" ") + title, bar + paint(styles.ColorText).Render(" ") + meta}
	for len(lines) < height {
		lines = append(lines, "")
	}
	if bg == nil {
		return FitBlock(strings.Join(lines, "\n"), width, height)
	}
	return FillBlock(strings.Join(lines, "\n"), width, height, bg)
}

func renderTaskCard(task *models.WorkItem, width int, bg color.Color) []string {
	on := func(fg color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fg).Background(bg)
	}
	bar := on(styles.TypeColor(string(task.Type))).Render("▌")
	assignee := task.AssignedTo
	fg := styles.ColorOffWhite
	if assignee == "" {
		assignee = "Unassigned"
		fg = styles.ColorTextMuted
	}
	line1 := on(styles.ColorLight).Render(fmt.Sprintf(" %d ", task.ID)) + on(styles.ColorText).Render(task.Title)
	line2 := on(fg).Render(" ◉ " + assignee)
	inner := max(width-1, 1)
	return []string{bar + FillBlock(line1, inner, 1, bg), bar + FillBlock(line2, inner, 1, bg)}
}

func (t *TaskboardPanel) GetAvailableActions() []string {
	return Hints(styles.ColorSurface,
		t.keys.View, t.keys.MoveItemLeft, t.keys.MoveItemRight, t.keys.ChangeState, t.keys.Assign,
		t.keys.CreateTask, t.keys.EditTask, t.keys.AddComment,
	)
}
