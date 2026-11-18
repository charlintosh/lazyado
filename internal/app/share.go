package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
)

const nothingToShare = "Nothing to share here"

type clipboardFallbackMsg struct {
	text   string
	notice string
}

func (a *App) share(t share.Target, now bool) tea.Cmd {
	o, ok := t.Default()
	switch {
	case !ok:
		return a.notify(nothingToShare)
	case now:
		return copyCmd(o.Text, t.Notice(o))
	}
	return a.openModal(modals.NewShareModal(a.env(), t))
}

func (a *App) isShareKey(msg tea.KeyPressMsg) bool {
	_, ok := panels.ShareRequested(a.keys, msg)
	return ok
}

func (a *App) shareNow(msg tea.KeyPressMsg) bool {
	now, _ := panels.ShareRequested(a.keys, msg)
	return now
}

func (a App) handleShareIntent(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case panels.ShareMsg:
		return a, a.share(msg.Target, msg.Now), true
	case panels.CopyTextMsg:
		return a, copyCmd(msg.Text, msg.Notice), true
	case panels.ShareReleaseMsg:
		r := models.Release{ID: msg.ReleaseID, Name: msg.Name, Definition: msg.Definition, WebURL: a.client.ReleaseWebURL(msg.ReleaseID)}
		return a, a.share(share.Release(r), msg.Now), true
	case modals.OpenURLRequestMsg:
		a.closeModal()
		return a, openBrowserCmd(msg.URL), true
	case clipboardFallbackMsg:
		return a, tea.Batch(tea.SetClipboard(msg.text), a.openModal(modals.NewClipboardModal(a.env(), msg.text, msg.notice))), true
	}
	return a, nil, false
}
