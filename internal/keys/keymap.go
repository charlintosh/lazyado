package keys

import "charm.land/bubbles/v2/key"

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
	View                key.Binding
	Search              key.Binding
	Refresh             key.Binding
	Help                key.Binding
	Back                key.Binding
	DismissNotification key.Binding
	Quit                key.Binding
	ChangeState         key.Binding
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
	QuickSearch         key.Binding

	SortByID    key.Binding
	SortByState key.Binding
	SortByType  key.Binding
	Panel1      key.Binding
	Panel2      key.Binding
	Panel3      key.Binding
	Panel4      key.Binding
	Panel5      key.Binding
	Panel6      key.Binding
	Panel7      key.Binding

	PickerUp   key.Binding
	PickerDown key.Binding

	HubBacklog    key.Binding
	HubBoard      key.Binding
	HubSprint     key.Binding
	PrevView      key.Binding
	NextView      key.Binding
	FilterBar     key.Binding
	ClearFilters  key.Binding
	Fullscreen    key.Binding
	ToggleRows    key.Binding
	DragTask      key.Binding
	DropTask      key.Binding
	MoveItemLeft  key.Binding
	MoveItemRight key.Binding
	HalfPageUp    key.Binding
	HalfPageDown  key.Binding

	HubPullRequests key.Binding
	OpenBrowser     key.Binding
	Vote            key.Binding
	Reply           key.Binding
	LikeComment     key.Binding
	ResolveThread   key.Binding
	NextFile        key.Binding
	PrevFile        key.Binding
	NextChange      key.Binding
	PrevChange      key.Binding
	Repository      key.Binding
	ToggleFavorite  key.Binding
	SelectLines     key.Binding
	LinkedWorkItems key.Binding
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
		PickerUp: key.NewBinding(
			key.WithKeys("up", "ctrl+p"),
			key.WithHelp("↑", "up"),
		),
		PickerDown: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↓", "down"),
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
			key.WithHelp("/", "keyword filter"),
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
		QuickSearch: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("Ctrl+f", "quick search"),
		),
		SortByID: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "sort by ID"),
		),
		Panel1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "iteration filter"),
		),
		SortByType: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "sort by type"),
		),
		Panel2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "person filter"),
		),
		SortByState: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "sort by state"),
		),
		Panel3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "state filter"),
		),
		Panel4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "area filter"),
		),
		Panel5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "main view"),
		),
		Panel6: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "details tab"),
		),
		Panel7: key.NewBinding(
			key.WithKeys("7"),
			key.WithHelp("7", "discussion tab"),
		),
		HubBacklog: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "backlog"),
		),
		HubBoard: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "board"),
		),
		HubSprint: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "sprint taskboard"),
		),
		PrevView: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous view"),
		),
		NextView: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next view"),
		),
		FilterBar: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter bar"),
		),
		ClearFilters: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "clear filters"),
		),
		Fullscreen: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "full screen"),
		),
		ToggleRows: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "collapse / expand all rows"),
		),
		DragTask: key.NewBinding(
			key.WithKeys("drag"),
			key.WithHelp("drag", "move a task (Sprints)"),
		),
		DropTask: key.NewBinding(
			key.WithKeys("release"),
			key.WithHelp("release", "drop on a state column"),
		),
		MoveItemLeft: key.NewBinding(
			key.WithKeys("<"),
			key.WithHelp("<", "move left"),
		),
		MoveItemRight: key.NewBinding(
			key.WithKeys(">"),
			key.WithHelp(">", "move right"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("Ctrl+u", "page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("Ctrl+d", "page down"),
		),
		HubPullRequests: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "pull requests"),
		),
		OpenBrowser: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open in browser"),
		),
		Vote: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "vote"),
		),
		Reply: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reply"),
		),
		LikeComment: key.NewBinding(
			key.WithKeys("+"),
			key.WithHelp("+", "like/unlike"),
		),
		ResolveThread: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "resolve/reopen"),
		),
		NextFile: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("J", "next file"),
		),
		PrevFile: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "previous file"),
		),
		NextChange: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next change"),
		),
		LinkedWorkItems: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "work items"),
		),
		SelectLines: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "select lines"),
		),
		ToggleFavorite: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("Ctrl+t", "favorite"),
		),
		Repository: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "repository"),
		),
		PrevChange: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "previous change"),
		),
	}
}

// FullHelp returns all key bindings for the help panel
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.HubBoard, k.HubBacklog, k.HubSprint, k.HubPullRequests, k.PrevView, k.NextView, k.Panel1, k.Panel2, k.Panel3, k.Panel4, k.Search, k.FilterBar, k.ClearFilters, k.Fullscreen},
		{k.Panel5, k.Panel6, k.Panel7},
		{k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom, k.HalfPageUp, k.HalfPageDown, k.NextPanel, k.PrevPanel},
		{k.View, k.Select, k.ChangeState, k.Assign, k.AddComment, k.CreateParent, k.CreateTask, k.EditTask, k.DeleteTask, k.Info},
		{k.MoveItemLeft, k.MoveItemRight, k.ToggleRows, k.DragTask, k.DropTask, k.EditComment, k.DeleteComment, k.QuickSearch, k.PickerUp, k.PickerDown, k.Refresh, k.Save, k.Confirm, k.Help, k.Back, k.Quit},
		{k.Repository, k.ToggleFavorite, k.Vote, k.AddComment, k.Reply, k.LikeComment, k.ResolveThread, k.SelectLines, k.LinkedWorkItems, k.OpenBrowser, k.NextFile, k.PrevFile, k.NextChange, k.PrevChange},
	}
}
