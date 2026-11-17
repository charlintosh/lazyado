package panels

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

type codeView struct {
	viewport viewport.Model
	keys     keys.KeyMap
	lines    []components.CodeLine
	rows     []string
	cursor   int
	focused  bool
	width    int
	height   int
}

func newCodeView(k keys.KeyMap) codeView {
	vp := widgets.NewViewport(k)
	vp.KeyMap = viewport.KeyMap{}
	return codeView{viewport: vp, keys: k}
}

func (v *codeView) setLines(lines []components.CodeLine) {
	v.lines, v.cursor = lines, 0
	v.render()
	v.viewport.GotoTop()
}

func (v *codeView) line() int { return v.cursor + 1 }

func (v *codeView) setFocused(focused bool) {
	if v.focused == focused {
		return
	}
	v.focused = focused
	v.renderRow(v.cursor)
	v.commit()
}

func (v *codeView) setSize(width, height int) {
	v.width, v.height = width, height
	v.viewport.SetWidth(max(width, 1))
	v.viewport.SetHeight(max(height, 1))
	v.render()
}

func (v *codeView) render() {
	if v.width <= 0 {
		return
	}
	v.rows = make([]string, len(v.lines))
	for i := range v.lines {
		v.renderRow(i)
	}
	v.commit()
}

func (v *codeView) renderRow(i int) {
	if i < 0 || i >= len(v.rows) {
		return
	}
	base := lipgloss.NewStyle()
	bar := " "
	if i == v.cursor {
		base = base.Background(styles.ColorHighlight)
		if v.focused {
			bar = railBar
		}
	}
	lead := base.Foreground(styles.ColorPrimary).Render(bar)
	gutter := base.Foreground(styles.ColorTextMuted).Render(lineNumber(i+1) + "  ")
	room := max(v.width-1-styles.DiffGutterWidth-2, 1)
	v.rows[i] = lead + gutter + v.lines[i].Render(base.Foreground(styles.ColorText), room)
}

func (v *codeView) commit() {
	v.viewport.SetContent(strings.Join(v.rows, "\n"))
	widgets.Reveal(&v.viewport, v.cursor, v.cursor)
}

func (v *codeView) moveTo(target int) {
	target = min(max(target, 0), max(len(v.lines)-1, 0))
	if target == v.cursor {
		return
	}
	old := v.cursor
	v.cursor = target
	v.renderRow(old)
	v.renderRow(target)
	v.commit()
}

func (v codeView) update(msg tea.Msg) codeView {
	k := v.keys
	page := max(v.height/2, 1)
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, k.Up):
			v.moveTo(v.cursor - 1)
		case key.Matches(msg, k.Down):
			v.moveTo(v.cursor + 1)
		case key.Matches(msg, k.HalfPageUp):
			v.moveTo(v.cursor - page)
		case key.Matches(msg, k.HalfPageDown):
			v.moveTo(v.cursor + page)
		case key.Matches(msg, k.Top):
			v.moveTo(0)
		case key.Matches(msg, k.Bottom):
			v.moveTo(len(v.lines) - 1)
		}
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			v.moveTo(v.cursor - 1)
		} else {
			v.moveTo(v.cursor + 1)
		}
	}
	return v
}

func (v *codeView) view() string {
	return Block(v.viewport.View(), v.width, v.height)
}
