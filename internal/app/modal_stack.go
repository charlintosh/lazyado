package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
)

type modalStack []modals.Modal

func (s modalStack) open() bool {
	return len(s) > 0
}

func (s modalStack) top() (modals.Modal, bool) {
	if len(s) == 0 {
		return nil, false
	}
	return s[len(s)-1], true
}

func topIs[T modals.Modal](s modalStack) bool {
	m, ok := s.top()
	if !ok {
		return false
	}
	_, ok = m.(T)
	return ok
}

func (s modalStack) push(m modals.Modal) modalStack {
	return append(slices.Clip(s), m)
}

func (s modalStack) pop() modalStack {
	if len(s) == 0 {
		return s
	}
	return slices.Clip(s[:len(s)-1])
}

func (s modalStack) update(msg tea.Msg) (modalStack, tea.Cmd) {
	m, ok := s.top()
	if !ok {
		return s, nil
	}
	next, cmd := m.Update(msg)
	out := slices.Clone(s)
	out[len(out)-1] = next
	return out, cmd
}

func (s modalStack) broadcast(msg tea.Msg) (modalStack, tea.Cmd) {
	out := make(modalStack, len(s))
	cmds := make([]tea.Cmd, len(s))
	for i, m := range s {
		out[i], cmds[i] = m.Update(msg)
	}
	return out, tea.Batch(cmds...)
}
