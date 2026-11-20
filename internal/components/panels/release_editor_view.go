package panels

import (
	"fmt"
	"image/color"
	"path"
	"regexp"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	editorTasksMinWidth = 30
	editorTasksMaxWidth = 48
	editorTasksRatio    = 35
	editorLabelMinWidth = 14
	editorLabelMaxWidth = 34
	editorLabelRatio    = 36
	editorStageBoxMin   = 22
	editorStageBoxMax   = 30
	editorSettableWidth = 11
	editorGutterWidth   = 2
	editorGlyphWidth    = 2
	editorEmptyValue    = "empty"
	editorSecretDots    = "●●●●●●●●"
	editorAzureTag      = "from Azure"
)

const (
	glyphChanged  = "▎"
	glyphCursor   = "▌"
	glyphNeeds    = "!"
	glyphOn       = "●"
	glyphOff      = "○"
	glyphPick     = "▾"
	glyphText     = "›"
	glyphTreeMid  = "├"
	glyphTreeEnd  = "└"
	glyphTreeBar  = "│"
	glyphLocked   = "⊘"
	glyphHidden   = "◌"
	glyphScope    = "■"
	glyphWas      = "−"
	glyphUnsaved  = "●"
	glyphJobGroup = "▾"
)

var (
	editorMuted   = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	editorText    = lipgloss.NewStyle().Foreground(styles.ColorText)
	editorBold    = lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	editorChanged = lipgloss.NewStyle().Foreground(styles.ColorOrange).Bold(true)
	editorNeeds   = lipgloss.NewStyle().Foreground(styles.ColorError).Bold(true)
	editorAccent  = lipgloss.NewStyle().Foreground(styles.ColorAccent)
	variableRef   = regexp.MustCompile(`\$\([^)]*\)`)
	stageScopes   = []color.Color{styles.ColorViolet, styles.ColorSecondary, styles.ColorPink, styles.ColorTypeTask}
)

func (v *ReleaseEditor) refresh() {
	if v.width <= 0 {
		return
	}
	width := max(v.width-2, 1)
	switch {
	case v.doc == nil:
		v.body.SetContent(Loading(v.spinner, loadingEditor))
	case v.tab == EditorPipelineTab:
		v.body.SetContent(v.pipelineContent(width))
	case v.tab == EditorHistoryTab:
		v.body.SetContent(v.historyContent(width))
	}
}

func paint(base lipgloss.Style, fg color.Color, text string) string {
	return base.Foreground(fg).Render(text)
}

func tag(text string, bg, fg color.Color) string {
	return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(true).Padding(0, 1).Render(text)
}

func groupColors(group string) (color.Color, color.Color) {
	switch group {
	case models.FieldGroupTask, models.FieldGroupJob:
		return styles.ColorDarkBlue, styles.ColorPrimaryLight
	case models.FieldGroupControl, models.FieldGroupArtifacts:
		return styles.ColorTealTint, styles.ColorTealLight
	case models.FieldGroupOptions:
		return styles.ColorChip, styles.ColorLight
	}
	return styles.ColorVioletTint, styles.ColorVioletLight
}

func sectionTag(title string, bg, fg color.Color, note string, width int) string {
	head := tag(strings.ToUpper(title), bg, fg)
	if note != "" {
		head += editorMuted.Render(" " + note)
	}
	fill := max(width-lipgloss.Width(head)-1, 0)
	return head + " " + lipgloss.NewStyle().Foreground(styles.ColorChip).Render(strings.Repeat(ruleGlyph, fill))
}

func (v *ReleaseEditor) tabLabels() string {
	k := v.keys
	names := []string{k.Panel1.Help().Key + " Pipeline", k.Panel2.Help().Key + " Tasks", k.Panel3.Help().Key + " Variables", k.Panel4.Help().Key + " History"}
	taskEdits := v.taskEditCount()
	counts := []int{0, taskEdits, len(v.edits) - taskEdits, 0}
	stale := []int{0, len(v.stale), 0, 0}
	var out []string
	for i, name := range names {
		st := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if EditorTab(i) == v.tab {
			st = st.Foreground(styles.ColorPrimary).Bold(true).Underline(true)
		}
		label := " " + st.Render(name)
		if counts[i] > 0 {
			label += editorChanged.Render(fmt.Sprintf(" %d", counts[i]))
		}
		if stale[i] > 0 {
			label += editorNeeds.Render(fmt.Sprintf(" %s%d", glyphNeeds, stale[i]))
		}
		out = append(out, label+" ")
	}
	return strings.Join(out, " ")
}

func (v *ReleaseEditor) header() string {
	inner := max(v.width-2, 1)
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat(ruleGlyph, v.width))
	if v.doc == nil {
		return strings.Join([]string{" " + Loading(v.spinner, loadingEditor), "", "", rule}, "\n")
	}
	d := v.doc
	badge := Badge("PIPELINE", styles.ColorChip, styles.ColorLight)
	sub := d.Folder
	if d.Kind == models.DocRelease {
		badge, sub = Badge("RELEASE", styles.ColorChip, styles.ColorLight), d.Definition
	}
	if v.editing {
		badge = Badge("EDITING", styles.ColorOrange, styles.ColorOnAccent)
	}
	title := badge + " " + lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Render(d.Title())
	if sub != "" {
		title += editorMuted.Render(" · " + sub)
	}
	var meta []string
	if v.Dirty() {
		meta = append(meta, editorChanged.Render(glyphUnsaved+" "+fmt.Sprint(len(v.edits))+" unsaved"))
	}
	if len(v.stale) > 0 {
		meta = append(meta, editorNeeds.Render(glyphNeeds+" "+fmt.Sprint(len(v.stale))+" need a value from Azure"))
	}
	if d.Kind == models.DocRelease {
		meta = append(meta, editorMuted.Render(fmt.Sprintf("from revision %d of the pipeline", d.Revision)))
		if v.editing {
			meta = append(meta, editorMuted.Render("saving changes this release only"))
		}
	} else {
		meta = append(meta, editorMuted.Render(fmt.Sprintf("Revision %d", d.Revision)))
		if v.editing {
			meta = append(meta, editorMuted.Render(fmt.Sprintf("saving creates revision %d", d.Revision+1)))
		} else if d.ModifiedBy != "" {
			meta = append(meta, editorMuted.Render("modified by "+d.ModifiedBy+" "+releaseStamp(d.Modified)))
		}
	}
	meta = append(meta, editorMuted.Render(plural1(len(d.Stages), "stage")+" · "+plural1(len(d.Artifacts), "artifact")))
	lines := []string{" " + Line(title, inner), " " + Line(strings.Join(meta, editorMuted.Render(" · ")), inner), Line(v.tabLabels(), v.width), rule}
	if d.Kind == models.DocRelease {
		banner := " " + glyphLocked + " Editing this release, not its pipeline. Stages that already deployed are read-only."
		lines = append(lines, lipgloss.NewStyle().Background(styles.ColorWarningTint).Foreground(styles.ColorWarning).
			Width(v.width).MaxWidth(v.width).Inline(true).Render(banner))
	}
	return strings.Join(lines, "\n")
}

func (v ReleaseEditor) View() string {
	if v.id == 0 || v.width <= 0 || v.height <= 0 {
		return ""
	}
	height := v.bodyHeight()
	var body string
	switch {
	case v.doc == nil || v.tab == EditorPipelineTab || v.tab == EditorHistoryTab:
		body = lipgloss.NewStyle().PaddingLeft(1).Render(Block(v.body.View(), max(v.width-2, 1), height))
	case v.tab == EditorTasksTab:
		body = v.tasksView(height)
	default:
		body = v.variablesView(height)
	}
	return Block(v.header()+"\n"+body, v.width, v.height)
}

func (v *ReleaseEditor) phaseWhere(p models.DocPhase) string {
	var parts []string
	if p.Kind != p.Name {
		parts = append(parts, p.Kind)
	}
	if q := v.refs.Queues[p.QueueID]; q.Name != "" {
		parts = append(parts, q.Name)
	}
	if p.Agent != "" {
		parts = append(parts, p.Agent)
	}
	return strings.Join(parts, " · ")
}

func (v *ReleaseEditor) stageStrip() string {
	hint := Muted(v.keys.PrevStage.Help().Key+" "+v.keys.NextStage.Help().Key+" switch stage") + " "
	var chips []string
	for i, s := range v.doc.Stages {
		bg, fg := styles.ColorChip, styles.ColorLight
		if i == v.stage {
			bg, fg = styles.ColorHighlight, styles.ColorPrimary
		}
		base := lipgloss.NewStyle().Background(bg)
		chip := paint(base.Bold(i == v.stage), fg, " "+s.Name)
		if s.Locked {
			chip = paint(base, styles.ColorTextMuted, " "+glyphLocked+" "+s.Name+" · deployed")
		}
		if n := v.edits.CountUnder(models.DocPath{"environments", s.Index}); n > 0 {
			chip += paint(base.Bold(true), styles.ColorOrange, fmt.Sprintf(" %s %d", glyphUnsaved, n))
		}
		if n := v.staleUnder(models.DocPath{"environments", s.Index}); n > 0 {
			chip += paint(base.Bold(true), styles.ColorError, fmt.Sprintf(" %s%d", glyphNeeds, n))
		}
		chips = append(chips, chip+base.Render(" "))
	}
	lead := " " + Muted("Stage") + "  "
	budget := max(v.width-lipgloss.Width(lead)-lipgloss.Width(hint)-1, 1)
	return spreadLine(lead+FitItems(chips, " ", budget), hint, v.width)
}

func (v *ReleaseEditor) paneWidths() (int, int) {
	left := min(max(v.width*editorTasksRatio/100, editorTasksMinWidth), editorTasksMaxWidth)
	left = min(left, max(v.width/2, 1))
	return left, max(v.width-left-1, 1)
}

func (v *ReleaseEditor) paneHeight() int { return max(v.bodyHeight()-1, 0) }

func (v *ReleaseEditor) tasksView(height int) string {
	strip := v.stageStrip()
	if len(v.doc.Stages) == 0 {
		return Block(strip+"\n "+Muted("This pipeline has no stages."), v.width, height)
	}
	lw, rw := v.paneWidths()
	ph := v.paneHeight()
	liw, lih := PaneContentSize(lw, ph)
	riw, rih := PaneContentSize(rw, ph)
	lines, _ := v.taskLines(liw)
	s, _ := v.currentStage()
	left := Pane(PaneTitle("", "Deployment process", v.pane == editorPaneTasks)+editorMuted.Render("  "+s.Summary()),
		window(lines, v.taskTop, lih), lw, ph, v.pane == editorPaneTasks)
	title, flines, _ := v.fieldLines(riw)
	right := Pane(title, window(flines, v.fieldTop, rih), rw, ph, v.pane == editorPaneFields)
	return Block(strip+"\n"+lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right), v.width, height)
}

func window(lines []string, top, height int) string {
	if height <= 0 {
		return ""
	}
	top = min(max(top, 0), max(len(lines)-height, 0))
	end := min(top+height, len(lines))
	return strings.Join(lines[top:end], "\n")
}

func revealTop(top, first, last, height int) int {
	if height <= 0 {
		return 0
	}
	if last >= top+height {
		top = last - height + 1
	}
	if first < top {
		top = first
	}
	return max(top, 0)
}

func (v *ReleaseEditor) reveal() {
	if v.doc == nil || v.width <= 0 {
		return
	}
	switch v.tab {
	case EditorTasksTab:
		lw, rw := v.paneWidths()
		liw, lih := PaneContentSize(lw, v.paneHeight())
		riw, rih := PaneContentSize(rw, v.paneHeight())
		if _, at := v.taskLines(liw); v.task < len(at) {
			v.taskTop = revealTop(v.taskTop, max(at[v.task]-1, 0), at[v.task]+1, lih)
		}
		if _, _, spans := v.fieldLines(riw); v.field < len(spans) {
			v.fieldTop = revealTop(v.fieldTop, spans[v.field][0], spans[v.field][1], rih)
		}
	case EditorVariablesTab:
		v.varTop = revealTop(v.varTop, v.variable, v.variable, v.variableRows())
	}
}

func rowStyle(selected, focused bool) lipgloss.Style {
	base := lipgloss.NewStyle()
	switch {
	case selected && focused:
		base = base.Background(styles.ColorHighlight)
	case selected:
		base = base.Background(styles.ColorHighlightDim)
	}
	return base
}

func fillRow(base lipgloss.Style, left, right string, width int) string {
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	row := left + base.Render(strings.Repeat(" ", gap)) + right
	if lipgloss.Width(row) > width {
		return ansi.Truncate(row, width, "…")
	}
	return row
}

func gutter(base lipgloss.Style, changed, needs, cursor bool) string {
	switch {
	case needs:
		return paint(base.Bold(true), styles.ColorError, glyphNeeds+" ")
	case changed:
		return paint(base, styles.ColorOrange, glyphChanged+" ")
	case cursor:
		return paint(base, styles.ColorPrimary, glyphCursor+" ")
	}
	return base.Render("  ")
}

func badgeOn(base lipgloss.Style, text string, bg, fg color.Color) string {
	return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(true).Render(" "+text+" ") + base.Render(" ")
}

func (v *ReleaseEditor) taskLines(width int) ([]string, []int) {
	s, _ := v.currentStage()
	var lines []string
	var at []int
	focused := v.pane == editorPaneTasks
	tree := lipgloss.NewStyle().Foreground(styles.ColorBorder)
	rows := v.taskRows()
	for ri, r := range rows {
		p := s.Phases[r.phase]
		selected := ri == v.task
		base := rowStyle(selected, focused)
		edits, stale := v.rowEdits(s, r), v.rowStale(s, r)
		var right string
		if edits > 0 {
			right += badgeOn(base, fmt.Sprint(edits), styles.ColorOrange, styles.ColorOnAccent)
		}
		if stale > 0 {
			right += badgeOn(base, glyphNeeds+fmt.Sprint(stale), styles.ColorError, styles.ColorOnAccent)
		}
		if r.task < 0 {
			if r.phase > 0 {
				lines = append(lines, "")
			}
			nameStyle := base.Foreground(styles.ColorText).Bold(true)
			if selected {
				nameStyle = nameStyle.Foreground(styles.ColorPrimary)
			}
			fields := v.doc.PhaseFields(s, p, v.refs.Queues, v.edits)
			name := v.fieldByKey(fields, "$name")
			left := gutter(base, edits > 0, stale > 0, selected && focused) + paint(base, styles.ColorPrimary, glyphJobGroup+" ") + nameStyle.Render(name)
			if agent := v.fieldByKey(fields, "$agent"); agent != "" {
				right += badgeOn(base, agent, styles.ColorDarkBlue, styles.ColorPrimaryLight)
			}
			at = append(at, len(lines))
			lines = append(lines, fillRow(base, left, right, width))
			detail := []string{v.queueName(v.fieldValue(fields, "$queue"))}
			if p.Kind != p.Name {
				detail = append([]string{p.Kind}, detail...)
			}
			detail = slices.DeleteFunc(detail, func(s string) bool { return s == "" })
			lines = append(lines, Line(tree.Render("  "+glyphTreeBar+" ")+editorMuted.Render(strings.Join(detail, " · ")), width))
			continue
		}
		t := p.Tasks[r.task]
		path := v.doc.TaskPath(s, p, t)
		enabled := models.Truthy(v.edits.Value(path.Child("enabled"), t.Enabled))
		branch := glyphTreeMid
		if r.task == len(p.Tasks)-1 {
			branch = glyphTreeEnd
		}
		nameStyle := base.Foreground(styles.ColorText)
		if selected {
			nameStyle = nameStyle.Bold(true).Foreground(styles.ColorPrimary)
		}
		dot := paint(base, styles.ColorSuccess, glyphOn+" ")
		if !enabled {
			nameStyle = nameStyle.Foreground(styles.ColorTextMuted).Strikethrough(true)
			dot = paint(base, styles.ColorTextMuted, glyphOff+" ")
			right = badgeOn(base, "OFF", styles.ColorChip, styles.ColorLight) + right
		}
		name := models.FormatDocValue(v.edits.Value(path.Child("name"), t.Name))
		left := gutter(base, edits > 0, stale > 0, selected && focused) + paint(base, styles.ColorBorder, branch+" ") + dot + nameStyle.Render(name)
		refColor := styles.ColorTextMuted
		if v.refs.Task(t) != nil {
			refColor = styles.ColorVioletLight
		}
		ref := paint(base, refColor, v.taskRef(t)+" ")
		if lipgloss.Width(left)+lipgloss.Width(right)+lipgloss.Width(ref)+1 <= width {
			right += ref
		}
		at = append(at, len(lines))
		lines = append(lines, fillRow(base, left, right, width))
	}
	if len(at) == 0 {
		lines = append(lines, Muted("No jobs in this stage."))
	}
	return lines, at
}

func (v *ReleaseEditor) fieldByKey(fields []models.TaskField, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return f.Text()
		}
	}
	return ""
}

func (v *ReleaseEditor) fieldValue(fields []models.TaskField, key string) any {
	for _, f := range fields {
		if f.Key == key {
			return f.Value
		}
	}
	return nil
}

func (v *ReleaseEditor) queueName(value any) string {
	id := models.QueueID(value)
	if q := v.refs.Queues[id]; q.Name != "" {
		return q.Name
	}
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("pool %d", id)
}

func (v *ReleaseEditor) taskRef(t models.DocTask) string {
	if t.TaskGroup() {
		return "task group"
	}
	if def := v.refs.Task(t); def != nil && def.Name != "" {
		return fmt.Sprintf("%s@%d", def.Name, t.Major())
	}
	return "v" + t.Version
}

func (v *ReleaseEditor) displayText(f models.TaskField) string {
	switch f.Kind {
	case models.InputEndpoint:
		if name := v.refs.Endpoints[strings.ToLower(f.Text())]; name != "" {
			return name
		}
	case models.InputQueue:
		return v.queueName(f.Value)
	case models.InputAzure:
		if name := v.refs.AzureNames[strings.ToLower(f.Text())]; name != "" {
			return name
		}
	}
	return f.Text()
}

func (v *ReleaseEditor) fieldText(f models.TaskField) (string, bool) {
	if f.Secret {
		return editorSecretDots, f.Changed()
	}
	text := v.displayText(f)
	if text == "" {
		return editorEmptyValue, false
	}
	if first, rest, multi := strings.Cut(text, "\n"); multi {
		return first + fmt.Sprintf("  (+%d lines)", strings.Count(rest, "\n")+1), true
	}
	return text, true
}

func (v *ReleaseEditor) beforeText(f models.TaskField) string {
	was := f
	was.Value = f.Original
	if f.Secret {
		return "secret"
	}
	text, filled := v.fieldText(was)
	if !filled {
		return editorEmptyValue
	}
	return text
}

func fieldGlyph(base lipgloss.Style, f models.TaskField) string {
	switch {
	case f.ReadOnly:
		return paint(base, styles.ColorTextMuted, glyphLocked+" ")
	case f.Kind == models.InputBool && (models.Truthy(f.Value) || f.Value == nil && models.Truthy(f.Default)):
		return paint(base, styles.ColorSuccess, glyphOn+" ")
	case f.Kind == models.InputBool:
		return paint(base, styles.ColorTextMuted, glyphOff+" ")
	case f.Kind == models.InputChoice, f.Kind == models.InputEndpoint, f.Kind == models.InputQueue, f.Kind == models.InputAzure:
		return paint(base, styles.ColorPrimary, glyphPick+" ")
	}
	return paint(base, styles.ColorTextMuted, glyphText+" ")
}

func (v *ReleaseEditor) valueText(base lipgloss.Style, f models.TaskField, stale bool, width int) string {
	text, filled := v.fieldText(f)
	text = ansi.Truncate(text, max(width, 1), "…")
	switch {
	case stale:
		return base.Background(styles.ColorErrorTint).Foreground(styles.ColorStaleText).Strikethrough(true).Render(text)
	case f.Secret:
		label := "secret"
		if filled {
			label = "new value"
		}
		return paint(base, styles.ColorMuted, text) + base.Render(" ") + lipgloss.NewStyle().Background(styles.ColorSecretTint).Foreground(styles.ColorSecretLight).Render(" "+label+" ")
	case !filled:
		return base.Foreground(styles.ColorTextMuted).Italic(true).Render(text)
	case f.Kind == models.InputBool && text == "Yes":
		return paint(base, styles.ColorSuccess, text)
	case f.Kind == models.InputBool:
		return paint(base, styles.ColorTextMuted, text)
	case f.Kind == models.InputEndpoint, f.Kind == models.InputQueue, f.Kind == models.InputAzure, f.Key == "$agent":
		return lipgloss.NewStyle().Background(styles.ColorDarkBlue).Foreground(styles.ColorPrimaryLight).Render(" " + text + " ")
	}
	return styledText(base, text)
}

func styledText(base lipgloss.Style, text string) string {
	if !strings.Contains(text, "$(") && strings.Contains(text, "/") && !strings.Contains(text, " ") {
		dir, file := path.Split(text)
		return paint(base, styles.ColorTextMuted, dir) + base.Foreground(styles.ColorText).Bold(true).Render(file)
	}
	var out strings.Builder
	last := 0
	for _, m := range variableRef.FindAllStringIndex(text, -1) {
		out.WriteString(paint(base, styles.ColorText, text[last:m[0]]))
		out.WriteString(paint(base, styles.ColorVioletLight, text[m[0]:m[1]]))
		last = m[1]
	}
	out.WriteString(paint(base, styles.ColorText, text[last:]))
	return out.String()
}

func (v *ReleaseEditor) fieldLines(width int) (string, []string, [][2]int) {
	focused := v.pane == editorPaneFields
	s, p, t, isTask, ok := v.currentRow()
	if !ok {
		return PaneTitle("", "Settings", focused), []string{Muted("Choose a job or a task.")}, nil
	}
	var fields []models.TaskField
	var title string
	hidden := 0
	var def *models.TaskDefinition
	if isTask {
		def = v.refs.Task(t)
		fields, hidden = v.doc.TaskFields(s, p, t, def, v.edits)
		kind := v.taskRef(t)
		if def != nil && def.FriendlyName != "" {
			kind = def.FriendlyName + " · " + t.Version
		}
		title = PaneTitle("", models.FormatDocValue(v.edits.Value(v.doc.TaskPath(s, p, t).Child("name"), t.Name)), focused) + " " +
			lipgloss.NewStyle().Background(styles.ColorVioletTint).Foreground(styles.ColorVioletLight).Padding(0, 1).Render(kind)
	} else {
		fields = v.doc.PhaseFields(s, p, v.refs.Queues, v.edits)
		title = PaneTitle("", v.fieldByKey(fields, "$name"), focused) + " " +
			lipgloss.NewStyle().Background(styles.ColorDarkBlue).Foreground(styles.ColorPrimaryLight).Padding(0, 1).Render(p.RunsOn())
	}
	changed := 0
	for _, f := range fields {
		if f.Changed() {
			changed++
		}
	}
	if changed > 0 {
		title += editorChanged.Render(fmt.Sprintf("  %d of %d changed", changed, len(fields)))
	}
	labelWidth := min(max(width*editorLabelRatio/100, editorLabelMinWidth), editorLabelMaxWidth)
	valueWidth := max(width-editorGutterWidth-labelWidth-editorGlyphWidth, 1)
	var lines []string
	spans := make([][2]int, 0, len(fields))
	group := ""
	for i, f := range fields {
		if f.Group != group {
			if group != "" {
				lines = append(lines, "")
			}
			group = f.Group
			bg, fg := groupColors(group)
			lines = append(lines, sectionTag(group, bg, fg, "", width))
		}
		cursor := i == v.field && focused
		base := rowStyle(cursor, focused)
		stale := v.isStale(f.Path)
		labelStyle := base.Foreground(styles.ColorTextMuted)
		switch {
		case stale:
			labelStyle = base.Foreground(styles.ColorError).Bold(true)
		case f.Changed():
			labelStyle = base.Foreground(styles.ColorOrange).Bold(true)
		}
		label := ansi.Truncate(f.Label, labelWidth-3, "…")
		cell := labelStyle.Render(label)
		if f.Required {
			cell += paint(base, styles.ColorError, " *")
		}
		cell += base.Render(strings.Repeat(" ", max(labelWidth-lipgloss.Width(cell), 0)))
		tail := ""
		if f.FromAzure() {
			tail = base.Render("  ") + lipgloss.NewStyle().Background(styles.ColorTealTint).Foreground(styles.ColorTealLight).Render(" "+editorAzureTag+" ")
		}
		value := v.valueText(base, f, stale, max(valueWidth-lipgloss.Width(tail), 1))
		start := len(lines)
		lines = append(lines, fillRow(base, gutter(base, f.Changed(), stale, cursor)+cell+fieldGlyph(base, f)+value+tail, "", width))
		pad := strings.Repeat(" ", editorGutterWidth+labelWidth)
		switch {
		case stale:
			cause := "listed for the old value"
			for _, st := range v.stale {
				if st.Path.String() == f.Path.String() {
					cause = "listed for the old " + st.Cause
				}
			}
			lines = append(lines, editorNeeds.Render(glyphNeeds)+pad[1:]+editorNeeds.Render(ansi.Truncate(cause+": pick again", valueWidth, "…")))
		case f.Changed():
			lines = append(lines, editorChanged.Render(glyphChanged)+pad[1:]+lipgloss.NewStyle().Foreground(styles.ColorError).Render(glyphWas+" ")+
				editorMuted.Strikethrough(true).Render(ansi.Truncate(v.beforeText(f), max(valueWidth-2, 1), "…")))
		}
		spans = append(spans, [2]int{start, len(lines) - 1})
	}
	var notes []string
	switch {
	case isTask && t.TaskGroup():
		notes = append(notes, "A task group: its own tasks are edited in Task groups on Azure DevOps.")
	case isTask && def == nil:
		notes = append(notes, "The task's details didn't load, so its inputs show by name.")
	}
	if hidden > 0 {
		notes = append(notes, glyphHidden+" "+plural1(hidden, "input")+" hidden by the values above.")
	}
	if len(notes) > 0 {
		lines = append(lines, "")
		for _, n := range notes {
			lines = append(lines, Line(Muted(n), width))
		}
	}
	return title, lines, spans
}

func (v *ReleaseEditor) scopeColor(stage int) color.Color {
	if stage < 0 {
		return styles.ColorPrimary
	}
	for i, s := range v.doc.Stages {
		if s.Index == stage {
			return stageScopes[i%len(stageScopes)]
		}
	}
	return styles.ColorTextMuted
}

func (v *ReleaseEditor) scopeLabels() []string {
	d := v.doc
	all := fmt.Sprintf("All %d", len(d.Variables))
	labels := []string{all, glyphScope + fmt.Sprintf(" %s %d", models.ReleaseScope, len(d.StageVariables(-1)))}
	for _, s := range d.Stages {
		labels = append(labels, glyphScope+fmt.Sprintf(" %s %d", s.Name, len(d.StageVariables(s.Index))))
	}
	return labels
}

func (v *ReleaseEditor) variableRows() int {
	groups := len(v.doc.Groups)
	reserved := 3
	if groups > 0 {
		reserved += min(groups, 4) + 2
	}
	return max(v.bodyHeight()-reserved, 1)
}

func (v *ReleaseEditor) variablesView(height int) string {
	width := max(v.width-2, 1)
	var chips []string
	for i, label := range v.scopeLabels() {
		fg := styles.ColorLight
		if i > 0 {
			fg = v.scopeColor(-1)
			if i > 1 {
				fg = v.scopeColor(v.doc.Stages[i-2].Index)
			}
		}
		if i == v.scope {
			chips = append(chips, lipgloss.NewStyle().Background(styles.ColorHighlight).Foreground(styles.ColorPrimary).Bold(true).Padding(0, 1).Render(label))
		} else {
			chips = append(chips, lipgloss.NewStyle().Background(styles.ColorChip).Foreground(fg).Padding(0, 1).Render(label))
		}
	}
	hint := v.keys.Left.Help().Key + "/" + v.keys.Right.Help().Key + " scope"
	lines := []string{spreadLine(Muted("Scope ")+FitItems(chips, " ", max(width-lipgloss.Width(hint)-8, 1)), Muted(hint), width)}
	nameW := min(max(width*26/100, 12), 36)
	scopeW := min(max(width*22/100, 10), 32)
	valueW := max(width-editorGutterWidth-nameW-scopeW-editorSettableWidth-3, 8)
	cell := func(s lipgloss.Style, text string, w int) string {
		return s.Width(w).MaxWidth(w).Inline(true).Render(ansi.Truncate(text, w-1, "…"))
	}
	head := editorMuted.Bold(true)
	lines = append(lines, "  "+cell(head, "Name", nameW)+" "+cell(head, "Value", valueW)+" "+cell(head, "Scope", scopeW)+" "+cell(head, "At release", editorSettableWidth))
	vars := v.scopedVariables()
	rows := v.variableRows()
	if len(vars) == 0 {
		lines = append(lines, Muted("  No variables in this scope."))
	}
	top := min(max(v.varTop, 0), max(len(vars)-rows, 0))
	for i := top; i < min(top+rows, len(vars)); i++ {
		variable := vars[i]
		f := v.doc.VariableField(variable, v.edits)
		selected := i == v.variable
		base := rowStyle(selected, true)
		nameStyle := base.Foreground(styles.ColorText)
		if selected {
			nameStyle = nameStyle.Foreground(styles.ColorPrimary).Bold(true)
		}
		if f.Changed() {
			nameStyle = nameStyle.Foreground(styles.ColorOrange).Bold(true)
		}
		value := v.valueText(base, f, false, valueW-1)
		value += base.Render(strings.Repeat(" ", max(valueW-lipgloss.Width(value), 0)))
		settable := paint(base, styles.ColorTextMuted, glyphOff+" no")
		if variable.Settable {
			settable = paint(base, styles.ColorSuccess, glyphOn+" yes")
		}
		scope := paint(base, v.scopeColor(variable.Stage), glyphScope+" ") + paint(base, styles.ColorTextMuted, ansi.Truncate(variable.Scope, scopeW-3, "…"))
		scope += base.Render(strings.Repeat(" ", max(scopeW-lipgloss.Width(scope), 0)))
		gap := base.Render(" ")
		row := gutter(base, f.Changed(), false, selected) + cell(nameStyle, variable.Name, nameW) + gap + value + gap + scope + gap + settable
		lines = append(lines, fillRow(base, row, "", width))
		if f.Changed() {
			lines = append(lines, editorChanged.Render(glyphChanged)+strings.Repeat(" ", nameW+2)+lipgloss.NewStyle().Foreground(styles.ColorError).Render(glyphWas+" ")+
				editorMuted.Strikethrough(true).Render(ansi.Truncate(v.beforeText(f), max(valueW-2, 1), "…")))
		}
	}
	if len(vars) > rows {
		lines = append(lines, Muted(fmt.Sprintf("  %d of %d", min(v.variable+1, len(vars)), len(vars))))
	}
	if len(v.doc.Groups) > 0 {
		lines = append(lines, "", sectionTag("Variable groups", styles.ColorChip, styles.ColorLight, "read-only · edited in Library", width))
		for _, g := range v.doc.Groups {
			info := v.refs.Groups[g.ID]
			name := g.Name
			if info.Name != "" {
				name = info.Name
			}
			if name == "" {
				name = fmt.Sprintf("group %d", g.ID)
			}
			detail := ""
			if info.Name != "" {
				detail = plural1(info.Variables, "variable") + "  "
			}
			lines = append(lines, Line(editorMuted.Render(glyphLocked+" ")+editorText.Render(name)+"  "+editorMuted.Render(detail)+
				lipgloss.NewStyle().Foreground(v.scopeColor(g.Stage)).Render(glyphScope+" ")+editorMuted.Render(g.Scope), width))
		}
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(Block(strings.Join(lines, "\n"), width, height))
}

func (v *ReleaseEditor) docStageBox(s models.DocStage, selected bool, width int) string {
	border := styles.ColorBorder
	if selected {
		border = styles.ColorPrimary
	}
	inner := max(width-4, 1)
	approval := editorMuted.Render("no approval")
	if len(s.PreApprovers) > 0 {
		approval = lipgloss.NewStyle().Foreground(styles.ColorOrange).Render(plural1(len(s.PreApprovers), "approver"))
	}
	lines := []string{
		Line(editorBold.Render(s.Name), inner),
		Line(editorAccent.Render(s.Summary()), inner),
		Line(editorMuted.Render(s.Trigger), inner),
		Line(approval, inner),
	}
	if v.doc.Kind == models.DocRelease {
		state := StageStateText(models.StageStateOf(s.Status, nil))
		if s.Locked {
			state += editorMuted.Render(" · read-only")
		}
		lines = append(lines, Line(state, inner))
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1).Width(width).
		Render(strings.Join(lines, "\n"))
}

func (v *ReleaseEditor) pipelineContent(width int) string {
	d := v.doc
	var parts []string
	var art []string
	for _, a := range d.Artifacts {
		kind := a.Type
		if a.Primary {
			kind += " · primary"
		}
		if a.Source != "" {
			kind += " · " + a.Source
		}
		art = append(art, Line(editorBold.Render(a.Alias)+editorMuted.Render("  "+kind), width))
	}
	if len(art) == 0 {
		art = append(art, Muted("No artifacts."))
	}
	if d.Kind == models.DocDefinition {
		triggers := d.Triggers
		if len(triggers) == 0 {
			triggers = []string{"No automatic triggers: releases are created by hand."}
		}
		for _, t := range triggers {
			art = append(art, Line(lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(t), width))
		}
	}
	parts = append(parts, sectionTitle("Artifacts and triggers", width)+"\n"+strings.Join(art, "\n"))
	if len(d.Stages) > 0 {
		boxWidth := min(max(width/max(len(d.Stages), 1)-1, editorStageBoxMin), editorStageBoxMax)
		perRow := max((width+1)/(boxWidth+1), 1)
		var rows []string
		for start := 0; start < len(d.Stages); start += perRow {
			var boxes []string
			for i := start; i < min(start+perRow, len(d.Stages)); i++ {
				if i > start {
					boxes = append(boxes, " ")
				}
				boxes = append(boxes, v.docStageBox(d.Stages[i], i == v.stage, boxWidth))
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, boxes...))
		}
		parts = append(parts, sectionTitle("Stages", width)+"\n"+strings.Join(rows, "\n"))
		parts = append(parts, v.stageDetails(d.Stages[v.stage], width))
	}
	details := [][2]string{}
	if d.Kind == models.DocDefinition {
		details = append(details,
			[2]string{"Revision", fmt.Sprint(d.Revision)},
			[2]string{"Modified", strings.TrimSpace(d.ModifiedBy + " " + releaseStamp(d.Modified))},
			[2]string{"Release name", d.NameFormat},
			[2]string{"Folder", d.Folder},
		)
	} else {
		details = append(details,
			[2]string{"Release", d.Name},
			[2]string{"Pipeline", fmt.Sprintf("%s · revision %d", d.Definition, d.Revision)},
		)
	}
	parts = append(parts, sectionTitle("Details", width)+"\n"+FieldRows(keptRows(details), fieldLabel, width))
	return strings.Join(parts, "\n\n")
}

func (v *ReleaseEditor) stageDetails(s models.DocStage, width int) string {
	d := v.doc
	gates := "off"
	if s.Gates {
		gates = "on"
	}
	rows := [][2]string{
		{"Trigger", s.Trigger},
		{"Before", "approval: " + s.Approval(true)},
		{"After", "approval: " + s.Approval(false)},
		{"Gates", gates},
	}
	for _, p := range s.Phases {
		rows = append(rows, [2]string{"Job", strings.Join(slices.DeleteFunc([]string{p.Name, v.phaseWhere(p), plural1(len(p.Tasks), "task")}, func(s string) bool { return s == "" }), " · ")})
	}
	vars := plural1(len(d.StageVariables(s.Index)), "variable")
	var groups []string
	for _, g := range d.StageGroups(s.Index) {
		name := g.Name
		if info := v.refs.Groups[g.ID]; info.Name != "" {
			name = info.Name
		}
		groups = append(groups, name)
	}
	if len(groups) > 0 {
		vars += " · groups " + strings.Join(groups, ", ")
	}
	rows = append(rows, [2]string{"Variables", vars}, [2]string{"Retention", s.Retention})
	title := s.Name
	if d.Kind == models.DocRelease {
		title += Muted(" · ") + StageStateText(models.StageStateOf(s.Status, nil))
		if s.Locked {
			rows = append(rows, [2]string{"Editing", "read-only: it has already deployed"})
		}
	}
	return sectionTitle(title, width) + "\n" + FieldRows(keptRows(rows), fieldLabel, width) + "\n" +
		Muted(v.keys.Select.Help().Key+" or "+v.keys.Panel2.Help().Key+" opens its tasks.")
}

func (v *ReleaseEditor) historyContent(width int) string {
	if v.kind == models.DocRelease {
		return Muted("History is kept for release pipelines. Press " + v.keys.EditPipeline.Help().Key + " on the pipeline in the hub to see it.")
	}
	switch {
	case v.revState == revisionsLoading:
		return Loading(v.spinner, "Loading history")
	case v.revErr != "":
		return lipgloss.NewStyle().Foreground(styles.ColorError).Render(v.revErr)
	case len(v.revisions) == 0:
		return Muted("No history yet.")
	}
	lines := []string{sectionTitle("History", width)}
	for _, r := range v.revisions {
		head := editorBold.Render(fmt.Sprintf("Revision %d", r.Revision)) + editorMuted.Render("  "+releaseStamp(r.Changed)+"  "+r.ChangedBy+"  "+r.ChangeType)
		lines = append(lines, Line(head, width))
		if r.Comment != "" {
			lines = append(lines, Line("  "+editorText.Render(r.Comment), width))
		}
	}
	return strings.Join(lines, "\n")
}
