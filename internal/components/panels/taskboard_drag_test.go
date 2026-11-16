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

func sprintItems(doneState models.WorkItemState) []models.WorkItem {
	items := []models.WorkItem{
		{ID: 10, Type: models.WorkItemTypePBI, Title: "Story", State: "Committed"},
		{ID: 11, Type: models.WorkItemTypeTask, Title: "Write it", State: "To Do", ParentID: 10},
		{ID: 12, Type: models.WorkItemTypeTask, Title: "Ship it", State: doneState, ParentID: 10},
	}
	items[0].Children = []*models.WorkItem{&items[1], &items[2]}
	return items
}

func dragBoard(t *testing.T) TaskboardPanel {
	t.Helper()
	tb := NewTaskboardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	tb.SetSize(120, 20)
	tb.SetOrigin(0, 0)
	tb.SetFocused(true)
	tb.SetItems(sprintItems("Done"), nil)
	return tb
}

func position(t *testing.T, tb TaskboardPanel, text string, header bool) (int, int) {
	t.Helper()
	for y, line := range strings.Split(ansi.Strip(tb.View()), "\n") {
		if header != isHeaderRow(line) {
			continue
		}
		if i := strings.Index(line, text); i >= 0 {
			return ansi.StringWidth(line[:i]) + 1, y
		}
	}
	t.Fatalf("%q is not on the board:\n%s", text, ansi.Strip(tb.View()))
	return 0, 0
}

func at(x, y int) tea.Mouse { return tea.Mouse{X: x, Y: y, Button: tea.MouseLeft} }

func isHeaderRow(line string) bool {
	return strings.Contains(line, collapseGlyph) || strings.Contains(line, expandGlyph)
}

func TestDragStateIsAValueSoOlderModelsDoNotChange(t *testing.T) {
	tb := dragBoard(t)
	x, y := position(t, tb, "11 ", false)
	dx, dy := position(t, tb, "Done  ", true)
	tb, _ = tb.Update(tea.MouseClickMsg(at(x, y)))
	before := tb
	after, _ := tb.Update(tea.MouseMotionMsg(at(dx, dy)))
	if !strings.Contains(after.Title(), "to Done") {
		t.Fatalf("the new model should target Done, title %q", after.Title())
	}
	if strings.Contains(before.Title(), "to Done") {
		t.Fatal("updating the model changed an earlier copy of it")
	}
}

func TestAReloadDuringADragUsesFreshData(t *testing.T) {
	tb := dragBoard(t)
	x, y := position(t, tb, "12 ", false)
	tx, ty := position(t, tb, "To Do  ", true)
	tb, _ = tb.Update(tea.MouseClickMsg(at(x, y)))
	tb, _ = tb.Update(tea.MouseMotionMsg(at(tx, ty)))
	tb.SetItems(sprintItems("To Do"), nil)
	if _, cmd := tb.Update(tea.MouseReleaseMsg(at(tx, ty))); cmd != nil {
		t.Fatalf("12 is already To Do after the reload, but the drop produced %#v", cmd())
	}

	tb = dragBoard(t)
	x, y = position(t, tb, "11 ", false)
	tb, _ = tb.Update(tea.MouseClickMsg(at(x, y)))
	tb.SetItems([]models.WorkItem{{ID: 10, Type: models.WorkItemTypePBI, Title: "Story", State: "Committed"}}, nil)
	if tb.Dragging() {
		t.Fatal("a drag must end when its task disappears in a reload")
	}
}

func TestMotionWithoutADragLeavesTheBoardAlone(t *testing.T) {
	tb := dragBoard(t)
	before := tb.View()
	next, cmd := tb.Update(tea.MouseMotionMsg(at(40, 5)))
	if cmd != nil || next.Dragging() || next.View() != before {
		t.Fatal("pointer motion without a drag must not change anything")
	}
}

func hasColor(s, rgb string) bool {
	return strings.Contains(s, "48;2;"+rgb)
}

func dragOver(t *testing.T, tb TaskboardPanel, task, header string) TaskboardPanel {
	t.Helper()
	x, y := position(t, tb, task, false)
	hx, hy := position(t, tb, header, true)
	tb, _ = tb.Update(tea.MouseClickMsg(at(x, y)))
	tb, _ = tb.Update(tea.MouseMotionMsg(at(hx, hy)))
	return tb
}

func TestHoverLiftsTheCardUnderThePointer(t *testing.T) {
	tb := dragBoard(t)
	x, y := position(t, tb, "11 ", false)
	tb, _ = tb.Update(tea.MouseMotionMsg(tea.Mouse{X: x, Y: y}))
	if !strings.Contains(ansi.Strip(tb.View()), gripGlyph) {
		t.Fatal("the hovered card should show its grip")
	}
	if hints := tb.KeyBindings(); hints[0].Help().Desc != "move 11" {
		t.Fatalf("first hint is %q, want the drag hint for 11", hints[0].Help().Desc)
	}
	tb, _ = tb.Update(tea.MouseMotionMsg(tea.Mouse{X: 0, Y: 0}))
	if strings.Contains(ansi.Strip(tb.View()), gripGlyph) {
		t.Fatal("moving off the card should drop the hover")
	}
}

func TestDraggingShowsGhostPhantomTintAndCounts(t *testing.T) {
	tb := dragOver(t, dragBoard(t), "11 ", "Done  ")
	view := tb.View()
	plain := ansi.Strip(view)
	for _, want := range []string{"┆ 11 Write it", ghostNote, "┇ 11 Write it", phantomNote + "Done", "To Do  1" + countArrow + "0", dropMarker + "Done  1" + countArrow + "2"} {
		if !strings.Contains(plain, want) {
			t.Errorf("dragging over Done should show %q:\n%s", want, plain)
		}
	}
	if !hasColor(view, "26;38;51") {
		t.Error("the target column should be tinted")
	}
	if tb.Title() != "Taskboard  moving 11 to Done" {
		t.Errorf("title %q", tb.Title())
	}
}

func TestNoPhantomWhenTheDropWouldDoNothing(t *testing.T) {
	cases := map[string]struct {
		header string
		title  string
	}{
		"own column":  {"To Do  ", "Taskboard  moving 11 · already To Do"},
		"lane column": {collapseGlyph, "Taskboard  moving 11 · " + dropPrompt},
	}
	for name, c := range cases {
		tb := dragOver(t, dragBoard(t), "11 ", c.header)
		plain := ansi.Strip(tb.View())
		if strings.Contains(plain, "┇") || hasColor(tb.View(), "26;38;51") {
			t.Errorf("%s: no phantom or tint expected:\n%s", name, plain)
		}
		if tb.Title() != c.title {
			t.Errorf("%s: title %q, want %q", name, tb.Title(), c.title)
		}
	}
}

func TestDragChipStaysOffTheTargetColumn(t *testing.T) {
	tb := dragOver(t, dragBoard(t), "11 ", "Done  ")
	chip, x, _, ok := tb.DragChip()
	if !ok || !strings.Contains(ansi.Strip(chip), chipMoveNote+"Done") {
		t.Fatalf("chip %q", ansi.Strip(chip))
	}
	left, right, _ := tb.columnSpan("Done")
	if end := x + lipgloss.Width(chip); x < right && end > left {
		t.Fatalf("chip spans %d-%d and covers the Done column %d-%d", x, end, left, right)
	}
}

func TestADropMovesTheCardBeforeTheServerAnswers(t *testing.T) {
	tb := dragOver(t, dragBoard(t), "11 ", "Done  ")
	hx, hy := position(t, tb, "Done  ", true)
	tb, cmd := tb.Update(tea.MouseReleaseMsg(at(hx, hy)))
	if move, ok := cmd().(MoveWorkItemMsg); !ok || move.Item.ID != 11 || move.NewState != "Done" {
		t.Fatalf("drop produced %#v", cmd())
	}
	if !tb.HasPendingMove(11) || !strings.Contains(ansi.Strip(tb.View()), savingNote+"Done…") {
		t.Fatalf("the card should already sit in Done and say it is saving:\n%s", ansi.Strip(tb.View()))
	}
	expire := tb.MoveSaved(11)
	if !strings.Contains(ansi.Strip(tb.View()), savedNote+"Done") || expire == nil {
		t.Fatal("a saved move flashes Moved to Done and schedules the flash to end")
	}
	tb.ExpireFlash(MoveFlashExpiredMsg{ID: 11, Seq: tb.flashSeq})
	if task, _ := tb.taskByID(11); tb.displayState(task) != "Done" {
		t.Fatal("until the reload arrives the card stays in Done")
	}
	tb.SetItems(sprintItems("Done"), nil)
	if task, _ := tb.taskByID(11); tb.displayState(task) != "Done" {
		t.Fatal("a reload that still has the old state must not snap the card back")
	}
	items := sprintItems("Done")
	items[1].State = "Done"
	tb.SetItems(items, nil)
	if len(tb.moves) != 0 {
		t.Fatalf("once Azure DevOps agrees the move is forgotten, got %v", tb.moves)
	}
}

func TestAFailedMoveGoesBack(t *testing.T) {
	tb := dragBoard(t)
	task, _ := tb.taskByID(11)
	tb.beginMove(task, "Done")
	tb.MoveFailed(11)
	plain := ansi.Strip(tb.View())
	if task, _ := tb.taskByID(11); tb.displayState(task) != "To Do" || !strings.Contains(plain, failedNote+"To Do") {
		t.Fatalf("a failed move returns to To Do in red:\n%s", plain)
	}
	tb.ExpireFlash(MoveFlashExpiredMsg{ID: 11, Seq: tb.flashSeq})
	if len(tb.moves) != 0 || strings.Contains(ansi.Strip(tb.View()), failedNote) {
		t.Fatal("the failure flash ends and the move is forgotten")
	}
}

func TestOldFlashTimersAreIgnoredAndMovesAreValues(t *testing.T) {
	tb := dragBoard(t)
	task, _ := tb.taskByID(11)
	tb.beginMove(task, "Done")
	before := tb
	tb.MoveSaved(11)
	old := tb.flashSeq
	if before.moves[11].flash != flashSaving {
		t.Fatal("settling a move changed an earlier copy of the model")
	}
	tb.beginMove(task, "To Do")
	tb.MoveSaved(11)
	tb.ExpireFlash(MoveFlashExpiredMsg{ID: 11, Seq: old})
	if tb.moves[11].flash != flashSaved {
		t.Fatal("an older timer must not end a newer flash")
	}
}

func TestKeyboardMovesAreOptimisticToo(t *testing.T) {
	tb := dragBoard(t)
	x, y := position(t, tb, "11 ", false)
	tb, _ = tb.Update(tea.MouseClickMsg(at(x, y)))
	tb, _ = tb.Update(tea.MouseReleaseMsg(at(x, y)))
	tb, cmd := tb.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	if move, ok := cmd().(MoveWorkItemMsg); !ok || move.Item.ID != 11 || !tb.HasPendingMove(11) {
		t.Fatalf("> produced %#v and pending=%v", cmd(), tb.HasPendingMove(11))
	}
}
