package panels

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	searchPrompt      = "/ "
	searchPlaceholder = "title, ID, branch or author"
	maxSuggestions    = 5
	serverTag         = "server"
)

type SearchPullRequestsMsg struct {
	Criteria models.PullRequestCriteria
}

type OpenPullRequestByIDMsg struct {
	ID int
}

type searchSuggestion struct {
	label  string
	detail string
	insert string
	server bool
	final  bool
}

var qualifierHelp = map[string][2]string{
	models.QualifierAuthor:                 {"name or @me", "created by"},
	models.QualifierReviewer:               {"name or @me", "has this reviewer"},
	models.QualifierSource:                 {"branch", "from this branch"},
	models.QualifierTarget:                 {"branch", "into this branch"},
	models.QualifierIs + models.DraftValue: {"", "drafts only"},
}

func (p *PullRequestsPanel) query() models.PullRequestQuery {
	return models.ParsePullRequestQuery(p.search.Value())
}

func (p *PullRequestsPanel) Searching() bool {
	return p.searching
}

func (p *PullRequestsPanel) SetMembers(members []models.TeamMember) {
	p.members = members
}

func (p *PullRequestsPanel) FocusSearch() tea.Cmd {
	p.searching = true
	p.layout()
	return p.search.Focus()
}

func (p *PullRequestsPanel) blurSearch() {
	p.searching = false
	p.search.Blur()
	p.layout()
}

func (p *PullRequestsPanel) searchShown() bool {
	return p.searching || p.search.Value() != ""
}

func (p *PullRequestsPanel) applyFilter() {
	selected := 0
	if current := p.Selected(); current != nil {
		selected = current.ID
	}
	q := p.query()
	p.items = p.items[:0:0]
	for _, pr := range p.all {
		if q.Matches(pr, p.me) {
			p.items = append(p.items, pr)
		}
	}
	p.cursor = 0
	for i := range p.items {
		if p.items[i].ID == selected {
			p.cursor = i
		}
	}
	p.layout()
}

func (p PullRequestsPanel) handleSearchKey(msg tea.KeyPressMsg) (PullRequestsPanel, tea.Cmd) {
	k := p.keys
	arrow := msg.Text == ""
	switch {
	case key.Matches(msg, k.Back):
		return p.clearOrLeave()
	case key.Matches(msg, k.Select):
		return p.submitSearch()
	case key.Matches(msg, k.NextPanel):
		p.complete()
		return p, nil
	case arrow && key.Matches(msg, k.Up):
		p.move(-1)
		return p, nil
	case arrow && key.Matches(msg, k.Down):
		p.move(1)
		return p, nil
	}
	return p.updateSearch(msg)
}

func (p PullRequestsPanel) updateSearch(msg tea.Msg) (PullRequestsPanel, tea.Cmd) {
	before := p.search.Value()
	var cmd tea.Cmd
	p.search, cmd = p.search.Update(msg)
	if p.search.Value() != before {
		p.searchErr = ""
		p.applyFilter()
	}
	return p, cmd
}

func (p PullRequestsPanel) clearOrLeave() (PullRequestsPanel, tea.Cmd) {
	if p.search.Value() == "" {
		p.blurSearch()
		return p, nil
	}
	p.search.SetValue("")
	p.searchErr = ""
	p.applyFilter()
	if p.applied.Empty() {
		return p, nil
	}
	p.applied = models.PullRequestCriteria{}
	return p, func() tea.Msg { return SearchPullRequestsMsg{} }
}

func (p PullRequestsPanel) submitSearch() (PullRequestsPanel, tea.Cmd) {
	q := p.query()
	criteria, err := q.Criteria(p.members, p.me)
	if err != nil {
		p.searchErr = err.Error()
		p.layout()
		return p, nil
	}
	if criteria != p.applied {
		p.applied = criteria
		p.blurSearch()
		return p, func() tea.Msg { return SearchPullRequestsMsg{Criteria: criteria} }
	}
	if pr := p.Selected(); pr != nil {
		chosen := *pr
		p.blurSearch()
		return p, func() tea.Msg { return ReviewPullRequestMsg{PullRequest: chosen} }
	}
	if id, ok := q.ID(); ok {
		p.blurSearch()
		return p, func() tea.Msg { return OpenPullRequestByIDMsg{ID: id} }
	}
	p.blurSearch()
	return p, nil
}

func (p *PullRequestsPanel) suggestions() []searchSuggestion {
	if !p.searching {
		return nil
	}
	value := p.search.Value()
	token := p.query().Last
	if value != "" && token == "" {
		return nil
	}
	lower := strings.ToLower(token)
	for _, prefix := range []string{models.QualifierAuthor, models.QualifierReviewer} {
		if strings.HasPrefix(lower, prefix) {
			return p.personSuggestions(prefix, token[len(prefix):])
		}
	}
	for _, prefix := range []string{models.QualifierSource, models.QualifierTarget} {
		if strings.HasPrefix(lower, prefix) {
			return p.branchSuggestions(prefix, token[len(prefix):])
		}
	}
	var out []searchSuggestion
	for _, q := range models.Qualifiers {
		if strings.HasPrefix(q, lower) && q != lower {
			help := qualifierHelp[q]
			out = append(out, searchSuggestion{label: q, detail: help[1], insert: q, server: !strings.HasPrefix(q, models.QualifierIs), final: strings.HasPrefix(q, models.QualifierIs)})
		}
	}
	return out
}

func (p *PullRequestsPanel) personSuggestions(prefix, partial string) []searchSuggestion {
	partial = strings.Trim(partial, `"`)
	var out []searchSuggestion
	if strings.HasPrefix(models.MeAlias, strings.ToLower(partial)) {
		out = append(out, searchSuggestion{label: prefix + models.MeAlias, detail: "you", insert: prefix + models.MeAlias, server: true, final: true})
	}
	for _, m := range models.MatchMembers(partial, p.members) {
		out = append(out, searchSuggestion{label: prefix + m.DisplayName, detail: m.UniqueName, insert: prefix + quoteIfSpaced(m.DisplayName), server: true, final: true})
	}
	return out[:min(len(out), maxSuggestions)]
}

func (p *PullRequestsPanel) branchSuggestions(prefix, partial string) []searchSuggestion {
	var branches []string
	for _, pr := range p.all {
		for _, b := range []string{pr.SourceBranch, pr.TargetBranch} {
			if strings.Contains(strings.ToLower(b), strings.ToLower(partial)) && !slices.Contains(branches, b) {
				branches = append(branches, b)
			}
		}
	}
	slices.Sort(branches)
	out := make([]searchSuggestion, 0, maxSuggestions)
	for _, b := range branches[:min(len(branches), maxSuggestions)] {
		out = append(out, searchSuggestion{label: prefix + b, insert: prefix + b, server: true, final: true})
	}
	return out
}

func quoteIfSpaced(s string) string {
	if strings.Contains(s, " ") {
		return `"` + s + `"`
	}
	return s
}

func (p *PullRequestsPanel) complete() {
	options := p.suggestions()
	if len(options) == 0 {
		return
	}
	value := p.search.Value()
	token := p.query().Last
	next := strings.TrimSuffix(value, token) + options[0].insert
	if options[0].final {
		next += " "
	}
	p.search.SetValue(next)
	p.search.CursorEnd()
	p.applyFilter()
}

func (p *PullRequestsPanel) searchChromeHeight() int {
	if !p.searchShown() {
		return 0
	}
	h := 2 + len(p.suggestions())
	if p.searchErr != "" {
		h++
	}
	return h
}

func (p *PullRequestsPanel) searchView() string {
	if !p.searchShown() {
		return ""
	}
	count := Muted(fmt.Sprintf("%d of %d", len(p.items), len(p.all)))
	if !p.applied.Empty() {
		count += Muted(" · " + serverTag)
	}
	bar := justify(p.search.View(), count, p.width)
	lines := []string{bar}
	for i, s := range p.suggestions() {
		lines = append(lines, p.suggestionRow(s, i == 0))
	}
	if p.searchErr != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorError).Render(ansi.Truncate(p.searchErr, max(p.width, 1), "…")))
	}
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat(ruleGlyph, max(p.width, 0)))
	return strings.Join(append(lines, rule), "\n")
}

func (p *PullRequestsPanel) suggestionRow(s searchSuggestion, first bool) string {
	base := lipgloss.NewStyle()
	bar := " "
	if first {
		base = base.Background(styles.ColorHighlight)
		bar = railBar
	}
	right := ""
	if s.server {
		right = Badge(serverTag, styles.ColorChip, styles.ColorLight)
	}
	left := base.Foreground(styles.ColorPrimary).Render(bar+" "+s.label) + base.Foreground(styles.ColorTextMuted).Render("  "+s.detail)
	return base.Width(p.width).Render(justify(left, right, p.width))
}

func (p *PullRequestsPanel) noMatches() string {
	q := p.query()
	if id, ok := q.ID(); ok {
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(fmt.Sprintf("!%d isn't in this view", id)),
			Muted("Press "+p.keys.Select.Help().Key+" to open it anyway."))
	}
	title := fmt.Sprintf("No pull requests match %q", strings.TrimSpace(p.search.Value()))
	hint := "Text only matches the " + strconv.Itoa(len(p.all)) + " pull requests loaded in this view, because Azure DevOps can't search text. " +
		models.QualifierAuthor + ", " + models.QualifierReviewer + ", " + models.QualifierSource + " and " + models.QualifierTarget + " search the server, or try another view."
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(ansi.Truncate(title, max(p.width-2, 1), "…")),
		lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(max(p.width-2, 1)).Render(hint))
}

func (p *PullRequestsPanel) searchBindings() []key.Binding {
	k := p.keys
	enter := widgets.Relabel(k.Select, "open review")
	if criteria, err := p.query().Criteria(p.members, p.me); err == nil && criteria != p.applied {
		enter = widgets.Relabel(k.Select, "search server")
	}
	bindings := []key.Binding{enter}
	if len(p.suggestions()) > 0 {
		bindings = append(bindings, widgets.Relabel(k.NextPanel, "complete"))
	}
	back := widgets.Relabel(k.Back, "done")
	if p.search.Value() != "" {
		back = widgets.Relabel(k.Back, "clear")
	}
	return append(bindings, widgets.Combine("move", k.PickerUp, k.PickerDown), back)
}

func highlight(text string, terms []string, base lipgloss.Style) string {
	if len(terms) == 0 {
		return base.Render(text)
	}
	lower := strings.ToLower(text)
	marks := make([]bool, len(text))
	for _, t := range terms {
		t = strings.TrimPrefix(t, "!")
		if t == "" || len(lower) != len(text) {
			continue
		}
		for from := 0; ; {
			i := strings.Index(lower[from:], t)
			if i < 0 {
				break
			}
			for j := from + i; j < from+i+len(t); j++ {
				marks[j] = true
			}
			from += i + len(t)
		}
	}
	hit := base.Background(styles.ColorMatch).Foreground(styles.ColorText)
	var b strings.Builder
	for start := 0; start < len(text); {
		end := start
		for end < len(text) && marks[end] == marks[start] {
			end++
		}
		style := base
		if marks[start] {
			style = hit
		}
		b.WriteString(style.Render(text[start:end]))
		start = end
	}
	return b.String()
}
