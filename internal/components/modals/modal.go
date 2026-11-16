package modals

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
)

type Modal interface {
	Init() tea.Cmd
	Update(tea.Msg) (Modal, tea.Cmd)
	View() string
}

type Env struct {
	Keys   keys.KeyMap
	Width  int
	Height int
}

func (e Env) resized(msg tea.WindowSizeMsg) Env {
	e.Width, e.Height = msg.Width, msg.Height
	return e
}

type ModalClosedMsg struct{}

func closeModal() tea.Msg { return ModalClosedMsg{} }

func send(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func itemLabel(item models.WorkItem) string {
	return fmt.Sprintf("#%d %s", item.ID, item.Title)
}
