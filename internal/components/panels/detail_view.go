package panels

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const dateFormat = "2006-01-02"

// fixedHeaderLines is the number of lines the non-scrollable header occupies inside
// the PanelActive content area: title (1) + subheader text (1) + bottom border (1).
const fixedHeaderLines = 3

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
}

// NewDetailView creates a new detail view.
func NewDetailView(st styles.Styles, ks keys.KeyMap) DetailView {
	return DetailView{
		viewport: viewport.New(0, 0),
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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Back), key.Matches(msg, d.keys.Quit):
			return d, func() tea.Msg { return CloseDetailViewMsg{} }
		case key.Matches(msg, d.keys.AddComment):
			if d.item != nil {
				return d, func() tea.Msg { return OpenCommentModalMsg{Item: *d.item} }
			}
		default:
			// Delegate all other key events to the viewport (j/k, up/down, page, …).
			d.viewport, cmd = d.viewport.Update(msg)
		}
	default:
		d.viewport, cmd = d.viewport.Update(msg)
	}

	return d, cmd
}

// View renders the detail view.
func (d DetailView) View() string {
	if d.item == nil {
		return ""
	}

	header := d.renderHeader()

	var body string
	if d.ready {
		body = d.viewport.View()
	}

	panelContent := lipgloss.JoinVertical(lipgloss.Left, header, body)

	mainArea := d.styles.PanelActive.
		Width(d.width - styles.PanelBorderOffset).
		Height(d.height - styles.PanelPaddedOffset).
		Render(panelContent)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, d.renderStatusBar())
}

// renderHeader renders the fixed (non-scrollable) title and subheader rows.
func (d *DetailView) renderHeader() string {
	innerWidth := d.viewport.Width
	if innerWidth <= 0 {
		innerWidth = d.width - styles.PanelPaddedOffset
	}

	typeStyle := d.styles.TypeBadge(string(d.item.Type))
	stateStyle := d.styles.StateBadge(string(d.item.State))

	idStr := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(
		fmt.Sprintf("#%d", d.item.ID),
	)
	typeBadge := typeStyle.Render(d.item.ShortType())
	stateBadge := stateStyle.Render(string(d.item.State))
	titleText := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorText).Render(d.item.Title)

	titleBar := lipgloss.NewStyle().
		Width(innerWidth).
		Render(idStr + "  " + typeBadge + "  " + stateBadge + "  " + titleText)

	assignedTo := d.item.AssignedTo
	if assignedTo == "" {
		assignedTo = "Unassigned"
	}
	labelSt := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	valueSt := lipgloss.NewStyle().Foreground(styles.ColorText)

	subHeaderText := strings.Join([]string{
		labelSt.Render("Assigned:") + " " + valueSt.Render(assignedTo),
		labelSt.Render("Sprint:") + " " + valueSt.Render(d.item.SprintName()),
		labelSt.Render("Area:") + " " + valueSt.Render(d.item.AreaName()),
	}, "    ")
	subHeader := lipgloss.NewStyle().
		Width(innerWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder).
		BorderBottom(true).
		Render(subHeaderText)

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, subHeader)
}

func (d *DetailView) renderStatusBar() string {
	return d.styles.StatusBar.
		Width(d.width).
		Render("Esc Back  c Add comment  j/k Scroll")
}

// SetSize updates the component dimensions and refreshes the viewport.
func (d *DetailView) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.refreshViewport()
}

// SetItem sets the work item and rebuilds the viewport content.
func (d *DetailView) SetItem(item *models.WorkItem) {
	d.item = item
	d.viewport.GotoTop()
	d.updateViewportContent()
}

// SetComments sets the comments list and rebuilds the viewport content.
func (d *DetailView) SetComments(comments []models.Comment) {
	d.comments = comments
	d.updateViewportContent()
}

// refreshViewport recalculates viewport dimensions from d.width / d.height.
func (d *DetailView) refreshViewport() {
	// In lipgloss, both Width(W) and Height(H) are pre-border; border adds +2 externally.
	// PanelActive uses Width(d.width-2)  → outer = d.width;     inner_w = d.width-4  (- padding 2).
	// PanelActive uses Height(d.height-4) → outer = d.height-2; inner_h = d.height-4 (no vert pad).
	// Subtract the fixed header lines to get the viewport height.
	vpWidth := d.width - styles.PanelPaddedOffset
	if vpWidth < styles.DetailMinViewportWidth {
		vpWidth = styles.DetailMinViewportWidth
	}
	vpHeight := d.height - styles.PanelPaddedOffset - fixedHeaderLines
	if vpHeight < styles.DetailMinViewportHeight {
		vpHeight = styles.DetailMinViewportHeight
	}

	d.viewport.Width = vpWidth
	d.viewport.Height = vpHeight
	d.updateViewportContent()
}

// updateViewportContent (re)builds the two-column layout and loads it into the viewport.
func (d *DetailView) updateViewportContent() {
	if d.item == nil || d.viewport.Width <= 0 {
		return
	}

	innerWidth := d.viewport.Width

	sidebarWidth := styles.DetailSidebarWidthDefault
	switch {
	case innerWidth > styles.DetailBreakpointLG:
		sidebarWidth = styles.DetailSidebarWidthLG
	case innerWidth < styles.DetailBreakpointSM:
		sidebarWidth = styles.DetailSidebarWidthSM
	}

	mainWidth := innerWidth - sidebarWidth
	if mainWidth < styles.MinBottomRowWidth {
		d.viewport.SetContent(d.renderSingleColumn(innerWidth))
	} else {
		left := d.renderMainContent(mainWidth)
		right := d.renderSidebar(sidebarWidth)
		d.viewport.SetContent(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
	}

	d.ready = true
}

// renderMainContent builds the scrollable left column.
func (d *DetailView) renderMainContent(width int) string {
	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(0, 1).
		Width(width - styles.PanelBorderOffset)

	titleSt := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorTextMuted)
	wrapWidth := width - styles.PanelPaddedOffset
	if wrapWidth < styles.DetailWrapMinWidth {
		wrapWidth = styles.DetailWrapMinWidth
	}

	sections := []string{
		sectionStyle.Render(titleSt.Render("DESCRIPTION") + "\n" + d.buildDescriptionBody(wrapWidth)),
	}

	if d.item.AcceptanceCriteria != "" {
		sections = append(sections, sectionStyle.Render(
			titleSt.Render("ACCEPTANCE CRITERIA")+"\n"+wordWrap(d.item.AcceptanceCriteria, wrapWidth),
		))
	}

	discussionTitle := fmt.Sprintf("DISCUSSION (%d)", len(d.comments))
	sections = append(sections, sectionStyle.Render(
		titleSt.Render(discussionTitle)+"\n"+d.buildDiscussionBody(wrapWidth),
	))

	return strings.Join(sections, "\n")
}

// buildDescriptionBody returns the rendered description text (or a placeholder).
func (d *DetailView) buildDescriptionBody(wrapWidth int) string {
	if d.item.Description == "" {
		return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Italic(true).Render("No description provided")
	}
	return wordWrap(d.item.Description, wrapWidth)
}

// buildDiscussionBody returns the rendered comment list (or a placeholder).
func (d *DetailView) buildDiscussionBody(wrapWidth int) string {
	if len(d.comments) == 0 {
		return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Italic(true).Render("No comments yet")
	}
	var parts []string
	for _, c := range d.comments {
		author := d.styles.CommentAuthor.Render(c.CreatedBy)
		date := d.styles.CommentDate.Render(formatCommentDate(c.CreatedDate))
		parts = append(parts, author+"  "+date+"\n"+wordWrap(c.Text, wrapWidth))
	}
	return strings.Join(parts, "\n\n")
}

// renderSidebar builds the scrollable right column.
func (d *DetailView) renderSidebar(width int) string {
	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(0, 1).
		Width(width - styles.PanelBorderOffset)

	titleSt := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorTextMuted)
	labelSt := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(styles.DetailLabelWidth)
	valueSt := lipgloss.NewStyle().Foreground(styles.ColorText)

	var sections []string

	// PLANNING
	priorityStr := "-"
	if d.item.Priority > 0 {
		priorityStr = fmt.Sprintf("%d", d.item.Priority)
	}
	effortStr := "-"
	if d.item.Effort > 0 {
		effortStr = fmt.Sprintf("%.1f pts", d.item.Effort)
	}
	sections = append(sections, sectionStyle.Render(strings.Join([]string{
		titleSt.Render("PLANNING"),
		labelSt.Render("Priority:") + valueSt.Render(priorityStr),
		labelSt.Render("Effort:") + valueSt.Render(effortStr),
	}, "\n")))

	// DETAILS
	sections = append(sections, sectionStyle.Render(strings.Join([]string{
		titleSt.Render("DETAILS"),
		labelSt.Render("Created:") + valueSt.Render(d.item.CreatedDate.Format(dateFormat)),
		labelSt.Render("Updated:") + valueSt.Render(d.item.ChangedDate.Format(dateFormat)),
	}, "\n")))

	// TAGS
	if len(d.item.Tags) > 0 {
		var tagStrs []string
		for _, tag := range d.item.Tags {
			tagStrs = append(tagStrs, d.styles.DetailTag.Render(tag))
		}
		sections = append(sections, sectionStyle.Render(
			titleSt.Render("TAGS")+"\n"+strings.Join(tagStrs, " "),
		))
	}

	// RELATED WORK
	if d.item.ParentID > 0 {
		parentRef := fmt.Sprintf("#%d", d.item.ParentID)
		if d.item.ParentTitle != "" {
			parentRef += " " + d.item.ParentTitle
		}
		sections = append(sections, sectionStyle.Render(
			titleSt.Render("RELATED WORK")+"\n"+
				lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render("Parent: ")+
				lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(parentRef),
		))
	}

	return strings.Join(sections, "\n")
}

// renderSingleColumn is a narrow-terminal fallback — everything stacked vertically.
func (d *DetailView) renderSingleColumn(width int) string {
	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(0, 1).
		Width(width - styles.PanelBorderOffset)

	titleSt := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorTextMuted)
	labelSt := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(styles.DetailLabelWidth)
	valueSt := lipgloss.NewStyle().Foreground(styles.ColorText)
	wrapWidth := width - styles.PanelPaddedOffset
	if wrapWidth < styles.DetailWrapMinWidth {
		wrapWidth = styles.DetailWrapMinWidth
	}

	var sections []string

	// Planning / details inline
	priorityStr := "-"
	if d.item.Priority > 0 {
		priorityStr = fmt.Sprintf("%d", d.item.Priority)
	}
	effortStr := "-"
	if d.item.Effort > 0 {
		effortStr = fmt.Sprintf("%.1f pts", d.item.Effort)
	}
	sections = append(sections, sectionStyle.Render(
		labelSt.Render("Priority:")+valueSt.Render(priorityStr)+"    "+
			labelSt.Render("Effort:")+valueSt.Render(effortStr)+"    "+
			labelSt.Render("Created:")+valueSt.Render(d.item.CreatedDate.Format(dateFormat))+"    "+
			labelSt.Render("Updated:")+valueSt.Render(d.item.ChangedDate.Format(dateFormat)),
	))

	// Tags
	if len(d.item.Tags) > 0 {
		var tagStrs []string
		for _, tag := range d.item.Tags {
			tagStrs = append(tagStrs, d.styles.DetailTag.Render(tag))
		}
		sections = append(sections, sectionStyle.Render(
			titleSt.Render("TAGS")+"\n"+strings.Join(tagStrs, " "),
		))
	}

	// Parent
	if d.item.ParentID > 0 {
		parentRef := fmt.Sprintf("#%d", d.item.ParentID)
		if d.item.ParentTitle != "" {
			parentRef += " " + d.item.ParentTitle
		}
		sections = append(sections, sectionStyle.Render(
			lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render("Parent: ")+
				lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(parentRef),
		))
	}

	sections = append(sections, sectionStyle.Render(
		titleSt.Render("DESCRIPTION")+"\n"+d.buildDescriptionBody(wrapWidth),
	))

	if d.item.AcceptanceCriteria != "" {
		sections = append(sections, sectionStyle.Render(
			titleSt.Render("ACCEPTANCE CRITERIA")+"\n"+wordWrap(d.item.AcceptanceCriteria, wrapWidth),
		))
	}

	discussionTitle := fmt.Sprintf("DISCUSSION (%d)", len(d.comments))
	sections = append(sections, sectionStyle.Render(
		titleSt.Render(discussionTitle)+"\n"+d.buildDiscussionBody(wrapWidth),
	))

	return strings.Join(sections, "\n")
}

// CloseDetailViewMsg is sent when the detail view should be closed.
type CloseDetailViewMsg struct{}

// OpenCommentModalMsg is sent when the comment modal should be opened.
type OpenCommentModalMsg struct {
	Item models.WorkItem
}
