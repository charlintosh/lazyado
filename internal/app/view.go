package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

func (a App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if a.tracksHover() {
		v.MouseMode = tea.MouseModeAllMotion
	}
	return v
}

func (a *App) tracksHover() bool {
	return a.viewMode == ViewMain && a.mainView() == viewTaskboard && a.activePanel == PanelWorkItems && !a.modals.open()
}

func (a *App) withDragChip(base string) string {
	if a.viewMode != ViewMain || a.mainView() != viewTaskboard {
		return base
	}
	chip, x, y, ok := a.taskboardPanel.DragChip()
	if !ok {
		return base
	}
	return widgets.Splice(base, chip, x, y, a.width, a.height-styles.StatusBarHeight)
}

func (a *App) render() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}
	if a.showSplash() {
		return a.splashScreen.View()
	}
	base := a.withDragChip(a.renderBase())
	m, ok := a.modals.top()
	if !ok {
		return base
	}
	if p, ok := m.(modals.FilterPickerModal); ok && a.viewMode == ViewMain {
		if x, y, ok := a.pickerAnchor(p.Group()); ok {
			return widgets.OverlayAt(base, m.View(), x, y, a.width, a.height-styles.StatusBarHeight)
		}
	}
	if _, ok := m.(modals.RepositoryPickerModal); ok && a.viewMode == ViewMain {
		if x, y, ok := a.pickerAnchor(repositoryGroup); ok {
			return widgets.OverlayAt(base, m.View(), x, y, a.width, a.height-styles.StatusBarHeight)
		}
	}
	return widgets.Overlay(base, m.View(), a.width, a.height)
}

func (a *App) renderBase() string {
	switch {
	case a.width < styles.ShellMinWidth || a.height < styles.ShellMinHeight:
		return a.renderTooSmall()
	case a.reviewing():
		return a.renderReview()
	case a.viewingRun():
		return a.renderRun()
	case a.viewingRelease():
		return a.renderRelease()
	case a.viewMode == ViewDetail:
		return a.renderDetailView()
	default:
		return a.renderMainView()
	}
}

func (a *App) renderTooSmall() string {
	msg := a.styles.TextMuted.Render(fmt.Sprintf("Terminal too small (%dx%d).\nResize to at least %dx%d.",
		a.width, a.height, styles.ShellMinWidth, styles.ShellMinHeight))
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, msg)
}

func (a *App) renderHeader(crumbs ...string) string {
	return panels.RenderHeader(panels.HeaderProps{
		Width:        a.width,
		Organization: a.client.Organization(),
		Project:      a.client.Project(),
		Crumbs:       crumbs,
		Loading:      a.loading || a.pr.loading || a.pl.loading || a.rl.loading,
		Spinner:      a.spinner.View(),
		Notice:       a.notification.View(),
		SearchKey:    a.keys.QuickSearch.Help().Key,
	})
}

func (a *App) renderMainView() string {
	l := a.shell
	var cols []string
	if l.railWidth > 0 {
		cols = append(cols, panels.RenderRail(panels.RailProps{
			Width:   l.railWidth,
			Height:  l.bodyHeight,
			Project: a.client.Project(),
			Hub:     a.hub,
			Compact: l.compactRail,
			Keys:    a.keys,
		}))
	}
	cols = append(cols, a.renderMainColumn(l))
	if l.showPane {
		cols = append(cols, a.renderPane(l.paneWidth, l.bodyHeight))
	}
	body := panels.Block(lipgloss.JoinHorizontal(lipgloss.Top, cols...), a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, a.renderHeader(a.hubCrumbs()...), body, a.renderStatusBar())
}

func (a *App) hubCrumbs() []string {
	crumbs := []string{a.hub.Section(), a.hub.Name()}
	if a.inPullRequests() && a.pr.repo.ID != "" {
		crumbs = append(crumbs, a.pr.repo.Name)
	}
	if a.inPipelines() {
		crumbs = append(crumbs, a.pipelineCrumb())
	}
	if a.inReleases() {
		crumbs = append(crumbs, a.releaseCrumb())
	}
	if a.inCode() && a.code.repo.ID != "" {
		crumbs = append(crumbs, a.code.repo.Name)
	}
	return crumbs
}

func (a *App) renderMainColumn(l shellLayout) string {
	var box string
	switch {
	case a.inReleases():
		box = a.renderReleasesBody(l.mainWidth, l.mainHeight())
	case a.inFiles():
		box = a.renderFilesBody(l.mainWidth, l.mainHeight())
	case a.inCommits():
		box = a.renderCommitsBody(l.mainWidth, l.mainHeight())
	case a.inBranches():
		box = a.renderBranchesBody(l.mainWidth, l.mainHeight())
	case a.inTags():
		box = a.renderTagsBody(l.mainWidth, l.mainHeight())
	case a.isPanePanel() && !l.showPane:
		box = a.renderPane(l.mainWidth, l.mainHeight())
	default:
		focused := a.activePanel == PanelWorkItems
		title, content := a.mainPanelView()
		box = panels.Pane(panels.PaneTitle(a.mainPaneKey(), title, focused), content, l.mainWidth, l.mainHeight(), focused)
	}
	header := panels.Block(a.header.View, l.mainWidth, l.headerHeight)
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (a *App) mainPaneKey() string {
	switch a.hub {
	case panels.HubBacklog, panels.HubBoard, panels.HubSprint:
		return a.keys.Panel5.Help().Key
	}
	return ""
}

func (a *App) mainPanelView() (string, string) {
	switch a.mainView() {
	case viewBoard:
		return a.boardPanel.Title(), a.boardPanel.View()
	case viewTaskboard:
		return a.taskboardPanel.Title(), a.taskboardPanel.View()
	case viewPullRequests:
		return a.pr.panel.Title(), a.pr.panel.View()
	case viewPipelines:
		return a.pl.panel.Title(), a.pl.panel.View()
	default:
		return a.workItemsPanel.Title(), a.workItemsPanel.View()
	}
}

func (a *App) currentIteration() *models.Iteration {
	selected := a.filters.GetSelectedSprint()
	for i := range a.iterations {
		if a.iterations[i].Path == selected {
			return &a.iterations[i]
		}
	}
	return nil
}

func (a *App) sprintName() string {
	if it := a.currentIteration(); it != nil {
		return it.Name
	}
	return allIterations
}

func (a *App) sprintDates() string {
	it := a.currentIteration()
	if it == nil || it.StartDate.IsZero() || it.FinishDate.IsZero() {
		return ""
	}
	out := "   " + it.StartDate.Format("Jan 2") + " - " + it.FinishDate.Format("Jan 2")
	if days := workingDaysLeft(time.Now(), it.FinishDate); days > 0 {
		out += fmt.Sprintf(", %d working days left", days)
	}
	return out
}

func workingDaysLeft(from, to time.Time) int {
	days := 0
	for d := from.Truncate(24 * time.Hour); !d.After(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}

func (a *App) renderPane(width, height int) string {
	if a.inPullRequests() {
		return a.renderPullRequestPane(width, height)
	}
	if a.inPipelines() {
		return a.renderPipelinePane(width, height)
	}
	focused := a.isPanePanel()
	discussion := "Discussion"
	if n := a.commentsPanel.Count(); n > 0 {
		discussion = "Discussion " + strconv.Itoa(n)
	}
	active := 0
	content := a.detailsPanel.View()
	if a.rightTab == PanelComments {
		active = 1
		content = a.commentsPanel.View()
		if a.commentsPanel.Loading() {
			cw, ch := panels.PaneContentSize(width, height)
			content = panels.Block(panels.Loading(a.spinner.View(), "Loading discussion"), cw, ch)
		}
	}
	title := panels.Tabs([]string{
		a.keys.Panel6.Help().Key + " Details",
		a.keys.Panel7.Help().Key + " " + discussion,
	}, active, focused)
	return panels.Pane(title, content, width, height, focused)
}

func (a *App) renderDetailView() string {
	l := a.shell
	id := ""
	if item := a.detailView.Item(); item != nil {
		id = strconv.Itoa(item.ID)
	}
	body := panels.Block(a.detailView.View(), a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, a.renderHeader("Boards", a.hub.Name(), id), body, a.renderStatusBar())
}

func (a *App) modeLabel() string {
	switch {
	case a.reviewing():
		return a.pr.view.Mode()
	case a.viewingRun():
		return a.pl.view.Mode()
	case a.viewingRelease():
		return a.rl.view.Mode()
	case a.viewMode == ViewDetail:
		return "WORK ITEM"
	case a.keyword.Focused():
		return "FILTER"
	case a.searchingPullRequests(), a.searchingPipelines(), a.searchingReleases(), a.searchingCommits(), a.searchingBranches(), a.searchingTags():
		return "SEARCH"
	case a.mainView() == viewTaskboard && a.taskboardPanel.Dragging():
		return "MOVING"
	case a.activePanel == PanelDetails:
		return "DETAILS"
	case a.activePanel == PanelComments:
		return "DISCUSSION"
	default:
		return a.hub.Mode()
	}
}

func (a *App) renderStatusBar() string {
	pill := lipgloss.NewStyle().Foreground(styles.ColorOnAccent).Background(styles.ColorPrimary).Bold(true).
		Padding(0, 1).Render(a.modeLabel())
	rightItems := panels.HelpItems(a.statusHelp, []key.Binding{a.keys.Help, a.keys.Quit})
	right := panels.FitItems(rightItems, "   ", max(a.width-lipgloss.Width(pill)-1, 0))
	budget := max(a.width-lipgloss.Width(pill)-lipgloss.Width(right)-styles.StatusBarGaps, 0)
	hints := panels.FitItems(panels.HelpItems(a.statusHelp, a.statusBindings()), "   ", budget)
	left := pill + " " + hints
	gap := max(a.width-lipgloss.Width(left)-lipgloss.Width(right)-1, 1)
	return lipgloss.NewStyle().Background(styles.ColorSurface).Width(a.width).MaxWidth(a.width).
		Render(left + strings.Repeat(" ", gap) + right + " ")
}

func (a *App) statusBindings() []key.Binding {
	switch {
	case a.reviewing():
		return a.pr.view.KeyBindings()
	case a.viewingRun():
		return a.pl.view.KeyBindings()
	case a.viewingRelease():
		return a.rl.view.KeyBindings()
	case a.viewMode == ViewDetail:
		return a.detailView.KeyBindings()
	case a.keyword.Focused():
		return []key.Binding{widgets.Relabel(a.keys.Select, "apply"), widgets.Relabel(a.keys.Back, "done")}
	case a.activePanel == PanelDetails:
		return a.detailsPanel.KeyBindings()
	case a.activePanel == PanelComments:
		return a.commentsPanel.KeyBindings()
	}
	switch a.mainView() {
	case viewBoard:
		return a.boardPanel.KeyBindings()
	case viewTaskboard:
		return a.taskboardPanel.KeyBindings()
	case viewPullRequests:
		return a.pullRequestBindings()
	case viewPipelines:
		return a.pl.panel.KeyBindings()
	case viewReleases:
		return a.releaseBindings()
	case viewFiles:
		return a.filesBindings()
	case viewCommits:
		return a.commitsBindings()
	case viewBranches:
		return a.branchesBindings()
	case viewTags:
		return a.tagsBindings()
	default:
		return a.workItemsPanel.KeyBindings()
	}
}
