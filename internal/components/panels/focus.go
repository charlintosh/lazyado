package panels

import (
	"fmt"
	"path"

	"github.com/charlintosh/lazyado/internal/models"
)

func LinesLabel(n int) string {
	if n == 1 {
		return "1 LINE"
	}
	return fmt.Sprintf("%d LINES", n)
}

func (l *runLogs) focusLabel() string {
	if l.pane == runPaneLog {
		if r, ok := l.selectedRecord(); ok {
			return "LOG · " + r.Name
		}
		return "LOG"
	}
	return "JOBS"
}

func (v *PipelineRunView) Focus() string {
	switch {
	case v.tab == RunLogsTab:
		return v.focusLabel()
	case v.focus == summaryPaneJobs:
		return "JOBS"
	}
	return "STAGES"
}

func (v *ReleaseView) Focus() string {
	if v.tab == ReleaseLogsTab {
		return v.focusLabel()
	}
	if s, ok := v.SelectedStage(); ok {
		return "STAGE " + s.Name
	}
	return "STAGES"
}

func (v *PullRequestView) Focus() (string, bool) {
	if v.tab != PullRequestFiles {
		return "OVERVIEW", false
	}
	if n := v.files.selectedLines(); n > 0 {
		return LinesLabel(n), true
	}
	if f, ok := v.files.file(); ok {
		return path.Base(f.Path), false
	}
	return "FILES", false
}

func (v *ReleaseEditor) Focus() string {
	if v.doc == nil {
		return "LOADING"
	}
	switch v.tab {
	case EditorPipelineTab:
		if s, ok := v.currentStage(); ok {
			return "STAGE " + s.Name
		}
		return "STAGES"
	case EditorVariablesTab:
		if vars := v.scopedVariables(); v.variable < len(vars) {
			return "VARIABLE " + vars[v.variable].Name
		}
		return "VARIABLES"
	case EditorHistoryTab:
		return "HISTORY"
	}
	if v.pane == editorPaneFields {
		if fields := v.fields(); v.field < len(fields) {
			return fields[v.field].Label
		}
	}
	s, p, t, isTask, ok := v.currentRow()
	switch {
	case !ok:
		return s.Name
	case isTask:
		return "TASK " + models.FormatDocValue(v.edits.Value(v.doc.TaskPath(s, p, t).Child("name"), t.Name))
	}
	return "JOB " + p.Name
}
