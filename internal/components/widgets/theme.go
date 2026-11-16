package widgets

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

func DisabledBinding() key.Binding {
	return key.NewBinding(key.WithDisabled())
}

func Relabel(b key.Binding, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(b.Keys()...), key.WithHelp(b.Help().Key, desc))
}

func Combine(desc string, bindings ...key.Binding) key.Binding {
	var ks, labels []string
	for _, b := range bindings {
		ks = append(ks, b.Keys()...)
		labels = append(labels, b.Help().Key)
	}
	return key.NewBinding(key.WithKeys(ks...), key.WithHelp(strings.Join(labels, "/"), desc))
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

func ListKeyMap(k keys.KeyMap) list.KeyMap {
	return list.KeyMap{
		CursorUp:             k.Up,
		CursorDown:           k.Down,
		PrevPage:             k.HalfPageUp,
		NextPage:             k.HalfPageDown,
		GoToStart:            k.Top,
		GoToEnd:              k.Bottom,
		Filter:               DisabledBinding(),
		ClearFilter:          DisabledBinding(),
		CancelWhileFiltering: DisabledBinding(),
		AcceptWhileFiltering: DisabledBinding(),
		ShowFullHelp:         DisabledBinding(),
		CloseFullHelp:        DisabledBinding(),
		Quit:                 DisabledBinding(),
		ForceQuit:            DisabledBinding(),
	}
}

func ListStyles(titleColor color.Color) list.Styles {
	s := list.DefaultStyles(true)
	s.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	s.Title = lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	s.NoItems = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	s.PaginationStyle = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	s.ArabicPagination = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	return s
}

func NewList(items []list.Item, delegate list.ItemDelegate, k keys.KeyMap, title string) list.Model {
	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.KeyMap = ListKeyMap(k)
	l.Styles = ListStyles(styles.ColorText)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.Paginator.Type = paginator.Arabic
	l.DisableQuitKeybindings()
	return l
}

func ViewportKeyMap(k keys.KeyMap) viewport.KeyMap {
	return viewport.KeyMap{
		PageDown:     DisabledBinding(),
		PageUp:       DisabledBinding(),
		HalfPageUp:   k.HalfPageUp,
		HalfPageDown: k.HalfPageDown,
		Up:           k.Up,
		Down:         k.Down,
		Left:         DisabledBinding(),
		Right:        DisabledBinding(),
	}
}

func NewViewport(k keys.KeyMap) viewport.Model {
	vp := viewport.New()
	vp.KeyMap = ViewportKeyMap(k)
	return vp
}

func Reveal(vp *viewport.Model, top, bottom int) {
	vp.EnsureVisible(bottom, 0, 0)
	vp.EnsureVisible(top, 0, 0)
}

func Hints(width int, bindings ...key.Binding) string {
	h := NewHelp()
	h.ShortSeparator = "  "
	h.SetWidth(width)
	return h.ShortHelpView(bindings)
}
