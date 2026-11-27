package panels

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	releaseRowHeight     = 2
	starredGroup         = "Starred"
	allReleasesGroup     = "All release pipelines"
	foundGroup           = "Found on Azure DevOps"
	releaseSearchHint    = "release pipeline name"
	loadingReleaseDefs   = "Loading release pipelines"
	loadingReleases      = "Loading releases"
	loadingApprovals     = "Loading approvals"
	releaseNameMinColumn = 14
	releaseCreatedColumn = 16
	releaseColumnGap     = 2
	releaseChipMin       = 9
	releaseChipMax       = 22
	releaseChipPadding   = 4
	releaseCreatedLayout = "2006-01-02 15:04"
	starGlyph            = "★"
	DefsEnterReleases    = "its releases"
	DefsEnterApprovals   = "its approvals"
)

type FocusReleasesMsg struct{}

type StarReleaseDefinitionMsg struct {
	Definition models.ReleaseDefinition
	Star       bool
}

type MoreReleaseDefinitionsMsg struct{}

type MoreReleasesMsg struct{}

type ReleaseSearchTypedMsg struct {
	Text string
}

type OpenLatestReleaseMsg struct {
	DefinitionID int
	Name         string
}

type ApprovalCount struct {
	Mine   int
	Others int
}

type ReleaseDefinitionsPanel struct {
	list          selectList
	counts        map[int]ApprovalCount
	latest        map[int]models.DefinitionDeployment
	enterLabel    string
	defs          []models.ReleaseDefinition
	found         []models.ReleaseDefinition
	foundFor      string
	searchPending bool
	starred       map[int]bool
	search        textinput.Model
	searching     bool
	loading       bool
	more          bool
	loadingMore   bool
	keys          keys.KeyMap
	spinner       string
	width         int
	height        int
}

func NewReleaseDefinitionsPanel(k keys.KeyMap) ReleaseDefinitionsPanel {
	return ReleaseDefinitionsPanel{list: newSelectList(k), keys: k, starred: map[int]bool{}, loading: true, enterLabel: DefsEnterReleases, search: widgets.SearchInput(searchPrompt, releaseSearchHint)}
}

func (p *ReleaseDefinitionsPanel) SetApprovalCounts(counts map[int]ApprovalCount) {
	p.counts = counts
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) SetEnterLabel(label string) { p.enterLabel = label }

func (p *ReleaseDefinitionsPanel) SetLatest(latest map[int]models.DefinitionDeployment) {
	p.latest = latest
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) SetDefinitions(defs []models.ReleaseDefinition, more bool) {
	p.defs, p.more, p.loading, p.loadingMore = defs, more, false, false
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) ShowLoading() {
	p.defs, p.loading = nil, true
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) SetStarred(starred map[int]bool) {
	p.starred = starred
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) Starred(id int) bool { return p.starred[id] }

func (p *ReleaseDefinitionsPanel) More() bool { return p.more }

func (p *ReleaseDefinitionsPanel) SetLoadingMore(loading bool) {
	p.loadingMore = loading
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) SetSearchPending(pending bool) {
	p.searchPending = pending
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) SetFound(text string, defs []models.ReleaseDefinition) {
	p.foundFor, p.found, p.searchPending = text, defs, false
	p.rebuild()
}

func (p *ReleaseDefinitionsPanel) Loading() bool {
	return p.loading || p.loadingMore || p.searchPending
}

func (p *ReleaseDefinitionsPanel) SetSpinner(frame string) {
	if p.spinner == frame {
		return
	}
	p.spinner = frame
	if p.searchPending || p.loadingMore {
		p.rebuild()
	}
}

func (p *ReleaseDefinitionsPanel) SetFocused(focused bool) { p.list.setFocused(focused) }

func (p *ReleaseDefinitionsPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.search.SetWidth(max(width-4, 1))
	p.layout()
}

func (p *ReleaseDefinitionsPanel) layout() {
	p.list.setSize(p.width, max(p.height-p.searchHeight(), 1))
}

func (p *ReleaseDefinitionsPanel) searchHeight() int {
	if p.searching {
		return 2
	}
	return 0
}

func (p *ReleaseDefinitionsPanel) Searching() bool { return p.searching }

func (p *ReleaseDefinitionsPanel) Filter() string { return strings.TrimSpace(p.search.Value()) }

func (p *ReleaseDefinitionsPanel) FocusSearch() tea.Cmd {
	p.searching = true
	p.search.CursorEnd()
	p.layout()
	return p.search.Focus()
}

func (p *ReleaseDefinitionsPanel) ClearFilter() tea.Cmd {
	if p.search.Value() == "" {
		return nil
	}
	p.search.SetValue("")
	p.found, p.foundFor, p.searchPending = nil, "", false
	p.rebuild()
	return func() tea.Msg { return ReleaseSearchTypedMsg{} }
}

func (p *ReleaseDefinitionsPanel) Selected() *models.ReleaseDefinition {
	id, err := strconv.Atoi(p.list.selectedKey())
	if err != nil {
		return nil
	}
	for _, group := range [][]models.ReleaseDefinition{p.defs, p.found} {
		for i := range group {
			if group[i].ID == id {
				return &group[i]
			}
		}
	}
	return nil
}

func (p *ReleaseDefinitionsPanel) Title() string {
	if p.loading {
		return "Release pipelines"
	}
	title := fmt.Sprintf("Release pipelines %d", len(p.defs))
	if p.more {
		title += "+"
	}
	if filter := p.Filter(); filter != "" {
		title = fmt.Sprintf("Release pipelines %d of %d", p.list.count(), len(p.defs))
		if !p.searching {
			title += " · /" + filter
		}
	}
	return title
}

func (p *ReleaseDefinitionsPanel) defItem(d models.ReleaseDefinition, terms []string) listItem {
	starred := p.starred[d.ID]
	latest, deployed := p.latest[d.ID]
	return listItem{key: strconv.Itoa(d.ID), render: func(base lipgloss.Style, selected bool, width int) []string {
		nameStyle := base.Foreground(styles.ColorText).Bold(true)
		if selected {
			nameStyle = nameStyle.Foreground(styles.ColorPrimary)
		}
		name := highlight(d.Name, terms, nameStyle)
		right := approvalCountBadges(p.counts[d.ID], base)
		if right != "" {
			right += base.Render(" ")
		}
		right += starSlot(base, starred)
		muted := base.Foreground(styles.ColorTextMuted)
		var second string
		switch {
		case deployed:
			glyph, mark := stageGlyph(latest.State)
			second = base.Foreground(mark).Bold(true).Render(glyph) + base.Render(" ") + base.Foreground(styles.ColorLight).Render(latest.Stage) +
				muted.Render(" · "+latest.Release+" · "+ShortAgo(latest.When))
		case d.LastRelease != "":
			second = muted.Render(d.LastRelease + " · " + ShortAgo(d.LastCreated))
		default:
			second = muted.Render("No releases yet")
		}
		return []string{cellsLine(name, right, base, width), cellsLine(second, "", base, width)}
	}}
}

func starSlot(base lipgloss.Style, starred bool) string {
	if !starred {
		return base.Render("  ")
	}
	return base.Foreground(styles.ColorWarning).Bold(true).Render(starGlyph) + base.Render(" ")
}

func approvalCountBadges(c ApprovalCount, base lipgloss.Style) string {
	var out []string
	if c.Mine > 0 {
		out = append(out, Badge("◐ "+strconv.Itoa(c.Mine), styles.ColorOrange, styles.ColorOnAccent))
	}
	if c.Others > 0 {
		out = append(out, Badge("◐ "+strconv.Itoa(c.Others), styles.ColorChip, styles.ColorLight))
	}
	return strings.Join(out, base.Render(" "))
}

func listGroup(title string, n int, bg, fg color.Color) listItem {
	return listItem{header: strings.ToUpper(title) + " " + strconv.Itoa(n), headerBg: bg, headerFg: fg}
}

func (p *ReleaseDefinitionsPanel) rebuild() {
	filter := p.Filter()
	terms := models.SearchTerms(filter)
	matches := func(d models.ReleaseDefinition) bool {
		return models.PipelineMatches(models.Pipeline{Name: d.Name, Folder: d.Folder}, terms)
	}
	sorted := models.SortDefinitions(p.defs, func(id int) bool { return p.starred[id] })
	var starred, rest []listItem
	loaded := make(map[int]bool, len(sorted))
	for _, d := range sorted {
		loaded[d.ID] = true
		if !matches(d) {
			continue
		}
		if p.starred[d.ID] {
			starred = append(starred, p.defItem(d, terms))
		} else {
			rest = append(rest, p.defItem(d, terms))
		}
	}
	var items []listItem
	if len(starred) > 0 || (filter == "" && !p.loading) {
		items = append(items, listGroup(starGlyph+" "+starredGroup, len(starred), styles.ColorWarningTint, styles.ColorWarning))
		if len(starred) == 0 {
			items = append(items, listItem{note: "None yet. " + p.keys.ToggleFavorite.Help().Key + " stars a pipeline."})
		}
		items = append(items, starred...)
	}
	if len(rest) > 0 {
		items = append(items, listGroup(allReleasesGroup, len(rest), styles.ColorChip, styles.ColorLight))
		items = append(items, rest...)
	}
	if filter != "" && len(starred)+len(rest) == 0 && len(p.defs) > 0 {
		items = append(items, listItem{note: fmt.Sprintf("None of the %d loaded match.", len(p.defs))})
	}
	if filter != "" && p.foundFor == filter {
		var found []listItem
		for _, d := range p.found {
			if !loaded[d.ID] {
				found = append(found, p.defItem(d, terms))
			}
		}
		if len(found) > 0 {
			items = append(items, listGroup(foundGroup, len(found), styles.ColorTealTint, styles.ColorTealLight))
			items = append(items, found...)
		}
	}
	p.list.footer = ""
	switch {
	case p.searchPending:
		p.list.footer = Loading(p.spinner, searchingServer)
	case p.loadingMore:
		p.list.footer = Loading(p.spinner, "Loading more")
	case p.more && filter == "":
		p.list.footer = Muted(moreOnScroll)
	}
	p.list.setItems(items)
}

func (p *ReleaseDefinitionsPanel) KeyBindings() []key.Binding {
	k := p.keys
	if p.searching {
		back := widgets.Relabel(k.Back, "done")
		if p.search.Value() != "" {
			back = widgets.Relabel(k.Back, "clear")
		}
		return []key.Binding{widgets.Combine("match", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, p.enterLabel), p.StarBinding(), back}
	}
	star := p.StarBinding()
	bindings := []key.Binding{widgets.Relabel(k.Select, p.enterLabel), widgets.Relabel(k.View, "open latest release")}
	if p.Filter() != "" {
		bindings = append(bindings, widgets.Relabel(k.Search, "edit filter"), widgets.Relabel(k.Back, "clear filter"))
	} else {
		bindings = append(bindings, widgets.Relabel(k.Search, "search"))
	}
	return append(bindings, widgets.Relabel(k.EditPipeline, "view/edit pipeline"), star, k.CreateRelease, k.Share, k.OpenBrowser, widgets.Relabel(k.NextPanel, "pane"), k.Refresh)
}

func (p ReleaseDefinitionsPanel) Update(msg tea.Msg) (ReleaseDefinitionsPanel, tea.Cmd) {
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
		delta := 1
		if msg.Button == tea.MouseWheelUp {
			delta = -1
		}
		p.list.move(delta)
		return p, p.moreIfAtEnd()
	}
	return p, nil
}

func (p *ReleaseDefinitionsPanel) moreIfAtEnd() tea.Cmd {
	if p.more && !p.loadingMore && p.Filter() == "" && p.list.atEnd() {
		return func() tea.Msg { return MoreReleaseDefinitionsMsg{} }
	}
	return nil
}

func (p ReleaseDefinitionsPanel) handleKey(msg tea.KeyPressMsg) (ReleaseDefinitionsPanel, tea.Cmd) {
	k := p.keys
	if _, handled := p.list.handleMove(msg, k, releaseRowHeight); handled {
		return p, p.moreIfAtEnd()
	}
	if key.Matches(msg, k.Back) {
		return p, p.ClearFilter()
	}
	d := p.Selected()
	if d == nil {
		return p, nil
	}
	chosen := *d
	switch {
	case key.Matches(msg, k.Select):
		return p, func() tea.Msg { return FocusReleasesMsg{} }
	case key.Matches(msg, k.View):
		return p, func() tea.Msg { return OpenLatestReleaseMsg{DefinitionID: chosen.ID, Name: chosen.Name} }
	case key.Matches(msg, k.EditPipeline):
		return p, func() tea.Msg { return OpenReleaseEditorMsg{Kind: models.DocDefinition, ID: chosen.ID} }
	case key.Matches(msg, k.ToggleFavorite):
		return p, p.starCmd()
	case key.Matches(msg, k.OpenBrowser):
		return p, func() tea.Msg { return BrowseReleaseMsg{DefinitionID: chosen.ID, URL: chosen.WebURL} }
	case key.Matches(msg, k.Share), key.Matches(msg, k.ShareLink):
		now, _ := ShareRequested(k, msg)
		return p, shareCmd(share.ReleaseDefinition(chosen), now)
	}
	return p, nil
}

func (p *ReleaseDefinitionsPanel) closeSearch() {
	p.searching = false
	p.search.Blur()
	p.layout()
	p.rebuild()
}

func (p ReleaseDefinitionsPanel) handleSearchKey(msg tea.KeyPressMsg) (ReleaseDefinitionsPanel, tea.Cmd) {
	k := p.keys
	arrow := msg.Text == ""
	switch {
	case key.Matches(msg, k.Back):
		if p.search.Value() == "" {
			p.closeSearch()
			return p, nil
		}
		return p, p.ClearFilter()
	case key.Matches(msg, k.Select):
		p.closeSearch()
		if p.Selected() == nil {
			return p, nil
		}
		return p, func() tea.Msg { return FocusReleasesMsg{} }
	case key.Matches(msg, k.PickerUp), arrow && key.Matches(msg, k.Up):
		p.list.move(-1)
		return p, nil
	case key.Matches(msg, k.PickerDown), arrow && key.Matches(msg, k.Down):
		p.list.move(1)
		return p, nil
	case key.Matches(msg, k.ToggleFavorite):
		return p, p.starCmd()
	}
	return p.updateSearch(msg)
}

func (p *ReleaseDefinitionsPanel) StarBinding() key.Binding {
	if d := p.Selected(); d != nil && p.starred[d.ID] {
		return widgets.Relabel(p.keys.ToggleFavorite, "unstar")
	}
	return widgets.Relabel(p.keys.ToggleFavorite, "star")
}

func (p *ReleaseDefinitionsPanel) starCmd() tea.Cmd {
	d := p.Selected()
	if d == nil {
		return nil
	}
	return StarCmd(*d, !p.starred[d.ID])
}

func StarCmd(d models.ReleaseDefinition, star bool) tea.Cmd {
	return func() tea.Msg { return StarReleaseDefinitionMsg{Definition: d, Star: star} }
}

func (p ReleaseDefinitionsPanel) updateSearch(msg tea.Msg) (ReleaseDefinitionsPanel, tea.Cmd) {
	before := p.search.Value()
	var cmd tea.Cmd
	p.search, cmd = p.search.Update(msg)
	if p.search.Value() == before {
		return p, cmd
	}
	p.rebuild()
	text := p.Filter()
	return p, tea.Batch(cmd, func() tea.Msg { return ReleaseSearchTypedMsg{Text: text} })
}

func (p ReleaseDefinitionsPanel) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	top := ""
	if p.searching {
		top = Line(p.search.View(), p.width) + "\n" + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(strings.Repeat(ruleGlyph, p.width)) + "\n"
	}
	h := max(p.height-p.searchHeight(), 1)
	switch {
	case len(p.list.items) > 0:
		return Block(top+p.list.view(), p.width, p.height)
	case p.loading:
		return Block(top+lipgloss.NewStyle().Padding(1, 1).Render(Loading(p.spinner, loadingReleaseDefs)), p.width, p.height)
	case p.Filter() != "":
		return Block(top+emptyState("No release pipelines match", "Press "+p.keys.Back.Help().Key+" to clear the filter.", p.width, h), p.width, p.height)
	}
	return emptyState("No release pipelines", "This project has no classic release pipelines.", p.width, p.height)
}

type ReleasesPanel struct {
	list        selectList
	releases    []models.Release
	loading     bool
	more        bool
	loadingMore bool
	definition  string
	keys        keys.KeyMap
	spinner     string
	width       int
	height      int
}

func NewReleasesPanel(k keys.KeyMap) ReleasesPanel {
	return ReleasesPanel{list: newSelectList(k), keys: k}
}

func (p *ReleasesPanel) SetReleases(definition string, releases []models.Release, more bool) {
	p.definition, p.releases, p.more, p.loading, p.loadingMore = definition, releases, more, false, false
	p.rebuild()
}

func (p *ReleasesPanel) ShowLoading(definition string) {
	p.definition, p.releases, p.loading, p.more = definition, nil, true, false
	p.rebuild()
}

func (p *ReleasesPanel) Releases() []models.Release { return p.releases }

func (p *ReleasesPanel) SetLoadingMore(loading bool) {
	p.loadingMore = loading
	p.rebuild()
}

func (p *ReleasesPanel) Loading() bool { return p.loading || p.loadingMore }

func (p *ReleasesPanel) SetSpinner(frame string) { p.spinner = frame }

func (p *ReleasesPanel) SetFocused(focused bool) { p.list.setFocused(focused) }

func (p *ReleasesPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.list.setSize(width, height)
	p.rebuild()
}

func (p *ReleasesPanel) Selected() *models.Release {
	id, err := strconv.Atoi(p.list.selectedKey())
	if err != nil {
		return nil
	}
	for i := range p.releases {
		if p.releases[i].ID == id {
			return &p.releases[i]
		}
	}
	return nil
}

func (p *ReleasesPanel) Title() string {
	if p.definition == "" {
		return "Releases"
	}
	if p.loading {
		return p.definition + " · Releases"
	}
	return fmt.Sprintf("%s · Releases %d", p.definition, len(p.releases))
}

type releaseColumns struct {
	name    int
	created int
	chip    int
	stages  int
}

func layoutReleaseColumns(releases []models.Release, width int) releaseColumns {
	longest, stages := 0, 0
	for _, r := range releases {
		stages = max(stages, len(r.Stages))
		for _, s := range r.Stages {
			longest = max(longest, lipgloss.Width(s.Name))
		}
	}
	c := releaseColumns{created: releaseCreatedColumn, stages: stages}
	c.chip = min(max(longest+releaseChipPadding, releaseChipMin), releaseChipMax)
	strip := func() int { return stages*c.chip + max(stages-1, 0)*lipgloss.Width(stageChipGap) }
	room := func() int { return width - c.created - strip() - 2*releaseColumnGap }
	for room() < releaseNameMinColumn && c.chip > releaseChipMin {
		c.chip--
	}
	if room() < releaseNameMinColumn {
		c.created = 0
	}
	c.name = max(room(), releaseNameMinColumn)
	if c.created == 0 {
		c.name = max(width-strip()-releaseColumnGap, releaseNameMinColumn)
	}
	return c
}

func padCell(base lipgloss.Style, text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = ansi.Truncate(text, width, "…")
	return text + base.Render(strings.Repeat(" ", max(width-lipgloss.Width(text), 0)))
}

func (p *ReleasesPanel) columnHeader(c releaseColumns, width int) string {
	head := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true)
	gap := strings.Repeat(" ", releaseColumnGap)
	line := padCell(lipgloss.NewStyle(), head.Render("Releases"), c.name) + gap
	if c.created > 0 {
		line += padCell(lipgloss.NewStyle(), head.Render("Created"), c.created) + gap
	}
	return Line(line+head.Render("Stages"), width)
}

func (p *ReleasesPanel) rebuild() {
	current := models.CurrentDeployments(p.releases)
	items := make([]listItem, 0, len(p.releases)+1)
	width := max(p.list.width-2, 1)
	releases := p.releases
	memo := map[int]releaseColumns{}
	colsFor := func(w int) releaseColumns {
		c, ok := memo[w]
		if !ok {
			c = layoutReleaseColumns(releases, w)
			memo[w] = c
		}
		return c
	}
	if len(p.releases) > 0 {
		items = append(items, listItem{line: p.columnHeader(colsFor(width), width)})
	}
	for i := range p.releases {
		r := p.releases[i]
		now := current[r.ID]
		items = append(items, listItem{key: strconv.Itoa(r.ID), render: func(base lipgloss.Style, selected bool, width int) []string {
			c := colsFor(width)
			gap := base.Render(strings.Repeat(" ", releaseColumnGap))
			muted := base.Foreground(styles.ColorTextMuted)
			first := padCell(base, base.Foreground(styles.ColorAccent).Bold(true).Render(r.Name), c.name) + gap
			meta := muted.Render(r.ArtifactSummary())
			second := padCell(base, meta, c.name) + gap
			if c.created > 0 {
				stamp := ""
				if !r.Created.IsZero() {
					stamp = r.Created.Local().Format(releaseCreatedLayout)
				}
				first += padCell(base, base.Foreground(styles.ColorLight).Render(stamp), c.created) + gap
				if lipgloss.Width(meta) > c.name {
					long := r.ArtifactSummary()
					if r.CreatedBy.DisplayName != "" {
						long += " · " + r.CreatedBy.DisplayName
					}
					second = padCell(base, muted.Render(long), c.name+releaseColumnGap+c.created) + gap
				} else {
					second += padCell(base, muted.Render(r.CreatedBy.DisplayName), c.created) + gap
				}
			}
			stripWidth := max(width-lipgloss.Width(first), 1)
			first += deploymentChips(r.Stages, now, c.chip, stripWidth, base)
			return []string{cellsLine(first, "", base, width), cellsLine(second, "", base, width)}
		}})
	}
	p.list.footer = ""
	switch {
	case p.loadingMore:
		p.list.footer = Loading(p.spinner, "Loading more")
	case p.more:
		p.list.footer = Muted(moreOnScroll)
	}
	p.list.setItems(items)
}

func (p *ReleasesPanel) KeyBindings() []key.Binding {
	k := p.keys
	if p.Selected() == nil {
		return []key.Binding{k.CreateRelease, widgets.Relabel(k.NextPanel, "pane"), k.Refresh}
	}
	bindings := []key.Binding{widgets.Relabel(k.Select, "open release"), widgets.Relabel(k.EditPipeline, "edit release"), k.CreateRelease}
	if r := p.Selected(); r != nil && hasPendingApproval(*r) {
		bindings = append(bindings, k.Approve)
	}
	return append(bindings, k.Share, k.OpenBrowser, widgets.Relabel(k.NextPanel, "pane"), widgets.Relabel(k.Back, "release pipelines"), k.Refresh)
}

func hasPendingApproval(r models.Release) bool {
	for _, s := range r.Stages {
		if _, ok := s.PendingApproval(); ok {
			return true
		}
	}
	return false
}

func (p ReleasesPanel) Update(msg tea.Msg) (ReleasesPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(msg)
	case tea.MouseWheelMsg:
		delta := 1
		if msg.Button == tea.MouseWheelUp {
			delta = -1
		}
		p.list.move(delta)
		return p, p.moreIfAtEnd()
	}
	return p, nil
}

func (p *ReleasesPanel) moreIfAtEnd() tea.Cmd {
	if p.more && !p.loadingMore && p.list.atEnd() {
		return func() tea.Msg { return MoreReleasesMsg{} }
	}
	return nil
}

func (p ReleasesPanel) handleKey(msg tea.KeyPressMsg) (ReleasesPanel, tea.Cmd) {
	k := p.keys
	if _, handled := p.list.handleMove(msg, k, releaseRowHeight); handled {
		return p, p.moreIfAtEnd()
	}
	r := p.Selected()
	if r == nil {
		return p, nil
	}
	chosen := *r
	switch {
	case key.Matches(msg, k.Select), key.Matches(msg, k.View):
		return p, func() tea.Msg { return OpenReleaseMsg{ReleaseID: chosen.ID} }
	case key.Matches(msg, k.EditPipeline):
		return p, func() tea.Msg { return OpenReleaseEditorMsg{Kind: models.DocRelease, ID: chosen.ID} }
	case key.Matches(msg, k.OpenBrowser):
		return p, func() tea.Msg { return BrowseReleaseMsg{ReleaseID: chosen.ID, URL: chosen.WebURL} }
	case key.Matches(msg, k.Share), key.Matches(msg, k.ShareLink):
		now, _ := ShareRequested(k, msg)
		return p, shareCmd(share.Release(chosen), now)
	case key.Matches(msg, k.Approve):
		for _, s := range chosen.Stages {
			if a, ok := s.PendingApproval(); ok {
				return p, func() tea.Msg { return ApproveReleaseMsg{Approval: a} }
			}
		}
		return p, func() tea.Msg { return ReleaseNoticeMsg{Text: chosen.Name + " has no approval waiting."} }
	}
	return p, nil
}

func (p ReleasesPanel) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	switch {
	case p.list.count() > 0:
		return p.list.view()
	case p.loading:
		return Block(lipgloss.NewStyle().Padding(1, 1).Render(Loading(p.spinner, loadingReleases)), p.width, p.height)
	case p.definition == "":
		return emptyState("Choose a release pipeline", "Its releases are listed here.", p.width, p.height)
	}
	return emptyState("No releases yet", "Press "+p.keys.CreateRelease.Help().Key+" to create the first one.", p.width, p.height)
}

type ApprovalsPanel struct {
	list      selectList
	mine      []models.ReleaseApproval
	others    []models.ReleaseApproval
	scopeID   int
	scopeName string
	loading   bool
	keys      keys.KeyMap
	spinner   string
	width     int
	height    int
}

func NewApprovalsPanel(k keys.KeyMap) ApprovalsPanel {
	return ApprovalsPanel{list: newSelectList(k), keys: k, loading: true}
}

func (p *ApprovalsPanel) SetApprovals(mine, others []models.ReleaseApproval) {
	p.mine, p.others, p.loading = mine, others, false
	p.rebuild()
}

func (p *ApprovalsPanel) SetScope(definitionID int, name string) {
	if p.scopeID == definitionID && p.scopeName == name {
		return
	}
	p.scopeID, p.scopeName = definitionID, name
	p.rebuild()
}

func (p *ApprovalsPanel) Scoped() bool { return p.scopeID != 0 }

func (p *ApprovalsPanel) Counts() map[int]ApprovalCount {
	out := map[int]ApprovalCount{}
	for _, a := range p.mine {
		c := out[a.DefinitionID]
		c.Mine++
		out[a.DefinitionID] = c
	}
	for _, a := range p.others {
		c := out[a.DefinitionID]
		c.Others++
		out[a.DefinitionID] = c
	}
	return out
}

func (p *ApprovalsPanel) inScope(approvals []models.ReleaseApproval) []models.ReleaseApproval {
	if p.scopeID == 0 {
		return approvals
	}
	var out []models.ReleaseApproval
	for _, a := range approvals {
		if a.DefinitionID == p.scopeID {
			out = append(out, a)
		}
	}
	return out
}

func (p *ApprovalsPanel) ShowLoading() {
	p.loading = true
	p.rebuild()
}

func (p *ApprovalsPanel) Loading() bool { return p.loading }

func (p *ApprovalsPanel) SetSpinner(frame string) { p.spinner = frame }

func (p *ApprovalsPanel) SetFocused(focused bool) { p.list.setFocused(focused) }

func (p *ApprovalsPanel) Mine() int { return len(p.mine) }

func (p *ApprovalsPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.list.setSize(width, height)
}

func (p *ApprovalsPanel) Selected() (*models.ReleaseApproval, bool) {
	key := p.list.selectedKey()
	for _, group := range []struct {
		items []models.ReleaseApproval
		mine  bool
	}{{p.mine, true}, {p.others, false}} {
		for i := range group.items {
			if approvalKey(group.items[i], group.mine) == key {
				return &group.items[i], group.mine
			}
		}
	}
	return nil, false
}

func approvalKey(a models.ReleaseApproval, mine bool) string {
	return fmt.Sprintf("%t-%d", mine, a.ID)
}

func (p *ApprovalsPanel) Title() string {
	shown := len(p.inScope(p.mine)) + len(p.inScope(p.others))
	if p.Scoped() {
		return fmt.Sprintf("Approvals · %s %d", p.scopeName, shown)
	}
	return fmt.Sprintf("Approvals %d · all pipelines", shown)
}

func (p *ApprovalsPanel) rebuild() {
	var items []listItem
	add := func(title string, approvals []models.ReleaseApproval, mine bool) {
		if len(approvals) == 0 {
			return
		}
		items = append(items, listItem{header: fmt.Sprintf("%s %d", title, len(approvals))})
		for i := range approvals {
			a := approvals[i]
			items = append(items, listItem{key: approvalKey(a, mine), render: func(base lipgloss.Style, selected bool, width int) []string {
				what := base.Foreground(styles.ColorText).Bold(true).Render(a.StageName) +
					base.Foreground(styles.ColorTextMuted).Render(" · ") + base.Foreground(styles.ColorText).Render(a.ReleaseName)
				if !p.Scoped() {
					what += base.Foreground(styles.ColorTextMuted).Render(" · " + a.Definition)
				}
				who := approverName(a)
				if mine {
					who += " · you can approve"
				}
				when := base.Foreground(styles.ColorTextMuted).Render(ShortAgo(a.Created))
				indent := base.Render(strings.Repeat(" ", runBadgeWidth+3))
				return []string{
					cellsLine(approvalBadge(a, mine)+base.Render(" ")+what, when, base, width),
					cellsLine(indent+base.Foreground(styles.ColorTextMuted).Render(who), "", base, width),
				}
			}})
		}
	}
	add("Waiting on you", p.inScope(p.mine), true)
	add("Waiting on others", p.inScope(p.others), false)
	p.list.setItems(items)
}

func (p *ApprovalsPanel) KeyBindings() []key.Binding {
	k := p.keys
	scope := p.ScopeBinding()
	a, mine := p.Selected()
	if a == nil {
		return []key.Binding{scope, widgets.Combine("view", k.Panel1, k.Panel2), k.Refresh}
	}
	bindings := []key.Binding{widgets.Relabel(k.Select, "open release")}
	if mine {
		bindings = append(bindings, k.Approve)
	}
	return append(bindings, scope, k.Share, k.OpenBrowser, widgets.Relabel(k.NextPanel, "pane"), widgets.Combine("view", k.Panel1, k.Panel2), k.Refresh)
}

func (p *ApprovalsPanel) ScopeBinding() key.Binding {
	if p.Scoped() {
		return widgets.Relabel(p.keys.AllApprovals, "all pipelines")
	}
	return widgets.Relabel(p.keys.AllApprovals, "this pipeline")
}

func (p ApprovalsPanel) Update(msg tea.Msg) (ApprovalsPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(msg)
	case tea.MouseWheelMsg:
		delta := 1
		if msg.Button == tea.MouseWheelUp {
			delta = -1
		}
		p.list.move(delta)
	}
	return p, nil
}

func (p ApprovalsPanel) handleKey(msg tea.KeyPressMsg) (ApprovalsPanel, tea.Cmd) {
	k := p.keys
	if _, handled := p.list.handleMove(msg, k, releaseRowHeight); handled {
		return p, nil
	}
	a, mine := p.Selected()
	if a == nil {
		return p, nil
	}
	chosen := *a
	switch {
	case key.Matches(msg, k.Select):
		return p, func() tea.Msg { return OpenReleaseMsg{ReleaseID: chosen.ReleaseID} }
	case key.Matches(msg, k.OpenBrowser):
		return p, func() tea.Msg { return BrowseReleaseMsg{ReleaseID: chosen.ReleaseID} }
	case key.Matches(msg, k.Share), key.Matches(msg, k.ShareLink):
		now, _ := ShareRequested(k, msg)
		return p, func() tea.Msg {
			return ShareReleaseMsg{ReleaseID: chosen.ReleaseID, Name: chosen.ReleaseName, Definition: chosen.Definition, Now: now}
		}
	case key.Matches(msg, k.Approve) && mine:
		return p, func() tea.Msg { return ApproveReleaseMsg{Approval: chosen} }
	case key.Matches(msg, k.Approve):
		return p, func() tea.Msg {
			return ReleaseNoticeMsg{Text: "This approval is assigned to " + chosen.Approver.DisplayName + ", not to you."}
		}
	}
	return p, nil
}

func (p ApprovalsPanel) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	switch {
	case p.list.count() > 0:
		return p.list.view()
	case p.loading:
		return Block(lipgloss.NewStyle().Padding(1, 1).Render(Loading(p.spinner, loadingApprovals)), p.width, p.height)
	}
	if p.Scoped() {
		return emptyState("No approvals waiting on "+p.scopeName, "Press "+p.keys.AllApprovals.Help().Key+" to see every pipeline's approvals.", p.width, p.height)
	}
	return emptyState("No approvals waiting", "Pending release approvals in this project show up here.", p.width, p.height)
}
