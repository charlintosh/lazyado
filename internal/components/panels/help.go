package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
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
	type section struct {
		title    string
		bindings []key.Binding
	}
	left := []section{
		{"Boards hubs", []key.Binding{h.keys.HubBacklog, h.keys.HubBoard, h.keys.HubSprint}},
		{"Navigation", []key.Binding{
			h.keys.Up, h.keys.Down, h.keys.Left, h.keys.Right, h.keys.Top, h.keys.Bottom,
			h.keys.HalfPageUp, h.keys.HalfPageDown, h.keys.NextPanel, h.keys.PrevPanel,
		}},
		{"Panels", []key.Binding{
			h.keys.Panel1, h.keys.Panel2, h.keys.Panel3, h.keys.Panel4,
			h.keys.Panel5, h.keys.Panel6, h.keys.Panel7,
		}},
	}
	right := []section{
		{"Work items", []key.Binding{
			h.keys.View, h.keys.Open, h.keys.ChangeState, h.keys.Assign, h.keys.AddComment,
			h.keys.CreateParent, h.keys.CreateTask, h.keys.EditTask, h.keys.DeleteTask,
			h.keys.CreateBranch, h.keys.Info,
		}},
		{"Board & taskboard", []key.Binding{h.keys.MoveItemLeft, h.keys.MoveItemRight}},
		{"Discussion", []key.Binding{h.keys.EditComment, h.keys.DeleteComment}},
		{"General", []key.Binding{
			h.keys.Search, h.keys.QuickSearch, h.keys.Refresh, h.keys.Save, h.keys.Confirm,
			h.keys.Help, h.keys.Back, h.keys.DismissNotification, h.keys.Quit,
		}},
	}
	render := func(secs []section) string {
		var b strings.Builder
		for i, sec := range secs {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(sec.title))
			b.WriteString("\n")
			for _, binding := range sec.bindings {
				hb := binding.Help()
				b.WriteString(h.styles.HelpKey.Width(styles.HelpKeyColumnWidth).Render(hb.Key))
				b.WriteString(h.styles.HelpDesc.Render(hb.Desc))
				b.WriteString("\n")
			}
		}
		return strings.TrimRight(b.String(), "\n")
	}
	colWidth := styles.HelpKeyColumnWidth + 26
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(colWidth).Render(render(left)),
		"  ",
		lipgloss.NewStyle().Width(colWidth).Render(render(right)),
	)
	footer := lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).
		Render(fmt.Sprintf("Press %s or %s to close", h.keys.Help.Help().Key, h.keys.Back.Help().Key))
	body += "\n\n" + footer

	width := min(lipgloss.Width(body)+styles.BoxPaddedSize, h.width)
	height := min(lipgloss.Height(body)+styles.BoxBorderSize, h.height)
	panel := RenderTitledBox(BoxTitle("", "Keyboard shortcuts", true), body, width, height, true)
	return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center, panel)
}

// SetVisible sets whether the help panel is visible
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
