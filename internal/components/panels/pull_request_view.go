package panels

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

type PullRequestTab int

const (
	PullRequestOverview PullRequestTab = iota
	PullRequestFiles
	pullRequestTabs
)

const (
	pullRequestHeaderHeight = 4
	generalComment          = "General comment"
	loadingActivity         = "Loading comments"
	noActivity              = "No comments yet."
)

type ClosePullRequestViewMsg struct{}

type VotePullRequestMsg struct {
	PullRequest models.PullRequest
}

type CommentPullRequestMsg struct {
	PullRequest models.PullRequest
	Anchor      *models.ThreadAnchor
}

type ReplyPullRequestMsg struct {
	PullRequest models.PullRequest
	Thread      models.PullRequestThread
	Comment     models.PullRequestComment
}

type LikePullRequestCommentMsg struct {
	PullRequest models.PullRequest
	Thread      models.PullRequestThread
	Comment     models.PullRequestComment
	Like        bool
}

type ResolvePullRequestThreadMsg struct {
	PullRequest models.PullRequest
	Thread      models.PullRequestThread
	Status      string
}

type EditPullRequestCommentMsg struct {
	PullRequest models.PullRequest
	Thread      models.PullRequestThread
	Comment     models.PullRequestComment
}

type DeletePullRequestCommentMsg struct {
	PullRequest models.PullRequest
	Thread      models.PullRequestThread
	Comment     models.PullRequestComment
}

type PullRequestBuildsMsg struct {
	PullRequest models.PullRequest
	Builds      []models.PullRequestCheck
}

type LinkedWorkItemsMsg struct {
	PullRequest models.PullRequest
	Items       []models.WorkItem
}

var mentionPattern = regexp.MustCompile(`@<([0-9A-Fa-f-]{36})>`)

type PullRequestView struct {
	names         map[string]string
	pr            *models.PullRequest
	threads       []models.PullRequestThread
	threadsLoaded bool
	checks        []models.PullRequestCheck
	checksLoaded  bool
	workItems     []models.WorkItem
	tab           PullRequestTab
	me            models.Identity
	keys          keys.KeyMap
	limits        models.DiffLimits
	spinner       string
	width         int
	height        int

	overview    viewport.Model
	selected    int
	comment     int
	focusTop    int
	focusBottom int
	expanded    map[int]bool

	files pullRequestFiles
}

func NewPullRequestView(k keys.KeyMap, limits models.DiffLimits) PullRequestView {
	return PullRequestView{keys: k, limits: limits, overview: widgets.NewViewport(k), files: newPullRequestFiles(k, limits)}
}

func (v *PullRequestView) Open(pr models.PullRequest) {
	*v = PullRequestView{
		pr:       &pr,
		me:       v.me,
		names:    v.names,
		keys:     v.keys,
		limits:   v.limits,
		spinner:  v.spinner,
		width:    v.width,
		height:   v.height,
		overview: widgets.NewViewport(v.keys),
		files:    newPullRequestFiles(v.keys, v.limits),
		expanded: map[int]bool{},
	}
	v.files.names, v.files.me, v.files.spinner, v.files.expanded = v.names, v.me, v.spinner, v.expanded
	v.layout()
}

func (v *PullRequestView) PullRequest() *models.PullRequest { return v.pr }

func (v *PullRequestView) Tab() PullRequestTab { return v.tab }

func (v *PullRequestView) Iteration() int { return v.files.changes.Iteration }

func (v *PullRequestView) Loading() bool {
	return v.pr != nil && (!v.threadsLoaded || !v.checksLoaded || !v.files.loaded)
}

func (v *PullRequestView) SetSpinner(frame string) {
	if v.spinner == frame {
		return
	}
	v.spinner, v.files.spinner = frame, frame
	if v.pr != nil && (!v.threadsLoaded || !v.checksLoaded) {
		v.renderOverview()
	}
}

func (v *PullRequestView) SetIdentity(me models.Identity) {
	v.me = me
	v.files.me = me
	v.files.rebuild()
	v.renderOverview()
}

func (v *PullRequestView) SetMembers(members []models.TeamMember) {
	v.names = make(map[string]string, len(members))
	for _, m := range members {
		v.names[strings.ToLower(m.ID)] = m.DisplayName
	}
	v.files.names = v.names
	v.renderOverview()
}

func (v *PullRequestView) Refresh(pr models.PullRequest) {
	if v.pr == nil || v.pr.ID != pr.ID {
		return
	}
	v.pr = &pr
	v.renderOverview()
}

func (v *PullRequestView) SetThreads(id int, threads []models.PullRequestThread) {
	if v.pr == nil || v.pr.ID != id {
		return
	}
	v.threads, v.threadsLoaded = threads, true
	t, ok := v.thread(v.selected)
	if !ok {
		v.selected, v.comment = 0, 0
	} else if _, found := t.Comment(v.comment); !found || t.Collapsed(v.expanded) {
		v.comment = t.Comments[0].ID
	}
	v.renderOverview()
	v.files.setThreads(threads)
}

func (v *PullRequestView) SetChecks(id int, checks []models.PullRequestCheck) {
	if v.pr == nil || v.pr.ID != id {
		return
	}
	v.checks, v.checksLoaded = checks, true
	v.renderOverview()
}

func (v *PullRequestView) SetWorkItems(id int, items []models.WorkItem) {
	if v.pr == nil || v.pr.ID != id {
		return
	}
	v.workItems = items
	v.renderOverview()
}

func (v *PullRequestView) SetChanges(id int, changes models.PullRequestChanges) {
	if v.pr == nil || v.pr.ID != id {
		return
	}
	v.files.setChanges(changes)
}

func (v *PullRequestView) SetFile(id int, path, oldText, newText string, err error) {
	if v.pr == nil || v.pr.ID != id {
		return
	}
	v.files.setFile(path, oldText, newText, err)
}

func (v *PullRequestView) WantedFile() (models.PullRequestFile, models.PullRequestChanges, bool) {
	if v.pr == nil || v.tab != PullRequestFiles {
		return models.PullRequestFile{}, models.PullRequestChanges{}, false
	}
	file, ok := v.files.wanted()
	return file, v.files.changes, ok
}

func (v *PullRequestView) SetSize(width, height int) {
	if v.width == width && v.height == height {
		return
	}
	v.width, v.height = width, height
	v.layout()
}

func (v *PullRequestView) bodyHeight() int {
	return max(v.height-pullRequestHeaderHeight, 0)
}

func (v *PullRequestView) sidebarWidth() int {
	if v.width < styles.PullRequestSidebarShow {
		return 0
	}
	return styles.PullRequestSidebarWidth
}

func (v *PullRequestView) mainWidth() int {
	if side := v.sidebarWidth(); side > 0 {
		return max(v.width-side-styles.DetailColumnGap-2, 1)
	}
	return max(v.width-2, 1)
}

func (v *PullRequestView) layout() {
	v.overview.SetWidth(v.mainWidth())
	v.overview.SetHeight(v.bodyHeight())
	v.files.setSize(max(v.width-2, 1), v.bodyHeight())
	v.renderOverview()
}

func (v *PullRequestView) comments() []models.PullRequestThread {
	out := make([]models.PullRequestThread, 0, len(v.threads))
	for _, t := range v.threads {
		if !t.System() {
			out = append(out, t)
		}
	}
	return out
}

func (v *PullRequestView) thread(id int) (models.PullRequestThread, bool) {
	for _, t := range v.threads {
		if t.ID == id && !t.System() {
			return t, id != 0
		}
	}
	return models.PullRequestThread{}, false
}

func (v *PullRequestView) selectedComment() (models.PullRequestThread, models.PullRequestComment, bool) {
	t, ok := v.thread(v.selected)
	if !ok {
		return t, models.PullRequestComment{}, false
	}
	c, ok := t.Comment(v.comment)
	return t, c, ok
}

func (v *PullRequestView) moveThread(delta int) {
	var stops []commentFocus
	index := -1
	for _, t := range v.comments() {
		for _, stop := range threadStops(t, v.expanded) {
			if stop.thread == v.selected && stop.comment == v.comment {
				index = len(stops)
			}
			stops = append(stops, stop)
		}
	}
	next := min(max(index+delta, -1), len(stops)-1)
	if next == index {
		return
	}
	v.selected, v.comment = 0, 0
	if next >= 0 {
		v.selected, v.comment = stops[next].thread, stops[next].comment
	}
	v.renderOverview()
	v.revealFocus()
}

func (v *PullRequestView) revealFocus() {
	if v.focusTop < 0 {
		v.overview.GotoTop()
		return
	}
	widgets.Reveal(&v.overview, v.focusTop, v.focusBottom)
}

func (v *PullRequestView) setExpanded(t models.PullRequestThread, open bool) {
	if open {
		v.expanded[t.ID] = true
	} else {
		delete(v.expanded, t.ID)
	}
	if v.selected == t.ID && t.Collapsed(v.expanded) {
		v.comment = t.Comments[0].ID
	}
	v.renderOverview()
	v.revealFocus()
	v.files.rebuild()
}

func (v *PullRequestView) threadStyle(t models.PullRequestThread, width int, anchored bool) threadStyle {
	st := threadStyle{width: width, anchored: anchored, names: v.names, me: v.me, expanded: v.expanded}
	if t.ID != v.selected || t.System() {
		return st
	}
	st.selected, st.comment = true, v.comment
	if c, ok := t.Comment(v.comment); ok {
		st.hints = commentHints(v.keys, v.me, t, c, v.expanded, width)
	}
	return st
}

func (v *PullRequestView) statusLine(width int) string {
	pr := *v.pr
	if pr.Status != models.PullRequestActive {
		return ""
	}
	ok := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true)
	bad := lipgloss.NewStyle().Foreground(styles.ColorError).Bold(true)
	merge := ok.Render("OK") + " No merge conflicts"
	if pr.HasConflicts() {
		merge = bad.Render("CONFLICTS") + " Resolve merge conflicts before completing"
	}
	policy := Loading(v.spinner, loadingChecks)
	if v.checksLoaded {
		policy = ok.Render("READY") + Muted(" required policies met")
		if n := models.BlockingChecks(v.checks); n > 0 {
			policy = bad.Render("BLOCKED") + Muted(fmt.Sprintf(" %d required not met", n))
		}
	}
	if lipgloss.Width(merge)+lipgloss.Width(policy)+2 > width {
		return Line(merge, width) + "\n" + Line(policy, width)
	}
	return merge + strings.Repeat(" ", width-lipgloss.Width(merge)-lipgloss.Width(policy)) + policy
}

func (v *PullRequestView) workItemsSection(width int) string {
	title := "Work items"
	if len(v.workItems) > 0 {
		title += " " + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(v.keys.LinkedWorkItems.Help().Key)
	}
	lines := []string{sectionTitle(title, width)}
	if len(v.workItems) == 0 {
		return strings.Join(append(lines, Muted("No linked work items.")), "\n")
	}
	for i := range v.workItems {
		item := &v.workItems[i]
		id := lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(fmt.Sprintf("%d", item.ID))
		lines = append(lines, Line(TypeText(item)+" "+id+" "+item.Title, width))
	}
	return strings.Join(lines, "\n")
}

func (v *PullRequestView) sideSections(width int) []string {
	sections := []string{
		RenderChecks(v.checks, v.checksLoaded, width, v.spinner, v.keys.OpenBuild.Help().Key),
		RenderReviewers(*v.pr, v.me, width),
		v.workItemsSection(width),
	}
	if len(v.pr.Labels) > 0 {
		sections = append(sections, sectionTitle("Tags", width)+"\n"+Chips(v.pr.Labels))
	}
	return sections
}

func (v *PullRequestView) renderOverview() {
	if v.pr == nil || v.width <= 0 {
		return
	}
	width := v.mainWidth()
	var parts []string
	if status := v.statusLine(width); status != "" {
		parts = append(parts, status)
	}
	if v.sidebarWidth() == 0 {
		parts = append(parts, v.sideSections(width)...)
	}
	parts = append(parts, sectionTitle("Description", width)+"\n"+PullRequestDescription(*v.pr, width))
	parts = append(parts, sectionTitle(fmt.Sprintf("Activity %d", len(v.threads)), width))
	line := 0
	for _, p := range parts {
		line += lipgloss.Height(p) + 1
	}
	line--
	v.focusTop, v.focusBottom = -1, -1
	switch {
	case !v.threadsLoaded:
		parts[len(parts)-1] += "\n" + Loading(v.spinner, loadingActivity)
	case len(v.threads) == 0:
		parts[len(parts)-1] += "\n" + Muted(noActivity)
	}
	blocks := make([]string, 0, len(v.threads))
	for _, t := range v.threads {
		r := renderThread(t, v.threadStyle(t, width, true))
		if r.focused() {
			v.focusTop, v.focusBottom = line+r.top, line+r.bottom
		}
		line += lipgloss.Height(r.text) + 1
		blocks = append(blocks, r.text)
	}
	content := strings.Join(parts, "\n\n")
	if len(blocks) > 0 {
		content += "\n" + strings.Join(blocks, "\n\n")
	}
	v.overview.SetContent(content)
}

func (v PullRequestView) Update(msg tea.Msg) (PullRequestView, tea.Cmd) {
	if v.pr == nil {
		return v, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		if v.tab == PullRequestFiles {
			v.files, cmd = v.files.update(msg)
			return v, cmd
		}
		v.overview, cmd = v.overview.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v PullRequestView) handleKey(msg tea.KeyPressMsg) (PullRequestView, tea.Cmd) {
	k := v.keys
	pr := *v.pr
	switch {
	case key.Matches(msg, k.Back) && v.tab == PullRequestFiles && v.files.selecting():
		v.files.clearSelection()
		return v, nil
	case key.Matches(msg, k.Back):
		return v, func() tea.Msg { return ClosePullRequestViewMsg{} }
	case key.Matches(msg, k.Share), key.Matches(msg, k.ShareLink):
		now, _ := ShareRequested(k, msg)
		return v, shareCmd(v.shareTarget(pr), now)
	case key.Matches(msg, k.Panel1):
		v.tab = PullRequestOverview
		return v, nil
	case key.Matches(msg, k.Panel2):
		v.tab = PullRequestFiles
		return v, nil
	case key.Matches(msg, k.NextView), key.Matches(msg, k.PrevView),
		v.tab == PullRequestOverview && (key.Matches(msg, k.NextPanel) || key.Matches(msg, k.PrevPanel)):
		v.tab = (v.tab + 1) % pullRequestTabs
		return v, nil
	case key.Matches(msg, k.Vote):
		return v, func() tea.Msg { return VotePullRequestMsg{PullRequest: pr} }
	case key.Matches(msg, k.OpenBrowser):
		pr.WebURL = v.browseURL(pr)
		return v, func() tea.Msg { return BrowsePullRequestMsg{PullRequest: pr} }
	case key.Matches(msg, k.LinkedWorkItems):
		items := v.workItems
		return v, func() tea.Msg { return LinkedWorkItemsMsg{PullRequest: pr, Items: items} }
	case key.Matches(msg, k.OpenBuild):
		builds := models.BuildChecks(v.checks)
		return v, func() tea.Msg { return PullRequestBuildsMsg{PullRequest: pr, Builds: builds} }
	}
	if v.tab == PullRequestFiles {
		return v.handleFilesKey(msg)
	}
	return v.handleOverviewKey(msg)
}

func (v *PullRequestView) shareTarget(pr models.PullRequest) share.Target {
	if v.tab != PullRequestFiles {
		return share.PullRequest(pr)
	}
	return v.files.shareTarget(pr)
}

func (v *PullRequestView) browseURL(pr models.PullRequest) string {
	if t, _, ok := v.FocusedComment(); ok {
		return fmt.Sprintf("%s?discussionId=%d", pr.WebURL, t.ID)
	}
	if file, ok := v.files.file(); ok && v.tab == PullRequestFiles {
		return pr.WebURL + "?_a=files&path=" + url.QueryEscape(file.Path)
	}
	return pr.WebURL
}

func (v *PullRequestView) FocusedComment() (models.PullRequestThread, models.PullRequestComment, bool) {
	if v.tab == PullRequestFiles {
		return v.files.focused()
	}
	return v.selectedComment()
}

func (v *PullRequestView) toggleThread(t models.PullRequestThread) bool {
	switch {
	case t.Collapsed(v.expanded):
		v.setExpanded(t, true)
	case t.Resolvable() && !t.Active() && (v.tab == PullRequestFiles || t.FilePath == ""):
		v.setExpanded(t, false)
	default:
		return false
	}
	return true
}

func (v PullRequestView) threadCmd(t models.PullRequestThread, c models.PullRequestComment, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	pr := *v.pr
	edit := key.Matches(msg, v.keys.EditComment)
	collapsed := t.Collapsed(v.expanded)
	switch {
	case collapsed && (edit || key.Matches(msg, v.keys.DeleteComment) || key.Matches(msg, v.keys.LikeComment)):
		return nil, true
	case edit || key.Matches(msg, v.keys.DeleteComment):
		switch {
		case !c.WrittenBy(v.me):
			known := v.me.Known()
			return func() tea.Msg { return CommentReadOnlyMsg{IdentityKnown: known} }, true
		case edit:
			return func() tea.Msg { return EditPullRequestCommentMsg{PullRequest: pr, Thread: t, Comment: c} }, true
		}
		return func() tea.Msg { return DeletePullRequestCommentMsg{PullRequest: pr, Thread: t, Comment: c} }, true
	case key.Matches(msg, v.keys.LikeComment):
		if c.WrittenBy(v.me) || !v.me.Known() {
			return nil, true
		}
		like := !c.LikedBy(v.me)
		return func() tea.Msg { return LikePullRequestCommentMsg{PullRequest: pr, Thread: t, Comment: c, Like: like} }, true
	case key.Matches(msg, v.keys.Reply):
		return func() tea.Msg { return ReplyPullRequestMsg{PullRequest: pr, Thread: t, Comment: c} }, true
	case key.Matches(msg, v.keys.ResolveThread):
		if !t.Resolvable() {
			return nil, true
		}
		status := models.ThreadFixed
		if !t.Active() {
			status = models.ThreadActive
		}
		return func() tea.Msg { return ResolvePullRequestThreadMsg{PullRequest: pr, Thread: t, Status: status} }, true
	}
	return nil, false
}

func (v PullRequestView) handleOverviewKey(msg tea.KeyPressMsg) (PullRequestView, tea.Cmd) {
	k := v.keys
	pr := *v.pr
	if t, c, ok := v.selectedComment(); ok {
		if cmd, handled := v.threadCmd(t, c, msg); handled {
			return v, cmd
		}
		if key.Matches(msg, k.Select) && !v.toggleThread(t) && t.FilePath != "" {
			v.tab = PullRequestFiles
			v.files.open(t.FilePath, t.Line, t.LeftSide)
			return v, nil
		}
		if key.Matches(msg, k.Select) {
			return v, nil
		}
	}
	switch {
	case key.Matches(msg, k.AddComment):
		return v, func() tea.Msg { return CommentPullRequestMsg{PullRequest: pr} }
	case key.Matches(msg, k.Down):
		v.moveThread(1)
	case key.Matches(msg, k.Up):
		v.moveThread(-1)
	case key.Matches(msg, k.Top):
		v.moveThread(-len(v.threads) - 1)
	case key.Matches(msg, k.Bottom):
		v.moveThread(len(v.threads))
	default:
		var cmd tea.Cmd
		v.overview, cmd = v.overview.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v PullRequestView) handleFilesKey(msg tea.KeyPressMsg) (PullRequestView, tea.Cmd) {
	pr := *v.pr
	if t, c, ok := v.files.focused(); ok {
		if key.Matches(msg, v.keys.Select) {
			v.toggleThread(t)
			return v, nil
		}
		if cmd, handled := v.threadCmd(t, c, msg); handled {
			return v, cmd
		}
	}
	if key.Matches(msg, v.keys.AddComment) {
		anchor, ok := v.files.anchor()
		if !ok {
			return v, nil
		}
		v.files.clearSelection()
		return v, func() tea.Msg { return CommentPullRequestMsg{PullRequest: pr, Anchor: &anchor} }
	}
	var cmd tea.Cmd
	v.files, cmd = v.files.update(msg)
	return v, cmd
}

func (v *PullRequestView) KeyBindings() []key.Binding {
	k := v.keys
	tabs := widgets.Combine("tab", k.Panel1, k.Panel2)
	back := widgets.Relabel(k.Back, "back")
	if v.tab == PullRequestFiles && v.files.selecting() {
		return v.files.keyBindings()
	}
	if v.tab == PullRequestFiles {
		return append(v.files.keyBindings(), k.Vote, tabs, back)
	}
	bindings := []key.Binding{k.Share, k.Vote, k.AddComment}
	if t, c, ok := v.selectedComment(); ok {
		bindings = append(bindings, commentActions(k, v.me, t, c, v.expanded)...)
		if t.FilePath != "" && !t.Collapsed(v.expanded) {
			bindings = append(bindings, widgets.Relabel(k.Select, "open file"))
		}
	}
	if len(v.workItems) > 0 {
		bindings = append(bindings, k.LinkedWorkItems)
	}
	if len(models.BuildChecks(v.checks)) > 0 {
		bindings = append(bindings, k.OpenBuild)
	}
	return append(bindings, widgets.Combine("comment", k.Up, k.Down), tabs, k.OpenBrowser, back)
}

func (v *PullRequestView) Mode() string {
	if n := v.files.selectedLines(); v.tab == PullRequestFiles && n > 0 {
		return fmt.Sprintf("SELECT %d LINES", n)
	}
	return "REVIEW"
}

func (v *PullRequestView) header() string {
	pr := *v.pr
	inner := max(v.width-2, 1)
	id := lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Render(fmt.Sprintf("%d", pr.ID))
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).
		Render(ansi.Truncate(pr.Title, max(inner-lipgloss.Width(id)-1, 1), "…"))
	meta := PullRequestBadge(pr) + " " + lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(pr.Author.DisplayName) +
		Muted(" proposes to merge ") + PullRequestBranches(pr)
	if v.threadsLoaded {
		meta += Muted(fmt.Sprintf(" · %d active comments", models.ActiveThreads(v.threads)))
	}
	files := "Files " + v.spinner
	if v.files.loaded {
		files = fmt.Sprintf("Files %d", len(v.files.changes.Files))
	}
	tabs := Tabs([]string{v.keys.Panel1.Help().Key + " Overview", v.keys.Panel2.Help().Key + " " + files}, int(v.tab), true)
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat(ruleGlyph, v.width))
	return strings.Join([]string{
		" " + Line(id+" "+title, inner),
		" " + Line(meta, inner),
		Line(tabs, v.width),
		rule,
	}, "\n")
}

func (v PullRequestView) View() string {
	if v.pr == nil || v.width <= 0 || v.height <= 0 {
		return ""
	}
	body := v.files.view()
	if v.tab == PullRequestOverview {
		body = v.overview.View()
		if side := v.sidebarWidth(); side > 0 {
			sidebar := Block(strings.Join(v.sideSections(side), "\n\n"), side, v.bodyHeight())
			body = lipgloss.JoinHorizontal(lipgloss.Top,
				Block(body, v.mainWidth(), v.bodyHeight()), strings.Repeat(" ", styles.DetailColumnGap), sidebar)
		}
	}
	body = lipgloss.NewStyle().PaddingLeft(1).Render(Block(body, max(v.width-2, 1), v.bodyHeight()))
	return Block(v.header()+"\n"+body, v.width, v.height)
}
