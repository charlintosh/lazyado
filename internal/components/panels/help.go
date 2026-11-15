package panels

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

// HelpPanel displays keyboard shortcuts
type HelpPanel struct {
	keys    keys.KeyMap
	styles  styles.Styles
	visible bool
	width   int
	height  int
}

// NewHelpPanel creates a new help panel
func NewHelpPanel(keys keys.KeyMap, styles styles.Styles) HelpPanel {
	return HelpPanel{
		keys:   keys,
		styles: styles,
	}
}

// View renders the help panel
func (h HelpPanel) View() string {
	if !h.visible {
		return ""
	}
	hm := NewHelp()
	hm.ShowAll = true
	body := hm.FullHelpView(h.keys.FullHelp())
	footer := Muted(fmt.Sprintf("Press %s or %s to close", h.keys.Help.Help().Key, h.keys.Back.Help().Key))
	body = lipgloss.JoinVertical(lipgloss.Left, body, "", footer)
	width := min(lipgloss.Width(body)+styles.BoxPaddedSize, h.width)
	height := min(lipgloss.Height(body)+styles.PaneChromeHeight, h.height)
	panel := Pane(PaneTitle("", "Keyboard shortcuts", true), body, width, height, true)
	return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center, panel)
}

func (h *HelpPanel) SetVisible(visible bool) {
	h.visible = visible
}

// IsVisible returns whether the help panel is visible
func (h *HelpPanel) IsVisible() bool {
	return h.visible
}

// Toggle toggles the visibility of the help panel
func (h *HelpPanel) Toggle() {
	h.visible = !h.visible
}

// SetSize sets the size of the help panel
func (h *HelpPanel) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// ShortHelp returns a short help string for the status bar
func ShortHelp(keys keys.KeyMap, styles styles.Styles) string {
	bindings := keys.ShortHelp()
	var parts []string

	for _, b := range bindings {
		help := b.Help()
		key := styles.HelpKey.Render(help.Key)
		desc := styles.HelpDesc.Render(help.Desc)
		parts = append(parts, key+" "+desc)
	}

	return strings.Join(parts, "  ")
}
