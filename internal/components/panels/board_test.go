package panels

import (
	"strings"

	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

func press(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(text)[0], Text: text}
}

func boardFixture() []models.WorkItem {
	items := []models.WorkItem{
		{ID: 1, Type: models.WorkItemTypePBI, Title: "First", State: "New"},
		{ID: 2, Type: models.WorkItemTypePBI, Title: "Second", State: "Committed"},
		{ID: 3, Type: models.WorkItemTypeBug, Title: "Third", State: "Committed"},
		{ID: 4, Type: models.WorkItemTypeTask, Title: "Task A", State: "To Do", ParentID: 2},
		{ID: 5, Type: models.WorkItemTypeTask, Title: "Task B", State: "Done", ParentID: 2},
	}
	items[1].Children = []*models.WorkItem{&items[3], &items[4]}
	return items
}

func TestBoardGroupsBacklogItemsByState(t *testing.T) {
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	b.SetItems(boardFixture(), nil)
	if len(b.columns) != 2 {
		t.Fatalf("got %d columns, want 2 (New, Committed)", len(b.columns))
	}
	if b.columns[0].state != "New" || b.columns[1].state != "Committed" {
		t.Fatalf("unexpected column order: %q, %q", b.columns[0].state, b.columns[1].state)
	}
	if b.Count() != 3 {
		t.Fatalf("tasks must not appear as cards, got %d cards", b.Count())
	}
}

func TestBoardUsesAzureDevOpsWorkflowOrder(t *testing.T) {
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	states := map[string][]models.WorkItemStateInfo{
		"Product Backlog Item": {
			{Name: "New", Category: "Proposed"},
			{Name: "Approved", Category: "Proposed"},
			{Name: "Committed", Category: "InProgress"},
			{Name: "Done", Category: "Completed"},
			{Name: "Removed", Category: "Removed"},
		},
	}
	b.SetItems(boardFixture(), states)
	var got []string
	for _, c := range b.columns {
		got = append(got, c.state)
	}
	want := []string{"New", "Approved", "Committed", "Done"}
	if len(got) != len(want) {
		t.Fatalf("columns = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("columns = %v, want %v", got, want)
		}
	}
}

func TestBoardMoveCardRequestsNextState(t *testing.T) {
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	b.SetItems(boardFixture(), nil)
	b.SetSize(80, 30)
	b.SetFocused(true)

	b, cmd := b.Update(press(">"))
	if cmd == nil {
		t.Fatal("expected a move command")
	}
	msg, ok := cmd().(MoveWorkItemMsg)
	if !ok {
		t.Fatalf("got %T, want MoveWorkItemMsg", cmd())
	}
	if msg.Item.ID != 1 || msg.NewState != "Committed" {
		t.Fatalf("got item %d → %q, want 1 → Committed", msg.Item.ID, msg.NewState)
	}

	if sel := b.SelectedItem(); sel == nil || sel.ID != 1 || b.columns[b.col].state != "Committed" {
		t.Fatalf("the moved card must follow into Committed and stay selected, got %v in %q", sel, b.columns[b.col].state)
	}
	if out := ansi.Strip(b.View()); !strings.Contains(out, savingNote+"Committed…") {
		t.Fatalf("the moved card must show it is saving:\n%s", out)
	}
	if _, cmd := b.Update(press(">")); cmd != nil {
		t.Fatal("moving past the last column must do nothing")
	}
}

func TestTaskboardNavigatesLanesAndMovesTasks(t *testing.T) {
	tb := NewTaskboardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	tb.SetItems(boardFixture(), nil)
	tb.SetSize(120, 30)
	tb.SetFocused(true)

	tb, _ = tb.Update(press("j"))
	tb, _ = tb.Update(press("l"))
	if sel := tb.SelectedItem(); sel == nil || sel.ID != 4 {
		t.Fatalf("selected %v, want task 4", sel)
	}
	_, cmd := tb.Update(press(">"))
	if cmd == nil {
		t.Fatal("expected a move command")
	}
	msg := cmd().(MoveWorkItemMsg)
	if msg.Item.ID != 4 || msg.NewState != "Done" {
		t.Fatalf("got task %d → %q, want 4 → Done", msg.Item.ID, msg.NewState)
	}
	done, total := tb.Progress()
	if done != 1 || total != 2 {
		t.Fatalf("progress = %d/%d, want 1/2", done, total)
	}
}

func TestPaneIsExactSize(t *testing.T) {
	out := Pane(PaneTitle("5", "A very long title that will not fit", true), "body\nline", 20, 6, true)
	lines := strings.Split(out, "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d lines, want 6", len(lines))
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 20 {
			t.Fatalf("line %d is %d cells, want 20", i, w)
		}
	}
}

func TestWorkItemsPanelKeepsSelectionAcrossResize(t *testing.T) {
	w := NewWorkItemsPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	w.SetSize(120, 20)
	w.SetItems([]models.WorkItem{{ID: 1, Title: "One"}, {ID: 2, Title: "Two"}})
	w.SetFocused(true)
	w, _ = w.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	w.SetSize(80, 20)
	if sel := w.SelectedItem(); sel == nil || sel.ID != 2 {
		t.Fatalf("selection after resize = %v, want item 2", sel)
	}
}

func dragStates() map[string][]models.WorkItemStateInfo {
	return map[string][]models.WorkItemStateInfo{
		"Product Backlog Item": {{Name: "New"}, {Name: "Approved"}, {Name: "Committed"}, {Name: "Done"}},
		"Bug":                  {{Name: "New"}, {Name: "Committed"}, {Name: "Done"}},
	}
}

func dragBoardPanel(t *testing.T, width int) BoardPanel {
	t.Helper()
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	b.SetSize(width, 30)
	b.SetOrigin(0, 0)
	b.SetFocused(true)
	b.SetItems(boardFixture(), dragStates())
	return b
}

func locateOnBoard(t *testing.T, b BoardPanel, text string) (int, int) {
	t.Helper()
	for y, line := range strings.Split(ansi.Strip(b.View()), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return ansi.StringWidth(line[:i]), y
		}
	}
	t.Fatalf("%q is not on the board:\n%s", text, ansi.Strip(b.View()))
	return 0, 0
}

func TestBoardRefusesAStateTheItemTypeLacks(t *testing.T) {
	b := dragBoardPanel(t, 140)
	x, y := locateOnBoard(t, b, "3 BUG")
	hx, hy := locateOnBoard(t, b, "Approved  ")
	b, _ = b.Update(tea.MouseClickMsg(at(x, y)))
	b, _ = b.Update(tea.MouseMotionMsg(at(hx, hy+6)))
	if got := b.Title(); got != "Board  moving 3 · Bug has no Approved state" {
		t.Fatalf("got title %q", got)
	}
	if chip, _, _, _ := b.DragChip(); !strings.Contains(ansi.Strip(chip), "✕ Bug has no Approved state") {
		t.Fatalf("the label must say why:\n%s", ansi.Strip(chip))
	}
	if _, cmd := b.Update(tea.MouseReleaseMsg(at(hx, hy+6))); cmd != nil {
		t.Fatal("dropping on a state the type lacks must not move it")
	}
}

func TestBoardScrollsToHiddenColumnsAtTheEdge(t *testing.T) {
	b := dragBoardPanel(t, 63)
	if start, count, _ := b.window(); start != 0 || count != 2 {
		t.Fatalf("expected two of four columns to fit, got start %d count %d", start, count)
	}
	x, y := locateOnBoard(t, b, "1 PBI")
	b, _ = b.Update(tea.MouseClickMsg(at(x, y)))
	b, cmd := b.Update(tea.MouseMotionMsg(at(62, y)))
	if cmd == nil {
		t.Fatal("holding at the right edge must schedule a scroll")
	}
	b.EdgeScroll(BoardEdgeScrollMsg{Seq: b.edgeSeq})
	if start, _, _ := b.window(); start != 1 {
		t.Fatalf("the board must scroll one column right, start is %d", start)
	}
	if chip, _, _, _ := b.DragChip(); !strings.Contains(ansi.Strip(chip), edgeNoteRight+"Done") {
		t.Fatalf("the label must name the next hidden column:\n%s", ansi.Strip(chip))
	}
	b.EdgeScroll(BoardEdgeScrollMsg{Seq: b.edgeSeq - 1})
	if start, _, _ := b.window(); start != 1 {
		t.Fatal("a stale scroll tick must be ignored")
	}
}

func TestBoardFailedMoveGoesBack(t *testing.T) {
	b := dragBoardPanel(t, 140)
	b, _ = b.Update(press(">"))
	if b.columns[b.col].state != "Approved" {
		t.Fatalf("the card must move at once, it is in %q", b.columns[b.col].state)
	}
	b.MoveFailed(1)
	if b.columns[0].state != "New" || len(b.columns[0].items) != 1 || b.columns[0].items[0].ID != 1 {
		t.Fatal("a failed move must put the card back in New")
	}
	if out := ansi.Strip(b.View()); !strings.Contains(out, failedNote+"New") {
		t.Fatalf("the card must say it was not saved:\n%s", out)
	}
}

func TestBoardDoubleClickOpensTheWorkItem(t *testing.T) {
	b := dragBoardPanel(t, 140)
	x, y := locateOnBoard(t, b, "2 PBI")
	b, _ = b.Update(tea.MouseClickMsg(at(x, y)))
	b, _ = b.Update(tea.MouseReleaseMsg(at(x, y)))
	_, cmd := b.Update(tea.MouseClickMsg(at(x, y)))
	if cmd == nil {
		t.Fatal("a double click must open the work item")
	}
	if view, ok := cmd().(ViewWorkItemMsg); !ok || view.Item.ID != 2 {
		t.Fatalf("got %#v", cmd())
	}
}

func TestBoardCardsAreFiveRowsWithTheirDetails(t *testing.T) {
	items := []models.WorkItem{{ID: 7, Type: models.WorkItemTypePBI, Title: "Guest checkout behind a feature flag for every store and every region we run today", State: "New",
		AssignedTo: "Dana Reyes", Effort: 5, Tags: []string{"checkout", "eu", "beta"}}}
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	b.SetSize(40, 20)
	b.SetItems(items, nil)
	out := ansi.Strip(b.View())
	for _, want := range []string{"┃ 7 PBI", "Effort 5", "Guest checkout behind a", "…", "checkout", "+1", "DR  Dana Reyes"} {
		if !strings.Contains(out, want) {
			t.Errorf("card is missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "┃") != styles.BoardCardHeight {
		t.Errorf("a card must be %d rows tall:\n%s", styles.BoardCardHeight, out)
	}
}
