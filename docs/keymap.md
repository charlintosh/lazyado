# KeyMap Reference and Usage

Purpose

- Centralize key bindings across components using the `KeyMap` type in [internal/keys](internal/keys).
- Use `DefaultKeyMap()` to create a canonical set of defaults and derive help text for UIs.

Location

- KeyMap type and `DefaultKeyMap()` live in [internal/keys/keymap.go](internal/keys/keymap.go).

Patterns

1. Defining and exposing the KeyMap

- Add any new binding to the `KeyMap` struct.
- Add a sensible default to `DefaultKeyMap()`.

Example:

```go
// internal/keys/keymap.go (excerpt)
type KeyMap struct {
    Save      key.Binding
    Back      key.Binding
    Select    key.Binding
    Up        key.Binding
    Down      key.Binding
    Search    key.Binding
    // Add new bindings here
}

func DefaultKeyMap() KeyMap {
    return KeyMap{
        Save: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save")),
        Back: key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q/esc", "back")),
        Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
        Up: key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
        Down: key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
        Search: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
    }
}
```

2. Using the KeyMap in a component

- Inject `keys.KeyMap` into component constructors.
- Match keys in `Update` using `key.Matches(msg, km.Whatever)`.
- Render help text using `km.Whatever.Help()` or component helpers that derive from the KeyMap.

Example:

```go
func (m MyComponent) Update(msg tea.Msg) (MyComponent, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, m.keys.Back):
            // handle back
        case key.Matches(msg, m.keys.Save):
            // handle save
        }
    }
    return m, nil
}
```

3. Exposing help text

- Build footers and status hints from bindings with `widgets.Hints(width, bindings...)`; it renders through the shared `help.Model`, so key labels always come from the `KeyMap`.
- Use `widgets.Relabel(binding, "desc")` to give a binding a contextual description (for example `Esc` → "cancel") and `widgets.Combine("move", k.PickerUp, k.PickerDown)` to show related keys as one hint.

Example assembly:

```go
footer := widgets.Hints(inner,
    widgets.Combine("move", k.PickerUp, k.PickerDown),
    widgets.Relabel(k.Select, "apply"),
    widgets.Relabel(k.Back, "cancel"),
)
```

4. Adding new key bindings safely

- Add the binding to the `KeyMap` struct.
- Provide a default in `DefaultKeyMap()`.
- Update [README.md](README.md) and the in-app help so users can discover the change.
  - Add new bindings to `KeyMap.FullHelp()` so the in-app help ([internal/components/modals/help_modal.go](internal/components/modals/help_modal.go)) lists them.

5. Testing key handling

- For `Update` logic, create unit tests that send `tea.KeyMsg{String(): "s"}` or `tea.KeyMsg{Type: tea.KeyEnter}` and assert state transitions.
- Use tea test harnesses (e.g., `teatest`) for integration tests where appropriate.

References

- See the `KeyMap` implementation: [internal/keys/keymap.go](internal/keys/keymap.go)
- Examples of `KeyMap` usage: [internal/components/modals/choice_modal.go](internal/components/modals/choice_modal.go) and [internal/components/modals/comment_modal.go](internal/components/modals/comment_modal.go)

Notes

- All keyboard input handling MUST use the centralized `KeyMap` (do not match raw strings in Update handlers).

- When bindings change, update [README.md](README.md) and `KeyMap.FullHelp()` (rendered by [internal/components/modals/help_modal.go](internal/components/modals/help_modal.go)) to keep docs and UI in sync.
