package ui

import (
	"slices"
	"strconv"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	releasesTab = iota
	approvalsTab
)

const (
	paneDefinitions = iota
	paneReleases
)

const (
	releasesPage          = 25
	releaseDefsMinWidth   = 30
	releaseDefsMaxWidth   = 48
	releaseDefsRatio      = 0.34
	releaseTwoPanesWidth  = 80
	approvalPreviewWidth  = 44
	approvalPreviewShow   = 110
	releaseSearchDebounce = 300 * time.Millisecond
)

var releasePivots = []string{"Releases", "Approvals"}

type releaseState struct {
	defsLoad  inflight
	search    inflight
	defs      panels.ReleaseDefinitionsPanel
	list      panels.ReleasesPanel
	approvals panels.ApprovalsPanel
	view      panels.ReleaseView
	pivot     int
	pane      int

	loaded      bool
	loading     bool
	seq         int
	searchSeq   int
	defsToken   string
	definitions []models.ReleaseDefinition
	adoStars    []int
	localStars  []int

	listFor   int
	listSeq   int
	listToken string
	releases  []models.Release
	pollSeq   int

	approvalsLoaded  bool
	approvalsLoading bool
	approvalsSeq     int
	previewFor       int
	preview          *models.Release

	viewSeq int
}

type releaseDefinitionsMsg struct {
	seq     int
	defs    []models.ReleaseDefinition
	token   string
	starred []int
	more    bool
	err     error
}

type releasesDueMsg struct {
	definition int
}

type releasesLoadedMsg struct {
	seq        int
	definition int
	releases   []models.Release
	token      string
	more       bool
	silent     bool
	err        error
}

type releasePollMsg struct {
	seq int
}

type approvalsLoadedMsg struct {
	seq    int
	mine   []models.ReleaseApproval
	others []models.ReleaseApproval
	err    error
}

type approvalPreviewDueMsg struct {
	release int
}

type approvalPreviewMsg struct {
	release models.Release
}

func loadReleaseDefinitionsCmd(client *api.Client, seq int, search, token string, more bool) tea.Cmd {
	return func() tea.Msg {
		msg := releaseDefinitionsMsg{seq: seq, more: more}
		msg.defs, msg.token, msg.err = client.ListReleaseDefinitions(search, token)
		if msg.err == nil && !more {
			msg.starred, _ = client.GetFavoriteReleaseDefinitionIDs()
		}
		return msg
	}
}

func loadReleasesCmd(client *api.Client, seq, definition int, token string, more, silent bool) tea.Cmd {
	return func() tea.Msg {
		msg := releasesLoadedMsg{seq: seq, definition: definition, more: more, silent: silent}
		msg.releases, msg.token, msg.err = client.ListReleases(definition, releasesPage, token)
		return msg
	}
}

func loadApprovalsCmd(client *api.Client, seq int, me models.Identity) tea.Cmd {
	return func() tea.Msg {
		msg := approvalsLoadedMsg{seq: seq}
		all, err := client.ListReleaseApprovals("")
		if err != nil {
			msg.err = err
			return msg
		}
		assignee := me.UniqueName
		if assignee == "" {
			assignee = me.ID
		}
		if assignee != "" {
			msg.mine, _ = client.ListReleaseApprovals(assignee)
		}
		mine := make(map[int]bool, len(msg.mine))
		for _, a := range msg.mine {
			mine[a.ID] = true
		}
		for _, a := range all {
			if !mine[a.ID] {
				msg.others = append(msg.others, a)
			}
		}
		return msg
	}
}

type releaseSearchDueMsg struct {
	seq  int
	text string
}

type releaseSearchMsg struct {
	seq  int
	text string
	defs []models.ReleaseDefinition
}

type latestReleaseMsg struct {
	id   int
	name string
	err  error
}

func scheduleReleaseSearchCmd(seq int, text string) tea.Cmd {
	return tea.Tick(releaseSearchDebounce, func(time.Time) tea.Msg { return releaseSearchDueMsg{seq: seq, text: text} })
}

func findReleaseDefinitionsCmd(client *api.Client, seq int, text string) tea.Cmd {
	return func() tea.Msg {
		defs, _, err := client.ListReleaseDefinitions(text, "")
		if err != nil {
			defs = nil
		}
		return releaseSearchMsg{seq: seq, text: text, defs: defs}
	}
}

func latestReleaseCmd(client *api.Client, definition int, name string) tea.Cmd {
	return func() tea.Msg {
		releases, _, err := client.ListReleases(definition, 1, "")
		msg := latestReleaseMsg{name: name, err: err}
		if err == nil && len(releases) > 0 {
			msg.id = releases[0].ID
		}
		return msg
	}
}

func scheduleReleasesCmd(definition int) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg { return releasesDueMsg{definition: definition} })
}

func scheduleReleasePollCmd(seq int) tea.Cmd {
	return tea.Tick(listPollInterval, func(time.Time) tea.Msg { return releasePollMsg{seq: seq} })
}

func scheduleApprovalPreviewCmd(release int) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg { return approvalPreviewDueMsg{release: release} })
}

func loadApprovalPreviewCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		r, err := client.GetRelease(id)
		if err != nil {
			return nil
		}
		return approvalPreviewMsg{release: r}
	}
}

func (a *App) inReleases() bool { return a.hub == panels.HubReleases }

func (a *App) searchingReleases() bool {
	return a.viewMode == ViewMain && a.inReleases() && !a.modals.open() && a.rl.defs.Searching()
}

func (a *App) openReleases() tea.Cmd {
	a.switchHub(panels.HubReleases)
	if a.rl.loaded || a.rl.loading {
		return nil
	}
	return a.loadReleaseDefinitions()
}

func (a *App) loadReleaseDefinitions() tea.Cmd {
	a.rl.seq++
	a.rl.loading, a.rl.loaded = true, false
	a.rl.defs.ShowLoading()
	return loadReleaseDefinitionsCmd(a.rl.defsLoad.restart(a.client), a.rl.seq, "", "", false)
}

func (a *App) starredReleases() map[int]bool {
	out := make(map[int]bool, len(a.rl.adoStars)+len(a.rl.localStars))
	for _, id := range a.rl.adoStars {
		out[id] = true
	}
	for _, id := range a.rl.localStars {
		out[id] = true
	}
	return out
}

func (a *App) refreshReleases() tea.Cmd {
	if a.rl.pivot == approvalsTab {
		return a.loadApprovals()
	}
	a.rl.listFor = 0
	return a.loadReleaseDefinitions()
}

func (a *App) loadApprovals() tea.Cmd {
	a.rl.approvalsSeq++
	a.rl.approvalsLoading = true
	a.rl.approvals.ShowLoading()
	return loadApprovalsCmd(a.client, a.rl.approvalsSeq, a.me)
}

func (a *App) selectReleasePivot(pivot int) tea.Cmd {
	if pivot == a.rl.pivot {
		return nil
	}
	a.rl.pivot = pivot
	if pivot == approvalsTab && !a.rl.approvalsLoaded && !a.rl.approvalsLoading {
		return a.loadApprovals()
	}
	return nil
}

func (a *App) syncReleaseSelection() tea.Cmd {
	if !a.inReleases() || a.viewMode != ViewMain {
		return nil
	}
	if a.rl.pivot == approvalsTab {
		return a.syncApprovalPreview()
	}
	d := a.rl.defs.Selected()
	if d == nil || d.ID == a.rl.listFor {
		return nil
	}
	a.rl.listFor, a.rl.releases, a.rl.listToken = d.ID, nil, ""
	a.rl.list.ShowLoading(d.Name)
	return scheduleReleasesCmd(d.ID)
}

func (a *App) syncApprovalPreview() tea.Cmd {
	ap, _ := a.rl.approvals.Selected()
	id := 0
	if ap != nil {
		id = ap.ReleaseID
	}
	if id == a.rl.previewFor {
		return nil
	}
	a.rl.previewFor, a.rl.preview = id, nil
	if id == 0 || a.shell.mainWidth < approvalPreviewShow {
		return nil
	}
	return scheduleApprovalPreviewCmd(id)
}

func (a *App) releaseLoading() bool {
	return a.rl.loading || a.rl.defs.Loading() || a.rl.list.Loading() || a.rl.approvalsLoading ||
		(a.rl.pivot == approvalsTab && a.rl.previewFor != 0 && a.rl.preview == nil && a.shell.mainWidth >= approvalPreviewShow)
}

func (a *App) activeReleases() bool {
	for _, r := range a.rl.releases {
		if r.Active() {
			return true
		}
	}
	return false
}

func (a *App) toggleReleaseStar(id int, star bool) tea.Cmd {
	a.rl.localStars = slices.DeleteFunc(slices.Clone(a.rl.localStars), func(v int) bool { return v == id })
	if star {
		a.rl.localStars = append(a.rl.localStars, id)
	} else if slices.Contains(a.rl.adoStars, id) {
		a.rl.defs.SetStarred(a.starredReleases())
		return tea.Batch(saveFiltersCmd(a.savedState()), a.info("This pipeline is starred in Azure DevOps. Unstar it there to move it out of Starred."))
	}
	a.rl.defs.SetStarred(a.starredReleases())
	return saveFiltersCmd(a.savedState())
}

func (a *App) searchReleaseDefinitions(text string) tea.Cmd {
	a.rl.searchSeq++
	if text == "" || !a.rl.defs.More() {
		a.rl.defs.SetFound("", nil)
		return nil
	}
	a.rl.defs.SetSearchPending(true)
	return scheduleReleaseSearchCmd(a.rl.searchSeq, text)
}

func (a *App) openLatestRelease(definition int, name string) tea.Cmd {
	if definition == a.rl.listFor && !a.rl.list.Loading() {
		if len(a.rl.releases) == 0 {
			return a.info(name + " has no releases yet. Press " + a.keys.CreateRelease.Help().Key + " to create one.")
		}
		return a.openRelease(a.rl.releases[0].ID, panels.ReleasePipelineTab)
	}
	return a.mutate(latestReleaseCmd(a.client, definition, name))
}

func (a *App) openCreateRelease(id int, name string) tea.Cmd {
	return a.openModal(modals.NewCreateReleaseForm(a.env(), id, name))
}

func (a *App) setReleasePane(pane int) {
	a.rl.pane = pane
}

func (a App) handleReleaseKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	k := a.keys
	if a.rl.defs.Searching() {
		var cmd tea.Cmd
		a.rl.defs, cmd = a.rl.defs.Update(msg)
		return a, cmd
	}
	for i, b := range a.filterKeys()[:len(releasePivots)] {
		if key.Matches(msg, b) {
			return a, a.selectReleasePivot(i)
		}
	}
	twoPanes := a.rl.pivot == releasesTab
	if hub, ok := a.hubFor(msg); ok {
		return a, a.goToHub(hub)
	}
	switch {
	case key.Matches(msg, k.PrevView), key.Matches(msg, k.NextView):
		return a, a.selectReleasePivot((a.rl.pivot + 1) % len(releasePivots))
	case twoPanes && (key.Matches(msg, k.NextPanel) || key.Matches(msg, k.PrevPanel)):
		a.setReleasePane(1 - a.rl.pane)
	case twoPanes && key.Matches(msg, k.Left):
		a.setReleasePane(paneDefinitions)
	case twoPanes && key.Matches(msg, k.Right):
		a.setReleasePane(paneReleases)
	case twoPanes && key.Matches(msg, k.Back) && a.rl.pane == paneReleases:
		a.setReleasePane(paneDefinitions)
	case key.Matches(msg, k.Fullscreen):
		a.fullscreen = !a.fullscreen
	case key.Matches(msg, k.QuickSearch):
		return a, a.openModal(modals.NewQuickSearchModal(a.env()))
	case key.Matches(msg, k.Search):
		a.rl.pivot, a.rl.pane = releasesTab, paneDefinitions
		return a, a.rl.defs.FocusSearch()
	case key.Matches(msg, k.CreateRelease):
		if d := a.rl.defs.Selected(); d != nil {
			return a, a.openCreateRelease(d.ID, d.Name)
		}
		return a, a.info("Choose a release pipeline first.")
	default:
		return a, a.updateReleasePanel(msg)
	}
	return a, nil
}

func (a *App) updateReleasePanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch {
	case a.rl.pivot == approvalsTab:
		a.rl.approvals, cmd = a.rl.approvals.Update(msg)
	case a.rl.pane == paneReleases:
		a.rl.list, cmd = a.rl.list.Update(msg)
	default:
		a.rl.defs, cmd = a.rl.defs.Update(msg)
	}
	return cmd
}

func (a App) handleReleaseHubResult(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case releaseDefinitionsMsg:
		if msg.seq != a.rl.seq {
			return a, nil, true
		}
		a.rl.loading = false
		if msg.more {
			a.rl.defs.SetLoadingMore(false)
		}
		if msg.err != nil {
			a.rl.loaded = true
			a.rl.defs.SetDefinitions(nil, false)
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		a.rl.loaded = true
		if msg.more {
			a.rl.definitions = append(a.rl.definitions, msg.defs...)
		} else {
			a.rl.definitions, a.rl.adoStars = msg.defs, msg.starred
		}
		a.rl.defsToken = msg.token
		a.rl.defs.SetStarred(a.starredReleases())
		a.rl.defs.SetDefinitions(a.rl.definitions, msg.token != "")
		return a, nil, true
	case panels.MoreReleaseDefinitionsMsg:
		if a.rl.defsToken == "" || a.rl.loading {
			return a, nil, true
		}
		a.rl.defs.SetLoadingMore(true)
		return a, loadReleaseDefinitionsCmd(a.client, a.rl.seq, "", a.rl.defsToken, true), true
	case panels.ReleaseSearchTypedMsg:
		return a, a.searchReleaseDefinitions(msg.Text), true
	case releaseSearchDueMsg:
		if msg.seq != a.rl.searchSeq {
			return a, nil, true
		}
		return a, findReleaseDefinitionsCmd(a.rl.search.restart(a.client), msg.seq, msg.text), true
	case releaseSearchMsg:
		if msg.seq != a.rl.searchSeq {
			return a, nil, true
		}
		a.rl.defs.SetFound(msg.text, msg.defs)
		return a, nil, true
	case panels.OpenLatestReleaseMsg:
		return a, a.openLatestRelease(msg.DefinitionID, msg.Name), true
	case latestReleaseMsg:
		a.loading = false
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		if msg.id == 0 {
			return a, a.info(msg.name + " has no releases yet. Press " + a.keys.CreateRelease.Help().Key + " to create one."), true
		}
		return a, a.openRelease(msg.id, panels.ReleasePipelineTab), true
	case panels.StarReleaseDefinitionMsg:
		return a, a.toggleReleaseStar(msg.ID, msg.Star), true
	case panels.FocusReleasesMsg:
		a.setReleasePane(paneReleases)
		return a, nil, true
	case releasesDueMsg:
		if msg.definition != a.rl.listFor {
			return a, nil, true
		}
		a.rl.listSeq++
		return a, loadReleasesCmd(a.client, a.rl.listSeq, msg.definition, "", false, false), true
	case panels.MoreReleasesMsg:
		if a.rl.listToken == "" {
			return a, nil, true
		}
		a.rl.list.SetLoadingMore(true)
		return a, loadReleasesCmd(a.client, a.rl.listSeq, a.rl.listFor, a.rl.listToken, true, false), true
	case releasesLoadedMsg:
		return a.setReleases(msg)
	case releasePollMsg:
		if msg.seq != a.rl.pollSeq || !a.inReleases() || a.viewMode != ViewMain || a.rl.listFor == 0 {
			return a, nil, true
		}
		return a, loadReleasesCmd(a.client, a.rl.listSeq, a.rl.listFor, "", false, true), true
	case approvalsLoadedMsg:
		if msg.seq != a.rl.approvalsSeq {
			return a, nil, true
		}
		a.rl.approvalsLoading, a.rl.approvalsLoaded = false, true
		a.rl.approvals.SetApprovals(msg.mine, msg.others)
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		return a, nil, true
	case approvalPreviewDueMsg:
		if msg.release != a.rl.previewFor {
			return a, nil, true
		}
		return a, loadApprovalPreviewCmd(a.client, msg.release), true
	case approvalPreviewMsg:
		if msg.release.ID == a.rl.previewFor {
			r := msg.release
			a.rl.preview = &r
		}
		return a, nil, true
	case panels.BrowseReleaseMsg:
		url := msg.URL
		switch {
		case url != "":
		case msg.ReleaseID != 0:
			url = a.client.ReleaseWebURL(msg.ReleaseID)
		default:
			url = a.client.ReleaseDefinitionWebURL(msg.DefinitionID)
		}
		return a, openBrowserCmd(url), true
	case panels.ReleaseNoticeMsg:
		return a, a.info(msg.Text), true
	}
	return a.handleReleaseViewResult(msg)
}

func (a App) setReleases(msg releasesLoadedMsg) (App, tea.Cmd, bool) {
	if msg.seq != a.rl.listSeq || msg.definition != a.rl.listFor {
		return a, nil, true
	}
	if msg.err != nil {
		a.rl.list.SetReleases(a.listDefinitionName(), a.rl.releases, false)
		if msg.silent {
			return a, nil, true
		}
		return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
	}
	switch {
	case msg.more:
		a.rl.releases = append(a.rl.releases, msg.releases...)
		a.rl.listToken = msg.token
	case msg.silent:
		a.rl.releases = mergeReleases(a.rl.releases, msg.releases)
	default:
		a.rl.releases, a.rl.listToken = msg.releases, msg.token
	}
	a.rl.list.SetReleases(a.listDefinitionName(), a.rl.releases, a.rl.listToken != "")
	if !a.activeReleases() {
		return a, nil, true
	}
	a.rl.pollSeq++
	return a, scheduleReleasePollCmd(a.rl.pollSeq), true
}

func mergeReleases(old, fresh []models.Release) []models.Release {
	seen := make(map[int]bool, len(fresh))
	for _, r := range fresh {
		seen[r.ID] = true
	}
	out := slices.Clone(fresh)
	for _, r := range old[min(len(fresh), len(old)):] {
		if !seen[r.ID] {
			out = append(out, r)
		}
	}
	return out
}

func (a *App) listDefinitionName() string {
	for _, d := range a.rl.definitions {
		if d.ID == a.rl.listFor {
			return d.Name
		}
	}
	return ""
}

func (a *App) releaseLayout(width int) (int, int) {
	if a.rl.pivot == approvalsTab {
		if width < approvalPreviewShow {
			return width, 0
		}
		return width - approvalPreviewWidth, approvalPreviewWidth
	}
	if width < releaseTwoPanesWidth {
		return width, 0
	}
	left := min(max(int(float64(width)*releaseDefsRatio), releaseDefsMinWidth), releaseDefsMaxWidth)
	return left, width - left
}

func (a *App) syncReleaseLayout(width, height int) {
	cw := func(w int) int { pw, _ := panels.PaneContentSize(w, height); return pw }
	_, ch := panels.PaneContentSize(width, height)
	left, right := a.releaseLayout(width)
	if a.rl.pivot == approvalsTab {
		a.rl.approvals.SetSize(cw(left), ch)
		return
	}
	if right == 0 {
		a.rl.defs.SetSize(cw(width), ch)
		a.rl.list.SetSize(cw(width), ch)
		return
	}
	a.rl.defs.SetSize(cw(left), ch)
	a.rl.list.SetSize(cw(right), ch)
}

func (a *App) syncReleaseFocus(main bool) {
	hub := main && a.inReleases() && a.viewMode == ViewMain
	a.rl.defs.SetFocused(hub && a.rl.pivot == releasesTab && a.rl.pane == paneDefinitions)
	a.rl.list.SetFocused(hub && a.rl.pivot == releasesTab && a.rl.pane == paneReleases)
	a.rl.approvals.SetFocused(hub && a.rl.pivot == approvalsTab)
}

func (a *App) renderReleasesBody(width, height int) string {
	left, right := a.releaseLayout(width)
	if a.rl.pivot == approvalsTab {
		list := panels.Pane(panels.PaneTitle("", a.rl.approvals.Title(), true), a.rl.approvals.View(), left, height, true)
		if right == 0 {
			return list
		}
		ap, mine := a.rl.approvals.Selected()
		cw, ch := panels.PaneContentSize(right, height)
		preview := panels.RenderApprovalPreview(panels.ApprovalPreviewProps{Approval: ap, Mine: mine, Release: a.rl.preview, Spinner: a.spinner.View(), Width: cw, Height: ch})
		return lipgloss.JoinHorizontal(lipgloss.Top, list, panels.Pane(panels.PaneTitle("", "Preview", false), preview, right, height, false))
	}
	defsFocused, listFocused := a.rl.pane == paneDefinitions, a.rl.pane == paneReleases
	defs := panels.Pane(panels.PaneTitle("", a.rl.defs.Title(), defsFocused), a.rl.defs.View(), left, height, defsFocused)
	listWidth := right
	if right == 0 {
		listWidth = width
	}
	list := panels.Pane(panels.PaneTitle("", a.rl.list.Title(), listFocused), a.rl.list.View(), listWidth, height, listFocused)
	switch {
	case right > 0:
		return lipgloss.JoinHorizontal(lipgloss.Top, defs, list)
	case listFocused:
		return list
	}
	return defs
}

func (a *App) releasePivotLabels() []string {
	labels := make([]string, len(releasePivots))
	for i, name := range releasePivots {
		labels[i] = a.filterKeys()[i].Help().Key + " " + name
	}
	if n := a.rl.approvals.Mine(); n > 0 {
		labels[approvalsTab] += " " + panels.Badge(strconv.Itoa(n), styles.ColorOrange, styles.ColorOnAccent)
	}
	return labels
}

func (a *App) releaseCrumb() string {
	if a.rl.pivot == approvalsTab {
		return releasePivots[approvalsTab]
	}
	if d := a.rl.defs.Selected(); d != nil {
		return d.Name
	}
	return releasePivots[releasesTab]
}

func (a *App) releaseBindings() []key.Binding {
	switch {
	case a.rl.pivot == approvalsTab:
		return a.rl.approvals.KeyBindings()
	case a.rl.pane == paneReleases:
		return a.rl.list.KeyBindings()
	}
	return a.rl.defs.KeyBindings()
}
