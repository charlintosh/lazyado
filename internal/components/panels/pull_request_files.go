package panels

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/diff"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	diffHeaderHeight   = 2
	rangeBar           = "│"
	maxPinnedRange     = 200
	loadingFiles       = "Loading files"
	loadingDiff        = "Loading "
	noFiles            = "This pull request has no file changes."
	notRenderedNote    = "not rendered"
	tooLargeTitle      = "This file is too large to show"
	tooLargeNote       = "too large"
	noTextChanges      = "No line changes to show (the file was renamed, or only its mode changed)."
	binaryFile         = "Binary file not shown."
	outsideChangesNote = "Comments on the file, or on lines that are no longer in it"
)

type diffSkip int

const (
	notSkipped diffSkip = iota
	skipLong
	skipLarge
	skipGenerated
	skipNoDiff
	skipConfigured
	skipTooBig
)

var skipFor = map[diff.Reason]diffSkip{
	diff.Generated:  skipGenerated,
	diff.NoDiff:     skipNoDiff,
	diff.Configured: skipConfigured,
}

var fileNotes = map[diff.Reason]string{
	diff.Generated:  "generated",
	diff.NoDiff:     "no diff",
	diff.Configured: "collapsed",
}

type fileDiff struct {
	skip    diffSkip
	forced  bool
	changed int
	length  int
	err     string
	oldText string
	newText string
	pins    string
	lines   []diff.Line
	added   int
	deleted int
	oldCode []components.CodeLine
	newCode []components.CodeLine
}

func (d fileDiff) code(l diff.Line) components.CodeLine {
	side, n := d.newCode, l.New
	if l.Kind == diff.Delete {
		side, n = d.oldCode, l.Old
	}
	if n < 1 || n > len(side) {
		return components.CodeLine{{Text: components.SanitizeCode(l.Text)}}
	}
	return side[n-1]
}

func (f *pullRequestFiles) highlighted(filePath string, d fileDiff) fileDiff {
	if d.newCode == nil && d.oldCode == nil && !d.collapsed() && len(d.lines) > 0 {
		d.oldCode = components.HighlightCode(filePath, d.oldText)
		d.newCode = components.HighlightCode(filePath, d.newText)
		f.diffs[filePath] = d
	}
	return d
}

func threadPins(threads []models.PullRequestThread) diff.Pins {
	var pins diff.Pins
	for _, t := range threads {
		if t.Line <= 0 {
			continue
		}
		for n := t.Line; n <= min(t.LastLine(), t.Line+maxPinnedRange); n++ {
			if t.LeftSide {
				pins.Old = append(pins.Old, n)
			} else {
				pins.New = append(pins.New, n)
			}
		}
		if t.LeftSide {
			pins.Old = append(pins.Old, t.LastLine())
		} else {
			pins.New = append(pins.New, t.LastLine())
		}
	}
	return pins
}

func (d fileDiff) collapsed() bool {
	return d.skip != notSkipped && !d.forced
}

func (d fileDiff) loadable() bool {
	return d.collapsed() && d.skip != skipTooBig
}

func (f *pullRequestFiles) pinned(filePath string, threads []models.PullRequestThread) (fileDiff, bool) {
	d, ok := f.diffs[filePath]
	if !ok || d.err != "" || d.collapsed() {
		return d, ok
	}
	pins := threadPins(threads)
	key := fmt.Sprint(pins.Old, pins.New)
	if d.pins == key {
		return d, true
	}
	d.pins = key
	d.lines = diff.Unified(d.oldText, d.newText, styles.DiffContextLines, pins)
	d.added, d.deleted = diff.Stats(d.lines)
	f.diffs[filePath] = d
	return d, true
}

type lineJump struct {
	path string
	line int
	left bool
}

type pullRequestFiles struct {
	keys       keys.KeyMap
	names      map[string]string
	me         models.Identity
	spinner    string
	limits     models.DiffLimits
	attributes diff.Attributes
	changes    models.PullRequestChanges
	loaded     bool
	threads    []models.PullRequestThread
	diffs      map[string]fileDiff
	requested  map[string]bool
	cursor     int
	line       int
	mark       int
	rows       []string
	offsets    []int
	jump       *lineJump
	focus      commentFocus
	focusRel   [2]int
	expanded   map[int]bool
	treeFocus  bool
	tree       viewport.Model
	diff       viewport.Model
	width      int
	height     int
}

func newPullRequestFiles(k keys.KeyMap, limits models.DiffLimits) pullRequestFiles {
	return pullRequestFiles{
		keys:      k,
		limits:    limits.WithDefaults(),
		mark:      -1,
		focusRel:  [2]int{-1, -1},
		expanded:  map[int]bool{},
		diffs:     map[string]fileDiff{},
		requested: map[string]bool{},
		tree:      widgets.NewViewport(k),
		diff:      widgets.NewViewport(k),
	}
}

func (f *pullRequestFiles) file() (models.PullRequestFile, bool) {
	if f.cursor < 0 || f.cursor >= len(f.changes.Files) {
		return models.PullRequestFile{}, false
	}
	return f.changes.Files[f.cursor], true
}

func (f *pullRequestFiles) current() (fileDiff, bool) {
	file, ok := f.file()
	if !ok {
		return fileDiff{}, false
	}
	d, ok := f.diffs[file.Path]
	return d, ok
}

func (f *pullRequestFiles) selecting() bool {
	return f.mark >= 0
}

func (f *pullRequestFiles) selection() (int, int) {
	if !f.selecting() {
		return f.line, f.line
	}
	return min(f.mark, f.line), max(f.mark, f.line)
}

func (f *pullRequestFiles) selectedLines() int {
	if !f.selecting() {
		return 0
	}
	lo, hi := f.selection()
	return hi - lo + 1
}

func (f *pullRequestFiles) clearSelection() {
	if !f.selecting() {
		return
	}
	f.mark = -1
	f.rebuild()
}

func (f *pullRequestFiles) startSelection() {
	d, loaded := f.current()
	if !loaded || !f.focus.none() || f.line >= len(d.lines) || d.lines[f.line].Kind == diff.Hunk {
		return
	}
	f.mark = f.line
	f.rebuild()
}

func (f *pullRequestFiles) hunkBounds(d fileDiff, index int) (int, int) {
	lo, hi := index, index
	for lo > 0 && d.lines[lo-1].Kind != diff.Hunk {
		lo--
	}
	for hi < len(d.lines)-1 && d.lines[hi+1].Kind != diff.Hunk {
		hi++
	}
	return lo, hi
}

func (f *pullRequestFiles) wanted() (models.PullRequestFile, bool) {
	file, ok := f.file()
	if !ok || f.requested[file.Path] {
		return models.PullRequestFile{}, false
	}
	f.requested[file.Path] = true
	return file, true
}

func (f *pullRequestFiles) setChanges(changes models.PullRequestChanges) {
	f.changes, f.loaded = changes, true
	f.attributes = diff.ParseAttributes(changes.Attributes).WithPatterns(f.limits.Collapse)
	f.cursor = min(f.cursor, max(len(changes.Files)-1, 0))
	f.applyJump()
	f.renderTree()
	f.rebuild()
}

func (f *pullRequestFiles) setThreads(threads []models.PullRequestThread) {
	f.threads = threads
	f.renderTree()
	f.rebuild()
}

func (f *pullRequestFiles) setFile(filePath, oldText, newText string, err error) {
	d := fileDiff{}
	switch {
	case errors.Is(err, models.ErrFileTooLarge):
		d.skip = skipTooBig
	case err != nil:
		d.err = err.Error()
	case strings.ContainsRune(oldText, 0) || strings.ContainsRune(newText, 0):
		d.err = binaryFile
	default:
		d.oldText, d.newText, d.pins = oldText, newText, "unset"
		d.changed = diff.Estimate(oldText, newText)
		d.length = max(diff.LineCount(oldText), diff.LineCount(newText))
		d.skip = f.skipFor(filePath, d)
	}
	f.diffs[filePath] = d
	f.renderTree()
	if file, ok := f.file(); ok && file.Path == filePath {
		f.pinned(filePath, f.fileThreads(filePath))
		f.mark, f.focus = -1, commentFocus{}
		f.line = f.firstChange()
		f.applyJump()
		f.rebuild()
	}
}

func (f *pullRequestFiles) skipFor(filePath string, d fileDiff) diffSkip {
	switch {
	case d.length > f.limits.MaxFileLines:
		return skipLong
	case d.changed > f.limits.MaxChangedLines:
		return skipLarge
	}
	if skip, ok := skipFor[f.attributes.Reason(filePath)]; ok {
		return skip
	}
	return notSkipped
}

func (f *pullRequestFiles) tooLarge(filePath string) bool {
	d, ok := f.diffs[filePath]
	return ok && d.collapsed() && (d.skip == skipTooBig || d.skip == skipLong || d.skip == skipLarge)
}

func (f *pullRequestFiles) loadDiff() {
	file, ok := f.file()
	d, loaded := f.current()
	if !ok || !loaded || !d.loadable() {
		return
	}
	d.forced, d.pins = true, "unset"
	f.diffs[file.Path] = d
	f.pinned(file.Path, f.fileThreads(file.Path))
	f.focus = commentFocus{}
	f.line = f.firstChange()
	f.rebuild()
}

func (f *pullRequestFiles) notice(d fileDiff) []string {
	if !d.collapsed() {
		return nil
	}
	k := f.keys
	browse := widgets.Relabel(k.OpenBrowser, "open in browser")
	title, detail := "", ""
	switch d.skip {
	case skipTooBig:
		title, detail = tooLargeTitle, fmt.Sprintf("Over %d KB; the limit is diff.max_file_kb in config.yaml.", f.limits.MaxFileKB)
	case skipLong:
		title, detail = tooLargeTitle, fmt.Sprintf("%d lines; the limit is %d (diff.max_file_lines in config.yaml).", d.length, f.limits.MaxFileLines)
	case skipLarge:
		title, detail = tooLargeTitle, fmt.Sprintf("About %d changed lines; the limit is %d (diff.max_changed_lines in config.yaml).", d.changed, f.limits.MaxChangedLines)
	case skipGenerated:
		title, detail = "Generated file not rendered", "Marked linguist-generated in .gitattributes."
	case skipNoDiff:
		title, detail = "Diff turned off for this file", "Marked -diff or binary in .gitattributes."
	case skipConfigured:
		title, detail = "Collapsed by your settings", "Matches a diff.collapse pattern in config.yaml."
	}
	hints := []key.Binding{browse}
	if d.loadable() {
		hints = []key.Binding{widgets.Relabel(k.Select, "load diff"), browse}
	}
	width := f.diffWidth()
	heading := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(title)
	return []string{
		boxEdge(topCorners, heading, "", width, false),
		boxRow(Muted(detail), width, false),
		boxEdge(bottomCorners, widgets.Hints(max(width-boxChrome-2, 1), hints...), "", width, false),
	}
}

func (f *pullRequestFiles) setSize(width, height int) {
	f.width, f.height = width, height
	f.tree.SetWidth(f.treeWidth())
	f.tree.SetHeight(height)
	f.diff.SetWidth(f.diffWidth())
	f.diff.SetHeight(max(height-diffHeaderHeight, 0))
	f.renderTree()
	f.rebuild()
}

func (f *pullRequestFiles) split() bool {
	return f.width >= styles.PullRequestTreeShow
}

func (f *pullRequestFiles) treeWidth() int {
	if f.split() {
		return styles.PullRequestTreeWidth
	}
	return f.width
}

func (f *pullRequestFiles) diffWidth() int {
	if f.split() {
		return max(f.width-styles.PullRequestTreeWidth-styles.DetailColumnGap, 1)
	}
	return f.width
}

func (f *pullRequestFiles) open(filePath string, line int, left bool) {
	f.jump = &lineJump{path: filePath, line: line, left: left}
	f.treeFocus = false
	f.mark, f.focus = -1, commentFocus{}
	f.applyJump()
	f.renderTree()
	f.rebuild()
}

func (f *pullRequestFiles) applyJump() {
	if f.jump == nil {
		return
	}
	for i, file := range f.changes.Files {
		if samePath(file.Path, f.jump.path) {
			f.cursor = i
		}
	}
	file, ok := f.file()
	d, loaded := f.current()
	if !ok || !samePath(file.Path, f.jump.path) || !loaded {
		return
	}
	for i, l := range d.lines {
		if l.Kind != diff.Hunk && ((f.jump.left && l.Old == f.jump.line) || (!f.jump.left && l.New == f.jump.line)) {
			f.line = i
		}
	}
	f.jump = nil
}

func (f *pullRequestFiles) firstChange() int {
	d, _ := f.current()
	for i, l := range d.lines {
		if l.Kind == diff.Add || l.Kind == diff.Delete {
			return i
		}
	}
	return 0
}

func (f *pullRequestFiles) fileThreads(filePath string) []models.PullRequestThread {
	var out []models.PullRequestThread
	for _, t := range f.threads {
		if samePath(t.FilePath, filePath) && !t.System() {
			out = append(out, t)
		}
	}
	return out
}

func sideLine(t models.PullRequestThread, l diff.Line) (int, bool) {
	switch {
	case l.Kind == diff.Hunk || t.Line == 0:
		return 0, false
	case t.LeftSide:
		return l.Old, l.Kind != diff.Add
	}
	return l.New, l.Kind != diff.Delete
}

func samePath(a, b string) bool {
	return a != "" && strings.EqualFold(strings.TrimPrefix(a, "/"), strings.TrimPrefix(b, "/"))
}

func anchoredAt(t models.PullRequestThread, l diff.Line) bool {
	n, ok := sideLine(t, l)
	return ok && n == t.LastLine()
}

func coveredBy(t models.PullRequestThread, l diff.Line) bool {
	n, ok := sideLine(t, l)
	return ok && t.MultiLine() && t.Active() && n >= t.Line && n <= t.EndLine
}

func shownIn(d fileDiff, t models.PullRequestThread) bool {
	for _, l := range d.lines {
		if anchoredAt(t, l) {
			return true
		}
	}
	return false
}

func (f *pullRequestFiles) thread(id int) (models.PullRequestThread, bool) {
	for _, t := range f.threads {
		if t.ID == id {
			return t, true
		}
	}
	return models.PullRequestThread{}, false
}

func (f *pullRequestFiles) focused() (models.PullRequestThread, models.PullRequestComment, bool) {
	if f.treeFocus || f.focus.none() {
		return models.PullRequestThread{}, models.PullRequestComment{}, false
	}
	t, ok := f.thread(f.focus.thread)
	if !ok {
		return t, models.PullRequestComment{}, false
	}
	c, ok := t.Comment(f.focus.comment)
	return t, c, ok
}

func (f *pullRequestFiles) validateFocus() {
	if f.focus.none() {
		return
	}
	t, ok := f.thread(f.focus.thread)
	if !ok || t.System() {
		f.focus = commentFocus{}
		return
	}
	if _, found := t.Comment(f.focus.comment); !found || t.Collapsed(f.expanded) {
		f.focus.comment = t.Comments[0].ID
	}
}

func (f *pullRequestFiles) lineStops(d fileDiff, threads []models.PullRequestThread, index int) []commentFocus {
	if index < 0 || index >= len(d.lines) {
		return nil
	}
	var stops []commentFocus
	for _, t := range threads {
		if anchoredAt(t, d.lines[index]) {
			stops = append(stops, threadStops(t, f.expanded)...)
		}
	}
	return stops
}

func (f *pullRequestFiles) orphanStops(d fileDiff, threads []models.PullRequestThread) []commentFocus {
	var stops []commentFocus
	for _, t := range threads {
		if !shownIn(d, t) {
			stops = append(stops, threadStops(t, f.expanded)...)
		}
	}
	return stops
}

func (f *pullRequestFiles) step(delta int) {
	file, ok := f.file()
	d, loaded := f.current()
	if !ok || !loaded || f.selecting() {
		f.moveLine(f.line + delta)
		return
	}
	threads := f.fileThreads(file.Path)
	orphans := f.orphanStops(d, threads)
	if f.focus.none() {
		f.stepFromCode(d, threads, orphans, delta)
		return
	}
	slot, orphan := f.lineStops(d, threads, f.line), false
	if !slices.Contains(slot, f.focus) {
		slot, orphan = orphans, true
	}
	next := slices.Index(slot, f.focus) + delta
	switch {
	case next >= 0 && next < len(slot):
		f.setFocus(f.line, slot[next])
	case delta < 0 && !orphan:
		f.setFocus(f.line, commentFocus{})
	case delta > 0 && orphan && len(d.lines) > 0:
		f.setFocus(0, commentFocus{})
	case delta > 0 && !orphan && f.line+1 < len(d.lines):
		f.setFocus(f.line+1, commentFocus{})
	}
}

func (f *pullRequestFiles) stepFromCode(d fileDiff, threads []models.PullRequestThread, orphans []commentFocus, delta int) {
	switch {
	case len(d.lines) == 0:
		if delta > 0 && len(orphans) > 0 {
			f.setFocus(0, orphans[0])
		}
	case delta > 0:
		if stops := f.lineStops(d, threads, f.line); len(stops) > 0 {
			f.setFocus(f.line, stops[0])
			return
		}
		f.moveLine(f.line + 1)
	case f.line == 0:
		if len(orphans) > 0 {
			f.setFocus(0, orphans[len(orphans)-1])
		}
	default:
		if stops := f.lineStops(d, threads, f.line-1); len(stops) > 0 {
			f.setFocus(f.line-1, stops[len(stops)-1])
			return
		}
		f.moveLine(f.line - 1)
	}
}

func (f *pullRequestFiles) setFocus(line int, focus commentFocus) {
	file, _ := f.file()
	d, _ := f.current()
	old := f.line
	f.line, f.focus = line, focus
	if len(d.lines) == 0 || len(f.rows) != len(d.lines) {
		f.rebuild()
		return
	}
	threads := f.fileThreads(file.Path)
	for _, i := range []int{0, old, line} {
		f.rows[i] = f.block(d, threads, i)
	}
	f.commit()
}

func (f *pullRequestFiles) threadStyle(t models.PullRequestThread, width int, anchored bool) threadStyle {
	st := threadStyle{width: width, anchored: anchored, names: f.names, me: f.me, expanded: f.expanded}
	if f.treeFocus || f.focus.thread != t.ID {
		return st
	}
	st.selected, st.comment = true, f.focus.comment
	if c, ok := t.Comment(f.focus.comment); ok {
		st.hints = commentHints(f.keys, f.me, t, c, f.expanded, width)
	}
	return st
}

func (f *pullRequestFiles) noteFocus(r threadRender, before []string) {
	if !r.focused() {
		return
	}
	offset := 0
	for _, p := range before {
		offset += strings.Count(p, "\n") + 1
	}
	f.focusRel = [2]int{offset + r.top, offset + r.bottom}
}

func (f *pullRequestFiles) anchor() (models.ThreadAnchor, bool) {
	file, ok := f.file()
	d, loaded := f.current()
	if !ok || !loaded || f.treeFocus || !f.focus.none() || f.line >= len(d.lines) {
		return models.ThreadAnchor{}, false
	}
	lo, hi := f.selection()
	var kept []diff.Line
	deleted := true
	for _, l := range d.lines[lo : hi+1] {
		if l.Kind != diff.Hunk {
			deleted = deleted && l.Kind == diff.Delete
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		return models.ThreadAnchor{}, false
	}
	anchor := models.ThreadAnchor{FilePath: file.Path, LeftSide: deleted}
	if deleted && file.OriginalPath != "" {
		anchor.FilePath = file.OriginalPath
	}
	for _, l := range kept {
		n := l.New
		if deleted {
			n = l.Old
		}
		if n == 0 {
			continue
		}
		if anchor.Line == 0 {
			anchor.Line = n
		}
		anchor.EndLine, anchor.EndOffset = n, len([]rune(l.Text))+1
	}
	if anchor.EndLine == anchor.Line {
		anchor.EndLine, anchor.EndOffset = 0, 0
	}
	return anchor, anchor.Line > 0
}

func (f *pullRequestFiles) shareTarget(pr models.PullRequest) share.Target {
	file, ok := f.file()
	if !ok {
		return share.PullRequest(pr)
	}
	if anchor, ok := f.anchor(); ok {
		commit := f.changes.HeadCommit
		if anchor.LeftSide {
			commit = f.changes.BaseCommit
		}
		lines := share.Lines{From: anchor.Line, To: max(anchor.EndLine, anchor.Line), Code: f.selectedCode(anchor.LeftSide)}
		return share.PullRequestLines(pr, commit, anchor.FilePath, lines)
	}
	if file.Kind == models.ChangeDelete {
		return share.PullRequestFile(pr, f.changes.BaseCommit, file.Path)
	}
	return share.PullRequestFile(pr, f.changes.HeadCommit, file.Path)
}

func (f *pullRequestFiles) selectedCode(leftSide bool) []string {
	d, _ := f.current()
	lo, hi := f.selection()
	var code []string
	for _, l := range d.lines[lo : hi+1] {
		n := l.New
		if leftSide {
			n = l.Old
		}
		if l.Kind != diff.Hunk && n > 0 {
			code = append(code, l.Text)
		}
	}
	return code
}

func lineNumber(n int) string {
	if n == 0 {
		return strings.Repeat(" ", styles.DiffGutterWidth)
	}
	return fmt.Sprintf("%*d", styles.DiffGutterWidth, n)
}

func (f *pullRequestFiles) codeRow(l diff.Line, code components.CodeLine, selected, covered bool) string {
	width := f.diffWidth()
	base := lipgloss.NewStyle()
	sign, signColor := " ", styles.ColorTextMuted
	switch l.Kind {
	case diff.Add:
		base, sign, signColor = base.Background(styles.ColorDiffAdd), "+", styles.ColorSuccess
	case diff.Delete:
		base, sign, signColor = base.Background(styles.ColorDiffDelete), "-", styles.ColorError
	case diff.Hunk:
		base = base.Background(styles.ColorDiffHunk)
	}
	bar := " "
	if selected {
		base = base.Background(styles.ColorHighlight)
		if !f.treeFocus {
			bar = railBar
		}
	}
	lead := base.Foreground(styles.ColorPrimary).Render(bar)
	if covered && bar == " " {
		lead = base.Foreground(styles.ColorOrange).Render(rangeBar)
	}
	if l.Kind == diff.Hunk {
		text := ansi.Truncate(l.Text, max(width-2, 1), "…")
		return lead + base.Foreground(styles.ColorAccent).Width(max(width-1, 1)).Render(" "+text)
	}
	gutter := base.Foreground(styles.ColorTextMuted).Render(lineNumber(l.Old) + " " + lineNumber(l.New) + " ")
	mark := base.Foreground(signColor).Bold(true).Render(sign + " ")
	room := max(width-1-2*styles.DiffGutterWidth-4, 1)
	return lead + gutter + mark + code.Render(base.Foreground(styles.ColorText), room)
}

func (f *pullRequestFiles) threadIndent() int {
	return min(2*styles.DiffGutterWidth+4, f.diffWidth()/4)
}

func (f *pullRequestFiles) block(d fileDiff, threads []models.PullRequestThread, index int) string {
	l := d.lines[index]
	lo, hi := f.selection()
	covered := false
	for _, t := range threads {
		covered = covered || coveredBy(t, l)
	}
	if index == f.line {
		f.focusRel = [2]int{-1, -1}
	}
	var parts []string
	if index == 0 {
		parts = f.orphans(d, threads)
	}
	parts = append(parts, f.codeRow(l, d.code(l), f.focus.none() && index >= lo && index <= hi, covered))
	indent := f.threadIndent()
	for _, t := range threads {
		if !anchoredAt(t, l) {
			continue
		}
		r := renderThread(t, f.threadStyle(t, f.diffWidth()-indent, false))
		f.noteFocus(r, parts)
		parts = append(parts, lipgloss.NewStyle().PaddingLeft(indent).Render(r.text))
	}
	return strings.Join(parts, "\n")
}

func (f *pullRequestFiles) orphans(d fileDiff, threads []models.PullRequestThread) []string {
	var parts []string
	for _, t := range threads {
		if shownIn(d, t) {
			continue
		}
		if len(parts) == 0 {
			parts = append(parts, Muted(outsideChangesNote))
		}
		r := renderThread(t, f.threadStyle(t, f.diffWidth(), true))
		f.noteFocus(r, parts)
		parts = append(parts, r.text)
	}
	if len(parts) > 0 {
		parts = append(parts, "")
	}
	return parts
}

func (f *pullRequestFiles) rebuild() {
	f.rows, f.offsets = nil, nil
	file, ok := f.file()
	threads := f.fileThreads(file.Path)
	d, loaded := f.pinned(file.Path, threads)
	if !ok || !loaded || f.width <= 0 {
		f.diff.SetContent("")
		return
	}
	d = f.highlighted(file.Path, d)
	f.line = min(max(f.line, 0), max(len(d.lines)-1, 0))
	f.validateFocus()
	f.rows = make([]string, len(d.lines))
	for i := range d.lines {
		f.rows[i] = f.block(d, threads, i)
	}
	if len(d.lines) == 0 || d.collapsed() {
		f.focusRel = [2]int{-1, -1}
		notice := f.notice(d)
		parts := append(notice, f.orphans(fileDiff{}, threads)...)
		if f.focusRel[0] >= 0 {
			f.focusRel = [2]int{f.focusRel[0] + len(notice), f.focusRel[1] + len(notice)}
		}
		f.rows = nil
		if len(parts) > 0 {
			f.rows = []string{strings.Join(parts, "\n")}
		}
	}
	f.commit()
}

func (f *pullRequestFiles) commit() {
	f.offsets = make([]int, len(f.rows))
	line := 0
	for i, r := range f.rows {
		f.offsets[i] = line
		line += strings.Count(r, "\n") + 1
	}
	f.diff.SetContent(strings.Join(f.rows, "\n"))
	if f.line < len(f.offsets) && !f.focus.none() && f.focusRel[0] >= 0 {
		widgets.Reveal(&f.diff, f.offsets[f.line]+f.focusRel[0], f.offsets[f.line]+f.focusRel[1])
		return
	}
	if f.line < len(f.offsets) {
		top := f.offsets[f.line]
		bottom := top + strings.Count(f.rows[f.line], "\n") + 1
		if next := f.line + 1; next < len(f.rows) {
			if end := f.offsets[next] + strings.Count(f.rows[next], "\n") + 1; end-top <= f.diff.Height() {
				bottom = end
			}
		}
		widgets.Reveal(&f.diff, top, bottom)
	}
}

func (f *pullRequestFiles) moveLine(to int) {
	file, ok := f.file()
	d, loaded := f.current()
	if !ok || !loaded || len(d.lines) == 0 {
		return
	}
	to = min(max(to, 0), len(d.lines)-1)
	if f.selecting() {
		lo, hi := f.hunkBounds(d, f.mark)
		to = min(max(to, lo), hi)
	}
	focused := !f.focus.none()
	f.focus = commentFocus{}
	if to == f.line && !focused {
		return
	}
	first, last := min(f.line, to), max(f.line, to)
	f.line = to
	threads := f.fileThreads(file.Path)
	for i := first; i <= last; i++ {
		f.rows[i] = f.block(d, threads, i)
	}
	if focused && first > 0 {
		f.rows[0] = f.block(d, threads, 0)
	}
	f.commit()
}

func (f *pullRequestFiles) jumpChange(delta int) {
	d, _ := f.current()
	for i := f.line + delta; i >= 0 && i < len(d.lines); i += delta {
		if d.lines[i].Kind == diff.Hunk {
			f.moveLine(min(i+1, len(d.lines)-1))
			return
		}
	}
}

func (f *pullRequestFiles) moveFile(delta int) {
	next := min(max(f.cursor+delta, 0), len(f.changes.Files)-1)
	if next == f.cursor || next < 0 {
		return
	}
	f.cursor = next
	f.mark, f.focus = -1, commentFocus{}
	f.line = f.firstChange()
	f.diff.GotoTop()
	f.renderTree()
	f.rebuild()
}

func (f *pullRequestFiles) focusTree(focus bool) {
	if f.treeFocus == focus {
		return
	}
	f.treeFocus = focus
	f.mark, f.focus = -1, commentFocus{}
	f.renderTree()
	f.rebuild()
}

func (f pullRequestFiles) update(msg tea.Msg) (pullRequestFiles, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		f.diff, cmd = f.diff.Update(msg)
		return f, cmd
	case tea.KeyPressMsg:
		f.handleKey(msg)
	}
	return f, nil
}

func (f *pullRequestFiles) handleKey(msg tea.KeyPressMsg) {
	k := f.keys
	switch {
	case key.Matches(msg, k.NextPanel), key.Matches(msg, k.PrevPanel):
		f.focusTree(!f.treeFocus)
		return
	case key.Matches(msg, k.NextFile):
		f.moveFile(1)
		return
	case key.Matches(msg, k.PrevFile):
		f.moveFile(-1)
		return
	}
	if f.treeFocus {
		switch {
		case key.Matches(msg, k.Down):
			f.moveFile(1)
		case key.Matches(msg, k.Up):
			f.moveFile(-1)
		case key.Matches(msg, k.Top):
			f.moveFile(-len(f.changes.Files))
		case key.Matches(msg, k.Bottom):
			f.moveFile(len(f.changes.Files))
		case key.Matches(msg, k.Select), key.Matches(msg, k.Right):
			f.focusTree(false)
		}
		return
	}
	d, _ := f.current()
	page := max(f.diff.Height()/2, 1)
	switch {
	case key.Matches(msg, k.Down):
		f.step(1)
	case key.Matches(msg, k.Up):
		f.step(-1)
	case key.Matches(msg, k.HalfPageDown):
		f.moveLine(f.line + page)
	case key.Matches(msg, k.HalfPageUp):
		f.moveLine(f.line - page)
	case key.Matches(msg, k.Top):
		f.moveLine(0)
	case key.Matches(msg, k.Bottom):
		f.moveLine(len(d.lines) - 1)
	case key.Matches(msg, k.SelectLines):
		if f.selecting() {
			f.clearSelection()
		} else {
			f.startSelection()
		}
	case key.Matches(msg, k.NextChange):
		f.jumpChange(1)
	case key.Matches(msg, k.PrevChange):
		f.jumpChange(-1)
	case key.Matches(msg, k.Left):
		f.focusTree(true)
	case key.Matches(msg, k.Select):
		f.loadDiff()
	}
}

func (f *pullRequestFiles) keyBindings() []key.Binding {
	k := f.keys
	files := widgets.Combine("file", k.PrevFile, k.NextFile)
	focus := widgets.Relabel(k.NextPanel, "files/diff")
	if f.treeFocus {
		return []key.Binding{widgets.Combine("file", k.Up, k.Down), widgets.Relabel(k.Select, "view diff"), focus}
	}
	if n := f.selectedLines(); n > 0 {
		return []key.Binding{
			widgets.Combine("extend", k.Up, k.Down),
			widgets.Relabel(k.Share, fmt.Sprintf("share %d lines", n)),
			widgets.Relabel(k.AddComment, fmt.Sprintf("comment on %d lines", n)),
			widgets.Relabel(k.Back, "cancel selection"),
		}
	}
	bindings := []key.Binding{k.Share, widgets.Relabel(k.AddComment, "comment on line"), k.SelectLines}
	if d, loaded := f.current(); loaded && d.collapsed() {
		bindings = []key.Binding{widgets.Relabel(k.OpenBrowser, "open in browser")}
		if d.loadable() {
			bindings = append([]key.Binding{widgets.Relabel(k.Select, "load diff")}, bindings...)
		}
	}
	if t, c, ok := f.focused(); ok {
		bindings = commentActions(k, f.me, t, c, f.expanded)
	}
	return append(bindings, widgets.Combine("line/comment", k.Up, k.Down), files, widgets.Combine("change", k.NextChange, k.PrevChange), focus)
}

func changeColor(kind models.ChangeKind) lipgloss.Style {
	st := lipgloss.NewStyle().Bold(true)
	switch kind {
	case models.ChangeAdd:
		return st.Foreground(styles.ColorSuccess)
	case models.ChangeDelete:
		return st.Foreground(styles.ColorError)
	case models.ChangeRename:
		return st.Foreground(styles.ColorPrimary)
	}
	return st.Foreground(styles.ColorWarning)
}

func (f *pullRequestFiles) treeRow(file models.PullRequestFile, selected bool) string {
	width := f.treeWidth()
	base := lipgloss.NewStyle()
	bar := " "
	if selected {
		base = base.Background(styles.ColorHighlight)
		if f.treeFocus || !f.split() {
			bar = railBar
		}
	}
	note, noteColor := "", styles.ColorTextMuted
	threads := f.fileThreads(file.Path)
	switch active := models.ActiveThreads(threads); {
	case active > 0:
		note, noteColor = fmt.Sprintf("%d active", active), styles.ColorOrange
	case len(threads) == 1:
		note = "1 comment"
	case len(threads) > 1:
		note = fmt.Sprintf("%d comments", len(threads))
	case f.tooLarge(file.Path):
		note = tooLargeNote
	default:
		note = fileNotes[f.attributes.Reason(file.Path)]
	}
	room := max(width-6-lipgloss.Width(note), 1)
	name := ansi.Truncate(path.Base(file.Path), room, "…")
	return base.Foreground(styles.ColorPrimary).Render(bar) + base.Render("  ") +
		changeColor(file.Kind).Inherit(base).Render(file.Kind.Letter()) + base.Render(" ") +
		base.Foreground(styles.ColorText).Bold(selected).Render(name) +
		base.Render(strings.Repeat(" ", max(room-lipgloss.Width(name)+1, 1))) +
		base.Foreground(noteColor).Render(note)
}

func (f *pullRequestFiles) renderTree() {
	if f.width <= 0 {
		return
	}
	var lines []string
	dir, top := "", 0
	for i, file := range f.changes.Files {
		if d := path.Dir(file.Path); d != dir || i == 0 {
			dir = d
			if i > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, Muted(ansi.Truncate(d, max(f.treeWidth()-1, 1), "…")))
		}
		if i == f.cursor {
			top = len(lines)
		}
		lines = append(lines, f.treeRow(file, i == f.cursor))
	}
	f.tree.SetContent(strings.Join(lines, "\n"))
	widgets.Reveal(&f.tree, top, top+1)
}

func (f *pullRequestFiles) diffPane() string {
	width, height := f.diffWidth(), f.height
	file, ok := f.file()
	if !ok {
		return Block("", width, height)
	}
	d, loaded := f.current()
	name := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(file.Kind.Letter() + " " + strings.TrimPrefix(file.Path, "/"))
	stats := Muted(fmt.Sprintf("file %d of %d", f.cursor+1, len(f.changes.Files)))
	if loaded && d.collapsed() {
		stats = Muted(notRenderedNote+"  ") + stats
	}
	if loaded && d.err == "" && !d.collapsed() {
		stats = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(fmt.Sprintf("+%d", d.added)) + " " +
			lipgloss.NewStyle().Foreground(styles.ColorError).Render(fmt.Sprintf("-%d", d.deleted)) + "  " + stats
	}
	name = ansi.Truncate(name, max(width-lipgloss.Width(stats)-1, 1), "…")
	gap := strings.Repeat(" ", max(width-lipgloss.Width(name)-lipgloss.Width(stats), 1))
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat(ruleGlyph, width))
	body := f.diff.View()
	switch {
	case !loaded:
		body = Loading(f.spinner, loadingDiff+path.Base(file.Path))
	case d.err != "":
		body = Muted(d.err)
	case len(d.lines) == 0 && len(f.rows) == 0:
		body = Muted(noTextChanges)
	}
	return Block(name+gap+stats+"\n"+rule+"\n"+body, width, height)
}

func (f *pullRequestFiles) view() string {
	switch {
	case f.width <= 0 || f.height <= 0:
		return ""
	case !f.loaded:
		return lipgloss.Place(f.width, f.height, lipgloss.Center, lipgloss.Center, Loading(f.spinner, loadingFiles))
	case len(f.changes.Files) == 0:
		return Block(Muted(noFiles), f.width, f.height)
	case !f.split() && f.treeFocus:
		return Block(f.tree.View(), f.width, f.height)
	case !f.split():
		return f.diffPane()
	}
	tree := Block(f.tree.View(), f.treeWidth(), f.height)
	return lipgloss.JoinHorizontal(lipgloss.Top, tree, strings.Repeat(" ", styles.DetailColumnGap), f.diffPane())
}
