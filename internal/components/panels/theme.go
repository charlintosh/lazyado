package panels

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

func disabledBinding() key.Binding {
	return key.NewBinding(key.WithDisabled())
}

func NewHelp() help.Model {
	h := help.New()
	h.ShortSeparator = "   "
	h.FullSeparator = "    "
	h.Ellipsis = "..."
	h.Styles = help.Styles{
		Ellipsis:       lipgloss.NewStyle().Foreground(styles.ColorTextMuted),
		ShortKey:       lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true),
		ShortDesc:      lipgloss.NewStyle().Foreground(styles.ColorLight),
		ShortSeparator: lipgloss.NewStyle().Foreground(styles.ColorTextMuted),
		FullKey:        lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true),
		FullDesc:       lipgloss.NewStyle().Foreground(styles.ColorLight),
		FullSeparator:  lipgloss.NewStyle().Foreground(styles.ColorTextMuted),
	}
	return h
}

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
		PageUp:       disabledBinding(),
		PageDown:     disabledBinding(),
		HalfPageUp:   k.HalfPageUp,
		HalfPageDown: k.HalfPageDown,
		GotoTop:      k.Top,
		GotoBottom:   k.Bottom,
	}
}

func listKeyMap(k keys.KeyMap) list.KeyMap {
	return list.KeyMap{
		CursorUp:             k.Up,
		CursorDown:           k.Down,
		PrevPage:             k.HalfPageUp,
		NextPage:             k.HalfPageDown,
		GoToStart:            k.Top,
		GoToEnd:              k.Bottom,
		Filter:               disabledBinding(),
		ClearFilter:          disabledBinding(),
		CancelWhileFiltering: disabledBinding(),
		AcceptWhileFiltering: disabledBinding(),
		ShowFullHelp:         disabledBinding(),
		CloseFullHelp:        disabledBinding(),
		Quit:                 disabledBinding(),
		ForceQuit:            disabledBinding(),
	}
}

func listStyles(titleColor color.Color) list.Styles {
	s := list.DefaultStyles(true)
	s.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	s.Title = lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	s.NoItems = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	s.PaginationStyle = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	s.ArabicPagination = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	return s
}

func newComponentList(items []list.Item, delegate list.ItemDelegate, k keys.KeyMap, title string) list.Model {
	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.KeyMap = listKeyMap(k)
	l.Styles = listStyles(styles.ColorText)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.Paginator.Type = paginator.Arabic
	l.DisableQuitKeybindings()
	return l
}

func newProgress(width int) progress.Model {
	return progress.New(
		progress.WithColors(styles.ColorSuccess),
		progress.WithoutPercentage(),
		progress.WithWidth(max(width, 1)),
	)
}
