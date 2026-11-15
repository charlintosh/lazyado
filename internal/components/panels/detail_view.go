package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const dateFormat = "2006-01-02"

const fixedHeaderLines = 6

// DetailView is the fullscreen detail view component.
type DetailView struct {
	item     *models.WorkItem
	comments []models.Comment
	viewport viewport.Model
	styles   styles.Styles
	keys     keys.KeyMap
	width    int
	height   int
	ready    bool

	commentCursor        int
	commentFocused       bool
	commentOffsets       []int
	discussionBaseOffset int
}

// NewDetailView creates a new detail view.
func NewDetailView(st styles.Styles, ks keys.KeyMap) DetailView {
	return DetailView{
		viewport: viewport.New(),
		styles:   st,
		keys:     ks,
	}
}

// Init initialises the detail view.
func (d DetailView) Init() tea.Cmd { return nil }

// Update handles incoming messages.
func (d DetailView) Update(msg tea.Msg) (DetailView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keys.Back):
			if d.commentFocused {
				d.commentFocused = false
				d.updateViewportContent()
				return d, nil
			}
			return d, func() tea.Msg { return CloseDetailViewMsg{} }
		case key.Matches(msg, d.keys.Quit):
			return d, func() tea.Msg { return CloseDetailViewMsg{} }
		case key.Matches(msg, d.keys.AddComment):
			if d.item != nil {
				return d, func() tea.Msg { return OpenCommentModalMsg{Item: *d.item} }
			}
		case key.Matches(msg, d.keys.NextPanel):
			if len(d.comments) > 0 {
				d.commentFocused = !d.commentFocused
				d.updateViewportContent()
				if d.commentFocused {
					d.scrollToComment()
				}
				return d, nil
			}
		default:
			if d.commentFocused && len(d.comments) > 0 {
				oldCursor := d.commentCursor
				switch {
				case key.Matches(msg, d.keys.Up):
					if d.commentCursor > 0 {
						d.commentCursor--
					}
				case key.Matches(msg, d.keys.Down):
					if d.commentCursor < len(d.comments)-1 {
						d.commentCursor++
					}
				case key.Matches(msg, d.keys.Top):
					d.commentCursor = 0
				case key.Matches(msg, d.keys.Bottom):
					d.commentCursor = len(d.comments) - 1
				case key.Matches(msg, d.keys.EditComment):
					if d.commentCursor < len(d.comments) {
						comment := d.comments[d.commentCursor]
						return d, func() tea.Msg {
							return EditCommentMsg{Comment: comment}
						}
					}
				case key.Matches(msg, d.keys.DeleteComment):
					if d.commentCursor < len(d.comments) {
						comment := d.comments[d.commentCursor]
						return d, func() tea.Msg {
							return DeleteCommentMsg{Comment: comment}
						}
					}
				}
				if d.commentCursor != oldCursor {
					d.updateViewportContent()
					d.scrollToComment()
				}
				return d, nil
			}
			// Delegate all other key events to the viewport (j/k, up/down, page, …).
			d.viewport, cmd = d.viewport.Update(msg)
		}
	default:
		d.viewport, cmd = d.viewport.Update(msg)
	}

	return d, cmd
}

func (d DetailView) View() string {
	if d.item == nil {
		return Block("", d.width, d.height)
	}
	var body string
	if d.ready {
		body = d.viewport.View()
	}
	return Block(lipgloss.JoinVertical(lipgloss.Left, d.renderHeader(), body), d.width, d.height)
}

func (d *DetailView) renderHeader() string {
	item := d.item
	inner := max(d.width-styles.CardChromeWidth, 1)
	bg := styles.ColorSurfaceAlt
	text := lipgloss.NewStyle().Background(bg).Foreground(styles.ColorText)
	label := text.Foreground(styles.ColorTextMuted)
	gap := text.Render("    ")

	badge := TypeBadge(string(item.Type), fmt.Sprintf("%s %d", styles.TypeLabel(string(item.Type)), item.ID))
	title := text.Bold(true).Render(item.Title)

	assigned := text.Render(item.AssignedTo)
	if item.AssignedTo == "" {
		assigned = label.Render("Unassigned")
	}
	people := assigned + gap + label.Render(fmt.Sprintf("%d comments", len(d.comments)))
	if len(item.Tags) > 0 {
		people += gap + Chips(item.Tags)
	}
	fields := label.Render("State ") + text.Foreground(styles.StateColor(string(item.State))).Render(string(item.State)) +
		gap + label.Render("Area ") + text.Render(item.AreaPath) +
		gap + label.Render("Iteration ") + text.Render(item.IterationPath)

	line := text.Width(inner).MaxWidth(inner).Inline(true)
	block := lipgloss.NewStyle().
		Background(bg).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(styles.TypeColor(string(item.Type))).
		BorderBackground(bg).
		PaddingLeft(1).
		Width(d.width).
		Render(lipgloss.JoinVertical(lipgloss.Left, line.Render(badge), line.Render(title), line.Render(people), line.Render(fields)))

	active := 0
	if d.commentFocused {
		active = 1
	}
	tabs := lipgloss.NewStyle().
		Width(d.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.ColorBorder).
		Render(Tabs([]string{"Details", fmt.Sprintf("Discussion %d", len(d.comments))}, active, true))
	return lipgloss.JoinVertical(lipgloss.Left, block, tabs)
}

func (d *DetailView) KeyBindings() []key.Binding {
	if d.commentFocused {
		return []key.Binding{d.keys.Up, d.keys.EditComment, d.keys.DeleteComment, d.keys.AddComment, d.keys.NextPanel, d.keys.Back}
	}
	return []key.Binding{d.keys.Up, d.keys.NextPanel, d.keys.AddComment, d.keys.Back}
}

// SetSize updates the component dimensions and refreshes the viewport.
func (d *DetailView) SetSize(width, height int) {
	if d.width == width && d.height == height && d.ready {
		return
	}
	d.width = width
	d.height = height
	d.refreshViewport()
}

// SetItem sets the work item and rebuilds the viewport content.
func (d *DetailView) SetItem(item *models.WorkItem) {
	d.item = item
	d.commentFocused = false
	d.commentCursor = 0
	d.viewport.GotoTop()
	d.updateViewportContent()
}

// SetComments sets the comments list and rebuilds the viewport content.
func (d *DetailView) SetComments(comments []models.Comment) {
	d.comments = comments
	d.commentCursor = 0
	d.updateViewportContent()
}

// refreshViewport recalculates viewport dimensions from d.width / d.height.
func (d *DetailView) refreshViewport() {
	vpWidth := max(d.width-1, styles.DetailMinViewportWidth)
	vpHeight := max(d.height-fixedHeaderLines, styles.DetailMinViewportHeight)
	d.viewport.SetWidth(vpWidth)
	d.viewport.SetHeight(vpHeight)
	d.updateViewportContent()
}

func (d *DetailView) updateViewportContent() {
	if d.item == nil || d.viewport.Width() <= 0 {
		return
	}
	innerWidth := d.viewport.Width()
	sidebarWidth := styles.DetailSidebarWidthDefault
	switch {
	case innerWidth > styles.DetailBreakpointLG:
		sidebarWidth = styles.DetailSidebarWidthLG
	case innerWidth < styles.DetailBreakpointSM:
		sidebarWidth = styles.DetailSidebarWidthSM
	}
	mainWidth := innerWidth - sidebarWidth - 1
	if mainWidth < styles.MinBottomRowWidth {
		d.viewport.SetContent(d.renderSingleColumn(innerWidth))
	} else {
		left := d.renderMainContent(mainWidth)
		right := d.renderSidebar(sidebarWidth)
		d.viewport.SetContent(lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right))
	}
	d.ready = true
}

func formSection(title, body string, width int, focused bool) string {
	return Section(title, body, width, focused)
}

func (d *DetailView) mainSections(width int) (string, int) {
	wrap := max(width-styles.BoxPaddedSize, styles.DetailWrapMinWidth)
	pre := []string{formSection(PaneTitle("", "Description", false), d.buildDescriptionBody(wrap), width, false)}
	if d.item.AcceptanceCriteria != "" {
		pre = append(pre, formSection(PaneTitle("", "Acceptance Criteria", false),
			renderHTMLContent(d.item.AcceptanceCriteriaHTML, d.item.AcceptanceCriteria, wrap), width, false))
	}
	preContent := strings.Join(pre, "\n")
	return preContent, wrap
}

func (d *DetailView) discussionSection(preContent string, width, wrap int) string {
	d.discussionBaseOffset = strings.Count(preContent, "\n") + styles.SectionBodyOffset
	title := PaneTitle("", fmt.Sprintf("Discussion  %d", len(d.comments)), d.commentFocused)
	hint := Button("Add a comment", d.keys.AddComment)
	body := d.buildDiscussionBody(wrap)
	return formSection(title, hint+"\n"+body, width, d.commentFocused)
}

func (d *DetailView) renderMainContent(width int) string {
	preContent, wrap := d.mainSections(width)
	return preContent + "\n" + d.discussionSection(preContent, width, wrap)
}

func (d *DetailView) buildDescriptionBody(wrapWidth int) string {
	if d.item.Description == "" {
		return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Italic(true).Render("No description provided")
	}
	return renderHTMLContent(d.item.DescriptionHTML, d.item.Description, wrapWidth)
}

// buildDiscussionBody returns the rendered comment list (or a placeholder).
func (d *DetailView) buildDiscussionBody(wrapWidth int) string {
	if len(d.comments) == 0 {
		d.commentOffsets = nil
		return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Italic(true).Render("No comments yet")
	}
	selected := -1
	if d.commentFocused {
		selected = d.commentCursor
	}
	content, offsets := RenderCommentList(d.comments, d.styles, CommentListConfig{
		Width:      wrapWidth,
		Selected:   selected,
		ShowEdited: true,
		ShowBoxes:  true,
	})
	d.commentOffsets = offsets
	return strings.TrimRight(content, "\n")
}

// scrollToComment scrolls the viewport so the selected comment is visible.
func (d *DetailView) scrollToComment() {
	if d.commentCursor < 0 || d.commentCursor >= len(d.commentOffsets) {
		return
	}

	topLine := d.discussionBaseOffset + d.commentOffsets[d.commentCursor]

	bottomLine := d.viewport.TotalLineCount() - 1
	if d.commentCursor+1 < len(d.commentOffsets) {
		bottomLine = d.discussionBaseOffset + d.commentOffsets[d.commentCursor+1] - 1
	}

	vpTop := d.viewport.YOffset()
	vpBottom := vpTop + d.viewport.Height() - 1

	if topLine < vpTop {
		d.viewport.SetYOffset(topLine)
	} else if bottomLine > vpBottom {
		d.viewport.SetYOffset(bottomLine - d.viewport.Height() + 1)
	}
}

func (d *DetailView) renderSidebar(width int) string {
	return strings.Join(d.sidebarSections(width), "\n")
}

func (d *DetailView) sidebarSections(width int) []string {
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(styles.DetailFieldLabelW + 4)
	value := lipgloss.NewStyle().Foreground(styles.ColorText)
	row := func(l, v string) string { return label.Render(l) + value.Render(v) }

	planning := []string{
		row("Priority", orDash(d.item.Priority > 0, fmt.Sprintf("%d", d.item.Priority))),
		row("Effort", orDash(d.item.Effort > 0, formatEffort(d.item.Effort))),
		row("Created", d.item.CreatedDate.Format(dateFormat)),
		row("Updated", d.item.ChangedDate.Format(dateFormat)),
	}
	sections := []string{formSection(PaneTitle("", "Planning", false), strings.Join(planning, "\n"), width, false)}

	branch := Button("Create branch", d.keys.CreateBranch)
	sections = append(sections, formSection(PaneTitle("", "Development", false), branch, width, false))

	var related []string
	if d.item.ParentID > 0 {
		parent := fmt.Sprintf("%d", d.item.ParentID)
		if d.item.ParentTitle != "" {
			parent += " " + d.item.ParentTitle
		}
		related = append(related, Heading("Parent"), lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(parent))
	}
	if children := renderChildList(d.item, max(width-styles.BoxPaddedSize, 1)); children != "" {
		if len(related) > 0 {
			related = append(related, "")
		}
		related = append(related, children)
	}
	if len(related) == 0 {
		related = append(related, Muted("No related work"))
	}
	sections = append(sections, formSection(PaneTitle("", "Related Work", false), strings.Join(related, "\n"), width, false))
	return sections
}

func (d *DetailView) renderSingleColumn(width int) string {
	side := strings.Join(d.sidebarSections(width), "\n")
	preContent, wrap := d.mainSections(width)
	preContent = side + "\n" + preContent
	return preContent + "\n" + d.discussionSection(preContent, width, wrap)
}

// CloseDetailViewMsg is sent when the detail view should be closed.
type CloseDetailViewMsg struct{}

// OpenCommentModalMsg is sent when the comment modal should be opened.
type OpenCommentModalMsg struct {
	Item models.WorkItem
}

func (d DetailView) Item() *models.WorkItem {
	return d.item
}
