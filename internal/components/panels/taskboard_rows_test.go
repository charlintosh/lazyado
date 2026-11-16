package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

func rowsItems(pendingDone bool) []models.WorkItem {
	thirdState := models.WorkItemState("To Do")
	if pendingDone {
		thirdState = "Done"
	}
	items := []models.WorkItem{
		{ID: 10, Type: models.WorkItemTypePBI, Title: "Pending story", State: "Committed"},
		{ID: 11, Type: models.WorkItemTypeTask, Title: "Write it", State: thirdState, ParentID: 10},
		{ID: 12, Type: models.WorkItemTypeTask, Title: "Ship it", State: "Done", ParentID: 10},
		{ID: 20, Type: models.WorkItemTypePBI, Title: "Finished story", State: "Committed"},
		{ID: 21, Type: models.WorkItemTypeTask, Title: "Built", State: "Done", ParentID: 20},
		{ID: 22, Type: models.WorkItemTypeTask, Title: "Tested", State: "Done", ParentID: 20},
		{ID: 30, Type: models.WorkItemTypePBI, Title: "Closed story", State: "Done"},
		{ID: 40, Type: models.WorkItemTypePBI, Title: "Fresh story", State: "Approved"},
	}
	items[0].Children = []*models.WorkItem{&items[1], &items[2]}
	items[3].Children = []*models.WorkItem{&items[4], &items[5]}
	return items
}

func rowsBoard(t *testing.T) TaskboardPanel {
	t.Helper()
	tb := NewTaskboardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	tb.SetSize(120, 30)
	tb.SetFocused(true)
	tb.SetItems(rowsItems(false), nil)
	return tb
}

func laneIndex(tb TaskboardPanel, parent int) int {
	for li, l := range tb.lanes {
		if l.parent.ID == parent {
			return li
		}
	}
	return -1
}

func typed(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(text)[0], Text: text}
}

func TestRowsWithPendingTasksStartOpenAndDoneRowsCollapsed(t *testing.T) {
	tb := rowsBoard(t)
	want := map[int]bool{10: false, 20: true, 30: true, 40: false}
	for parent, collapsed := range want {
		if got := tb.isCollapsed(laneIndex(tb, parent)); got != collapsed {
			t.Errorf("row %d collapsed=%v, want %v", parent, got, collapsed)
		}
	}
	plain := ansi.Strip(tb.View())
	if strings.Contains(plain, "21 Built") || !strings.Contains(plain, chevronClosed+" 20 Finished story") || !strings.Contains(plain, allDoneGlyph+"2/2") {
		t.Errorf("a done row is one line with its count:\n%s", plain)
	}
	if !strings.Contains(tb.Title(), "· 2 collapsed") {
		t.Errorf("title %q should count collapsed rows", tb.Title())
	}
}

func TestRowDefaultsApplyOnlyTheFirstTime(t *testing.T) {
	tb := rowsBoard(t)
	tb.setCollapsed(laneIndex(tb, 20), false)
	tb.SetItems(rowsItems(true), nil)
	if tb.isCollapsed(laneIndex(tb, 20)) {
		t.Error("a row the user opened stays open after a reload")
	}
	if tb.isCollapsed(laneIndex(tb, 10)) {
		t.Error("a row whose last task finishes later is not collapsed under the user")
	}
}

func TestKeysCollapseAndExpandRows(t *testing.T) {
	tb := rowsBoard(t)
	tb, _ = tb.Update(typed("h"))
	if !tb.isCollapsed(0) {
		t.Fatal("h on a backlog item collapses its row")
	}
	tb, _ = tb.Update(typed("l"))
	if tb.isCollapsed(0) {
		t.Fatal("l on a collapsed row expands it")
	}
	tb, _ = tb.Update(typed("l"))
	if tb.SelectedItem().ID != 11 {
		t.Fatalf("l on an open row moves into its tasks, selected %d", tb.SelectedItem().ID)
	}
	tb, _ = tb.Update(typed("C"))
	if tb.anyExpanded() || tb.cell != 0 || tb.SelectedItem().ID != 10 {
		t.Fatal("C collapses every row and moves the selection to the backlog item")
	}
	tb, _ = tb.Update(typed("C"))
	if tb.collapsedCount() != 0 {
		t.Fatal("C again expands every row")
	}
}

func TestPointerTogglesRowsAndIgnoresHiddenTasks(t *testing.T) {
	tb := rowsBoard(t)
	x, y := position(t, tb, "20 Finished story", false)
	cx, _ := position(t, tb, "Done  ", true)
	tb, _ = tb.Update(tea.MouseClickMsg(at(cx, y)))
	if tb.Dragging() {
		t.Fatal("a collapsed row has no cards to drag")
	}
	tb, _ = tb.Update(tea.MouseClickMsg(at(x, y)))
	if tb.isCollapsed(laneIndex(tb, 20)) {
		t.Fatal("clicking a collapsed row expands it")
	}
	_, cy := position(t, tb, "Finished story", false)
	tb, _ = tb.Update(tea.MouseClickMsg(at(chevronFrom+1, cy)))
	if !tb.isCollapsed(laneIndex(tb, 20)) {
		t.Fatal("clicking ▾ collapses the row")
	}
	hx, hy := position(t, tb, collapseGlyph, true)
	tb, _ = tb.Update(tea.MouseClickMsg(at(hx, hy)))
	if tb.anyExpanded() {
		t.Fatal("clicking the header collapses every row")
	}
}

func TestDroppingOnACollapsedRowUsesItsColumn(t *testing.T) {
	tb := rowsBoard(t)
	x, y := position(t, tb, "11 ", false)
	cx, _ := position(t, tb, "Done  ", true)
	_, ry := position(t, tb, "20 Finished story", false)
	tb, _ = tb.Update(tea.MouseClickMsg(at(x, y)))
	tb, _ = tb.Update(tea.MouseMotionMsg(at(cx, ry)))
	_, cmd := tb.Update(tea.MouseReleaseMsg(at(cx, ry)))
	if move, ok := cmd().(MoveWorkItemMsg); !ok || move.Item.ID != 11 || move.NewState != "Done" {
		t.Fatalf("dropping on a collapsed row's Done cell produced %#v", cmd())
	}
}

func TestCollapsedRowsAreValues(t *testing.T) {
	tb := rowsBoard(t)
	before := tb
	tb.toggleAllRows()
	if !before.anyExpanded() || tb.anyExpanded() {
		t.Fatal("collapsing rows changed an earlier copy of the model")
	}
}
