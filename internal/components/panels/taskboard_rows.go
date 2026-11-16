package panels

import (
	"fmt"
	"maps"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	chevronOpen    = "▾"
	chevronClosed  = "▸"
	collapseGlyph  = "⊟ Collapse all"
	expandGlyph    = "⊞ Expand all"
	taskCountGlyph = "▪ "
	allDoneGlyph   = "✓ "
	chevronFrom    = 2
	chevronTo      = 6
)

func (t TaskboardPanel) laneDone(l taskLane) bool {
	if models.IsCompletedState(string(l.parent.State)) {
		return true
	}
	tasks := 0
	for ci, col := range l.cols {
		tasks += len(col)
		if len(col) > 0 && !models.IsCompletedState(t.states[ci]) {
			return false
		}
	}
	return tasks > 0
}

func (t *TaskboardPanel) seedCollapsed() {
	var next map[int]bool
	for _, l := range t.lanes {
		if _, seen := t.collapsed[l.parent.ID]; seen {
			continue
		}
		if next == nil {
			next = maps.Clone(t.collapsed)
			if next == nil {
				next = map[int]bool{}
			}
		}
		next[l.parent.ID] = t.laneDone(l)
	}
	if next != nil {
		t.collapsed = next
	}
}

func (t TaskboardPanel) isCollapsed(li int) bool {
	return li >= 0 && li < len(t.lanes) && t.collapsed[t.lanes[li].parent.ID]
}

func (t TaskboardPanel) collapsedCount() int {
	n := 0
	for li := range t.lanes {
		if t.isCollapsed(li) {
			n++
		}
	}
	return n
}

func (t TaskboardPanel) anyExpanded() bool {
	return t.collapsedCount() < len(t.lanes)
}

func (t *TaskboardPanel) setCollapsed(li int, collapsed bool) {
	if li < 0 || li >= len(t.lanes) || t.isCollapsed(li) == collapsed {
		return
	}
	next := maps.Clone(t.collapsed)
	if next == nil {
		next = map[int]bool{}
	}
	next[t.lanes[li].parent.ID] = collapsed
	t.collapsed = next
	t.keepSelectionVisible()
	t.render()
}

func (t *TaskboardPanel) toggleAllRows() {
	collapse := t.anyExpanded()
	next := maps.Clone(t.collapsed)
	if next == nil {
		next = map[int]bool{}
	}
	for _, l := range t.lanes {
		next[l.parent.ID] = collapse
	}
	t.collapsed = next
	t.keepSelectionVisible()
	t.render()
}

func (t *TaskboardPanel) keepSelectionVisible() {
	if t.isCollapsed(t.lane) {
		t.cell = 0
	}
}

func (t TaskboardPanel) rowsLabel() string {
	label := collapseGlyph
	if !t.anyExpanded() {
		label = expandGlyph
	}
	return label + "  " + t.keys.ToggleRows.Help().Key
}

func (t TaskboardPanel) laneSummary(l taskLane) (int, int) {
	done, total := 0, 0
	for ci, col := range l.cols {
		total += len(col)
		if models.IsCompletedState(t.states[ci]) {
			done += len(col)
		}
	}
	return done, total
}

func (t TaskboardPanel) renderCollapsedParent(l taskLane, width int, selected bool) string {
	p := l.parent
	done, total := t.laneSummary(l)
	summary := Muted(fmt.Sprintf("%d/%d", done, total))
	switch {
	case total == 0:
		summary = StateText(string(p.State))
	case done == total:
		summary = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(fmt.Sprintf("%s%d/%d", allDoneGlyph, done, total))
	}
	id := lipgloss.NewStyle().Foreground(styles.ColorLight).Render(fmt.Sprintf("%d", p.ID))
	edge := styles.TypeColor(string(p.Type))
	if selected && t.focused {
		id = Badge(fmt.Sprintf("%d", p.ID), styles.ColorPrimary, styles.ColorOnAccent)
		edge = styles.ColorPrimary
	}
	inner := max(width-styles.CardChromeWidth, 1)
	head := Muted(chevronClosed) + " " + id + " "
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(p.Title)
	room := max(inner-lipgloss.Width(head)-lipgloss.Width(summary)-1, 1)
	line := head + Line(title, room) + " " + summary
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(edge).
		PaddingLeft(1).
		Width(width).
		Render(Line(line, inner))
}

func (t TaskboardPanel) renderTaskCount(l taskLane, ci, width int, tinted bool) string {
	count := ""
	if n := len(l.cols[ci]); n > 0 {
		color := styles.ColorLight
		if models.IsCompletedState(t.states[ci]) {
			color = styles.ColorSuccess
		}
		count = lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%s%d", taskCountGlyph, n))
	}
	fill := lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true)
	if tinted {
		fill = fill.Background(styles.ColorDropTint)
	}
	return fill.Render(count)
}
