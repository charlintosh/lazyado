package panels

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
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

	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorText).
		MarginBottom(1).
		Render("Keyboard Shortcuts")

	b.WriteString(title)
	b.WriteString("\n\n")

	// Group shortcuts by category
	sections := []struct {
		title      string
		bindings   []key.Binding
		customKeys []struct {
			keys string
			help string
		}
	}{
		{
			title: "Navigation",
			bindings: []key.Binding{
				h.keys.Up,
				h.keys.Down,
				h.keys.Left,
				h.keys.Right,
				h.keys.Top,
				h.keys.Bottom,
				h.keys.NextPanel,
				h.keys.PrevPanel,
				h.keys.Panel1,
				h.keys.Panel2,
				h.keys.Panel3,
				h.keys.Panel4,
				h.keys.Panel5,
				h.keys.Panel6,
			},
		},
		{
			title: "Work Item Actions",
			bindings: []key.Binding{
				h.keys.Open,
				h.keys.View,
				h.keys.ChangeState,
				h.keys.Assign,
				h.keys.AddComment,
				h.keys.CreateParent,
				h.keys.CreateTask,
				h.keys.EditTask,
				h.keys.DeleteTask,
				h.keys.CreateBranch,
			},
		},
		{
			title: "Comment Actions",
			bindings: []key.Binding{
				h.keys.EditComment,
				h.keys.DeleteComment,
			},
		},
		{
			title: "Modals",
			bindings: []key.Binding{
				h.keys.Confirm,
				h.keys.Save,
			},
		},
		{
			title: "General",
			bindings: []key.Binding{
				h.keys.Select,
				h.keys.Search,
				h.keys.Refresh,
				h.keys.Help,
				h.keys.Back,
				h.keys.Quit,
			},
		},
	}

	for i, section := range sections {
		// Section title
		sectionTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorPrimary).
			Render(section.title)
		b.WriteString(sectionTitle)
		b.WriteString("\n")

		// Key bindings
		for _, binding := range section.bindings {
			keyStyle := h.styles.HelpKey.Width(12)
			descStyle := h.styles.HelpDesc

			help := binding.Help()
			line := keyStyle.Render(help.Key) + descStyle.Render(help.Desc)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Additional custom keys were removed in favor of centralized KeyMap bindings.

		// Add spacing between sections
		if i < len(sections)-1 {
			b.WriteString("\n")
		}
	}

	// Footer: render using KeyMap help labels for consistency
	b.WriteString("\n")
	helpKey := h.keys.Help.Help().Key
	backKey := h.keys.Back.Help().Key
	footer := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Italic(true).
		Render(fmt.Sprintf("Press %s or %s to close", helpKey, backKey))
	b.WriteString(footer)

	content := b.String()

	// Center the help panel
	helpWidth := 40
	helpHeight := 25

	panel := h.styles.HelpPanel.
		Width(helpWidth).
		Height(helpHeight).
		Render(content)

	// Create overlay positioning
	return lipgloss.Place(
		h.width,
		h.height,
		lipgloss.Center,
		lipgloss.Center,
		panel,
	)
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
