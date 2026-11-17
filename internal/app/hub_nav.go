package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/panels"
)

func (a *App) hubFor(msg tea.KeyPressMsg) (panels.Hub, bool) {
	k := a.keys
	hubs := []struct {
		binding key.Binding
		hub     panels.Hub
	}{
		{k.HubBacklog, panels.HubBacklog},
		{k.HubBoard, panels.HubBoard},
		{k.HubSprint, panels.HubSprint},
		{k.HubFiles, panels.HubFiles},
		{k.HubCommits, panels.HubCommits},
		{k.HubBranches, panels.HubBranches},
		{k.HubTags, panels.HubTags},
		{k.HubPullRequests, panels.HubPullRequests},
		{k.HubBuilds, panels.HubBuilds},
		{k.HubReleases, panels.HubReleases},
	}
	for _, h := range hubs {
		if key.Matches(msg, h.binding) {
			return h.hub, true
		}
	}
	return 0, false
}

func (a *App) goToHub(hub panels.Hub) tea.Cmd {
	switch hub {
	case panels.HubBacklog, panels.HubBoard, panels.HubSprint:
		a.switchHub(hub)
		return nil
	}
	if hub == a.hub {
		return nil
	}
	switch hub {
	case panels.HubFiles:
		return a.openFiles()
	case panels.HubCommits:
		return a.openCommits()
	case panels.HubBranches, panels.HubTags:
		return a.openCodeHub(hub)
	case panels.HubPullRequests:
		return a.openPullRequests()
	case panels.HubBuilds:
		return a.openPipelines()
	case panels.HubReleases:
		return a.openReleases()
	}
	return nil
}
