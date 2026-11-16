# Architecture

lazyado is a single Bubble Tea program that follows The Elm Architecture: one root model (`internal/app.App`), messages in, a new model and commands out, and a pure `View`.

## Packages

| Package                           | Role                                                                                       |
| --------------------------------- | ------------------------------------------------------------------------------------------ |
| `internal/app`                    | Root model, message routing, modal stack, layout reconciliation, commands                 |
| `internal/components/panels`      | Long-lived screen regions: navigation rail, hub page header, backlog table, board, taskboard, details, comments |
| `internal/components/modals`      | Short-lived dialogs implementing `modals.Modal`                                            |
| `internal/components/widgets`     | Reusable building blocks on top of Bubbles: `Picker`, `Dropdown`, `Dialog`, `Overlay`/`OverlayAt`, `Hints`, list/viewport/help factories |
| `internal/components`             | Markdown rendering (`RenderMarkdown`, `RenderHTML`, `CommentMarkdown`) and HTML conversion |
| `internal/keys`                   | The only source of key bindings                                                            |
| `internal/styles`                 | Colors and layout constants                                                                |

Every interactive element is a Bubbles component: `table` (backlog), `list` (board columns, every picker), `viewport` (details, discussion, detail view, taskboard, scrolling dialogs and forms), `textinput`, `textarea`, `spinner`, `paginator`, `progress` and `help`.

## The update cycle

`App.Update` is a thin wrapper:

```go
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := a.update(msg)
	return next, tea.Batch(cmd, next.reconcile())
}
```

1. `update` dispatches by message kind: window size, keys, mouse, spinner ticks, then results (`update_results.go`), and finally anything left over goes to the open modal (paste, cursor blink, quick-search results).
2. `reconcile` derives child state from the root state after every message: panel focus from `activePanel`/`hub`, panel sizes from the terminal size, the details pane and comment loading from the current selection, and whether the shared spinner must tick. Child setters are no-ops when nothing changed, so this is cheap and there is exactly one place that keeps children in sync.
3. `View` only renders. Values that are derived at render time (header breadcrumbs, spinner frame, notification text) are passed as props, for example `panels.RenderHeader(panels.HeaderProps{...})`.

## Commands and stale results

- Commands capture plain values, never pointers into the model (`queryFor(filters, me)` is evaluated in `Update`, not inside the goroutine).
- Iteration, State and Area go to the server (WIQL). Person and keyword filter the loaded items locally with `models.FilterTree`, which keeps the parents of matching children; `applyFilters` only reloads when the server query changed. When the current user cannot be resolved, `@Me` falls back to `@me` in WIQL.
- Every side effect is a command: HTTP calls, saving filters, opening the browser, copying to the clipboard.
- Work item reloads carry a sequence number (`loadSeq`); outdated responses are dropped.
- The signed-in user is the identity behind the PAT, read once from `_apis/connectionData` (`authenticatedUser`, falling back to `authorizedUser`) into `App.me`. `models.Identity.Owns` decides comment ownership by identity ID, then by unique name, and never by display name. `CommentThread` only emits edit/delete intents for owned comments, and `handlePanelIntent` checks again before opening a modal. An unknown identity makes comments read-only; `Ctrl+r` retries it.
- Comment loads carry the work item ID and are debounced (`commentsDueMsg`) while the cursor moves; responses for another item are ignored by both the Discussion tab and the detail view.
- Notification timers carry the notification ID, so an older timer never hides a newer toast.

## Modals

```go
type Modal interface {
	Init() tea.Cmd
	Update(tea.Msg) (Modal, tea.Cmd)
	View() string
}
```

- The app keeps a `modalStack`. The top modal receives every key and every message the app does not handle itself; `WindowSizeMsg` is broadcast to the whole stack. An error raised while a form is open stacks on top of it instead of replacing it.
- Modals never hide themselves. They return an intent message (`StateChangeRequestMsg`, `ParentCreateRequestMsg`, `ModalClosedMsg`, …) and the app pops the stack and runs the command.
- Modals render only their box with `widgets.Dialog`; the app composites it over the current screen with `widgets.Overlay` (Lip Gloss layers). The filter picker is the exception: the app anchors it under its header or filter bar control with `widgets.OverlayAt`, using the positions `panels.RenderPageHeader` reports.
- Constructors receive everything they need (`modals.Env` with keys and terminal size, plus data). There are no `SetVisible`/`SetItem` mutators.

### Adding a modal

- **Pick one option** (state, assign, info): build a `choiceSpec` and call `newChoiceModal`; `choose` maps the picked `widgets.Option` to your intent message. Set `search: true` for type-to-filter.
- **Confirmation**: return a `ConfirmModal` with the intent message to send on `y`.
- **Form**: add a constructor in `work_item_form.go` that lists `formEntry` values (`textField`, `areaField`, `choiceField`), and extend `request()` to build the intent message.
- Open it from `update_keys.go` (or `update_results.go`) with `a.openModal(...)`, then handle the intent in `handleModalIntent`/`mutationFor`.

## Widgets

- `widgets.Picker` wraps `list.Model` with the app keymap, a one-line delegate that highlights fuzzy matches, built-in pagination and an optional always-on filter input. Owners decide what `Enter` means by reading `Selected()`.
- `widgets.Dropdown` is a compact searchable menu for anchored pickers: sections with divider rows, a leading badge, a note and a right-aligned count per row. The filter pickers use it.
- `widgets.Hints(width, bindings...)` renders footers through the shared `help.Model`; use `Relabel` and `Combine` for contextual descriptions.
- `widgets.Reveal(&viewport, top, bottom)` scrolls a line range into view with `viewport.EnsureVisible`.

## Testing

- `internal/app/app_test.go` drives the root model with messages; `drain` runs returned commands with a short timeout so timers (cursor blink, debounce, toasts) are skipped.
- `internal/app/layout_test.go` asserts every screen and every modal fits 50×14 up to 200×50. Set `LAYOUT_DUMP_DIR=/some/dir` to write the rendered frames as `.ans` files for inspection.
