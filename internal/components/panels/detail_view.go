package panels

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DetailView is the fullscreen detail view component
type DetailView struct {
	item         *models.WorkItem
	comments     []models.Comment
	styles       styles.Styles
	keys         keys.KeyMap
	width        int
	height       int
	scrollOffset int
	maxScroll    int
}

// NewDetailView creates a new detail view
func NewDetailView(styles styles.Styles, keys keys.KeyMap) DetailView {
	return DetailView{
		styles: styles,
		keys:   keys,
	}
}

// Init initializes the detail view
func (d DetailView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the detail view
func (d DetailView) Update(msg tea.Msg) (DetailView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Back):
			return d, func() tea.Msg { return CloseDetailViewMsg{} }
		case key.Matches(msg, d.keys.Quit):
			return d, func() tea.Msg { return CloseDetailViewMsg{} }
		case key.Matches(msg, d.keys.AddComment):
			if d.item != nil {
				return d, func() tea.Msg { return OpenCommentModalMsg{Item: *d.item} }
			}
		case key.Matches(msg, d.keys.Up):
			if d.scrollOffset > 0 {
				d.scrollOffset--
			}
		case key.Matches(msg, d.keys.Down):
			if d.scrollOffset < d.maxScroll {
				d.scrollOffset++
			}
		}
	}

	return d, nil
}

// View renders the detail view
func (d DetailView) View() string {
	if d.item == nil {
		return ""
	}

	var sections []string

	// Title bar
	title := fmt.Sprintf("#%d %s", d.item.ID, d.item.Title)
	titleBar := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorText).
		Padding(0, 1).
		Width(d.width - 2).
		Render(title)
	sections = append(sections, titleBar)

	// Metadata section
	metadataContent := d.renderMetadata()
	metadataSection := d.styles.DetailSection.
		Width(d.width - 6).
		Render("METADATA\n" + metadataContent)
	sections = append(sections, metadataSection)

	// Parent section (if exists)
	if d.item.ParentID > 0 {
		parentContent := fmt.Sprintf("#%d", d.item.ParentID)
		if d.item.ParentTitle != "" {
			parentContent += " " + d.item.ParentTitle
		}
		parentSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("PARENT\n" + parentContent)
		sections = append(sections, parentSection)
	}

	// Description section
	if d.item.Description != "" {
		desc := wordWrap(d.item.Description, d.width-10)
		descSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("DESCRIPTION\n" + desc)
		sections = append(sections, descSection)
	}

	// Acceptance Criteria section
	if d.item.AcceptanceCriteria != "" {
		acCriteria := wordWrap(d.item.AcceptanceCriteria, d.width-10)
		acSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("ACCEPTANCE CRITERIA\n" + acCriteria)
		sections = append(sections, acSection)
	}

	// Tags section
	if len(d.item.Tags) > 0 {
		var tagStrings []string
		for _, tag := range d.item.Tags {
			tagStrings = append(tagStrings, d.styles.DetailTag.Render(tag))
		}
		tagsSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("TAGS\n" + strings.Join(tagStrings, " "))
		sections = append(sections, tagsSection)
	}

	// Comments section
	if len(d.comments) > 0 {
		var commentLines []string
		for _, comment := range d.comments {
			commentText := wordWrap(comment.Text, d.width-12)
			commentHeader := lipgloss.NewStyle().
				Bold(true).
				Foreground(styles.ColorPrimary).
				Render(comment.CreatedBy + " - " + formatCommentDate(comment.CreatedDate))
			commentLines = append(commentLines, commentHeader+"\n"+commentText)
		}
		commentsSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("COMMENTS (" + fmt.Sprintf("%d", len(d.comments)) + ")\n" + strings.Join(commentLines, "\n\n"))
		sections = append(sections, commentsSection)
	}

	// Join all sections
	content := strings.Join(sections, "\n\n")

	// Calculate scrolling
	contentLines := strings.Split(content, "\n")
	viewableHeight := d.height - 4
	d.maxScroll = len(contentLines) - viewableHeight
	if d.maxScroll < 0 {
		d.maxScroll = 0
	}

	// Apply scrolling
	if d.scrollOffset > 0 && d.scrollOffset < len(contentLines) {
		contentLines = contentLines[d.scrollOffset:]
	}
	if len(contentLines) > viewableHeight {
		contentLines = contentLines[:viewableHeight]
	}

	scrolledContent := strings.Join(contentLines, "\n")

	// Status bar
	statusBar := d.renderStatusBar()

	// Build final view
	mainContent := d.styles.PanelActive.
		Width(d.width).
		Height(d.height - 2).
		Render(scrolledContent)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusBar,
	)
}

func (d *DetailView) renderMetadata() string {
	typeStyle := d.styles.TypeBadge(string(d.item.Type))
	stateStyle := d.styles.StateBadge(string(d.item.State))

	labelWidth := 12
	valueWidth := 20

	label := func(s string) string {
		return d.styles.DetailLabel.Width(labelWidth).Render(s)
	}
	value := func(s string) string {
		return d.styles.DetailValue.Width(valueWidth).Render(s)
	}

	rows := []string{
		label("Type:") + typeStyle.Render(d.item.ShortType()) + "     " + label("ID:") + value(fmt.Sprintf("#%d", d.item.ID)),
		label("State:") + stateStyle.Render(string(d.item.State)) + "     " + label("Created:") + value(d.item.CreatedDate.Format("2006-01-02")),
	}

	assignedTo := d.item.AssignedTo
	if assignedTo == "" {
		assignedTo = "Unassigned"
	}
	rows = append(rows, label("Assigned:")+value(assignedTo)+"  "+label("Updated:")+value(d.item.ChangedDate.Format("2006-01-02")))
	rows = append(rows, label("Sprint:")+value(d.item.SprintName())+"  "+label("Priority:")+value(fmt.Sprintf("%d", d.item.Priority)))
	rows = append(rows, label("Area:")+value(d.item.AreaName()))

	return strings.Join(rows, "\n")
}

func (d *DetailView) renderStatusBar() string {
	help := "Esc Back  c Add comment  j/k Scroll"
	return d.styles.StatusBar.
		Width(d.width).
		Render(help)
}

// SetItem sets the work item to display
func (d *DetailView) SetItem(item *models.WorkItem) {
	d.item = item
	d.scrollOffset = 0
}

// SetComments sets the comments to display
func (d *DetailView) SetComments(comments []models.Comment) {
	d.comments = comments
}

// SetSize sets the size of the detail view
func (d *DetailView) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// CloseDetailViewMsg is sent when the detail view should be closed
type CloseDetailViewMsg struct{}

// OpenCommentModalMsg is sent when the comment modal should be opened
type OpenCommentModalMsg struct {
	Item models.WorkItem
}
