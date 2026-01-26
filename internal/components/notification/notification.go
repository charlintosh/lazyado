package notification

import (
	"fmt"
	"time"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	notificationDuration = 8 * time.Second
)

// NotificationType represents the type of notification
type NotificationType int

const (
	NotificationSuccess NotificationType = iota
	NotificationError
	NotificationInfo
)

// notificationIcons maps notification types to their display icons
var notificationIcons = map[NotificationType]string{
	NotificationSuccess: "✓",
	NotificationError:   "✗",
	NotificationInfo:    "ℹ",
}

// Notification displays auto-dismissable messages in the top-right corner
type Notification struct {
	visible   bool
	message   string
	notifType NotificationType
	styles    styles.Styles
	keys      keys.KeyMap
	width     int
	height    int
}

// NewNotification creates a new notification
func NewNotification(st styles.Styles, k keys.KeyMap) Notification {
	return Notification{
		styles: st,
		keys:   k,
	}
}

// AutoDismissMsg is sent after the notification duration
type AutoDismissMsg struct{}

func autoDismissCmd() tea.Cmd {
	return tea.Tick(notificationDuration, func(t time.Time) tea.Msg {
		return AutoDismissMsg{}
	})
}

// Update handles messages
func (n Notification) Update(msg tea.Msg) (Notification, tea.Cmd) {
	if !n.visible {
		return n, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Dismiss with ESC, Back key, or comma
		if key.Matches(msg, n.keys.Back) || key.Matches(msg, n.keys.DismissNotification) {
			n.visible = false
			return n, nil
		}
	case AutoDismissMsg:
		n.visible = false
		return n, nil
	}

	return n, nil
}

// View renders the notification
func (n Notification) View() string {
	if !n.visible || n.message == "" {
		return ""
	}

	// Choose background color based on notification type
	var bgColor lipgloss.Color
	switch n.notifType {
	case NotificationSuccess:
		bgColor = styles.ColorSuccess
	case NotificationError:
		bgColor = styles.ColorError
	case NotificationInfo:
		bgColor = styles.ColorPrimary
	default:
		bgColor = styles.ColorSuccess
	}

	// Simple text with background color, no borders
	notifStyle := lipgloss.NewStyle().
		Background(bgColor).
		Foreground(styles.ColorText).
		Bold(true).
		Padding(0, 2)

	return notifStyle.Render(n.message)
}

// Show displays a notification message with the specified type
// The type determines the icon prefix (✓ for success, ✗ for error, ℹ for info)
func (n *Notification) Show(notifType NotificationType, message string) tea.Cmd {
	icon := notificationIcons[notifType]
	n.visible = true
	n.notifType = notifType
	n.message = fmt.Sprintf("%s %s", icon, message)
	return autoDismissCmd()
}

// SetSize updates the notification dimensions
func (n *Notification) SetSize(width, height int) {
	n.width = width
	n.height = height
}

// IsVisible returns whether the notification is visible
func (n Notification) IsVisible() bool {
	return n.visible
}

// Hide hides the notification
func (n *Notification) Hide() {
	n.visible = false
	n.message = ""
}
