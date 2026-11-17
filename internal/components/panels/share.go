package panels

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/share"
)

type ShareMsg struct {
	Target share.Target
	Now    bool
}

type ShareReleaseMsg struct {
	ReleaseID  int
	Name       string
	Definition string
	Now        bool
}

func ShareRequested(k keys.KeyMap, msg tea.KeyPressMsg) (now, ok bool) {
	switch {
	case key.Matches(msg, k.ShareLink):
		return true, true
	case key.Matches(msg, k.Share):
		return false, true
	}
	return false, false
}

func shareCmd(t share.Target, now bool) tea.Cmd {
	return func() tea.Msg { return ShareMsg{Target: t, Now: now} }
}
