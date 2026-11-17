package panels

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const emptyBoard = "No backlog items match the current filters."

type boardColumn struct {
	state  string
	items  []*models.WorkItem
	cursor int
	top    int
}

func (c *boardColumn) selected() *models.WorkItem {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return nil
	}
	return c.items[c.cursor]
}

func (c *boardColumn) clamp(visible int) {
	c.cursor = min(max(c.cursor, 0), max(len(c.items)-1, 0))
	if c.cursor < c.top {
		c.top = c.cursor
	}
	if c.cursor >= c.top+visible {
		c.top = c.cursor - visible + 1
	}
	c.top = min(max(c.top, 0), max(len(c.items)-visible, 0))
}

type BoardPanel struct {
	columns      []*boardColumn
	items        []models.WorkItem
	statesByType map[string][]models.WorkItemStateInfo
	moves        map[int]taskMove
	flashSeq     int
	drag         boardDrag
	edgeSeq      int
	hover        int
	lastClick    int
	lastClickAt  time.Time
	styles       *styles.Styles
	keys         keys.KeyMap
	col          int
	start        int
	originX      int
	originY      int
	width        int
	height       int
	focused      bool
}

func NewBoardPanel(st *styles.Styles, k keys.KeyMap) BoardPanel {
	return BoardPanel{styles: st, keys: k}
}

func (b *BoardPanel) SetOrigin(x, y int) {
	b.originX, b.originY = x, y
}

func (b *BoardPanel) SetItems(items []models.WorkItem, statesByType map[string][]models.WorkItemStateInfo) {
	b.items = items
	b.statesByType = statesByType
	b.settleMoves()
	b.rebuild()
	b.reconcileDrag()
}

func (b BoardPanel) displayState(item *models.WorkItem) string {
	if m, ok := b.moves[item.ID]; ok && m.to != "" {
		return m.to
	}
	return string(item.State)
}

func (b *BoardPanel) rebuild() {
	selectedID := 0
	if sel := b.SelectedItem(); sel != nil {
		selectedID = sel.ID
	}
	backlog := backlogItems(b.items)
	var present []string
	seen := map[string]bool{}
	for _, it := range backlog {
		if state := b.displayState(it); !seen[state] {
			seen[state] = true
			present = append(present, state)
		}
	}
	states := models.BoardColumns(b.statesByType, backlogType(backlog, b.statesByType), present)
	grouped := map[string][]*models.WorkItem{}
	for _, it := range backlog {
		state := b.displayState(it)
		grouped[state] = append(grouped[state], it)
	}
	previous := map[string]*boardColumn{}
	for _, c := range b.columns {
		previous[c.state] = c
	}
	b.columns = make([]*boardColumn, len(states))
	for i, st := range states {
		c := &boardColumn{state: st, items: grouped[st]}
		if old, ok := previous[st]; ok {
			c.cursor, c.top = old.cursor, old.top
		}
		b.columns[i] = c
	}
	b.col = min(b.col, max(len(states)-1, 0))
	if selectedID > 0 {
		b.selectByID(selectedID)
	}
	b.ensureVisible()
}

func (b *BoardPanel) selectByID(id int) {
	for ci, c := range b.columns {
		for ri, it := range c.items {
			if it.ID == id {
				b.col, c.cursor = ci, ri
				return
			}
		}
	}
}

func (b BoardPanel) itemByID(id int) *models.WorkItem {
	for _, c := range b.columns {
		for _, it := range c.items {
			if it.ID == id {
				return it
			}
		}
	}
	return nil
}

func (b BoardPanel) allows(item *models.WorkItem, state string) bool {
	infos := b.statesByType[string(item.Type)]
	if len(infos) == 0 {
		return true
	}
	for _, s := range infos {
		if s.Name == state {
			return true
		}
	}
	return false
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
	return b.columns[b.col].selected()
}

func (b BoardPanel) Count() int {
	n := 0
	for _, c := range b.columns {
		n += len(c.items)
	}
	return n
}

func (b BoardPanel) Title() string {
	switch b.dragMode() {
	case dragValid:
		return fmt.Sprintf("Board  moving %d to %s", b.drag.id, b.drag.target)
	case dragSame:
		return fmt.Sprintf("Board  moving %d · already %s", b.drag.id, b.drag.from)
	case dragInvalid:
		reason := dropPrompt
		if b.drag.reason != "" {
			reason = b.drag.reason
		}
		return fmt.Sprintf("Board  moving %d · %s", b.drag.id, reason)
	}
	start, count, _ := b.window()
	if count < len(b.columns) {
		return fmt.Sprintf("Board  columns %d-%d of %d  %s/%s to scroll", start+1, start+count, len(b.columns),
			ShortKey(b.keys.Left), ShortKey(b.keys.Right))
	}
	return fmt.Sprintf("Board  %d items", b.Count())
}

func (b *BoardPanel) SetSize(width, height int) {
	if b.width == width && b.height == height {
		return
	}
	b.width, b.height = width, height
	b.ensureVisible()
}

func (b *BoardPanel) SetFocused(focused bool) {
	b.focused = focused
	if !focused {
		b.hover = 0
	}
}

func (b BoardPanel) window() (int, int, int) {
	n := max(len(b.columns), 1)
	fit := max((b.width+styles.BoardColumnGap)/(styles.BoardColumnMinWidth+styles.BoardColumnGap), 1)
	count := min(n, fit)
	start := min(max(b.start, 0), n-count)
	width := (b.width - (count-1)*styles.BoardColumnGap) / count
	return start, count, width
}

func (b *BoardPanel) ensureVisible() {
	start, count, _ := b.window()
	switch {
	case b.col < start:
		start = b.col
	case b.col >= start+count:
		start = b.col - count + 1
	}
	b.start = start
	visible := b.visibleCards()
	for _, c := range b.columns {
		c.clamp(visible)
	}
}

func (b BoardPanel) visibleCards() int {
	avail := b.height - 2 - styles.BoardColumnHeader
	return max((avail+styles.BoardCardGap)/(styles.BoardCardHeight+styles.BoardCardGap), 1)
}

func (b BoardPanel) Update(msg tea.Msg) (BoardPanel, tea.Cmd) {
	if !b.focused || len(b.columns) == 0 {
		return b, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return b.handleKey(msg)
	case tea.MouseWheelMsg:
		b.wheel(msg)
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			return b, b.press(msg.X, msg.Y)
		}
	case tea.MouseMotionMsg:
		return b, b.pointerAt(msg.X, msg.Y)
	case tea.MouseReleaseMsg:
		return b, b.drop(msg.X, msg.Y)
	}
	return b, nil
}

func (b BoardPanel) handleKey(msg tea.KeyPressMsg) (BoardPanel, tea.Cmd) {
	k := b.keys
	if b.drag.active {
		if key.Matches(msg, k.Back) {
			b.drag = boardDrag{}
		}
		return b, nil
	}
	c := b.columns[b.col]
	page := max(b.visibleCards()/2, 1)
	switch {
	case key.Matches(msg, k.Left):
		b.col = max(b.col-1, 0)
	case key.Matches(msg, k.Right):
		b.col = min(b.col+1, len(b.columns)-1)
	case key.Matches(msg, k.Up):
		c.cursor--
	case key.Matches(msg, k.Down):
		c.cursor++
	case key.Matches(msg, k.HalfPageUp):
		c.cursor -= page
	case key.Matches(msg, k.HalfPageDown):
		c.cursor += page
	case key.Matches(msg, k.Top):
		c.cursor = 0
	case key.Matches(msg, k.Bottom):
		c.cursor = len(c.items) - 1
	case key.Matches(msg, k.MoveItemLeft):
		return b, b.moveCard(-1)
	case key.Matches(msg, k.MoveItemRight):
		return b, b.moveCard(1)
	case key.Matches(msg, k.Select):
		if item := b.SelectedItem(); item != nil {
			opened := *item
			return b, func() tea.Msg { return OpenWorkItemMsg{Item: opened} }
		}
	case key.Matches(msg, k.View):
		if item := b.SelectedItem(); item != nil {
			viewed := *item
			return b, func() tea.Msg { return ViewWorkItemMsg{Item: viewed} }
		}
	}
	b.ensureVisible()
	return b, nil
}

func (b *BoardPanel) wheel(msg tea.MouseWheelMsg) {
	if lx, _, inside := b.local(msg.X, msg.Y); inside {
		if col, ok := b.columnAt(lx); ok {
			b.col = col
		}
	}
	c := b.columns[b.col]
	switch msg.Button {
	case tea.MouseWheelUp:
		c.cursor--
	case tea.MouseWheelDown:
		c.cursor++
	}
	b.ensureVisible()
}

func (b *BoardPanel) moveCard(delta int) tea.Cmd {
	item := b.SelectedItem()
	target := b.col + delta
	if item == nil || target < 0 || target >= len(b.columns) || !b.allows(item, b.columns[target].state) {
		return nil
	}
	return b.beginMove(item, b.columns[target].state)
}

func (b BoardPanel) View() string {
	if b.width <= 0 || b.height <= 0 {
		return ""
	}
	if len(b.columns) == 0 {
		return Block(Muted(emptyBoard), b.width, b.height)
	}
	start, count, width := b.window()
	gap := Block("", styles.BoardColumnGap, b.height)
	blocks := make([]string, 0, 2*count)
	for i := start; i < start+count; i++ {
		if i > start {
			blocks = append(blocks, gap)
		}
		blocks = append(blocks, b.renderColumn(i, width))
	}
	return Block(lipgloss.JoinHorizontal(lipgloss.Top, blocks...), b.width, b.height)
}

type boardCardView struct {
	item *models.WorkItem
	look cardLook
}

func (b BoardPanel) columnCards(i int) ([]boardCardView, int) {
	c := b.columns[i]
	var out []boardCardView
	top := c.top
	if b.dragMode() == dragValid && b.drag.target == c.state {
		if item := b.itemByID(b.drag.id); item != nil {
			out = append(out, boardCardView{item: item, look: lookPhantom})
		}
		top = 0
	}
	for _, it := range c.items {
		look := b.cardLook(it)
		out = append(out, boardCardView{item: it, look: look})
	}
	return out, top
}

func (b BoardPanel) cardLook(item *models.WorkItem) cardLook {
	if b.drag.active && b.drag.id == item.ID {
		return lookGhost
	}
	if m, ok := b.moves[item.ID]; ok {
		switch m.flash {
		case flashSaving:
			return lookSaving
		case flashSaved:
			return lookSaved
		case flashFailed:
			return lookFailed
		}
	}
	if !b.drag.active && item.ID == b.hover {
		return lookHover
	}
	return lookNormal
}

func (b BoardPanel) renderColumn(i, width int) string {
	c := b.columns[i]
	inner := max(width-2, 1)
	innerHeight := max(b.height-2, 0)
	target := b.dragMode() == dragValid && b.drag.target == c.state
	tint := lipgloss.NewStyle()
	border := styles.ColorBorder
	if target {
		tint = tint.Background(styles.ColorDropTint)
		border = styles.ColorPrimary
	}
	blank := tint.Render(strings.Repeat(" ", inner))
	cards, top := b.columnCards(i)
	visible := b.visibleCards()
	lines := []string{
		b.columnHeader(i, inner, tint, target, top, len(cards)),
		tint.Foreground(styles.ColorChip).Render(strings.Repeat("─", inner)),
		blank,
	}
	pad := tint.Render(strings.Repeat(" ", styles.BoardColumnPadding))
	cardWidth := max(inner-2*styles.BoardColumnPadding, 1)
	end := min(top+visible, len(cards))
	for j := top; j < end; j++ {
		if j > top {
			lines = append(lines, blank)
		}
		cv := cards[j]
		selected := cv.look != lookPhantom && i == b.col && cv.item == c.selected()
		for _, l := range b.renderCard(cv.item, cardWidth, cv.look, selected) {
			lines = append(lines, pad+l+pad)
		}
	}
	for len(lines) < innerHeight {
		lines = append(lines, blank)
	}
	lines = lines[:innerHeight]
	edge := lipgloss.NewStyle().Foreground(border)
	side := edge.Render("│")
	out := make([]string, 0, innerHeight+2)
	out = append(out, edge.Render("╭"+strings.Repeat("─", inner)+"╮"))
	for _, l := range lines {
		out = append(out, side+l+side)
	}
	out = append(out, edge.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return strings.Join(out, "\n")
}

func (b BoardPanel) columnHeader(i, inner int, tint lipgloss.Style, target bool, top, total int) string {
	c := b.columns[i]
	name := tint.Bold(true).Foreground(styles.ColorText)
	if target || (i == b.col && b.focused && !b.drag.active) {
		name = name.Foreground(styles.ColorPrimary)
	}
	count := fmt.Sprint(len(c.items))
	if b.dragMode() == dragValid {
		switch c.state {
		case b.drag.from:
			count += countArrow + fmt.Sprint(len(c.items)-1)
		case b.drag.target:
			count += countArrow + fmt.Sprint(len(c.items)+1)
		}
	}
	prefix := " "
	if target {
		prefix += dropMarker
	}
	chip := lipgloss.NewStyle().Background(styles.ColorChip).Foreground(styles.ColorLight).Render(" " + count + " ")
	left := tint.Foreground(styles.ColorPrimary).Render(prefix) + name.Render(c.state) + tint.Render("  ") + chip
	var hidden []string
	if top > 0 {
		hidden = append(hidden, fmt.Sprintf("↑ %d", top))
	}
	if below := total - top - b.visibleCards(); below > 0 {
		hidden = append(hidden, fmt.Sprintf("%d more ↓", below))
	}
	right := tint.Foreground(styles.ColorTextMuted).Render(strings.Join(hidden, "  "))
	return columnSpread(left, right, inner, tint)
}

func columnSpread(left, right string, width int, base lipgloss.Style) string {
	room := width - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if right == "" || room < 1 {
		return fit(left, width, base)
	}
	return left + base.Render(strings.Repeat(" ", room)) + right + base.Render(" ")
}

func (b *BoardPanel) KeyBindings() []key.Binding {
	k := b.keys
	if b.drag.active {
		drop := widgets.Relabel(k.DropTask, "cancel")
		switch b.dragMode() {
		case dragValid:
			drop = widgets.Relabel(k.DropTask, "move to "+b.drag.target)
		case dragSame:
			drop = widgets.Relabel(k.DropTask, "no change")
		}
		return []key.Binding{drop, widgets.Relabel(k.Back, "cancel")}
	}
	bindings := []key.Binding{
		k.View, k.MoveItemLeft, k.MoveItemRight, k.ChangeState, k.Assign,
		k.CreateTask, k.AddComment, k.CreateParent,
	}
	if b.hover != 0 {
		return append([]key.Binding{widgets.Relabel(k.DragTask, fmt.Sprintf("move %d", b.hover))}, bindings...)
	}
	return append([]key.Binding{widgets.Relabel(k.DragTask, "move a card")}, bindings...)
}

type MoveWorkItemMsg struct {
	Item     models.WorkItem
	NewState string
}
