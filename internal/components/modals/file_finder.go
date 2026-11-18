package modals

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	finderTitle        = "Go to file"
	finderPrompt       = "› "
	finderPlaceholder  = "type a file name…"
	finderRest         = 250 * time.Millisecond
	finderPreviewMin   = 90
	finderListShare    = 45
	finderPreviewLines = 400
	finderChrome       = 4
	finderGap          = " │ "
	finderGutter       = 5
)

type FileFinderPreviewRequestMsg struct {
	Path string
}

type FileFinderPreviewMsg struct {
	Path string
	Text string
	Err  error
}

type FileFinderIndexMsg struct {
	Key   string
	Index *models.FileIndex
	Err   error
}

type FileFinderRebuildMsg struct{}

type FileChosenMsg struct {
	Path string
}

type fileFinderRestMsg struct {
	seq int
}

type FileFinderProps struct {
	Key     string
	Version string
	Index   *models.FileIndex
	Known   []models.GitItem
	Recent  []string
	Err     error
}

type finderPreview struct {
	path     string
	lines    []components.CodeLine
	note     string
	rendered string
	width    int
	height   int
}

type FileFinder struct {
	env       Env
	props     FileFinderProps
	input     textinput.Model
	spinner   spinner.Model
	source    *models.FileIndex
	matches   []models.FuzzyMatch
	recent    []string
	lastQuery string
	cursor    int
	restSeq   int
	asked     map[string]bool
	previews  map[string]*finderPreview
}

func NewFileFinder(env Env, props FileFinderProps) FileFinder {
	in := textinput.New()
	in.Placeholder = finderPlaceholder
	in.CharLimit = styles.SearchCharLimit
	in.Prompt = finderPrompt
	st := in.Styles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true)
	st.Focused.Text = lipgloss.NewStyle().Foreground(styles.ColorText)
	in.SetStyles(st)
	in.Focus()
	m := FileFinder{
		env:      env,
		props:    props,
		input:    in,
		spinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary))),
		asked:    map[string]bool{},
		previews: map[string]*finderPreview{},
	}
	m.useSource()
	m.layout()
	return m
}

func (m FileFinder) Key() string { return m.props.Key }

func (m FileFinder) indexing() bool { return m.props.Index == nil && m.props.Err == nil }

func (m FileFinder) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.scheduleRest()}
	if m.indexing() {
		cmds = append(cmds, m.spinner.Tick)
	}
	return tea.Batch(cmds...)
}

func (m *FileFinder) useSource() {
	if m.props.Index != nil {
		m.source = m.props.Index
	} else {
		m.source = models.NewFileIndex(m.props.Known, "", false)
	}
	m.recent = m.recent[:0]
	for _, p := range m.props.Recent {
		if m.props.Index == nil || m.props.Index.Has(p) {
			m.recent = append(m.recent, p)
		}
	}
	m.lastQuery = ""
	m.refresh()
}

func (m *FileFinder) refresh() {
	query := strings.TrimSpace(m.input.Value())
	var within []int
	if m.lastQuery != "" && strings.HasPrefix(query, m.lastQuery) {
		within = make([]int, len(m.matches))
		for i, mt := range m.matches {
			within[i] = mt.Index
		}
	}
	if query == "" {
		m.matches = nil
	} else {
		m.matches = m.source.Match(query, within)
	}
	m.lastQuery = query
	m.cursor = 0
}

func (m FileFinder) count() int {
	if strings.TrimSpace(m.input.Value()) == "" {
		return len(m.recent)
	}
	return len(m.matches)
}

func (m FileFinder) selectedPath() string {
	switch {
	case m.cursor >= m.count():
		return ""
	case strings.TrimSpace(m.input.Value()) == "":
		return m.recent[m.cursor]
	}
	return "/" + m.source.File(m.matches[m.cursor].Index)
}

func (m *FileFinder) scheduleRest() tea.Cmd {
	m.restSeq++
	seq := m.restSeq
	return tea.Tick(finderRest, func(time.Time) tea.Msg { return fileFinderRestMsg{seq: seq} })
}

func (m FileFinder) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		if !m.indexing() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case FileFinderIndexMsg:
		if msg.Key != m.props.Key {
			return m, nil
		}
		m.props.Index, m.props.Err = msg.Index, msg.Err
		m.useSource()
		return m, m.scheduleRest()
	case fileFinderRestMsg:
		path := m.selectedPath()
		if msg.seq != m.restSeq || path == "" || m.asked[path] {
			return m, nil
		}
		m.asked[path] = true
		return m, send(FileFinderPreviewRequestMsg{Path: path})
	case FileFinderPreviewMsg:
		m.asked[msg.Path] = true
		m.previews[msg.Path] = newPreview(msg)
		return m, nil
	case tea.PasteMsg:
		return m.typed(msg)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func newPreview(msg FileFinderPreviewMsg) *finderPreview {
	p := &finderPreview{path: msg.Path}
	switch {
	case msg.Err != nil:
		p.note = msg.Err.Error()
	case models.IsBinary(msg.Text):
		p.note = "Binary file"
	default:
		lines := strings.SplitN(msg.Text, "\n", finderPreviewLines+1)
		p.lines = components.HighlightCode(msg.Path, strings.Join(lines[:min(len(lines), finderPreviewLines)], "\n"))
	}
	return p
}

func (m FileFinder) handleKey(msg tea.KeyPressMsg) (Modal, tea.Cmd) {
	k := m.env.Keys
	arrow := msg.Text == ""
	switch {
	case key.Matches(msg, k.Back):
		return m, closeModal
	case key.Matches(msg, k.Select):
		if path := m.selectedPath(); path != "" {
			return m, send(FileChosenMsg{Path: path})
		}
		return m, nil
	case key.Matches(msg, k.Refresh):
		m.props.Index, m.props.Err = nil, nil
		m.useSource()
		return m, tea.Batch(send(FileFinderRebuildMsg{}), m.spinner.Tick)
	case key.Matches(msg, k.PickerUp), arrow && key.Matches(msg, k.Up):
		return m.moveCursor(-1)
	case key.Matches(msg, k.PickerDown), arrow && key.Matches(msg, k.Down):
		return m.moveCursor(1)
	}
	return m.typed(msg)
}

func (m FileFinder) typed(msg tea.Msg) (Modal, tea.Cmd) {
	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() == before {
		return m, cmd
	}
	m.refresh()
	return m, tea.Batch(cmd, m.scheduleRest())
}

func (m FileFinder) moveCursor(delta int) (Modal, tea.Cmd) {
	next := min(max(m.cursor+delta, 0), max(m.count()-1, 0))
	if next == m.cursor {
		return m, nil
	}
	m.cursor = next
	return m, m.scheduleRest()
}

func (m *FileFinder) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.input.SetWidth(max(inner/2, 10))
}

func (m FileFinder) width() int {
	return widgets.DialogWidth(styles.ModalWidthFinder, m.env.Width)
}

func (m FileFinder) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	height := max(widgets.DialogBodyHeight(m.env.Height, false)-finderChrome, 3)
	listWidth, previewWidth := inner, 0
	if inner >= finderPreviewMin {
		listWidth = inner * finderListShare / 100
		previewWidth = inner - listWidth - lipgloss.Width(finderGap)
	}
	list := m.listView(listWidth, height)
	body := list
	if previewWidth > 0 {
		gap := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(finderGap)
		body = widgets.JoinColumns(list, strings.TrimSuffix(strings.Repeat(gap+"\n", height), "\n"), m.previewView(previewWidth, height))
	}
	head := spread(m.input.View(), muted(m.countLabel()), inner)
	k := m.env.Keys
	footer := widgets.Hints(inner, widgets.Relabel(k.Select, "open"), widgets.Combine("move", k.PickerUp, k.PickerDown),
		widgets.Relabel(k.Refresh, "rebuild index"), widgets.Relabel(k.Back, "close"))
	parts := []string{head, "", body, m.statusLine(inner)}
	return widgets.Dialog{Title: finderTitle, Body: strings.Join(parts, "\n"), Footer: footer, Width: m.width()}.View()
}

func (m FileFinder) countLabel() string {
	total := m.source.Len()
	if strings.TrimSpace(m.input.Value()) == "" {
		return fmt.Sprintf("%d files", total)
	}
	return fmt.Sprintf("%d of %d files", len(m.matches), total)
}

func (m FileFinder) statusLine(width int) string {
	switch {
	case m.props.Err != nil:
		return lipgloss.NewStyle().Foreground(styles.ColorError).Render(ansi.Truncate("Index failed: "+m.props.Err.Error()+" · ctrl+r to retry", width, "…"))
	case m.indexing():
		return m.spinner.View() + muted(" Indexing "+m.props.Version+" · showing files from folders you've opened")
	}
	x := m.props.Index
	text := "Indexed " + m.props.Version
	if x.Commit != "" {
		text += " @ " + shortID(x.Commit)
	}
	if x.Truncated {
		return lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("! ") + muted(ansi.Truncate(fmt.Sprintf("indexed the first %d files · %s", x.Len(), text), width-2, "…"))
	}
	return muted(ansi.Truncate(text, width, "…"))
}

func shortID(id string) string {
	return id[:min(len(id), 7)]
}

func (m FileFinder) listView(width, height int) string {
	var lines []string
	query := strings.TrimSpace(m.input.Value())
	switch {
	case query == "" && len(m.recent) > 0:
		lines = append(lines, heading("Recently opened"))
		for i, p := range m.recent {
			lines = append(lines, finderRow(strings.TrimPrefix(p, "/"), nil, i == m.cursor, width))
		}
	case query == "":
		lines = append(lines, muted("Type part of a file name or path."), muted("Letters match in order, with gaps."), muted("Spaces split words; a trailing / picks a folder."))
	case len(m.matches) == 0 && m.indexing():
		lines = append(lines, muted("Nothing in the folders you've opened yet."), muted("Waiting for the index…"))
	case len(m.matches) == 0:
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render("No file matches "+query), muted("Letters must appear in order: rgate finds release/gate.go."))
	default:
		start := min(max(m.cursor-height/2, 0), max(len(m.matches)-height, 0))
		for i := start; i < len(m.matches) && len(lines) < height; i++ {
			mt := m.matches[i]
			lines = append(lines, finderRow(m.source.File(mt.Index), mt.Positions, i == m.cursor, width))
		}
	}
	return fitLines(lines, width, height)
}

func fitLines(lines []string, width, height int) string {
	for i, l := range lines {
		if ansi.StringWidth(l) > width {
			lines[i] = ansi.Truncate(l, width, "…")
		}
	}
	return widgets.Block(strings.Join(lines[:min(len(lines), height)], "\n"), width, height)
}

func finderRow(path string, positions []int, selected bool, width int) string {
	base := lipgloss.NewStyle()
	bar := " "
	nameStyle := base.Foreground(styles.ColorText).Bold(true)
	dirStyle := base.Foreground(styles.ColorTextMuted)
	if selected {
		base, bar = base.Background(styles.ColorHighlight), "▌"
		dirStyle = base.Foreground(styles.ColorLight)
		nameStyle = base.Foreground(styles.ColorText).Bold(true)
	}
	hit := base.Foreground(styles.ColorWarning).Bold(true)
	runes := []rune(path)
	split := 0
	if cut := strings.LastIndex(path, "/"); cut >= 0 {
		split = len([]rune(path[:cut+1]))
	}
	var b strings.Builder
	b.WriteString(base.Foreground(styles.ColorPrimary).Bold(true).Render(bar))
	b.WriteString(base.Render(" "))
	paintRuns(&b, runes, split, len(runes), positions, nameStyle, hit)
	if split > 0 {
		b.WriteString(base.Render("  "))
		paintRuns(&b, runes, 0, split, positions, dirStyle, hit)
	}
	row := b.String()
	if w := ansi.StringWidth(row); w > width {
		return ansi.Truncate(row, width, "…")
	} else if selected {
		return row + base.Render(strings.Repeat(" ", width-w))
	}
	return row
}

func paintRuns(b *strings.Builder, runes []rune, from, to int, positions []int, plain, hit lipgloss.Style) {
	p := 0
	for p < len(positions) && positions[p] < from {
		p++
	}
	for start := from; start < to; {
		marked := p < len(positions) && positions[p] == start
		end := start + 1
		if marked {
			for p++; end < to && p < len(positions) && positions[p] == end; p++ {
				end++
			}
		} else {
			for end < to && (p >= len(positions) || positions[p] != end) {
				end++
			}
		}
		st := plain
		if marked {
			st = hit
		}
		b.WriteString(st.Render(string(runes[start:end])))
		start = end
	}
}

func (m FileFinder) previewView(width, height int) string {
	path := m.selectedPath()
	preview, ready := m.previews[path]
	var lines []string
	switch {
	case path == "":
	case !ready:
		lines = append(lines, muted(ansi.Truncate(strings.TrimPrefix(path, "/"), width, "…")), "", muted("Preview loads when you stop on a file."))
	case preview.note != "":
		lines = append(lines, muted(ansi.Truncate(strings.TrimPrefix(path, "/"), width, "…")), "", muted(preview.note))
	default:
		return preview.render(width, height)
	}
	return fitLines(lines, width, height)
}

func (p *finderPreview) render(width, height int) string {
	if p.rendered != "" && p.width == width && p.height == height {
		return p.rendered
	}
	code := max(width-finderGutter-1, 1)
	num := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	text := lipgloss.NewStyle().Foreground(styles.ColorText)
	lines := make([]string, 0, min(len(p.lines), height))
	for i, l := range p.lines[:min(len(p.lines), height)] {
		lines = append(lines, num.Render(fmt.Sprintf("%*d ", finderGutter, i+1))+l.Render(text, code))
	}
	p.rendered, p.width, p.height = fitLines(lines, width, height), width, height
	return p.rendered
}

func muted(s string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(s)
}

func heading(s string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(strings.ToUpper(s))
}

func spread(left, right string, width int) string {
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return ansi.Truncate(left+strings.Repeat(" ", gap)+right, width, "…")
}
