package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
)

const throttleNotice = "Azure DevOps is slowing down requests for your account. lazyado is pacing itself; screens may take longer to load."

func (a *App) syncThrottle() tea.Cmd {
	t := a.client.Throttle()
	if t.Episode == a.throttleSeen {
		return nil
	}
	a.throttleSeen = t.Episode
	return a.info(throttleText(t))
}

func throttleText(t api.Throttle) string {
	if t.Resource == "" {
		return throttleNotice
	}
	return throttleNotice + " (" + t.Resource + ")"
}
