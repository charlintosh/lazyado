package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard shortcuts
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Top       key.Binding
	Bottom    key.Binding
	NextPanel key.Binding
	PrevPanel key.Binding

	Select              key.Binding
	Open                key.Binding
	View                key.Binding
	Search              key.Binding
	Refresh             key.Binding
	Help                key.Binding
	Back                key.Binding
	DismissNotification key.Binding
	Quit                key.Binding
	ChangeState         key.Binding
	CreateBranch        key.Binding
	Assign              key.Binding
	CreateTask          key.Binding
	CreateParent        key.Binding
	EditTask            key.Binding
	DeleteTask          key.Binding
	AddComment          key.Binding
	EditComment         key.Binding
	DeleteComment       key.Binding
	Info                key.Binding
	Save                key.Binding
	Confirm             key.Binding

	SortByID    key.Binding
	SortByState key.Binding
	SortByType  key.Binding
	Panel1      key.Binding
	Panel2      key.Binding
	Panel3      key.Binding
	Panel4      key.Binding
	Panel5      key.Binding
	Panel6      key.Binding

	ModalLeft  key.Binding
	ModalRight key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		ModalLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "change"),
		),
		ModalRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "change"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		NextPanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next"),
		),
		PrevPanel: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", "prev"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "select"),
		),
		View: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "view details"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("Ctrl+r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "back"),
		),
		DismissNotification: key.NewBinding(
			key.WithKeys(","),
			key.WithHelp(",", "dismiss notification"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		ChangeState: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "change state"),
		),
		CreateBranch: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "create branch"),
		),
		Assign: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "assign"),
		),
		CreateTask: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "new child task"),
		),
		CreateParent: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new parent item"),
		),
		EditTask: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		DeleteTask: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		AddComment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "add comment"),
		),
		EditComment: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit comment"),
		),
		DeleteComment: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete comment"),
		),
		Info: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "info"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("Ctrl+s", "save"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "confirm"),
		),
		SortByID: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "sort by ID"),
		),
		Panel1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "panel 1"),
		),
		SortByType: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "sort by type"),
		),
		Panel2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "panel 2"),
		),
		SortByState: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "sort by state"),
		),
		Panel3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "panel 3"),
		),
		Panel4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "panel 4"),
		),
		Panel5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "panel 5"),
		),
		Panel6: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "panel 6"),
		),
	}
}

// ShortHelp returns a short help text for the status bar
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.NextPanel,
		k.Up, k.Down,
		k.Top, k.Bottom,
		k.Open,
		k.Help,
	}
}

// FullHelp returns all key bindings for the help panel
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.NextPanel, k.PrevPanel},
		{k.Select, k.Open, k.View},
		{k.ChangeState, k.CreateBranch, k.Assign},
		{k.CreateTask, k.EditTask, k.DeleteTask},
		{k.AddComment, k.SortByID, k.SortByType, k.SortByState},
		{k.Search, k.Refresh},
		{k.Help, k.Back, k.Quit},
	}
}
