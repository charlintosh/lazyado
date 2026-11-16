package panels

import (
	"maps"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/models"
)

const moveFlashDuration = 1500 * time.Millisecond

type moveFlash int

const (
	flashNone moveFlash = iota
	flashSaving
	flashSaved
	flashFailed
)

type taskMove struct {
	to     string
	from   string
	target string
	flash  moveFlash
	seq    int
}

type MoveFlashExpiredMsg struct {
	ID  int
	Seq int
}

func (t *TaskboardPanel) setMove(id int, m taskMove) {
	next := maps.Clone(t.moves)
	if next == nil {
		next = map[int]taskMove{}
	}
	next[id] = m
	t.moves = next
}

func (t *TaskboardPanel) forgetMove(id int) {
	next := maps.Clone(t.moves)
	delete(next, id)
	t.moves = next
}

func (t *TaskboardPanel) beginMove(task *models.WorkItem, to string) tea.Cmd {
	if task == nil {
		return nil
	}
	t.setMove(task.ID, taskMove{to: to, from: t.displayState(task), target: to, flash: flashSaving})
	t.hover = 0
	t.rebuild()
	moved := *task
	return func() tea.Msg { return MoveWorkItemMsg{Item: moved, NewState: to} }
}

func (t TaskboardPanel) HasPendingMove(id int) bool {
	m, ok := t.moves[id]
	return ok && m.flash == flashSaving
}

func (t *TaskboardPanel) MoveSaved(id int) tea.Cmd {
	return t.settle(id, flashSaved, false)
}

func (t *TaskboardPanel) MoveFailed(id int) tea.Cmd {
	return t.settle(id, flashFailed, true)
}

func (t *TaskboardPanel) settle(id int, flash moveFlash, revert bool) tea.Cmd {
	m, ok := t.moves[id]
	if !ok {
		return nil
	}
	t.flashSeq++
	m.flash, m.seq = flash, t.flashSeq
	if revert {
		m.to = ""
	}
	t.setMove(id, m)
	t.rebuild()
	seq := m.seq
	return tea.Tick(moveFlashDuration, func(time.Time) tea.Msg { return MoveFlashExpiredMsg{ID: id, Seq: seq} })
}

func (t *TaskboardPanel) ExpireFlash(msg MoveFlashExpiredMsg) {
	m, ok := t.moves[msg.ID]
	if !ok || m.seq != msg.Seq || m.flash == flashSaving {
		return
	}
	if task, found := findTask(t.items, msg.ID); m.to == "" || !found || string(task.State) == m.to {
		t.forgetMove(msg.ID)
	} else {
		m.flash = flashNone
		t.setMove(msg.ID, m)
	}
	t.render()
}

func (t *TaskboardPanel) settleMoves() {
	if len(t.moves) == 0 {
		return
	}
	next := make(map[int]taskMove, len(t.moves))
	for id, m := range t.moves {
		task, ok := findTask(t.items, id)
		if !ok || (m.flash == flashNone && string(task.State) == m.to) {
			continue
		}
		next[id] = m
	}
	t.moves = next
}

func findTask(items []models.WorkItem, id int) (*models.WorkItem, bool) {
	for i := range items {
		if items[i].ID == id {
			return &items[i], true
		}
	}
	return nil, false
}
