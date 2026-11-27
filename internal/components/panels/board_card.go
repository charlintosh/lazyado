package panels

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	cardEdge       = "┃"
	unassignedNote = "Unassigned"
)

var cardTypeLabels = map[models.WorkItemType]string{
	models.WorkItemTypePBI:   "PBI",
	models.WorkItemTypeStory: "STORY",
	models.WorkItemTypeBug:   "BUG",
}

func cardTypeLabel(t models.WorkItemType) string {
	if l, ok := cardTypeLabels[t]; ok {
		return l
	}
	return strings.ToUpper(string(t))
}

type cardPalette struct {
	base   lipgloss.Style
	text   lipgloss.Style
	muted  lipgloss.Style
	id     lipgloss.Style
	kind   lipgloss.Style
	edge   lipgloss.Style
	glyph  string
	note   string
	noteSt lipgloss.Style
	grip   bool
	plain  bool
}

func (b BoardPanel) cardPalette(item *models.WorkItem, look cardLook, selected bool) cardPalette {
	bg := styles.ColorSurface
	switch {
	case look == lookPhantom:
		bg = styles.ColorPhantom
	case selected && b.focused:
		bg = styles.ColorHighlight
	case selected:
		bg = styles.ColorSurfaceAlt
	case look == lookHover:
		bg = styles.ColorChip
	}
	base := lipgloss.NewStyle().Background(bg)
	p := cardPalette{
		base:  base,
		text:  base.Foreground(styles.ColorText),
		muted: base.Foreground(styles.ColorTextMuted),
		id:    base.Foreground(styles.ColorLight),
		kind:  base.Foreground(styles.TypeColor(string(item.Type))),
		edge:  base.Foreground(styles.TypeColor(string(item.Type))),
		glyph: cardEdge,
		grip:  look == lookHover,
	}
	if selected && b.focused {
		p.id = base.Foreground(styles.ColorPrimary).Bold(true)
	}
	m := b.moves[item.ID]
	switch look {
	case lookGhost:
		dim := lipgloss.NewStyle().Foreground(styles.ColorMuted)
		p.base, p.text, p.muted, p.id, p.kind = lipgloss.NewStyle(), dim, dim, dim, dim
		p.edge, p.glyph = lipgloss.NewStyle().Foreground(styles.ColorTextMuted), ghostBorder.Left
		p.note, p.noteSt, p.plain = ghostNote, dim, true
	case lookPhantom:
		light := base.Foreground(styles.ColorPrimaryLight)
		p.text, p.id, p.kind = light, light, light
		p.edge, p.glyph = base.Foreground(styles.ColorPrimary), phantomBorder.Left
		p.note, p.noteSt = phantomNote+b.drag.target, base.Foreground(styles.ColorPrimary).Bold(true)
	case lookSaving:
		p.edge = base.Foreground(styles.ColorPrimary)
		p.note, p.noteSt = savingNote+m.target+"…", base.Foreground(styles.ColorPrimary).Bold(true)
	case lookSaved:
		p.edge = base.Foreground(styles.ColorSuccess)
		p.note, p.noteSt = savedNote+m.target, base.Foreground(styles.ColorSuccess).Bold(true)
	case lookFailed:
		p.edge = base.Foreground(styles.ColorError)
		p.note, p.noteSt = failedNote+m.from, base.Foreground(styles.ColorError).Bold(true)
	}
	return p
}

func (b BoardPanel) renderCard(item *models.WorkItem, width int, look cardLook, selected bool) []string {
	p := b.cardPalette(item, look, selected)
	inner := max(width-styles.BoardCardChrome, 1)
	first, second := cardTitle(item.Title, inner)
	rows := []string{
		cardHeader(item, p, inner),
		p.text.Bold(!p.plain).Render(first),
		p.text.Bold(!p.plain).Render(second),
		cardTags(item.Tags, p, inner),
		cardFooter(item, p, inner),
	}
	pad := p.base.Render(" ")
	for i, r := range rows {
		rows[i] = p.edge.Render(p.glyph) + pad + fit(r, inner, p.base) + pad
	}
	return rows
}

func cardHeader(item *models.WorkItem, p cardPalette, inner int) string {
	left := p.id.Render(strconv.Itoa(item.ID)) + p.base.Render(" ") + p.kind.Render(cardTypeLabel(item.Type))
	if p.grip {
		left = p.muted.Render(gripGlyph+" ") + left
	}
	right := ""
	if item.Effort > 0 {
		right = p.muted.Render("Effort ") + p.text.Render(formatEffort(item.Effort))
	}
	return spreadExact(left, right, inner, p.base)
}

func cardTitle(title string, inner int) (string, string) {
	lines := strings.Split(ansi.Wrap(title, inner, ""), "\n")
	switch len(lines) {
	case 0:
		return "", ""
	case 1:
		return lines[0], ""
	case 2:
		return lines[0], lines[1]
	}
	return lines[0], ansi.Truncate(strings.Join(lines[1:], " "), inner, "…")
}

func cardTags(tags []string, p cardPalette, inner int) string {
	if len(tags) == 0 {
		return ""
	}
	chip := lipgloss.NewStyle().Background(styles.ColorChip).Foreground(styles.ColorOffWhite).Padding(0, 1)
	if p.plain {
		chip = p.muted
	}
	for shown := min(len(tags), styles.BoardTagChips); shown >= 0; shown-- {
		parts := make([]string, 0, shown+1)
		for _, t := range tags[:shown] {
			parts = append(parts, chip.Render(t))
		}
		if rest := len(tags) - shown; rest > 0 {
			parts = append(parts, p.muted.Render(fmt.Sprintf("+%d", rest)))
		}
		line := strings.Join(parts, p.base.Render(" "))
		if lipgloss.Width(line) <= inner {
			return line
		}
	}
	return ""
}

func cardFooter(item *models.WorkItem, p cardPalette, inner int) string {
	if p.note != "" {
		return p.noteSt.Render(p.note)
	}
	left := p.muted.Render(unassignedNote)
	if item.AssignedTo != "" {
		left = initialsBadge(item.AssignedTo, p.plain) + p.base.Render(" ") + p.text.Render(item.AssignedTo)
	}
	right := ""
	if len(item.Children) > 0 {
		done, total := DoneCount(item.Children)
		right = p.muted.Render("Tasks ") + p.text.Render(fmt.Sprintf("%d/%d", done, total))
	}
	return spreadExact(left, right, inner, p.base)
}

func initialsBadge(name string, plain bool) string {
	bg := styles.AvatarColor(name)
	if plain {
		bg = styles.ColorChip
	}
	return lipgloss.NewStyle().Background(bg).Foreground(styles.ColorOnAccent).Bold(true).Padding(0, 1).Render(widgets.Initials(name))
}

func spreadExact(left, right string, width int, base lipgloss.Style) string {
	room := width - lipgloss.Width(left) - lipgloss.Width(right)
	if right == "" || room < 1 {
		return left
	}
	return left + base.Render(strings.Repeat(" ", room)) + right
}

func fit(s string, width int, base lipgloss.Style) string {
	if w := lipgloss.Width(s); w > width {
		s = ansi.Truncate(s, width, "…")
	}
	if gap := width - lipgloss.Width(s); gap > 0 {
		s += base.Render(strings.Repeat(" ", gap))
	}
	return s
}
