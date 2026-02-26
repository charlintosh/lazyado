package panels

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

type PRFilterMode int

const (
	PRFilterAll PRFilterMode = iota
	PRFilterMine
	PRFilterAssignedToMe
	PRFilterByRepo
)

const (
	prColIDWidth     = 7
	prColRepoWidth   = 14
	prColStatusWidth = 10
	prColAuthorWidth = 14
	prColVotesWidth  = 10
)

type PRListPanel struct {
	items          []models.PullRequest
	filteredItems  []models.PullRequest
	table          table.Model
	styles         styles.Styles
	keys           keys.KeyMap
	width          int
	height         int
	focused        bool
	filterMode     PRFilterMode
	userID         string
	filterRepoID   string
	filterRepoName string
}

type PRsLoadedMsg struct {
	Items []models.PullRequest
}

type OpenPRMsg struct {
	Item models.PullRequest
}

type ViewPRMsg struct {
	Item models.PullRequest
}

type PRVoteRequestMsg struct {
	PR   models.PullRequest
	Vote models.PRVote
}

type OpenPRFilterMsg struct{}

func NewPRListPanel(s styles.Styles, k keys.KeyMap) PRListPanel {
	tableStyles := table.DefaultStyles()
	tableStyles.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorAccent).
		Padding(0, 1)
	tableStyles.Cell = lipgloss.NewStyle().
		Padding(0, 1)
	tableStyles.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)

	km := table.DefaultKeyMap()
	km.PageUp = key.NewBinding(key.WithKeys("pgup"))
	km.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"))

	cols := []table.Column{
		{Title: "ID", Width: prColIDWidth},
		{Title: "REPO", Width: prColRepoWidth},
		{Title: "STATUS", Width: prColStatusWidth},
		{Title: "AUTHOR", Width: prColAuthorWidth},
		{Title: "VOTES", Width: prColVotesWidth},
		{Title: "TITLE", Width: 30},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
		table.WithStyles(tableStyles),
		table.WithKeyMap(km),
		table.WithHeight(10),
	)

	return PRListPanel{
		items:  []models.PullRequest{},
		table:  t,
		styles: s,
		keys:   k,
	}
}

func (p *PRListPanel) SetItems(items []models.PullRequest) {
	p.items = items
	p.applyFilter()
}

func (p *PRListPanel) SetUserID(id string) {
	p.userID = id
}

func (p *PRListPanel) CycleFilter() {
	switch p.filterMode {
	case PRFilterAll:
		p.filterMode = PRFilterMine
	case PRFilterMine:
		p.filterMode = PRFilterAssignedToMe
	case PRFilterAssignedToMe:
		p.filterMode = PRFilterAll
	case PRFilterByRepo:
		p.filterMode = PRFilterAll
		p.filterRepoID = ""
		p.filterRepoName = ""
	}
	p.applyFilter()
}

func (p *PRListPanel) FilterMode() PRFilterMode {
	return p.filterMode
}

func (p *PRListPanel) SetRepoFilter(repoID, repoName string) {
	if repoID == "" {
		p.filterMode = PRFilterAll
		p.filterRepoID = ""
		p.filterRepoName = ""
	} else {
		p.filterMode = PRFilterByRepo
		p.filterRepoID = repoID
		p.filterRepoName = repoName
	}
	p.applyFilter()
}

func (p *PRListPanel) SelectedRepoID() string {
	return p.filterRepoID
}

func (p *PRListPanel) Repositories() []models.Repository {
	seen := make(map[string]bool)
	var repos []models.Repository
	for _, pr := range p.items {
		if !seen[pr.Repository.ID] {
			seen[pr.Repository.ID] = true
			repos = append(repos, pr.Repository)
		}
	}
	return repos
}

func (p *PRListPanel) applyFilter() {
	switch p.filterMode {
	case PRFilterMine:
		filtered := make([]models.PullRequest, 0)
		for _, pr := range p.items {
			if pr.CreatedByID == p.userID {
				filtered = append(filtered, pr)
			}
		}
		p.filteredItems = filtered
	case PRFilterAssignedToMe:
		filtered := make([]models.PullRequest, 0)
		for _, pr := range p.items {
			for _, r := range pr.Reviewers {
				if r.ID == p.userID {
					filtered = append(filtered, pr)
					break
				}
			}
		}
		p.filteredItems = filtered
	case PRFilterByRepo:
		filtered := make([]models.PullRequest, 0)
		for _, pr := range p.items {
			if pr.Repository.ID == p.filterRepoID {
				filtered = append(filtered, pr)
			}
		}
		p.filteredItems = filtered
	default:
		p.filteredItems = p.items
	}
	p.rebuildTable()
}

func (p *PRListPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	innerWidth := width - styles.PanelPaddedOffset

	tableHeight := height - 4
	if tableHeight < 3 {
		tableHeight = 3
	}

	p.table.SetWidth(innerWidth)
	p.table.SetHeight(tableHeight)
	p.table.SetColumns(p.buildColumns())
	p.rebuildTable()
}

func (p *PRListPanel) buildColumns() []table.Column {
	innerWidth := p.width - styles.PanelPaddedOffset
	fixedWidth := prColIDWidth + prColRepoWidth + prColStatusWidth + prColAuthorWidth + prColVotesWidth + 12
	titleWidth := innerWidth - fixedWidth
	if titleWidth < 10 {
		titleWidth = 10
	}

	return []table.Column{
		{Title: "ID", Width: prColIDWidth},
		{Title: "REPO", Width: prColRepoWidth},
		{Title: "STATUS", Width: prColStatusWidth},
		{Title: "AUTHOR", Width: prColAuthorWidth},
		{Title: "VOTES", Width: prColVotesWidth},
		{Title: "TITLE", Width: titleWidth},
	}
}

func (p *PRListPanel) SetFocused(focused bool) {
	p.focused = focused
	if focused {
		p.table.Focus()
	} else {
		p.table.Blur()
	}
}

func (p *PRListPanel) SelectedItem() *models.PullRequest {
	idx := p.table.Cursor()
	if idx < 0 || idx >= len(p.filteredItems) {
		return nil
	}
	return &p.filteredItems[idx]
}

func (p PRListPanel) Update(msg tea.Msg) (PRListPanel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, p.keys.Open) {
			if item := p.SelectedItem(); item != nil {
				return p, func() tea.Msg { return OpenPRMsg{Item: *item} }
			}
		}

		if key.Matches(msg, p.keys.View) {
			if item := p.SelectedItem(); item != nil {
				return p, func() tea.Msg { return ViewPRMsg{Item: *item} }
			}
		}

		if key.Matches(msg, p.keys.Search) {
			return p, func() tea.Msg { return OpenPRFilterMsg{} }
		}

		p.table, cmd = p.table.Update(msg)
	}

	return p, cmd
}

func (p PRListPanel) View() string {
	panel := p.styles.PanelInactive
	if p.focused {
		panel = p.styles.PanelActive
	}

	var filterLabel string
	switch p.filterMode {
	case PRFilterMine:
		filterLabel = " (Created by me)"
	case PRFilterAssignedToMe:
		filterLabel = " (Assigned to me)"
	case PRFilterByRepo:
		filterLabel = " (" + p.filterRepoName + ")"
	}

	counter := fmt.Sprintf("  %d", len(p.filteredItems))
	if len(p.filteredItems) != len(p.items) {
		counter = fmt.Sprintf("  %d/%d", len(p.filteredItems), len(p.items))
	}

	var b strings.Builder
	b.WriteString(p.styles.PanelTitle.Render("Pull Requests" + filterLabel))
	b.WriteString(p.styles.Subtitle.Render(counter))
	b.WriteString("\n")
	b.WriteString(p.table.View())

	return panel.
		Width(p.width).
		Height(p.height).
		Render(b.String())
}

func (p PRListPanel) GetAvailableActions() []string {
	return []string{
		p.styles.HelpKey.Render("enter") + p.styles.HelpDesc.Render(":open"),
		p.styles.HelpKey.Render("v") + p.styles.HelpDesc.Render(":view"),
		p.styles.HelpKey.Render("a") + p.styles.HelpDesc.Render(":approve"),
		p.styles.HelpKey.Render("/") + p.styles.HelpDesc.Render(":search repo"),
		p.styles.HelpKey.Render("f") + p.styles.HelpDesc.Render(":filter"),
		p.styles.HelpKey.Render("Ctrl+r") + p.styles.HelpDesc.Render(":refresh"),
	}
}

func (p *PRListPanel) rebuildTable() {
	cols := p.buildColumns()
	titleWidth := cols[len(cols)-1].Width

	rows := make([]table.Row, 0, len(p.filteredItems))
	for _, pr := range p.filteredItems {
		id := fmt.Sprintf("#%d", pr.ID)

		repo := string(truncate.StringWithTail(pr.Repository.Name, uint(prColRepoWidth), "…"))

		status := pr.StatusLabel()
		if pr.IsDraft {
			status = "Draft"
		}

		author := string(truncate.StringWithTail(pr.CreatedBy, uint(prColAuthorWidth), "…"))

		votes := prVoteSummary(pr.Reviewers)

		title := string(truncate.StringWithTail(pr.Title, uint(titleWidth), "…"))

		rows = append(rows, table.Row{id, repo, status, author, votes, title})
	}
	p.table.SetRows(rows)
}

func prVoteSummary(reviewers []models.PRReviewer) string {
	approved := 0
	rejected := 0
	waiting := 0
	for _, r := range reviewers {
		if r.IsContainer {
			continue
		}
		switch {
		case r.Vote >= 10:
			approved++
		case r.Vote == 5:
			approved++
		case r.Vote <= -10:
			rejected++
		case r.Vote == -5:
			waiting++
		}
	}

	var parts []string
	if approved > 0 {
		parts = append(parts, fmt.Sprintf("%d✓", approved))
	}
	if rejected > 0 {
		parts = append(parts, fmt.Sprintf("%d✗", rejected))
	}
	if waiting > 0 {
		parts = append(parts, fmt.Sprintf("%d⏳", waiting))
	}
	if len(parts) == 0 {
		return "--"
	}
	return strings.Join(parts, " ")
}
