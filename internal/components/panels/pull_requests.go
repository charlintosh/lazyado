package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	loadingPullRequests = "Loading pull requests"
	searchingServer     = "Searching Azure DevOps"
)

type ReviewPullRequestMsg struct {
	PullRequest models.PullRequest
}

type BrowsePullRequestMsg struct {
	PullRequest models.PullRequest
}

type PullRequestsPanel struct {
	all       []models.PullRequest
	items     []models.PullRequest
	members   []models.TeamMember
	search    textinput.Model
	searching bool
	searchErr string
	applied   models.PullRequestCriteria
	cursor    int
	viewport  viewport.Model
	keys      keys.KeyMap
	me        models.Identity
	loading   bool
	spinner   string
	unscoped  bool
	oneRepo   bool
	focused   bool
	width     int
	height    int
}

func NewPullRequestsPanel(k keys.KeyMap) PullRequestsPanel {
	return PullRequestsPanel{viewport: widgets.NewViewport(k), keys: k, unscoped: true, search: widgets.SearchInput(searchPrompt, searchPlaceholder)}
}

func (p *PullRequestsPanel) SetItems(items []models.PullRequest) {
	p.all = items
	p.applyFilter()
}

func (p *PullRequestsPanel) SetLoading(loading bool) { p.loading = loading }

func (p *PullRequestsPanel) SetSpinner(frame string) { p.spinner = frame }

func (p *PullRequestsPanel) SetScope(chosen, oneRepository bool) {
	if p.unscoped == !chosen && p.oneRepo == oneRepository {
		return
	}
	p.unscoped, p.oneRepo = !chosen, oneRepository
	p.render()
}

func (p *PullRequestsPanel) SetIdentity(me models.Identity) {
	p.me = me
	p.render()
}

func (p *PullRequestsPanel) SetFocused(focused bool) {
	if p.focused == focused {
		return
	}
	p.focused = focused
	p.render()
}

func (p *PullRequestsPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.search.SetWidth(max(width/2, 1))
	p.layout()
}

func (p *PullRequestsPanel) layout() {
	p.viewport.SetWidth(p.width)
	p.viewport.SetHeight(max(p.height-p.searchChromeHeight(), 1))
	p.render()
}

func (p *PullRequestsPanel) Selected() *models.PullRequest {
	if p.cursor < 0 || p.cursor >= len(p.items) {
		return nil
	}
	return &p.items[p.cursor]
}

func (p *PullRequestsPanel) Title() string {
	if p.query().Active() {
		return fmt.Sprintf("Pull requests %d of %d", len(p.items), len(p.all))
	}
	return fmt.Sprintf("Pull requests %d", len(p.items))
}

func (p *PullRequestsPanel) KeyBindings() []key.Binding {
	if p.searching {
		return p.searchBindings()
	}
	search := widgets.Relabel(p.keys.Search, "search")
	if len(p.items) == 0 {
		return []key.Binding{p.keys.NewPullRequest, search, p.keys.Repository, widgets.Combine("view", p.keys.PrevPanel, p.keys.NextPanel), p.keys.Refresh}
	}
	return []key.Binding{
		widgets.Relabel(p.keys.Select, "review"), p.keys.NewPullRequest, search, p.keys.Vote, p.keys.AddComment, p.keys.Repository, p.keys.OpenBrowser,
		widgets.Combine("view", p.keys.PrevPanel, p.keys.NextPanel), p.keys.Refresh,
	}
}

func (p PullRequestsPanel) Update(msg tea.Msg) (PullRequestsPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if p.searching {
			return p.handleSearchKey(msg)
		}
		return p.handleKey(msg)
	case tea.PasteMsg:
		if p.searching {
			return p.updateSearch(msg)
		}
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			p.move(-1)
		case tea.MouseWheelDown:
			p.move(1)
		}
	}
	return p, nil
}

func (p PullRequestsPanel) handleKey(msg tea.KeyPressMsg) (PullRequestsPanel, tea.Cmd) {
	k := p.keys
	page := max(p.height/(styles.PullRequestRowHeight+styles.PullRequestRowGap)/2, 1)
	switch {
	case key.Matches(msg, k.Up):
		p.move(-1)
	case key.Matches(msg, k.Down):
		p.move(1)
	case key.Matches(msg, k.HalfPageUp):
		p.move(-page)
	case key.Matches(msg, k.HalfPageDown):
		p.move(page)
	case key.Matches(msg, k.Top):
		p.move(-len(p.items))
	case key.Matches(msg, k.Bottom):
		p.move(len(p.items))
	case key.Matches(msg, k.Select), key.Matches(msg, k.View):
		if pr := p.Selected(); pr != nil {
			chosen := *pr
			return p, func() tea.Msg { return ReviewPullRequestMsg{PullRequest: chosen} }
		}
	case key.Matches(msg, k.OpenBrowser):
		if pr := p.Selected(); pr != nil {
			chosen := *pr
			return p, func() tea.Msg { return BrowsePullRequestMsg{PullRequest: chosen} }
		}
	}
	return p, nil
}

func (p *PullRequestsPanel) move(delta int) {
	if len(p.items) == 0 {
		return
	}
	next := min(max(p.cursor+delta, 0), len(p.items)-1)
	if next == p.cursor {
		return
	}
	p.cursor = next
	p.render()
}

func (p *PullRequestsPanel) render() {
	if p.width <= 0 {
		return
	}
	rows := make([]string, len(p.items))
	for i := range p.items {
		rows[i] = p.row(p.items[i], i == p.cursor)
	}
	p.viewport.SetContent(strings.Join(rows, strings.Repeat("\n", styles.PullRequestRowGap+1)))
	top := p.cursor * (styles.PullRequestRowHeight + styles.PullRequestRowGap)
	widgets.Reveal(&p.viewport, top, top+styles.PullRequestRowHeight)
}

func (p *PullRequestsPanel) reviewerChips(pr models.PullRequest, base lipgloss.Style) string {
	chips := make([]string, 0, styles.PullRequestMaxChips)
	for i, r := range pr.Reviewers {
		if i == styles.PullRequestMaxChips {
			break
		}
		chips = append(chips, VoteChip(r, p.me.Is(r.ID, r.UniqueName)))
	}
	return strings.Join(chips, base.Render(" "))
}

func (p *PullRequestsPanel) row(pr models.PullRequest, selected bool) string {
	base := lipgloss.NewStyle()
	bar := " "
	if selected {
		base = base.Background(styles.ColorHighlight)
		if p.focused {
			bar = railBar
		}
	}
	pad := func(n int) string { return base.Render(strings.Repeat(" ", max(n, 0))) }
	lead := base.Foreground(styles.ColorPrimary).Render(bar) + pad(1)
	indent := styles.PullRequestColID + 1

	right := ""
	if p.width >= styles.PullRequestAgeWidth {
		age := formatCommentDate(pr.CreatedDate)
		right = pad(styles.PullRequestColAge-lipgloss.Width(age)) + base.Foreground(styles.ColorTextMuted).Render(age)
	}
	if p.width >= styles.PullRequestReviewersWidth {
		chips := p.reviewerChips(pr, base)
		right = pad(1) + chips + pad(styles.PullRequestColReviewers-lipgloss.Width(chips)) + right
	}

	id := fmt.Sprintf("%-*d", styles.PullRequestColID, pr.ID)
	room := max(p.width-2-indent-lipgloss.Width(right)-1, 1)
	draft := ""
	if pr.IsDraft {
		draft = Badge(draftLabel, styles.ColorChip, styles.ColorLight) + pad(1)
	}
	terms := p.query().Terms
	plain := ansi.Truncate(pr.Title, max(room-lipgloss.Width(draft), 1), "…")
	title := highlight(plain, terms, base.Foreground(styles.ColorText).Bold(selected))
	first := lead + base.Foreground(styles.ColorAccent).Render(id) + pad(1) + draft + title +
		pad(room-lipgloss.Width(draft)-lipgloss.Width(title)+1) + right

	detailRoom := max(p.width-2-indent, 1)
	summary := pr.Author.DisplayName + " · " + pr.SourceBranch + branchJoin + pr.TargetBranch
	if !p.oneRepo {
		summary = pr.RepositoryName + " · " + summary
	}
	detail := ansi.Truncate(summary, detailRoom, "…")
	second := lead + pad(indent) + highlight(detail, terms, base.Foreground(styles.ColorTextMuted)) + pad(detailRoom-lipgloss.Width(detail))
	return first + "\n" + second
}

func (p PullRequestsPanel) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	top := p.searchView()
	if top != "" {
		top += "\n"
	}
	if len(p.items) > 0 {
		return Block(top+p.viewport.View(), p.width, p.height)
	}
	if len(p.all) > 0 && p.query().Active() {
		return Block(top+lipgloss.NewStyle().Padding(1, 1).Render(p.noMatches()), p.width, p.height)
	}
	title, hint := "No pull requests in this view", "Switch views with "+p.keys.NextPanel.Help().Key+" or "+p.keys.Panel1.Help().Key+"-"+p.keys.Panel4.Help().Key+"."
	switch {
	case p.unscoped:
		title = "Choose a repository"
		hint = "Pull requests are listed one repository at a time. Press " + p.keys.Repository.Help().Key + " to pick one."
	case p.loading && !p.applied.Empty():
		return Block(top+lipgloss.NewStyle().Padding(1, 1).Render(Loading(p.spinner, searchingServer)), p.width, p.height)
	case p.loading:
		return Block(top+lipgloss.NewStyle().Padding(1, 1).Render(Loading(p.spinner, loadingPullRequests)), p.width, p.height)
	case !p.applied.Empty():
		title, hint = "No pull requests match these filters", "Press "+p.keys.Search.Help().Key+" and "+p.keys.Back.Help().Key+" to clear the search."
	}
	msg := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(title),
		lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(max(p.width-2, 1)).Render(hint))
	return Block(top+lipgloss.NewStyle().Padding(1, 1).Render(msg), p.width, p.height)
}
