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

type boardColumn struct {
	state  string
	items  []*models.WorkItem
	row    int
	offset int
}

type BoardPanel struct {
	columns []boardColumn
	styles  styles.Styles
	keys    keys.KeyMap
	col     int
	width   int
	height  int
	focused bool
}

func NewBoardPanel(st styles.Styles, k keys.KeyMap) BoardPanel {
	return BoardPanel{styles: st, keys: k}
}

func (b *BoardPanel) SetItems(items []models.WorkItem, statesByType map[string][]models.WorkItemStateInfo) {
	selectedID := 0
	if sel := b.SelectedItem(); sel != nil {
		selectedID = sel.ID
	}
	backlog := backlogItems(items)
	var present []string
	seen := map[string]bool{}
	for _, it := range backlog {
		if !seen[string(it.State)] {
			seen[string(it.State)] = true
			present = append(present, string(it.State))
		}
	}
	states := models.BoardColumns(statesByType, backlogType(backlog, statesByType), present)
	cols := make([]boardColumn, len(states))
	index := map[string]int{}
	for i, st := range states {
		cols[i] = boardColumn{state: st}
		index[st] = i
	}
	for _, it := range backlog {
		if i, ok := index[string(it.State)]; ok {
			cols[i].items = append(cols[i].items, it)
		}
	}
	b.columns = cols
	b.col = min(b.col, max(len(cols)-1, 0))
	if selectedID > 0 {
		b.selectByID(selectedID)
	}
	b.clampRows()
}

func (b *BoardPanel) selectByID(id int) {
	for ci, c := range b.columns {
		for ri, it := range c.items {
			if it.ID == id {
				b.col = ci
				b.columns[ci].row = ri
				return
			}
		}
	}
}

func (b *BoardPanel) clampRows() {
	for i := range b.columns {
		c := &b.columns[i]
		c.row = max(min(c.row, len(c.items)-1), 0)
	}
	b.ensureVisible()
}

func backlogItems(items []models.WorkItem) []*models.WorkItem {
	var out []*models.WorkItem
	for i := range items {
		switch items[i].Type {
		case models.WorkItemTypePBI, models.WorkItemTypeStory, models.WorkItemTypeBug:
			out = append(out, &items[i])
		}
	}
	if len(out) > 0 {
		return out
	}
	for i := range items {
		if items[i].Type != models.WorkItemTypeTask {
			out = append(out, &items[i])
		}
	}
	return out
}

func backlogType(items []*models.WorkItem, statesByType map[string][]models.WorkItemStateInfo) models.WorkItemType {
	for _, t := range []models.WorkItemType{models.WorkItemTypePBI, models.WorkItemTypeStory} {
		if _, ok := statesByType[string(t)]; ok {
			return t
		}
	}
	if len(items) > 0 {
		return items[0].Type
	}
	return models.WorkItemTypePBI
}

func (b *BoardPanel) SelectedItem() *models.WorkItem {
	if b.col < 0 || b.col >= len(b.columns) {
		return nil
	}
	c := b.columns[b.col]
	if c.row < 0 || c.row >= len(c.items) {
		return nil
	}
	return c.items[c.row]
}

func (b BoardPanel) Count() int {
	n := 0
	for _, c := range b.columns {
		n += len(c.items)
	}
	return n
}

func (b BoardPanel) Title() string {
	return fmt.Sprintf("Board · %d items", b.Count())
}

func (b *BoardPanel) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.ensureVisible()
}

func (b *BoardPanel) SetFocused(focused bool) {
	b.focused = focused
}

func (b BoardPanel) Update(msg tea.Msg) (BoardPanel, tea.Cmd) {
	if !b.focused || len(b.columns) == 0 {
		return b, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return b.handleKey(msg)
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			b.moveRow(-1)
		case tea.MouseWheelDown:
			b.moveRow(1)
		}
	}
	return b, nil
}

func (b BoardPanel) handleKey(msg tea.KeyPressMsg) (BoardPanel, tea.Cmd) {
	switch {
	case key.Matches(msg, b.keys.Left):
		b.col = max(b.col-1, 0)
	case key.Matches(msg, b.keys.Right):
		b.col = min(b.col+1, len(b.columns)-1)
	case key.Matches(msg, b.keys.Up):
		b.moveRow(-1)
	case key.Matches(msg, b.keys.Down):
		b.moveRow(1)
	case key.Matches(msg, b.keys.Top):
		b.columns[b.col].row = 0
	case key.Matches(msg, b.keys.Bottom):
		b.columns[b.col].row = max(len(b.columns[b.col].items)-1, 0)
	case key.Matches(msg, b.keys.MoveItemLeft):
		return b, b.moveCard(-1)
	case key.Matches(msg, b.keys.MoveItemRight):
		return b, b.moveCard(1)
	case key.Matches(msg, b.keys.Select):
		if item := b.SelectedItem(); item != nil {
			return b, func() tea.Msg { return OpenWorkItemMsg{Item: *item} }
		}
	case key.Matches(msg, b.keys.View):
		if item := b.SelectedItem(); item != nil {
			return b, func() tea.Msg { return ViewWorkItemMsg{Item: *item} }
		}
	}
	b.ensureVisible()
	return b, nil
}

func (b *BoardPanel) moveRow(delta int) {
	c := &b.columns[b.col]
	c.row = max(min(c.row+delta, len(c.items)-1), 0)
	b.ensureVisible()
}

func (b BoardPanel) moveCard(delta int) tea.Cmd {
	item := b.SelectedItem()
	target := b.col + delta
	if item == nil || target < 0 || target >= len(b.columns) {
		return nil
	}
	state := b.columns[target].state
	moved := *item
	return func() tea.Msg { return MoveWorkItemMsg{Item: moved, NewState: state} }
}

func (b *BoardPanel) cardsPerColumn() int {
	return max((b.height-styles.BoardHeaderLines+styles.BoardCardGap)/(styles.BoardCardHeight+styles.BoardCardGap), 1)
}

func (b *BoardPanel) ensureVisible() {
	per := b.cardsPerColumn()
	for i := range b.columns {
		c := &b.columns[i]
		if c.row < c.offset {
			c.offset = c.row
		}
		if c.row >= c.offset+per {
			c.offset = c.row - per + 1
		}
		c.offset = max(min(c.offset, len(c.items)-per), 0)
	}
}

func (b BoardPanel) visibleColumns() (int, int, int) {
	n := len(b.columns)
	maxCols := max((b.width+styles.BoardColumnGap)/(styles.BoardColumnMinWidth+styles.BoardColumnGap), 1)
	count := min(n, maxCols)
	start := 0
	if b.col >= count {
		start = b.col - count + 1
	}
	width := (b.width - (count-1)*styles.BoardColumnGap) / max(count, 1)
	return start, count, width
}

func (b BoardPanel) View() string {
	if b.width <= 0 || b.height <= 0 {
		return ""
	}
	if len(b.columns) == 0 {
		return FitBlock(b.styles.TextMuted.Render("No backlog items match the current filters."), b.width, b.height)
	}
	start, count, colWidth := b.visibleColumns()
	var blocks []string
	for i := start; i < start+count; i++ {
		if i > start {
			blocks = append(blocks, FitBlock("", styles.BoardColumnGap, b.height))
		}
		blocks = append(blocks, b.renderColumn(i, colWidth))
	}
	out := lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
	if start > 0 || start+count < len(b.columns) {
		lines := strings.Split(out, "\n")
		marker := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).
			Render(fmt.Sprintf("columns %d–%d of %d · h/l to scroll", start+1, start+count, len(b.columns)))
		lines[len(lines)-1] = marker
		out = strings.Join(lines, "\n")
	}
	return FitBlock(out, b.width, b.height)
}

func (b BoardPanel) renderColumn(i, width int) string {
	c := b.columns[i]
	active := i == b.col
	nameStyle := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	ruleColor := styles.ColorBorder
	if active && b.focused {
		ruleColor = styles.ColorPrimary
		countStyle = countStyle.Foreground(styles.ColorPrimary)
	}
	header := nameStyle.Render(c.state)
	count := countStyle.Render(fmt.Sprintf("%d", len(c.items)))
	gap := width - lipgloss.Width(header) - lipgloss.Width(count)
	lines := []string{
		header + strings.Repeat(" ", max(gap, 1)) + count,
		lipgloss.NewStyle().Foreground(ruleColor).Render(strings.Repeat("─", width)),
	}
	per := b.cardsPerColumn()
	end := min(c.offset+per, len(c.items))
	for r := c.offset; r < end; r++ {
		if r > c.offset {
			lines = append(lines, "")
		}
		lines = append(lines, b.renderCard(c.items[r], width, active && r == c.row)...)
	}
	if len(c.items) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("No items"))
	}
	if end < len(c.items) {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorTextMuted).
			Render(fmt.Sprintf("+ %d more", len(c.items)-end)))
	}
	return FitBlock(strings.Join(lines, "\n"), width, b.height)
}

func (b BoardPanel) renderCard(item *models.WorkItem, width int, selected bool) []string {
	bg := styles.ColorSurface
	if selected {
		bg = styles.ColorSurfaceAlt
		if b.focused {
			bg = styles.ColorHighlight
		}
	}
	return renderWorkItemCard(item, width, bg, selected && b.focused)
}

func renderWorkItemCard(item *models.WorkItem, width int, bg color.Color, highlight bool) []string {
	on := func(fg color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fg).Background(bg)
	}
	bar := on(styles.TypeColor(string(item.Type))).Render("▌")
	inner := max(width-1, 1)
	idColor := styles.ColorLight
	if highlight {
		idColor = styles.ColorPrimary
	}
	id := fmt.Sprintf("%d ", item.ID)
	titleWidth := max(inner-1-len(id), 1)
	wrapped := strings.Split(lipgloss.NewStyle().Width(titleWidth).Render(item.Title), "\n")
	line1 := on(idColor).Render(" "+id) + on(styles.ColorText).Bold(true).Render(strings.TrimRight(wrapped[0], " "))
	line2 := ""
	if len(wrapped) > 1 {
		line2 = on(styles.ColorText).Bold(true).Render(strings.Repeat(" ", len(id)+1) + strings.TrimRight(wrapped[1], " "))
	}

	assignee := item.AssignedTo
	assigneeColor := styles.ColorOffWhite
	if assignee == "" {
		assignee = "Unassigned"
		assigneeColor = styles.ColorTextMuted
	}
	effort := ""
	if item.Effort > 0 {
		effort = formatEffort(item.Effort) + " "
	}
	left := on(assigneeColor).Render(" ◉ " + assignee)
	right := on(styles.ColorLight).Render(effort)
	line3 := left + on(styles.ColorText).Render(strings.Repeat(" ", max(inner-lipgloss.Width(left)-lipgloss.Width(right), 1))) + right

	var meta []string
	if len(item.Children) > 0 {
		done, total := DoneCount(item.Children)
		meta = append(meta, on(styles.ColorTextMuted).Render(fmt.Sprintf("☑ %d/%d", done, total)))
	}
	if len(item.Tags) > 0 {
		meta = append(meta, Chip(item.Tags[0]))
	}
	line4 := on(styles.ColorText).Render(" ") + strings.Join(meta, on(styles.ColorText).Render(" "))
	if len(meta) == 0 {
		line4 = StateDot(string(item.State), bg)
		line4 = on(styles.ColorText).Render(" ") + line4
	}

	lines := []string{line1, line2, line3, line4}
	for i, l := range lines {
		lines[i] = bar + FillBlock(l, inner, 1, bg)
	}
	return lines
}

func (b *BoardPanel) GetAvailableActions() []string {
	return Hints(styles.ColorSurface,
		b.keys.View, b.keys.MoveItemLeft, b.keys.MoveItemRight, b.keys.ChangeState, b.keys.Assign,
		b.keys.CreateTask, b.keys.AddComment, b.keys.CreateParent,
	)
}

type MoveWorkItemMsg struct {
	Item     models.WorkItem
	NewState string
}
