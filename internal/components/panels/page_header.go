package panels

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	headerGap      = "   "
	headerPivotY   = 1
	headerBarY     = 3
	maxFullBarRows = 2
	ruleGlyph      = "─"
	underline      = "━"
)

type HeaderControl struct {
	Prefix string
	Label  string
	Short  string
	Key    key.Binding
	Open   bool
	Set    bool
	Group  int
}

type HeaderAnchor struct {
	Group int
	X     int
	Y     int
}

type PageHeaderProps struct {
	Width      int
	Title      string
	Action     HeaderControl
	Pivots     []string
	Pivot      int
	Controls   []HeaderControl
	ShowBar    bool
	Keyword    string
	KeywordKey key.Binding
	Bar        []HeaderControl
	Clear      HeaderControl
}

type PageHeader struct {
	View    string
	Height  int
	Anchors []HeaderAnchor
}

func RenderPageHeader(p PageHeaderProps) PageHeader {
	pivots, anchors := p.pivotRow()
	lines := []string{p.titleRow(), pivots, p.ruleRow()}
	if p.ShowBar {
		bar, barAnchors := p.barRows()
		lines = append(lines, bar...)
		anchors = append(anchors, barAnchors...)
	}
	return PageHeader{View: strings.Join(lines, "\n"), Height: len(lines), Anchors: anchors}
}

func (h PageHeader) Anchor(group int) (HeaderAnchor, bool) {
	for _, a := range h.Anchors {
		if a.Group == group {
			return a, true
		}
	}
	return HeaderAnchor{}, false
}

func (c HeaderControl) view(short bool) string {
	label, prefix := c.Label, c.Prefix
	if short {
		prefix = ""
		if c.Short != "" {
			label = c.Short
		}
	}
	bg := styles.ColorChip
	if c.Open {
		bg = styles.ColorHighlight
	}
	fg := styles.ColorText
	if c.Set {
		fg = styles.ColorPrimary
	}
	text := ""
	if prefix != "" {
		text = lipgloss.NewStyle().Background(bg).Foreground(styles.ColorTextMuted).Render(prefix)
	}
	text += lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(c.Set).Render(label)
	chip := lipgloss.NewStyle().Background(bg).Padding(0, 1).Render(text)
	return chip + " " + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(c.Key.Help().Key)
}

func joinControls(controls []HeaderControl, short bool) (string, []int) {
	out := ""
	offsets := make([]int, len(controls))
	for i, c := range controls {
		if i > 0 {
			out += headerGap
		}
		offsets[i] = lipgloss.Width(out)
		out += c.view(short)
	}
	return out, offsets
}

func fits(width int, parts ...string) bool {
	total := 2
	for _, p := range parts {
		total += lipgloss.Width(p)
	}
	return total < width
}

func spread(width int, left, right string) string {
	gap := max(width-2-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return Line(" "+left+strings.Repeat(" ", gap)+right+" ", width)
}

func (p PageHeaderProps) titleRow() string {
	action := p.Action.view(false)
	if !fits(p.Width, p.Title, " ", action) {
		action = ""
	}
	return spread(p.Width, p.Title, action)
}

func (p PageHeaderProps) pivotLabels() []string {
	out := make([]string, len(p.Pivots))
	for i, label := range p.Pivots {
		st := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if i == p.Pivot {
			st = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
		}
		out[i] = st.Render(label)
	}
	return out
}

func (p PageHeaderProps) pivotRow() (string, []HeaderAnchor) {
	pivots := strings.Join(p.pivotLabels(), headerGap)
	controls := p.Controls
	for {
		for _, short := range []bool{false, true} {
			right, offsets := joinControls(controls, short)
			if fits(p.Width, pivots, " ", right) {
				x := p.Width - 1 - lipgloss.Width(right)
				return spread(p.Width, pivots, right), anchorsFor(controls, offsets, x, headerPivotY)
			}
		}
		if len(controls) == 0 {
			return spread(p.Width, pivots, ""), nil
		}
		controls = controls[:len(controls)-1]
	}
}

func PivotTabs(labels []string, active, width int) (string, string) {
	p := PageHeaderProps{Width: width, Pivots: labels, Pivot: active}
	return spread(width, strings.Join(p.pivotLabels(), headerGap), ""), p.ruleRow()
}

func (p PageHeaderProps) ruleRow() string {
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder)
	if p.Pivot < 0 || p.Pivot >= len(p.Pivots) {
		return rule.Render(strings.Repeat(ruleGlyph, max(p.Width, 0)))
	}
	start := 1
	for _, label := range p.Pivots[:p.Pivot] {
		start += lipgloss.Width(label) + len(headerGap)
	}
	start = min(start, p.Width)
	length := min(lipgloss.Width(p.Pivots[p.Pivot]), p.Width-start)
	return rule.Render(strings.Repeat(ruleGlyph, start)) +
		lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(strings.Repeat(underline, length)) +
		rule.Render(strings.Repeat(ruleGlyph, max(p.Width-start-length, 0)))
}

func (p PageHeaderProps) barRows() ([]string, []HeaderAnchor) {
	rows, anchors := p.flowBar(false)
	if len(rows) > maxFullBarRows {
		if short, shortAnchors := p.flowBar(true); len(short) < len(rows) {
			return short, shortAnchors
		}
	}
	return rows, anchors
}

func (p PageHeaderProps) flowBar(short bool) ([]string, []HeaderAnchor) {
	var rows []string
	var anchors []HeaderAnchor
	row := p.Keyword + " " + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(p.KeywordKey.Help().Key)
	for _, c := range p.Bar {
		view := c.view(short)
		if lipgloss.Width(view)+2 > p.Width {
			view = c.view(true)
		}
		if row != "" && !fits(p.Width, row, headerGap, view) {
			rows = append(rows, row)
			row = ""
		}
		if row != "" {
			row += headerGap
		}
		if c.Group >= 0 {
			anchors = append(anchors, HeaderAnchor{Group: c.Group, X: 1 + lipgloss.Width(row), Y: headerBarY + len(rows)})
		}
		row += view
	}
	clear := p.Clear.view(short)
	if !fits(p.Width, row, " ", clear) {
		rows = append(rows, row)
		row = ""
	}
	rows = append(rows, row)
	out := make([]string, len(rows))
	for i, r := range rows {
		right := ""
		if i == len(rows)-1 {
			right = clear
		}
		out[i] = spread(p.Width, r, right)
	}
	return out, anchors
}

func anchorsFor(controls []HeaderControl, offsets []int, x, y int) []HeaderAnchor {
	var out []HeaderAnchor
	for i, c := range controls {
		if c.Group >= 0 {
			out = append(out, HeaderAnchor{Group: c.Group, X: x + offsets[i], Y: y})
		}
	}
	return out
}
