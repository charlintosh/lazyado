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

- Use `binding := m.keys.Save.Help()` to get `{Key, Desc}` for assembling the instruction line shown in modals.
- Prefer deriving help from the `KeyMap` rather than hardcoding key names inside views.

Example assembly:

```go
save := m.keys.Save.Help()
back := m.keys.Back.Help()
instr := fmt.Sprintf("%s: %s  %s: %s", save.Key, save.Desc, back.Key, back.Desc)
view.WriteString(m.styles.ModalInstructions.Render(instr))
```

4. Adding new key bindings safely

- Add the binding to the `KeyMap` struct.
- Provide a default in `DefaultKeyMap()`.
- Update [README.md](README.md) and the in-app help so users can discover the change.
  - Update [internal/components/panels/help.go](internal/components/panels/help.go) (the in-app help renderer) to include new bindings.

5. Testing key handling

- For `Update` logic, create unit tests that send `tea.KeyMsg{String(): "s"}` or `tea.KeyMsg{Type: tea.KeyEnter}` and assert state transitions.
- Use tea test harnesses (e.g., `teatest`) for integration tests where appropriate.

References

- See the `KeyMap` implementation: [internal/keys/keymap.go](internal/keys/keymap.go)
- Examples of `KeyMap` usage: [internal/components/modals/assign_modal.go](internal/components/modals/assign_modal.go) and [internal/components/modals/comment_modal.go](internal/components/modals/comment_modal.go)

Notes

- All keyboard input handling MUST use the centralized `KeyMap` (do not match raw strings in Update handlers).

- When bindings change, update [README.md](README.md) and [internal/components/panels/help.go](internal/components/panels/help.go) to keep docs and UI in sync.
