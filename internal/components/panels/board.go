package panels

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type cardItem struct {
	item *models.WorkItem
}

func (c cardItem) FilterValue() string { return c.item.Title }

type cardDelegate struct {
	active  bool
	focused bool
}

func (d cardDelegate) Height() int                             { return styles.BoardCardHeight }
func (d cardDelegate) Spacing() int                            { return styles.BoardCardGap }
func (d cardDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d cardDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	c, ok := li.(cardItem)
	if !ok {
		return
	}
	selected := index == m.Index() && d.active
	_, _ = io.WriteString(w, RenderCard(c.item, m.Width(), selected, selected && d.focused))
}

func RenderCard(item *models.WorkItem, width int, selected, focused bool) string {
	bg := styles.ColorSurface
	if selected {
		bg = styles.ColorSurfaceAlt
	}
	if focused {
		bg = styles.ColorHighlight
	}
	inner := max(width-styles.CardChromeWidth, 1)
	text := lipgloss.NewStyle().Background(bg).Foreground(styles.ColorText)
	muted := text.Foreground(styles.ColorTextMuted)

	idStyle := text.Foreground(styles.ColorLight)
	if focused {
		idStyle = idStyle.Foreground(styles.ColorPrimary).Bold(true)
	}
	title := text.Bold(true).Width(inner).MaxHeight(2).Render(idStyle.Render(fmt.Sprintf("%d ", item.ID)) + text.Bold(true).Render(item.Title))
	if lipgloss.Height(title) < 2 {
		title = lipgloss.JoinVertical(lipgloss.Left, title, text.Width(inner).Render(""))
	}

	assignee := text.Render(item.AssignedTo)
	if item.AssignedTo == "" {
		assignee = muted.Render("Unassigned")
	}
	effort := ""
	if item.Effort > 0 {
		effort = muted.Render("Effort ") + text.Render(formatEffort(item.Effort))
	}
	people := lipgloss.JoinHorizontal(lipgloss.Top,
		text.Width(max(inner-lipgloss.Width(effort), 1)).Render(assignee),
		effort,
	)

	meta := []string{text.Foreground(styles.StateColor(string(item.State))).Render(string(item.State))}
	if len(item.Children) > 0 {
		done, total := DoneCount(item.Children)
		meta = append(meta, muted.Render(fmt.Sprintf("Tasks %d/%d", done, total)))
	}
	if len(item.Tags) > 0 {
		meta = append(meta, Chip(item.Tags[0]))
	}
	metaLine := text.Width(inner).MaxWidth(inner).Render(strings.Join(meta, text.Render("  ")))

	body := lipgloss.JoinVertical(lipgloss.Left, title, people, metaLine)
	return lipgloss.NewStyle().
		Background(bg).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(styles.TypeColor(string(item.Type))).
		BorderBackground(bg).
		PaddingLeft(1).
		Width(width).
		MaxHeight(styles.BoardCardHeight).
		Render(body)
}

type boardColumn struct {
	state string
	list  list.Model
}

type BoardPanel struct {
	columns []*boardColumn
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
	grouped := map[string][]list.Item{}
	for _, it := range backlog {
		grouped[string(it.State)] = append(grouped[string(it.State)], cardItem{item: it})
	}
	b.columns = make([]*boardColumn, len(states))
	for i, st := range states {
		l := widgets.NewList(grouped[st], cardDelegate{}, b.keys, st)
		l.SetStatusBarItemName("card", "cards")
		b.columns[i] = &boardColumn{state: st, list: l}
	}
	b.col = min(b.col, max(len(states)-1, 0))
	if selectedID > 0 {
		b.selectByID(selectedID)
	}
	b.refresh()
}

func (b *BoardPanel) selectByID(id int) {
	for ci, c := range b.columns {
		for ri, li := range c.list.Items() {
			if li.(cardItem).item.ID == id {
				b.col = ci
				c.list.Select(ri)
				return
			}
		}
	}
}

func (b *BoardPanel) refresh() {
	_, _, width := b.visibleColumns()
	for i, c := range b.columns {
		c.list.SetDelegate(cardDelegate{active: i == b.col, focused: b.focused})
		c.list.Title = fmt.Sprintf("%s  %d", c.state, len(c.list.Items()))
		titleColor := styles.ColorText
		if i == b.col && b.focused {
			titleColor = styles.ColorPrimary
		}
		c.list.Styles = widgets.ListStyles(titleColor)
		c.list.SetSize(width, b.height)
	}
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
	if c, ok := b.columns[b.col].list.SelectedItem().(cardItem); ok {
		return c.item
	}
	return nil
}

func (b BoardPanel) Count() int {
	n := 0
	for _, c := range b.columns {
		n += len(c.list.Items())
	}
	return n
}

func (b BoardPanel) Title() string {
	start, count, _ := b.visibleColumns()
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
	b.width = width
	b.height = height
	b.refresh()
}

func (b *BoardPanel) SetFocused(focused bool) {
	if b.focused == focused {
		return
	}
	b.focused = focused
	b.refresh()
}

func (b BoardPanel) Update(msg tea.Msg) (BoardPanel, tea.Cmd) {
	if !b.focused || len(b.columns) == 0 {
		return b, nil
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, b.keys.Left):
			b.col = max(b.col-1, 0)
			b.refresh()
			return b, nil
		case key.Matches(k, b.keys.Right):
			b.col = min(b.col+1, len(b.columns)-1)
			b.refresh()
			return b, nil
		case key.Matches(k, b.keys.MoveItemLeft):
			return b, b.moveCard(-1)
		case key.Matches(k, b.keys.MoveItemRight):
			return b, b.moveCard(1)
		case key.Matches(k, b.keys.Select):
			if item := b.SelectedItem(); item != nil {
				opened := *item
				return b, func() tea.Msg { return OpenWorkItemMsg{Item: opened} }
			}
			return b, nil
		case key.Matches(k, b.keys.View):
			if item := b.SelectedItem(); item != nil {
				viewed := *item
				return b, func() tea.Msg { return ViewWorkItemMsg{Item: viewed} }
			}
			return b, nil
		}
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok {
		c := &b.columns[b.col].list
		switch wheel.Button {
		case tea.MouseWheelUp:
			c.CursorUp()
		case tea.MouseWheelDown:
			c.CursorDown()
		}
		return b, nil
	}
	var cmd tea.Cmd
	b.columns[b.col].list, cmd = b.columns[b.col].list.Update(msg)
	return b, cmd
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

func (b BoardPanel) visibleColumns() (int, int, int) {
	n := max(len(b.columns), 1)
	maxCols := max((b.width+styles.BoardColumnGap)/(styles.BoardColumnMinWidth+styles.BoardColumnGap), 1)
	count := min(n, maxCols)
	start := 0
	if b.col >= count {
		start = b.col - count + 1
	}
	width := (b.width - (count-1)*styles.BoardColumnGap) / count
	return start, count, width
}

func (b BoardPanel) View() string {
	if b.width <= 0 || b.height <= 0 {
		return ""
	}
	if len(b.columns) == 0 {
		return Block(Muted("No backlog items match the current filters."), b.width, b.height)
	}
	start, count, colWidth := b.visibleColumns()
	gap := Block("", styles.BoardColumnGap, b.height)
	var blocks []string
	for i := start; i < min(start+count, len(b.columns)); i++ {
		if i > start {
			blocks = append(blocks, gap)
		}
		blocks = append(blocks, Block(b.columns[i].list.View(), colWidth, b.height))
	}
	return Block(lipgloss.JoinHorizontal(lipgloss.Top, blocks...), b.width, b.height)
}

func (b *BoardPanel) KeyBindings() []key.Binding {
	return []key.Binding{
		b.keys.View, b.keys.MoveItemLeft, b.keys.MoveItemRight, b.keys.ChangeState, b.keys.Assign,
		b.keys.CreateTask, b.keys.AddComment, b.keys.CreateParent,
	}
}

type MoveWorkItemMsg struct {
	Item     models.WorkItem
	NewState string
}
