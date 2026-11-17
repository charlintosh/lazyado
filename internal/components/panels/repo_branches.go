package panels

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	BranchesMine = iota
	BranchesAll
	BranchesStale
)

const (
	branchSearchHint  = "branch name"
	branchAuthorWidth = 18
	branchDateWidth   = 12
	branchAheadWidth  = 16
	branchPRWidth     = 16
	loadingBranches   = "Loading branches"
	behindWidth       = len("Behind")
)

type BranchesPanel struct {
	list      denseList
	keys      keys.KeyMap
	branches  []models.Branch
	visible   []int
	view      int
	search    textinput.Model
	searching bool
	loading   bool
	spinner   string
	now       time.Time
	width     int
	height    int
}

func NewBranchesPanel(k keys.KeyMap) BranchesPanel {
	return BranchesPanel{list: newDenseList(k), keys: k, loading: true, search: widgets.SearchInput(searchPrompt, branchSearchHint)}
}

func (p *BranchesPanel) ShowLoading() {
	p.branches, p.loading = nil, true
	p.rebuild()
}

func (p *BranchesPanel) SetBranches(branches []models.Branch, now time.Time) {
	p.branches, p.loading, p.now = branches, false, now
	p.rebuild()
}

func (p *BranchesPanel) SetView(view int) {
	if p.view == view {
		return
	}
	p.view = view
	p.list.cursor = 0
	p.rebuild()
}

func (p *BranchesPanel) Pivot() int { return p.view }

func (p *BranchesPanel) Loading() bool { return p.loading }

func (p *BranchesPanel) Searching() bool { return p.searching }

func (p *BranchesPanel) Filter() string { return strings.TrimSpace(p.search.Value()) }

func (p *BranchesPanel) Selected() *models.Branch {
	if p.list.cursor < 0 || p.list.cursor >= len(p.visible) {
		return nil
	}
	b := p.branches[p.visible[p.list.cursor]]
	return &b
}

func (p *BranchesPanel) Counts() [3]int {
	var out [3]int
	for _, b := range p.branches {
		for v := range out {
			if p.inView(b, v) {
				out[v]++
			}
		}
	}
	return out
}

func (p *BranchesPanel) inView(b models.Branch, view int) bool {
	switch view {
	case BranchesMine:
		return b.Mine || b.Default
	case BranchesStale:
		return b.Stale(p.now)
	}
	return true
}

func (p *BranchesPanel) SetFocused(focused bool) {
	if p.list.focused == focused {
		return
	}
	p.list.focused = focused
	p.rebuild()
}

func (p *BranchesPanel) SetSpinner(frame string) {
	if p.spinner == frame {
		return
	}
	p.spinner = frame
	if p.loading {
		p.rebuild()
	}
}

func (p *BranchesPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.search.SetWidth(max(width-4, 1))
	p.layout()
}

func (p *BranchesPanel) layout() {
	extra := 1
	if p.searching {
		extra += 2
	}
	p.list.setSize(p.width, max(p.height-extra, 1))
	p.rebuild()
}

func (p *BranchesPanel) FocusSearch() tea.Cmd {
	p.searching = true
	p.search.CursorEnd()
	p.layout()
	return p.search.Focus()
}

func (p *BranchesPanel) ClearFilter() bool {
	if p.search.Value() == "" {
		return false
	}
	p.search.SetValue("")
	p.rebuild()
	return true
}

func (p *BranchesPanel) Title(base string) string {
	if p.loading {
		return "Branches"
	}
	title := fmt.Sprintf("Branches %d", len(p.visible))
	if filter := p.Filter(); filter != "" && !p.searching {
		title += " · /" + filter
	}
	if base != "" && p.width >= 60 {
		title += " · ahead and behind " + base
	}
	return title
}

func (p *BranchesPanel) columns() []column {
	return fitColumns(p.width, 0,
		column{title: "Branch"},
		column{title: "Commit", width: commitIDWidth},
		column{title: "Author", width: branchAuthorWidth},
		column{title: "Updated", width: branchDateWidth},
		column{title: "Behind | Ahead", width: branchAheadWidth},
		column{title: "Pull request", width: branchPRWidth},
	)
}

func (p *BranchesPanel) rebuild() {
	filter := strings.ToLower(p.Filter())
	p.visible = p.visible[:0]
	for i, b := range p.branches {
		if p.inView(b, p.view) && strings.Contains(strings.ToLower(b.Name), filter) {
			p.visible = append(p.visible, i)
		}
	}
	p.list.setCount(len(p.visible))
	cols := p.columns()
	rows := make([]string, len(p.visible))
	for i, idx := range p.visible {
		rows[i] = p.row(p.branches[idx], i, cols)
	}
	footer := ""
	switch {
	case p.loading:
		footer = Loading(p.spinner, loadingBranches)
	case len(p.visible) == 0 && filter != "":
		footer = Muted("No branch matches.")
	case len(p.visible) == 0 && p.view == BranchesStale:
		footer = Muted("No stale branches: every branch has a commit from the last 3 months.")
	case len(p.visible) == 0:
		footer = Muted("No branches here yet. " + p.keys.NewRef.Help().Key + " creates one.")
	}
	p.list.render(rows, footer)
}

func (p *BranchesPanel) row(b models.Branch, i int, cols []column) string {
	base, lead := p.list.rowStyle(i)
	selected := i == p.list.cursor
	name := base.Foreground(styles.ColorText).Bold(selected).Render(b.Name)
	if b.Default {
		name += base.Render(" ") + chipOn(base, "Default", styles.ColorDarkBlue, styles.ColorPrimaryLight)
	}
	if b.Locked {
		name += base.Render(" ") + chipOn(base, "Locked", styles.ColorChip, styles.ColorLight)
	}
	out := lead
	for j, col := range cols {
		switch j {
		case 0:
			out += cellText(name, col.width, base)
		case 1:
			out += cellText(models.ShortCommit(b.Last.ID), col.width, base.Foreground(styles.ColorViolet))
		case 2:
			out += cellText(b.Last.Author, col.width, base.Foreground(styles.ColorText))
		case 3:
			out += cellText(ShortAgo(b.Last.Date), col.width, base.Foreground(styles.ColorTextMuted))
		case 4:
			out += cellText(aheadBehind(base, b), col.width, base)
		case 5:
			out += cellText(pullRequestCell(base, b), col.width, base)
		}
	}
	return fill(out, "", base, p.width)
}

func chipOn(base lipgloss.Style, text string, bg, fg color.Color) string {
	return base.Background(bg).Foreground(fg).Padding(0, 1).Render(text)
}

func aheadBehind(base lipgloss.Style, b models.Branch) string {
	muted := base.Foreground(styles.ColorMuted)
	if b.Default {
		return base.Render(strings.Repeat(" ", behindWidth+1)) + muted.Render("-")
	}
	return base.Foreground(styles.ColorWarning).Render(fmt.Sprintf("%*d", behindWidth, b.Behind)) + muted.Render(" | ") +
		base.Foreground(styles.ColorSuccess).Render(strconv.Itoa(b.Ahead))
}

func pullRequestCell(base lipgloss.Style, b models.Branch) string {
	if b.PullRequestID == 0 {
		return ""
	}
	return base.Foreground(styles.ColorPrimary).Render("!"+strconv.Itoa(b.PullRequestID)) + base.Foreground(styles.ColorTextMuted).Render(" active")
}

func (p *BranchesPanel) KeyBindings() []key.Binding {
	k := p.keys
	if p.searching {
		return []key.Binding{widgets.Relabel(k.Select, "done"), widgets.Relabel(k.Back, "clear")}
	}
	bindings := []key.Binding{widgets.Relabel(k.NewRef, "new branch")}
	if b := p.Selected(); b != nil && !b.Default {
		bindings = append(bindings, widgets.Relabel(k.DeleteRef, "delete"))
	}
	bindings = append(bindings, widgets.Relabel(k.Select, "files"), widgets.Relabel(k.HubCommits, "commits"))
	if b := p.Selected(); b != nil && !b.Default {
		bindings = append(bindings, k.PullRequestFrom)
	}
	if p.Filter() != "" {
		bindings = append(bindings, widgets.Relabel(k.Back, "clear filter"))
	}
	return append(bindings, widgets.Relabel(k.Search, "search"), widgets.Relabel(k.Copy, "copy name"), k.Repository, k.OpenBrowser, k.Refresh)
}

func (p BranchesPanel) Update(msg tea.Msg) (BranchesPanel, tea.Cmd) {
	if p.searching {
		if k, ok := msg.(tea.KeyPressMsg); ok {
			switch {
			case key.Matches(k, p.keys.Select):
				p.searching = false
				p.search.Blur()
				p.layout()
				return p, nil
			case key.Matches(k, p.keys.Back):
				p.searching = false
				p.search.Blur()
				p.search.SetValue("")
				p.layout()
				return p, nil
			}
		}
		var cmd tea.Cmd
		p.search, cmd = p.search.Update(msg)
		p.list.cursor = 0
		p.rebuild()
		return p, cmd
	}
	if moved, handled := p.list.handleMove(msg, p.keys); handled && moved {
		p.rebuild()
	}
	return p, nil
}

func (p *BranchesPanel) View() string {
	parts := []string{}
	if p.searching {
		parts = append(parts, Line(p.search.View(), p.width), "")
	}
	parts = append(parts, Line(columnHeader(p.columns()), p.width), p.list.view())
	return Block(strings.Join(parts, "\n"), p.width, p.height)
}

func (p *BranchesPanel) SelectName(name string) {
	for i, idx := range p.visible {
		if p.branches[idx].Name == name {
			p.list.cursor = i
			p.rebuild()
			return
		}
	}
}
