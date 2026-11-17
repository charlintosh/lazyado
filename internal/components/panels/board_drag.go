package panels

import (
	"fmt"
	"maps"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	boardEdgeDelay   = 500 * time.Millisecond
	boardDoubleClick = 400 * time.Millisecond
	edgeNoteRight    = "› scrolling to "
	edgeNoteLeft     = "‹ scrolling to "
)

type boardDrag struct {
	active  bool
	id      int
	from    string
	target  string
	reason  string
	pointer bool
	x       int
	y       int
	edge    int
}

type BoardEdgeScrollMsg struct {
	Seq int
}

type BoardFlashExpiredMsg struct {
	ID  int
	Seq int
}

type boardHit struct {
	col   int
	index int
	item  *models.WorkItem
}

func (b BoardPanel) Dragging() bool {
	return b.drag.active
}

func (b BoardPanel) dragMode() dragMode {
	d := b.drag
	switch {
	case !d.active:
		return dragNone
	case d.target == "" || d.reason != "":
		return dragInvalid
	case d.target == d.from:
		return dragSame
	}
	return dragValid
}

func (b BoardPanel) local(x, y int) (int, int, bool) {
	lx, ly := x-b.originX, y-b.originY
	return lx, ly, lx >= 0 && ly >= 0 && lx < b.width && ly < b.height
}

func (b BoardPanel) columnAt(lx int) (int, bool) {
	start, count, width := b.window()
	for k := range count {
		x0 := k * (width + styles.BoardColumnGap)
		if lx >= x0 && lx < x0+width {
			return start + k, true
		}
	}
	return 0, false
}

func (b BoardPanel) stateAt(x, y int) string {
	lx, _, inside := b.local(x, y)
	if !inside {
		return ""
	}
	if col, ok := b.columnAt(lx); ok && col < len(b.columns) {
		return b.columns[col].state
	}
	return ""
}

func (b BoardPanel) hit(x, y int) (boardHit, bool) {
	lx, ly, inside := b.local(x, y)
	if !inside {
		return boardHit{}, false
	}
	col, ok := b.columnAt(lx)
	if !ok || col >= len(b.columns) {
		return boardHit{}, false
	}
	c := b.columns[col]
	row := ly - 1 - styles.BoardColumnHeader
	stride := styles.BoardCardHeight + styles.BoardCardGap
	if row < 0 || row%stride >= styles.BoardCardHeight {
		return boardHit{col: col, index: -1}, true
	}
	j := row / stride
	index := c.top + j
	if j >= b.visibleCards() || index >= len(c.items) {
		return boardHit{col: col, index: -1}, true
	}
	return boardHit{col: col, index: index, item: c.items[index]}, true
}

func (b *BoardPanel) press(x, y int) tea.Cmd {
	hit, ok := b.hit(x, y)
	if !ok {
		return nil
	}
	b.col, b.hover, b.drag = hit.col, 0, boardDrag{}
	if hit.item == nil {
		b.ensureVisible()
		return nil
	}
	b.columns[hit.col].cursor = hit.index
	b.ensureVisible()
	now := time.Now()
	if b.lastClick == hit.item.ID && now.Sub(b.lastClickAt) < boardDoubleClick {
		b.lastClick = 0
		viewed := *hit.item
		return func() tea.Msg { return ViewWorkItemMsg{Item: viewed} }
	}
	b.lastClick, b.lastClickAt = hit.item.ID, now
	state := b.displayState(hit.item)
	b.drag = boardDrag{active: true, id: hit.item.ID, from: state, target: state, pointer: true, x: x, y: y}
	return nil
}

func (b *BoardPanel) pointerAt(x, y int) tea.Cmd {
	if !b.drag.active {
		b.hoverAt(x, y)
		return nil
	}
	b.drag.pointer, b.drag.x, b.drag.y = true, x, y
	b.retarget()
	return b.edgeAt(x)
}

func (b *BoardPanel) hoverAt(x, y int) {
	b.hover = 0
	if hit, ok := b.hit(x, y); ok && hit.item != nil {
		b.hover = hit.item.ID
	}
}

func (b *BoardPanel) retarget() {
	b.drag.target = b.stateAt(b.drag.x, b.drag.y)
	b.drag.reason = ""
	item := b.itemByID(b.drag.id)
	if item != nil && b.drag.target != "" && b.drag.target != b.drag.from && !b.allows(item, b.drag.target) {
		b.drag.reason = fmt.Sprintf("%s has no %s state", item.Type, b.drag.target)
	}
}

func (b *BoardPanel) edgeAt(x int) tea.Cmd {
	lx := x - b.originX
	start, count, _ := b.window()
	dir := 0
	switch {
	case lx < styles.BoardEdgeZone && start > 0:
		dir = -1
	case lx >= b.width-styles.BoardEdgeZone && start+count < len(b.columns):
		dir = 1
	}
	if dir == b.drag.edge {
		return nil
	}
	b.drag.edge = dir
	if dir == 0 {
		return nil
	}
	return b.scheduleEdge()
}

func (b *BoardPanel) scheduleEdge() tea.Cmd {
	b.edgeSeq++
	seq := b.edgeSeq
	return tea.Tick(boardEdgeDelay, func(time.Time) tea.Msg { return BoardEdgeScrollMsg{Seq: seq} })
}

func (b *BoardPanel) EdgeScroll(msg BoardEdgeScrollMsg) tea.Cmd {
	if !b.drag.active || msg.Seq != b.edgeSeq || b.drag.edge == 0 {
		return nil
	}
	start, count, _ := b.window()
	next := start + b.drag.edge
	if next < 0 || next+count > len(b.columns) {
		b.drag.edge = 0
		return nil
	}
	b.start = next
	b.retarget()
	return b.scheduleEdge()
}

func (b BoardPanel) edgeTarget() (string, string) {
	start, count, _ := b.window()
	switch {
	case b.drag.edge > 0 && start+count < len(b.columns):
		return edgeNoteRight, b.columns[start+count].state
	case b.drag.edge < 0 && start > 0:
		return edgeNoteLeft, b.columns[start-1].state
	}
	return "", ""
}

func (b *BoardPanel) drop(x, y int) tea.Cmd {
	if !b.drag.active {
		return nil
	}
	b.drag.x, b.drag.y = x, y
	b.retarget()
	d, mode := b.drag, b.dragMode()
	b.drag = boardDrag{}
	item := b.itemByID(d.id)
	if item == nil || mode != dragValid || d.target == b.displayState(item) {
		return nil
	}
	return b.beginMove(item, d.target)
}

func (b *BoardPanel) reconcileDrag() {
	if !b.drag.active {
		return
	}
	item := b.itemByID(b.drag.id)
	if item == nil {
		b.drag = boardDrag{}
		return
	}
	b.drag.from = b.displayState(item)
	b.retarget()
}

func (b BoardPanel) DragChip() (string, int, int, bool) {
	if !b.drag.active || !b.drag.pointer {
		return "", 0, 0, false
	}
	item := b.itemByID(b.drag.id)
	if item == nil {
		return "", 0, 0, false
	}
	color := styles.ColorError
	note := lipgloss.NewStyle().Foreground(styles.ColorError).Bold(true).Render(chipInvalidNote)
	if b.drag.reason != "" {
		note = lipgloss.NewStyle().Foreground(styles.ColorError).Bold(true).Render("✕ " + b.drag.reason)
	}
	switch b.dragMode() {
	case dragValid:
		color = styles.ColorPrimary
		note = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(chipMoveNote + b.drag.target)
	case dragSame:
		color = styles.ColorBorder
		note = Muted(chipSameNote + b.drag.from)
	}
	if prefix, state := b.edgeTarget(); state != "" {
		color = styles.ColorPrimary
		note = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(prefix + state)
	}
	inner := styles.DragChipWidth - styles.BoxPaddedSize
	head := Muted(gripGlyph) + " " + lipgloss.NewStyle().Foreground(styles.ColorLight).Render(strconv.Itoa(item.ID)) + " " +
		lipgloss.NewStyle().Foreground(styles.ColorText).Render(item.Title)
	chip := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1).
		Width(styles.DragChipWidth).
		Render(Line(head, inner) + "\n" + Line(note, inner))
	width := lipgloss.Width(chip)
	x := b.drag.x + 2
	if x+width > b.originX+b.width {
		x = max(b.drag.x-width-1, b.originX)
	}
	return chip, x, b.drag.y + 1, true
}

func (b *BoardPanel) setMove(id int, m taskMove) {
	next := maps.Clone(b.moves)
	if next == nil {
		next = map[int]taskMove{}
	}
	next[id] = m
	b.moves = next
}

func (b *BoardPanel) forgetMove(id int) {
	next := maps.Clone(b.moves)
	delete(next, id)
	b.moves = next
}

func (b *BoardPanel) beginMove(item *models.WorkItem, to string) tea.Cmd {
	if item == nil {
		return nil
	}
	b.setMove(item.ID, taskMove{to: to, from: b.displayState(item), target: to, flash: flashSaving})
	b.hover = 0
	moved := *item
	b.rebuild()
	b.selectByID(moved.ID)
	b.ensureVisible()
	return func() tea.Msg { return MoveWorkItemMsg{Item: moved, NewState: to} }
}

func (b BoardPanel) HasPendingMove(id int) bool {
	m, ok := b.moves[id]
	return ok && m.flash == flashSaving
}

func (b *BoardPanel) MoveSaved(id int) tea.Cmd {
	return b.settle(id, flashSaved, false)
}

func (b *BoardPanel) MoveFailed(id int) tea.Cmd {
	return b.settle(id, flashFailed, true)
}

func (b *BoardPanel) settle(id int, flash moveFlash, revert bool) tea.Cmd {
	m, ok := b.moves[id]
	if !ok {
		return nil
	}
	b.flashSeq++
	m.flash, m.seq = flash, b.flashSeq
	if revert {
		m.to = ""
	}
	b.setMove(id, m)
	b.rebuild()
	seq := m.seq
	return tea.Tick(moveFlashDuration, func(time.Time) tea.Msg { return BoardFlashExpiredMsg{ID: id, Seq: seq} })
}

func (b *BoardPanel) ExpireFlash(msg BoardFlashExpiredMsg) {
	m, ok := b.moves[msg.ID]
	if !ok || m.seq != msg.Seq || m.flash == flashSaving {
		return
	}
	if item, found := findTask(b.items, msg.ID); m.to == "" || !found || string(item.State) == m.to {
		b.forgetMove(msg.ID)
	} else {
		m.flash = flashNone
		b.setMove(msg.ID, m)
	}
	b.rebuild()
}

func (b *BoardPanel) settleMoves() {
	if len(b.moves) == 0 {
		return
	}
	next := make(map[int]taskMove, len(b.moves))
	for id, m := range b.moves {
		item, ok := findTask(b.items, id)
		if !ok || (m.flash == flashNone && string(item.State) == m.to) {
			continue
		}
		next[id] = m
	}
	b.moves = next
}
