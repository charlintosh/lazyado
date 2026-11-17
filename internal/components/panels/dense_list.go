package panels

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const flexColumnMin = 16

type column struct {
	title string
	width int
}

type denseList struct {
	viewport viewport.Model
	cursor   int
	count    int
	focused  bool
	width    int
	height   int
}

func newDenseList(k keys.KeyMap) denseList {
	return denseList{viewport: widgets.NewViewport(k)}
}

func (l *denseList) setCount(n int) {
	l.count = n
	l.cursor = min(max(l.cursor, 0), max(n-1, 0))
}

func (l *denseList) setSize(width, height int) {
	l.width, l.height = width, height
	l.viewport.SetWidth(max(width, 1))
	l.viewport.SetHeight(max(height, 1))
}

func (l *denseList) move(delta int) bool {
	target := min(max(l.cursor+delta, 0), max(l.count-1, 0))
	if target == l.cursor {
		return false
	}
	l.cursor = target
	return true
}

func (l *denseList) atEnd() bool { return l.count > 0 && l.cursor == l.count-1 }

func (l *denseList) handleMove(msg tea.Msg, k keys.KeyMap) (bool, bool) {
	page := max(l.height/2, 1)
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, k.Up):
			return l.move(-1), true
		case key.Matches(msg, k.Down):
			return l.move(1), true
		case key.Matches(msg, k.HalfPageUp):
			return l.move(-page), true
		case key.Matches(msg, k.HalfPageDown):
			return l.move(page), true
		case key.Matches(msg, k.Top):
			return l.move(-l.count), true
		case key.Matches(msg, k.Bottom):
			return l.move(l.count), true
		}
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			return l.move(-1), true
		}
		return l.move(1), true
	}
	return false, false
}

func (l *denseList) render(rows []string, footer string) {
	if l.width <= 0 {
		return
	}
	if footer != "" {
		rows = append(rows, "", footer)
	}
	l.viewport.SetContent(strings.Join(rows, "\n"))
	widgets.Reveal(&l.viewport, l.cursor, l.cursor)
}

func (l *denseList) rowStyle(i int) (lipgloss.Style, string) {
	base := lipgloss.NewStyle()
	bar := " "
	if i == l.cursor {
		base = base.Background(styles.ColorHighlight)
		if l.focused {
			bar = railBar
		}
	}
	return base, base.Foreground(styles.ColorPrimary).Render(bar) + base.Render(" ")
}

func (l *denseList) view() string {
	return Block(l.viewport.View(), l.width, l.height)
}

func cellText(s string, width int, st lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	return st.Render(Line(ansi.Truncate(s, width-1, "…"), width))
}

func columnHeader(cols []column) string {
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	var b strings.Builder
	b.WriteString("  ")
	for _, c := range cols {
		b.WriteString(cellText(c.title, c.width, muted))
	}
	return b.String()
}

func fitColumns(width, flex int, cols ...column) []column {
	out := slices.Clone(cols)
	for {
		used := 2
		for i, c := range out {
			if i != flex {
				used += c.width
			}
		}
		room := width - used
		if room >= flexColumnMin || len(out) <= flex+1 {
			out[flex].width = max(room, 1)
			return out
		}
		out = out[:len(out)-1]
	}
}
