package ui

import (
	"fmt"
	"image/color"
	"path"
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	statusItemMax   = 28
	statusSeparator = "│"
	statusHintGap   = "   "
)

type focusTone int

const (
	toneArea focusTone = iota
	toneEdit
	toneSearch
	toneMove
	toneSelect
)

type statusFocus struct {
	area   string
	screen string
	item   string
	tone   focusTone
}

func workItemLabel(it *models.WorkItem) string {
	if it == nil {
		return ""
	}
	return strings.ToUpper(it.ShortType()) + fmt.Sprintf(" #%d", it.ID)
}

func (a *App) statusFocus() statusFocus {
	f := statusFocus{area: a.hub.Section(), screen: a.hub.Mode()}
	switch {
	case a.reviewing():
		f.area, f.screen = panels.HubPullRequests.Section(), "REVIEW"
		if pr := a.pr.view.PullRequest(); pr != nil {
			f.screen = fmt.Sprintf("REVIEW !%d", pr.ID)
		}
		item, selecting := a.pr.view.Focus()
		f.item = item
		if selecting {
			f.tone = toneSelect
		}
	case a.viewingRun():
		f.area, f.screen = panels.HubBuilds.Section(), "RUN"
		if r := a.pl.view.Run(); r != nil && r.Number != "" {
			f.screen = "RUN " + r.Number
		}
		f.item = a.pl.view.Focus()
	case a.viewingRelease():
		f.area, f.screen = panels.HubReleases.Section(), "RELEASE"
		if r := a.rl.view.Release(); r != nil {
			f.screen = r.Name
		}
		f.item = a.rl.view.Focus()
	case a.viewingReleaseEditor():
		e := &a.rl.editor
		f.area, f.screen, f.item = panels.HubReleases.Section(), e.Mode(), e.Focus()
		if d := e.Doc(); d != nil && !e.Editing() && d.Kind == models.DocRelease {
			f.screen = d.Name
		}
		if e.Editing() {
			f.tone = toneEdit
		}
	case a.viewMode == ViewDetail:
		f.screen, f.item = "WORK ITEM", workItemLabel(a.detailView.Item())
	default:
		a.mainFocus(&f)
	}
	f.item = ansi.Truncate(f.item, statusItemMax, "")
	return f
}

func (a *App) mainFocus(f *statusFocus) {
	switch {
	case a.keyword.Focused():
		f.item, f.tone = "FILTER", toneSearch
	case a.searchingReleases():
		f.item, f.tone = "SEARCH /"+a.rl.defs.Filter(), toneSearch
	case a.searchingPullRequests(), a.searchingPipelines(), a.searchingCommits(), a.searchingBranches(), a.searchingTags():
		f.item, f.tone = "SEARCH", toneSearch
	case a.mainView() == viewTaskboard && a.taskboardPanel.Dragging(), a.mainView() == viewBoard && a.boardPanel.Dragging():
		f.item, f.tone = "MOVING", toneMove
		if it := a.selectedItem(); it != nil {
			f.item += fmt.Sprintf(" #%d", it.ID)
		}
	case a.activePanel == PanelDetails:
		f.item = "DETAILS"
	case a.activePanel == PanelComments:
		f.item = "DISCUSSION"
	default:
		f.item = a.hubItem(f)
	}
}

func (a *App) hubItem(f *statusFocus) string {
	switch a.hub {
	case panels.HubPullRequests:
		if pr := a.pr.panel.Selected(); pr != nil {
			return fmt.Sprintf("!%d", pr.ID)
		}
	case panels.HubBuilds:
		if r := a.pl.panel.SelectedRun(); r != nil {
			return "RUN " + r.Number
		}
		if p := a.pl.panel.SelectedPipeline(); p != nil {
			return p.Name
		}
	case panels.HubReleases:
		return a.releaseItem()
	case panels.HubFiles:
		if a.code.files.pane == paneContent && a.code.files.content.Selecting() {
			if lines, ok := a.code.files.content.SelectedLines(); ok {
				f.tone = toneSelect
				return panels.LinesLabel(lines.To - lines.From + 1)
			}
		}
		if it := a.code.files.tree.Selected(); it != nil {
			return path.Base(it.Path)
		}
	case panels.HubCommits:
		if c := a.commits.panel.Selected(); c != nil && len(c.ID) >= 8 {
			return c.ID[:8]
		}
	case panels.HubBranches:
		if b := a.branches.panel.Selected(); b != nil {
			return b.Name
		}
	case panels.HubTags:
		if t := a.tags.panel.Selected(); t != nil {
			return t.Name
		}
	default:
		return workItemLabel(a.selectedItem())
	}
	return ""
}

func (a *App) releaseItem() string {
	switch {
	case a.rl.pane == paneDefinitions:
		if d := a.rl.defs.Selected(); d != nil {
			return d.Name
		}
	case a.rl.pivot == approvalsTab:
		if ap, ok := a.rl.approvals.Selected(); ok && ap != nil {
			return "APPROVAL " + ap.StageName
		}
	default:
		if r := a.rl.list.Selected(); r != nil {
			return r.Name
		}
	}
	return ""
}

func areaColors(area string) (color.Color, color.Color, color.Color) {
	switch area {
	case panels.HubPullRequests.Section():
		return styles.ColorViolet, styles.ColorVioletTint, styles.ColorVioletLight
	case panels.HubBuilds.Section():
		return styles.ColorSecondary, styles.ColorTealTint, styles.ColorTealLight
	}
	return styles.ColorPrimary, styles.ColorDarkBlue, styles.ColorPrimaryLight
}

func (f statusFocus) colors() (screenBg, screenFg, itemBg, itemFg color.Color) {
	screenBg, itemBg, itemFg = areaColors(f.area)
	screenFg = styles.ColorOnAccent
	switch f.tone {
	case toneEdit:
		screenBg, itemBg, itemFg = styles.ColorOrange, styles.ColorWaitTint, styles.ColorWaitLight
	case toneSearch:
		screenBg, itemBg, itemFg = styles.ColorWarning, styles.ColorWarningTint, styles.ColorWarning
	case toneMove:
		screenBg, screenFg, itemBg, itemFg = styles.ColorPink, styles.ColorText, styles.ColorSecretTint, styles.ColorSecretLight
	case toneSelect:
		itemBg, itemFg = styles.ColorMatch, styles.ColorText
	}
	return screenBg, screenFg, itemBg, itemFg
}

func (f statusFocus) pill(compact bool) string {
	screenBg, screenFg, itemBg, itemFg := f.colors()
	screen := lipgloss.NewStyle().Background(screenBg).Foreground(screenFg).Bold(true).Padding(0, 1).Render(f.screen)
	if f.item == "" {
		return screen
	}
	item := lipgloss.NewStyle().Background(itemBg).Foreground(itemFg).Bold(true).Padding(0, 1).Render("› " + f.item)
	if compact {
		return lipgloss.NewStyle().Background(screenBg).Foreground(screenFg).Bold(true).Render(" ") + item
	}
	return screen + item
}

func keySet(keys []string) string {
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	return strings.Join(slices.Compact(sorted), ",")
}

func navigationKeys(k keys.KeyMap) map[string]bool {
	nav := []key.Binding{
		k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom, k.NextPanel, k.PrevPanel, k.PickerUp, k.PickerDown,
		k.HalfPageUp, k.HalfPageDown, k.Panel1, k.Panel2, k.Panel3, k.Panel4, k.Panel5, k.Panel6, k.Panel7,
		k.PrevView, k.NextView, k.PrevStage, k.NextStage, k.Back, k.Search, k.NextFile, k.PrevFile,
	}
	sets := map[string]bool{}
	for i, x := range nav {
		sets[keySet(x.Keys())] = true
		for _, y := range nav[i+1:] {
			sets[keySet(append(slices.Clone(x.Keys()), y.Keys()...))] = true
		}
	}
	navSingles := map[string]bool{}
	for _, x := range nav {
		navSingles[keySet(x.Keys())+"|"+x.Help().Desc] = true
	}
	for _, group := range k.FullHelp() {
		for _, b := range group {
			if !navSingles[keySet(b.Keys())+"|"+b.Help().Desc] {
				delete(sets, keySet(b.Keys()))
			}
		}
	}
	return sets
}

func (a *App) splitBindings(bindings []key.Binding) ([]key.Binding, []key.Binding) {
	var actions, nav []key.Binding
	for _, b := range bindings {
		if a.navKeys[keySet(b.Keys())] {
			nav = append(nav, b)
		} else {
			actions = append(actions, b)
		}
	}
	return actions, nav
}

func dimHelp(h help.Model) help.Model {
	st := h.Styles
	st.ShortKey = lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true)
	st.ShortDesc = lipgloss.NewStyle().Foreground(styles.ColorMuted)
	h.Styles = st
	return h
}

func (a *App) renderStatusBar() string {
	f := a.statusFocus()
	rightItems := panels.HelpItems(a.statusHelp, []key.Binding{a.keys.Help, a.keys.Quit})
	pill := f.pill(false)
	if lipgloss.Width(pill) > a.width/2 {
		pill = f.pill(true)
	}
	right := panels.FitItems(rightItems, statusHintGap, max(a.width-lipgloss.Width(pill)-1, 0))
	budget := max(a.width-lipgloss.Width(pill)-lipgloss.Width(right)-styles.StatusBarGaps, 0)
	actions, nav := a.splitBindings(a.statusBindings())
	sep := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(" " + statusSeparator + " ")
	navItems := panels.HelpItems(a.statusNavHelp, nav)
	reserve := 0
	if len(navItems) > 0 {
		reserve = min(lipgloss.Width(strings.Join(navItems, statusHintGap))+lipgloss.Width(sep), budget/3)
	}
	hints := panels.FitItems(panels.HelpItems(a.statusHelp, actions), statusHintGap, budget-reserve)
	if room := budget - lipgloss.Width(hints) - lipgloss.Width(sep); len(nav) > 0 && room > 0 {
		if navHints := panels.FitItems(navItems, statusHintGap, room); navHints != "" {
			if hints != "" {
				hints += sep
			}
			hints += navHints
		}
	}
	left := pill + " " + hints
	gap := max(a.width-lipgloss.Width(left)-lipgloss.Width(right)-1, 1)
	return lipgloss.NewStyle().Background(styles.ColorSurface).Width(a.width).MaxWidth(a.width).
		Render(left + strings.Repeat(" ", gap) + right + " ")
}
