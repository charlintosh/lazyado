package panels

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const noColumn = -1

type dragMode int

const (
	dragNone dragMode = iota
	dragValid
	dragSame
	dragInvalid
)

const (
	chipMoveNote    = "→ "
	chipSameNote    = "· already in "
	chipInvalidNote = "✕ not a state column"
)

type taskDrag struct {
	active  bool
	taskID  int
	lane    int
	from    string
	target  string
	pointer bool
	x       int
	y       int
}

type boardCell struct {
	lane int
	cell int
	col  int
	row  int
	item *models.WorkItem
}

func (t *TaskboardPanel) SetOrigin(x, y int) {
	t.originX, t.originY = x, y
}

func (t TaskboardPanel) Dragging() bool {
	return t.drag.active
}

func (t TaskboardPanel) dragMode() dragMode {
	d := t.drag
	switch {
	case !d.active:
		return dragNone
	case d.target == "":
		return dragInvalid
	case d.target == d.from:
		return dragSame
	}
	return dragValid
}

func (t TaskboardPanel) dragTarget() (string, bool) {
	return t.drag.target, t.dragMode() == dragValid
}

func (t TaskboardPanel) local(x, y int) (int, int, bool) {
	lx, ly := x-t.originX, y-t.originY
	return lx, ly, lx >= 0 && ly >= 0 && lx < t.width && ly < t.height
}

func (t TaskboardPanel) columnAt(lx int) (int, bool) {
	start, count := t.stateWindow()
	laneWidth, colWidth := t.columnWidths(count)
	x := 1 + laneWidth + styles.TaskCellPadding
	if lx < x {
		return noColumn, lx >= 1
	}
	for i := range count {
		x++
		if lx >= x && lx < x+colWidth+styles.TaskCellPadding {
			return start + i, true
		}
		x += colWidth + styles.TaskCellPadding
	}
	return noColumn, false
}

func (t TaskboardPanel) columnSpan(state string) (int, int, bool) {
	start, count := t.stateWindow()
	laneWidth, colWidth := t.columnWidths(count)
	x := 1 + laneWidth + styles.TaskCellPadding
	for i := range count {
		x++
		if start+i < len(t.states) && t.states[start+i] == state {
			return t.originX + x, t.originX + x + colWidth + styles.TaskCellPadding, true
		}
		x += colWidth + styles.TaskCellPadding
	}
	return 0, 0, false
}

func (t TaskboardPanel) chipX(width int) int {
	right := t.originX + t.width
	if target, ok := t.dragTarget(); ok {
		if left, end, found := t.columnSpan(target); found {
			if end+1+width <= right {
				return end + 1
			}
			if left-1-width >= t.originX {
				return left - 1 - width
			}
		}
	}
	if x := t.drag.x + 2; x+width <= right {
		return x
	}
	return max(t.drag.x-width-1, t.originX)
}

func (t TaskboardPanel) stateAt(x, y int) string {
	lx, _, inside := t.local(x, y)
	if !inside || len(t.lanes) == 0 {
		return ""
	}
	if col, ok := t.columnAt(lx); ok && col != noColumn {
		return t.states[col]
	}
	return ""
}

func (t TaskboardPanel) cellAt(x, y int) (boardCell, bool) {
	lx, ly, inside := t.local(x, y)
	if !inside {
		return boardCell{}, false
	}
	col, ok := t.columnAt(lx)
	if !ok {
		return boardCell{}, false
	}
	row := ly + t.viewport.YOffset()
	top := styles.TaskTableHeaderLines
	for li := range t.lanes {
		height := t.laneHeight(li)
		if row < top {
			return boardCell{}, false
		}
		if row < top+height {
			if t.isCollapsed(li) && col != noColumn {
				return boardCell{}, false
			}
			return t.lanes[li].cellAtRow(li, col, row-top)
		}
		top += height + 1
	}
	return boardCell{}, false
}

func (l *taskLane) cellAtRow(lane, col, offset int) (boardCell, bool) {
	if col == noColumn {
		return boardCell{lane: lane, col: noColumn, row: offset, item: l.parent}, offset < styles.TaskParentHeight
	}
	stride := styles.TaskCardHeight + styles.TaskCardGap
	card := offset / stride
	if offset%stride >= styles.TaskCardHeight || card >= len(l.cols[col]) {
		return boardCell{}, false
	}
	cell := 1 + card
	for _, c := range l.cols[:col] {
		cell += len(c)
	}
	return boardCell{lane: lane, cell: cell, col: col, item: l.cols[col][card]}, true
}

func (t TaskboardPanel) taskByID(id int) (*models.WorkItem, bool) {
	for _, l := range t.lanes {
		for _, col := range l.cols {
			for _, task := range col {
				if task.ID == id {
					return task, true
				}
			}
		}
	}
	return nil, false
}

func (t TaskboardPanel) laneOf(id int) int {
	for li, l := range t.lanes {
		for _, col := range l.cols {
			for _, task := range col {
				if task.ID == id {
					return li
				}
			}
		}
	}
	return -1
}

func (t *TaskboardPanel) pointerAt(x, y int) {
	if !t.drag.active {
		t.hoverAt(x, y)
		return
	}
	t.drag.pointer, t.drag.x, t.drag.y = true, x, y
	if target := t.stateAt(x, y); target != t.drag.target {
		t.drag.target = target
		t.render()
	}
}

func (t *TaskboardPanel) hoverAt(x, y int) {
	id := 0
	if hit, ok := t.cellAt(x, y); ok && hit.col != noColumn {
		id = hit.item.ID
	}
	if id != t.hover {
		t.hover = id
		t.render()
	}
}

func (t TaskboardPanel) onRowsHeader(x, y int) bool {
	lx, ly, inside := t.local(x, y)
	if !inside || ly+t.viewport.YOffset() != 1 {
		return false
	}
	col, ok := t.columnAt(lx)
	return ok && col == noColumn
}

func (t TaskboardPanel) onChevron(x int) bool {
	lx := x - t.originX
	return lx >= chevronFrom && lx < chevronTo
}

func (t *TaskboardPanel) press(x, y int) {
	if t.onRowsHeader(x, y) {
		t.toggleAllRows()
		return
	}
	hit, ok := t.cellAt(x, y)
	if !ok {
		return
	}
	t.lane, t.cell = hit.lane, hit.cell
	t.drag, t.hover = taskDrag{}, 0
	if hit.col == noColumn {
		switch {
		case t.isCollapsed(hit.lane):
			t.setCollapsed(hit.lane, false)
		case hit.row == 0 && t.onChevron(x):
			t.setCollapsed(hit.lane, true)
		}
		t.render()
		return
	}
	if hit.col != noColumn {
		state := t.displayState(hit.item)
		t.drag = taskDrag{active: true, taskID: hit.item.ID, lane: hit.lane, from: state, target: state, pointer: true, x: x, y: y}
	}
	t.render()
}

func (t *TaskboardPanel) drop(x, y int) tea.Cmd {
	if !t.drag.active {
		return nil
	}
	id, state := t.drag.taskID, t.stateAt(x, y)
	t.drag = taskDrag{}
	task, ok := t.taskByID(id)
	if !ok || state == "" || state == t.displayState(task) {
		t.render()
		return nil
	}
	return t.beginMove(task, state)
}

func (t *TaskboardPanel) cancelDrag() {
	t.drag = taskDrag{}
	t.render()
}

func (t *TaskboardPanel) reconcileDrag() {
	if !t.drag.active {
		return
	}
	task, ok := t.taskByID(t.drag.taskID)
	if !ok {
		t.drag = taskDrag{}
		return
	}
	t.drag.from = t.displayState(task)
	t.drag.lane = t.laneOf(task.ID)
}

func (t TaskboardPanel) DragChip() (string, int, int, bool) {
	if !t.drag.active || !t.drag.pointer {
		return "", 0, 0, false
	}
	task, ok := t.taskByID(t.drag.taskID)
	if !ok {
		return "", 0, 0, false
	}
	color, note := styles.ColorError, lipgloss.NewStyle().Foreground(styles.ColorError).Bold(true).Render(chipInvalidNote)
	switch t.dragMode() {
	case dragValid:
		color = styles.ColorPrimary
		note = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(chipMoveNote + t.drag.target)
	case dragSame:
		color = styles.ColorBorder
		note = Muted(chipSameNote + t.drag.from)
	}
	inner := styles.DragChipWidth - styles.BoxPaddedSize
	head := Muted(gripGlyph) + " " + lipgloss.NewStyle().Foreground(styles.ColorLight).Render(strconv.Itoa(task.ID)) + " " +
		lipgloss.NewStyle().Foreground(styles.ColorText).Render(task.Title)
	chip := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1).
		Width(styles.DragChipWidth).
		Render(Line(head, inner) + "\n" + Line(note, inner))
	return chip, t.chipX(lipgloss.Width(chip)), t.drag.y + 1, true
}
