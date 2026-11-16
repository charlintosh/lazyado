package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charmbracelet/x/ansi"
)

func locate(t testing.TB, a App, text string) (int, int, bool) {
	t.Helper()
	for y, line := range strings.Split(ansi.Strip(a.render()), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return ansi.StringWidth(line[:i]), y, true
		}
	}
	return 0, 0, false
}

func locateHeader(t testing.TB, a App, state string) (int, int, bool) {
	t.Helper()
	for y, line := range strings.Split(ansi.Strip(a.render()), "\n") {
		if !strings.Contains(line, "Collapse all") && !strings.Contains(line, "Expand all") {
			continue
		}
		if i := strings.Index(line, state+"  "); i >= 0 {
			return ansi.StringWidth(line[:i]), y, true
		}
	}
	return 0, 0, false
}

func mouse(x, y int) tea.Mouse {
	return tea.Mouse{X: x, Y: y, Button: tea.MouseLeft}
}

func dragTo(t *testing.T, a App, fromX, fromY, toX, toY int) (App, []tea.Msg) {
	t.Helper()
	a, _ = step(t, a, tea.MouseClickMsg(mouse(fromX, fromY)))
	a, _ = step(t, a, tea.MouseMotionMsg(mouse(toX, toY)))
	return step(t, a, tea.MouseReleaseMsg(mouse(toX, toY)))
}

func taskboardApp(t testing.TB, width, height int, keys ...string) App {
	t.Helper()
	a := newTestApp(t, width, height)
	for _, k := range append([]string{"S"}, keys...) {
		a, _ = step(t, a, press(k))
	}
	return a
}

func TestDraggingATaskToAColumnChangesItsState(t *testing.T) {
	layouts := map[string]struct {
		width, height int
		keys          []string
	}{
		"wide":               {160, 45, nil},
		"two states visible": {120, 40, nil},
		"compact rail":       {90, 34, nil},
		"rail hidden":        {140, 40, []string{"z"}},
		"filter bar open":    {160, 45, []string{"f"}},
	}
	for name, l := range layouts {
		t.Run(name, func(t *testing.T) {
			a := taskboardApp(t, l.width, l.height, l.keys...)
			x, y, ok := locate(t, a, "1052 ")
			if !ok {
				t.Fatal("task 1052 is not on screen")
			}
			target := ""
			var tx, ty int
			for _, state := range []string{"Done", "In Progress"} {
				if hx, hy, found := locateHeader(t, a, state); found {
					target, tx, ty = state, hx+1, hy
					break
				}
			}
			if target == "" {
				t.Fatal("no other state column is visible")
			}
			a, _ = step(t, a, tea.MouseClickMsg(mouse(x+2, y)))
			a, _ = step(t, a, tea.MouseMotionMsg(mouse(tx, ty)))
			screen := ansi.Strip(a.render())
			if a.modeLabel() != "MOVING" || !strings.Contains(screen, "moving 1052 to "+target) || !strings.Contains(screen, "→ "+target) {
				t.Fatalf("dragging 1052 over %s shows no feedback:\n%s", target, screen)
			}
			_, msgs := step(t, a, tea.MouseReleaseMsg(mouse(tx, ty)))
			move, ok := find[panels.MoveWorkItemMsg](msgs)
			if !ok || move.Item.ID != 1052 || move.NewState != target {
				t.Fatalf("drop produced %v, want 1052 moved to %s", msgs, target)
			}
		})
	}
}

func TestDroppingAnywhereInAColumnUsesThatColumnsState(t *testing.T) {
	a := taskboardApp(t, 160, 45)
	x, y, _ := locate(t, a, "1052 ")
	dx, _, _ := locateHeader(t, a, "Done")
	_, ly, _ := locate(t, a, "1062 ")
	_, msgs := dragTo(t, a, x, y, dx+3, ly)
	if move, ok := find[panels.MoveWorkItemMsg](msgs); !ok || move.NewState != "Done" {
		t.Fatalf("dropping in another lane's Done cell produced %v, want Done", msgs)
	}
}

func TestDropsThatChangeNothingAreIgnored(t *testing.T) {
	a := taskboardApp(t, 160, 45)
	x, y, _ := locate(t, a, "1052 ")
	_, ly, _ := locate(t, a, "1062 ")
	cases := map[string][2]int{
		"same column":       {x + 4, ly},
		"backlog item lane": {3 + 28, y},
		"outside the board": {x, 0},
	}
	for name, to := range cases {
		next, msgs := dragTo(t, a, x+2, y, to[0], to[1])
		if _, ok := find[panels.MoveWorkItemMsg](msgs); ok || next.taskboardPanel.Dragging() {
			t.Errorf("%s: produced %v, want no move and no drag left over", name, msgs)
		}
	}
}

func TestEscapeCancelsADrag(t *testing.T) {
	a := taskboardApp(t, 160, 45)
	x, y, _ := locate(t, a, "1052 ")
	dx, dy, _ := locateHeader(t, a, "Done")
	a, _ = step(t, a, tea.MouseClickMsg(mouse(x+2, y)))
	a, _ = step(t, a, tea.MouseMotionMsg(mouse(dx+1, dy)))
	a, _ = step(t, a, special(tea.KeyEscape))
	if a.taskboardPanel.Dragging() {
		t.Fatal("Esc must cancel the drag")
	}
	if _, msgs := step(t, a, tea.MouseReleaseMsg(mouse(dx+1, dy))); len(msgs) > 0 {
		if _, ok := find[panels.MoveWorkItemMsg](msgs); ok {
			t.Fatal("a cancelled drag must not move the task")
		}
	}
}

func TestClickingSelectsWithoutMoving(t *testing.T) {
	a := taskboardApp(t, 160, 45)
	x, y, _ := locate(t, a, "1062 ")
	a, msgs := dragTo(t, a, x+2, y, x+2, y)
	if _, ok := find[panels.MoveWorkItemMsg](msgs); ok {
		t.Fatal("a click must not move the task")
	}
	if sel := a.selectedItem(); sel == nil || sel.ID != 1062 {
		t.Fatalf("clicking 1062 selected %v", sel)
	}
	px, py, _ := locate(t, a, "1060 ")
	a, _ = step(t, a, tea.MouseClickMsg(mouse(px+2, py)))
	if sel := a.selectedItem(); sel == nil || sel.ID != 1060 || a.taskboardPanel.Dragging() {
		t.Fatal("clicking a backlog item selects it and never starts a drag")
	}
}

func TestDragAndDropOnlyWorksOnTheSprintTaskboard(t *testing.T) {
	for _, hub := range []string{"B", "L"} {
		a := newTestApp(t, 160, 45)
		a, _ = step(t, a, press(hub))
		x, y, ok := locate(t, a, "1043 ")
		if !ok {
			t.Fatalf("%s: 1043 is not on screen", hub)
		}
		_, msgs := dragTo(t, a, x+2, y, x+40, y)
		if _, ok := find[panels.MoveWorkItemMsg](msgs); ok {
			t.Fatalf("%s: dragging must not change state outside Sprints", hub)
		}
	}
}

func TestDragWorksWhenTheBoardIsScrolled(t *testing.T) {
	a := taskboardApp(t, 160, 14)
	if _, _, visible := locate(t, a, "1062 "); visible {
		t.Fatal("the second lane should start below the fold at this height")
	}
	a, _ = step(t, a, press("j"))
	x, y, ok := locate(t, a, "1062 ")
	dx, _, okDone := locate(t, a, "1061 ")
	if !ok || !okDone {
		t.Fatal("scrolling to the second lane should show its To Do and Done tasks")
	}
	if _, _, header := locateHeader(t, a, "Done"); header {
		t.Fatal("the header row should have scrolled away")
	}
	_, msgs := dragTo(t, a, x+2, y, dx+2, y)
	if move, ok := find[panels.MoveWorkItemMsg](msgs); !ok || move.Item.ID != 1062 || move.NewState != "Done" {
		t.Fatalf("dragging in a scrolled board produced %v, want 1062 moved to Done", msgs)
	}
}

func BenchmarkDragMotion(b *testing.B) {
	a := taskboardApp(b, 160, 45, "f")
	x, y, okTask := locate(b, a, "1052 ")
	dx, dy, okDone := locateHeader(b, a, "Done")
	px, py, okProgress := locateHeader(b, a, "In Progress")
	if !okTask || !okDone || !okProgress {
		b.Fatal("the board is missing the task or its target columns")
	}
	m, _ := a.Update(tea.MouseClickMsg(mouse(x+2, y)))
	targets := []tea.Mouse{mouse(dx+1, dy), mouse(px+1, py), mouse(px+3, py)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		m, _ = m.Update(tea.MouseMotionMsg(targets[i%len(targets)]))
		_ = m.View()
	}
}

func TestHoverTrackingOnlyRunsOnTheSprintTaskboard(t *testing.T) {
	a := taskboardApp(t, 160, 45)
	if a.View().MouseMode != tea.MouseModeAllMotion {
		t.Fatal("the Sprints taskboard needs pointer motion for hover")
	}
	for _, k := range []string{"B", "L"} {
		other, _ := step(t, a, press(k))
		if other.View().MouseMode != tea.MouseModeCellMotion {
			t.Fatalf("%s should only report motion while a button is held", k)
		}
	}
	withModal, _ := step(t, a, press("?"))
	if withModal.View().MouseMode != tea.MouseModeCellMotion {
		t.Fatal("hover tracking pauses while a dialog is open")
	}
}

func droppedOnDone(t *testing.T) App {
	t.Helper()
	a := taskboardApp(t, 160, 45)
	x, y, _ := locate(t, a, "1052 ")
	dx, dy, _ := locateHeader(t, a, "Done")
	a, _ = dragTo(t, a, x+2, y, dx+1, dy)
	return a
}

func TestASavedMoveConfirmsOnTheCard(t *testing.T) {
	a := droppedOnDone(t)
	if !strings.Contains(ansi.Strip(a.render()), "Saving to Done") {
		t.Fatal("the dropped card should say it is saving")
	}
	a, _ = step(t, a, workItemMovedMsg{id: 1052, state: "Done"})
	screen := ansi.Strip(a.render())
	if !strings.Contains(screen, "Moved to Done") || !strings.Contains(screen, "State changed to Done") || !a.loading {
		t.Fatalf("a saved move flashes on the card, notifies and reloads:\n%s", screen)
	}
}

func TestAFailedMoveReturnsTheCardWithANotice(t *testing.T) {
	a := droppedOnDone(t)
	a, _ = step(t, a, workItemMoveFailedMsg{id: 1052, state: "Done", err: errors.New("TF401320: rule error")})
	screen := ansi.Strip(a.render())
	if a.modals.open() || !strings.Contains(screen, "Not saved") || !strings.Contains(screen, "Couldn't move #1052 to Done") {
		t.Fatalf("a failed drag returns the card and explains in the top bar:\n%s", screen)
	}
	board, _ := step(t, newTestApp(t, 160, 45), workItemMoveFailedMsg{id: 1043, state: "Done", err: errors.New("boom")})
	if !board.modals.open() {
		t.Fatal("a failed move without a pending card (the Kanban board) still shows the error dialog")
	}
}

func BenchmarkHoverMotion(b *testing.B) {
	a := taskboardApp(b, 160, 45)
	x, y, okTask := locate(b, a, "1052 ")
	ox, oy, okOther := locate(b, a, "1062 ")
	if !okTask || !okOther {
		b.Fatal("the board is missing its tasks")
	}
	var m tea.Model = a
	targets := []tea.Mouse{{X: x + 2, Y: y}, {X: x + 4, Y: y}, {X: ox + 2, Y: oy}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		m, _ = m.Update(tea.MouseMotionMsg(targets[i%len(targets)]))
		_ = m.View()
	}
}
