package notification

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
)

const notificationDuration = 8 * time.Second

type NotificationType int

const (
	NotificationSuccess NotificationType = iota
	NotificationError
	NotificationInfo
)

var notificationLabels = map[NotificationType]string{
	NotificationSuccess: "Done",
	NotificationError:   "Error",
	NotificationInfo:    "Info",
}

var notificationColors = map[NotificationType]color.Color{
	NotificationSuccess: styles.ColorSuccess,
	NotificationError:   styles.ColorError,
	NotificationInfo:    styles.ColorPrimary,
}

type AutoDismissMsg struct {
	ID int
}

type Notification struct {
	id        int
	visible   bool
	message   string
	notifType NotificationType
}

func (n Notification) Show(t NotificationType, message string) (Notification, tea.Cmd) {
	n.id++
	n.visible = true
	n.notifType = t
	n.message = notificationLabels[t] + ": " + message
	id := n.id
	return n, tea.Tick(notificationDuration, func(time.Time) tea.Msg { return AutoDismissMsg{ID: id} })
}

func (n Notification) Update(msg tea.Msg) Notification {
	if m, ok := msg.(AutoDismissMsg); ok && m.ID == n.id {
		return n.Dismiss()
	}
	return n
}

func (n Notification) Dismiss() Notification {
	n.visible = false
	n.message = ""
	return n
}

func (n Notification) IsVisible() bool {
	return n.visible
}

func (n Notification) View() string {
	if !n.visible {
		return ""
	}
	return lipgloss.NewStyle().
		Background(notificationColors[n.notifType]).
		Foreground(styles.ColorOnAccent).
		Bold(true).
		Padding(0, 1).
		Render(n.message)
}
