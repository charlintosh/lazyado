package ui

import (
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/styles"
)

type shellLayout struct {
	bodyHeight   int
	headerHeight int
	railWidth    int
	compactRail  bool
	mainWidth    int
	showPane     bool
	paneWidth    int
}

func (a *App) layout() shellLayout {
	l := shellLayout{
		bodyHeight:   max(a.height-styles.TopBarHeight-styles.StatusBarHeight, 0),
		headerHeight: styles.PageHeaderHeight,
	}
	switch {
	case a.fullscreen:
	case a.width >= styles.RailShowWidth:
		l.railWidth = styles.RailWidth
	case a.width >= styles.RailCompactShowWidth:
		l.railWidth, l.compactRail = styles.RailCompactWidth, true
	}
	rest := a.width - l.railWidth
	view := a.mainView()
	l.showPane = (view == viewList || view == viewPullRequests || view == viewPipelines) && a.width >= styles.RightPaneShowWidth
	if l.showPane {
		pw := int(float64(a.width) * styles.RightPaneWidthRatio)
		l.paneWidth = max(min(pw, styles.RightPaneMaxWidth), styles.RightPaneMinWidth)
		rest -= l.paneWidth
	}
	l.mainWidth = max(rest, 0)
	return l
}

func (l shellLayout) mainHeight() int {
	return max(l.bodyHeight-l.headerHeight, 0)
}

func (a *App) syncLayout() {
	if a.width == 0 || a.height == 0 {
		return
	}
	l := a.layout()
	a.header = a.pageHeader(l.mainWidth)
	l.headerHeight = a.header.Height
	a.shell = l
	cw, ch := panels.PaneContentSize(l.mainWidth, l.mainHeight())
	a.workItemsPanel.SetSize(cw, ch)
	a.boardPanel.SetSize(cw, ch)
	a.taskboardPanel.SetSize(cw, ch)
	a.pr.panel.SetSize(cw, ch)
	a.pl.panel.SetSize(cw, ch)
	a.syncReleaseLayout(l.mainWidth, l.mainHeight())
	a.syncFilesLayout(l.mainWidth, l.mainHeight())
	a.syncCommitsLayout(l.mainWidth, l.mainHeight())
	a.syncBranchesLayout(l.mainWidth, l.mainHeight())
	a.syncTagsLayout(l.mainWidth, l.mainHeight())
	originX, originY := l.railWidth+styles.PaneContentInsetX, styles.TopBarHeight+l.headerHeight+styles.PaneContentInsetY
	a.taskboardPanel.SetOrigin(originX, originY)
	a.boardPanel.SetOrigin(originX, originY)

	pw, ph := cw, ch
	if l.showPane {
		pw, ph = panels.PaneContentSize(l.paneWidth, l.bodyHeight)
	}
	a.detailsPanel.SetSize(pw, ph)
	a.commentsPanel.SetSize(pw, ph)
	a.detailView.SetSize(a.width, l.bodyHeight)
	a.pr.view.SetSize(a.width, l.bodyHeight)
	a.pl.view.SetSize(a.width, l.bodyHeight)
	a.rl.view.SetSize(a.width, l.bodyHeight)
}

func (a *App) syncFocus() {
	main := a.activePanel == PanelWorkItems
	view := a.mainView()
	a.workItemsPanel.SetFocused(main && view == viewList)
	a.boardPanel.SetFocused(main && view == viewBoard)
	a.taskboardPanel.SetFocused(main && view == viewTaskboard)
	a.pr.panel.SetFocused(main && view == viewPullRequests)
	a.pl.panel.SetFocused(main && view == viewPipelines)
	a.syncReleaseFocus(main)
	a.syncFilesFocus(main)
	a.commits.panel.SetFocused(main && a.inCommits() && a.viewMode == ViewMain)
	a.branches.panel.SetFocused(main && a.inBranches() && a.viewMode == ViewMain)
	a.tags.panel.SetFocused(main && a.inTags() && a.viewMode == ViewMain)
	a.detailsPanel.SetFocused(a.activePanel == PanelDetails)
	a.commentsPanel.SetFocused(a.activePanel == PanelComments)
	if a.isPanePanel() {
		a.rightTab = a.activePanel
	}
}

func (a *App) isPanePanel() bool {
	return a.activePanel == PanelDetails || a.activePanel == PanelComments
}
