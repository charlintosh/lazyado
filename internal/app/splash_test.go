package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/models"
)

func newLoadingApp(t *testing.T) App {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	client := api.NewClient(&config.Config{Organization: "acme", Project: "storefront", Team: "Storefront Team"})
	var m tea.Model = NewApp(client, models.DiffLimits{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 34})
	return m.(App)
}

func TestAnyKeySkipsTheSplash(t *testing.T) {
	a := newLoadingApp(t)
	if !a.showSplash() {
		t.Fatal("the splash is not shown while the first load runs")
	}
	next, cmd := a.handleKey(press("j"))
	if next.showSplash() {
		t.Error("a key press left the splash on screen")
	}
	if cmd != nil {
		t.Error("the key that skipped the splash also did something else")
	}
}

func TestQuitStillQuitsDuringTheSplash(t *testing.T) {
	a := newLoadingApp(t)
	_, cmd := a.handleKey(press("q"))
	if cmd == nil {
		t.Fatal("q during the splash did nothing")
	}
	if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Error("q during the splash did not quit")
	}
}
