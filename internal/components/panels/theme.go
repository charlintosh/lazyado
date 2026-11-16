package panels

import (
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

func tableStyles() table.Styles {
	return table.Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorTextMuted).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(styles.ColorBorder),
		Cell: lipgloss.NewStyle().Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorText).
			Background(styles.ColorHighlight),
	}
}

func tableKeyMap(k keys.KeyMap) table.KeyMap {
	return table.KeyMap{
		LineUp:       k.Up,
		LineDown:     k.Down,
		PageUp:       widgets.DisabledBinding(),
		PageDown:     widgets.DisabledBinding(),
		HalfPageUp:   k.HalfPageUp,
		HalfPageDown: k.HalfPageDown,
		GotoTop:      k.Top,
		GotoBottom:   k.Bottom,
	}
}

func newProgress(width int) progress.Model {
	return progress.New(
		progress.WithColors(styles.ColorSuccess),
		progress.WithoutPercentage(),
		progress.WithWidth(max(width, 1)),
	)
}
