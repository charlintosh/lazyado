# Architecture

lazyado is a single Bubble Tea program that follows The Elm Architecture: one root model (`internal/app.App`), messages in, a new model and commands out, and a pure `View`.

## Packages

| Package                           | Role                                                                                       |
| --------------------------------- | ------------------------------------------------------------------------------------------ |
| `internal/app`                    | Root model, message routing, modal stack, layout reconciliation, commands                 |
| `internal/components/panels`      | Long-lived screen regions: navigation rail, hub page header, backlog table, board, taskboard, details, comments, pull request and pipeline lists and views |
| `internal/components/modals`      | Short-lived dialogs implementing `modals.Modal`                                            |
| `internal/components/widgets`     | Reusable building blocks on top of Bubbles: `Picker`, `Dropdown`, `Dialog`, `Overlay`/`OverlayAt`, `Hints`, list/viewport/help factories |
| `internal/components`             | Markdown rendering (`RenderMarkdown`, `RenderHTML`, `CommentMarkdown`) and HTML conversion |
| `internal/keys`                   | The only source of key bindings                                                            |
| `internal/cache`                  | `LRU`: a size- or count-bounded least-recently-used cache with entry ages, used for stale-while-revalidate data and render caches |
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

## Keeping Update and View cheap

Bubble Tea calls `View` after every `Update`, and some messages arrive in bursts (mouse motion while dragging, cursor blinks, spinner ticks), so both run on every one of them.

- `Update` and `View` keep value receivers: Bubble Tea's contract is unchanged, `Update` returns a new model and never mutates the old one. Every other method on `App` takes `*App`, so helpers never copy the model. Methods that only read state stay read-only.
- Immutable configuration is shared, not copied: every panel holds the one `*styles.Styles` from `styles.DefaultStyles()`. The model was 227 KB when each panel carried its own copy, and copies of more than 64 KB go to the heap.
- Derived layout is computed once per message in `reconcile` (`a.shell` and the page header `a.header`). `View`, the picker anchor and the panels read those values instead of recomputing them.
- Child state that changes during a gesture is a plain value (`taskDrag`), never a pointer shared between model copies, and it refers to items by ID and state name rather than by index, so a reload during the gesture cannot point it at the wrong item.
- Panels re-render their content only when something visible changed (a drag re-renders the board when the drop column changes, not on every motion event).
- `panels.Block` and `panels.Pane` skip Lip Gloss wrapping when content already fits, and fall back to Lip Gloss otherwise, with identical output (`frame_test.go`). Use them for content that is already laid out.
- Hover needs `tea.MouseModeAllMotion`, which reports every pointer move; `View` only asks for it while the Sprints taskboard is focused with no dialog open, and uses cell motion everywhere else.
- Small overlays that move with the pointer (the drag chip) use `widgets.Splice`, which rewrites only the rows they cover. `OverlayAt` re-composites the whole screen and is meant for dialogs.
- Optimistic task moves live in the taskboard as a copy-on-write map of pending moves keyed by task ID; a reload that still has the old state does not snap the card back, and flash timers carry a sequence number so an older timer never ends a newer flash.
- `BenchmarkDragMotion` and `BenchmarkHoverMotion` measure one motion event through the whole app with the filter bar open; run `go test ./internal/app -run '^$' -bench DragMotion` after changing the render path.

## Commands and stale results

- Commands capture plain values, never pointers into the model (`queryFor(filters, me)` is evaluated in `Update`, not inside the goroutine).
- Iteration, State and Area go to the server (WIQL). Person and keyword filter the loaded items locally with `models.FilterTree`, which keeps the parents of matching children; `applyFilters` only reloads when the server query changed. When the current user cannot be resolved, `@Me` falls back to `@me` in WIQL.
- Every side effect is a command: HTTP calls, saving filters, opening the browser, copying to the clipboard.
- Mouse events reach only the focused main panel. The taskboard knows its content origin on screen (`SetOrigin`, set next to `SetSize` in `syncLayout`) and maps a pointer to a card or state column with the same widths it renders with, including the viewport scroll offset. A drag ends in the existing `MoveWorkItemMsg`, so it uses the same state-change path as `<` / `>`.
- Work item reloads carry a sequence number (`loadSeq`); outdated responses are dropped.
- The signed-in user is the identity behind the PAT, read once from `_apis/connectionData` (`authenticatedUser`, falling back to `authorizedUser`) into `App.me`. `models.Identity.Owns` decides comment ownership by identity ID, then by unique name, and never by display name. `CommentThread` only emits edit/delete intents for owned comments, and `handlePanelIntent` checks again before opening a modal. An unknown identity makes comments read-only; `Ctrl+r` retries it.
- Comment loads carry the work item ID and are debounced (`commentsDueMsg`) while the cursor moves; responses for another item are ignored by both the Discussion tab and the detail view.
- Notification timers carry the notification ID, so an older timer never hides a newer toast.
- Comments, pull request checks and pull request lists are stale-while-revalidate: the cached value (`internal/cache`, entry age from `Lookup`) is shown at once and refetched when older than `revalidateAfter` (30 s). Pending or running checks are always refetched. Your own changes drop the cached entry (`commentChangedMsg`, `reviewChangedMsg`, `pullRequestCreatedMsg`). A failed load never replaces cached data, and a failed *background* pull request list load is silent. `Ctrl+r` always fetches.
- Memory is bounded: the Files hub's file text is a 32 MB LRU (at least twice `diff.max_file_kb`, so any allowed file fits); `requested` only marks files in flight, so an evicted file loads again when shown. Highlighted code and rendered Markdown are byte-bounded LRUs, glamour renderers are kept for three widths, and the Builds stage cache is pruned to the visible runs past 300 entries.

## Pull requests

- The Repos › Pull requests hub is a fourth `panels.Hub`. Its state lives in `App.pr` (`internal/app/pull_requests.go`): the list panel, the loaded status, and the policies of the selected row, which load debounced like comments. Mine and Active share one `active` load; Mine filters locally by the PAT identity.
- `ViewPullRequest` shows `panels.PullRequestView` (`internal/app/pull_request_review.go`). Opening it loads threads, policy evaluations plus pull request statuses, the pull request itself, linked work items and the latest iteration's changes in parallel; every result carries the pull request ID and is dropped if another one is open.
- Azure DevOps has no diff endpoint, so the Files tab fetches both versions of a file (merge base and source commit of the latest iteration) through the Items API the first time the file is selected and diffs them with `internal/diff` (Myers, unified hunks). `reconcile` asks the view which file it still needs (`WantedFile`), so file loads stay commands.
- Diff rows are rendered once per file and only the two rows under the old and new cursor are re-rendered on a move.
- The view never calls the API. It emits intents (`VotePullRequestMsg`, `CommentPullRequestMsg`, `ReplyPullRequestMsg`, `ResolvePullRequestThreadMsg`); the app opens the dialog or runs the command, then reloads threads, the pull request and its policies.

## Repos: Files, Commits, Branches and Tags

- The Repos › Files hub is `panels.HubFiles`. Repos hubs share `App.code` (`internal/app/repo_scope.go`): their own repository (resolved from `pr.repos`, which carry the clone and web URLs), the branch or tag being browsed (`models.GitVersion`) and the repository's branch and tag names for the `b` picker. Changing the repository or the version bumps `code.seq`, and every folder and file response carrying an older `seq` is dropped.
- `panels.RepoTreePanel` owns the tree: loaded children per folder, expanded folders and the cursor. It never calls the API; `reconcile` (`syncFiles`) asks it for `WantedFolders()` (expanded but not loaded) and loads them right away, while the selected folder's children or the selected file's text load after the comment debounce (`repoFileDueMsg`). `App.code.files.requested` keeps each path from being requested twice.
- `panels.RepoContentPanel` renders a `panels.RepoContent` snapshot built by the app (`fileContent`). The app passes a revision counter (`files.rev`, bumped on every load) so the panel re-renders only when the selection or the data changed.
- The Commits hub (`panels.HubCommits`, `App.commits` in `internal/app/repo_commits.go`) reloads whenever the wanted `commitsKey` (repository, version, `models.CommitCriteria`) differs from the loaded one, which `syncCommits` checks in `reconcile`; pages carry `commits.seq`. Text in the search bar filters the loaded commits in `panels.CommitsPanel`; `author:` and `path:` become criteria. `H` in Files sets `criteria.Path` before switching hubs. The preview's detail (`GetCommitDetail`) loads debounced and is cached by commit ID.
- The Branches hub (`App.branches`, `internal/app/repo_branches.go`) loads once per repository (`loadedFor`) through `ListBranchDetails` and joins the active pull requests by source branch inside the command. Create and delete are intents from `modals.RefForm` / `ConfirmModal` (`CreateBranchRequestMsg`, `DeleteBranchRequestMsg`); the result is a `refChangedMsg` that closes the form on success (an error stacks over it), reloads the branches and the `b` picker's refs, and selects a newly created branch.
- The Tags hub (`App.tags`, `internal/app/repo_tags.go`) loads once per repository, like Branches. An annotated tag's message and tagger load debounced for the preview and are cached by tag object ID; a failed load is cached too, so the preview shows the error instead of loading forever (Commits does the same). Creating a tag returns the same `refChangedMsg` with `tag: true`.
- `modals.RefForm` is a small form (fields plus a `build` func that validates and returns the intent) shared by the new branch and new tag forms.
- `panels.denseList` is the single-line list (cursor, viewport, column fitting) shared by Repos lists.
- Code is highlighted by `components.HighlightCode(path, text)`: Chroma (glamour's engine) picks the lexer from the file name, falling back to content analysis or plain text, and the colors come from the glamour style in `markdown.go` (`CodeBlock.Chroma`), so code looks the same everywhere. It returns one `CodeLine` (colored tokens, no background) per source line, cached by a hash of path and text; `CodeLine.Render(base, width)` draws it on the caller's background, padded and truncated. The pull request diff highlights each side of a file once and picks a diff line's tokens by its old or new line number; the Files hub's `panels.codeView` (gutter, line cursor, focus bar) uses the same rows. Rendering whole files through `glamour.Render` would wrap lines and emit its own backgrounds, which can't sit under diff colors or a cursor row.
- File text comes from the Items API (`GetItemText`, `$format=text`, capped by `diff.max_file_kb`); `GetFileContent` used by the pull request diff is the same call pinned to a commit.

## Pipelines

- The Pipelines › Builds hub is a fifth `panels.Hub` (`HubBuilds`). Its state lives in `App.pl` (`internal/app/pipelines.go`). One `panels.PipelinesPanel` renders every list (`ListRecent`, `ListAll`, `ListRuns`, `ListPipelineRuns`, `ListBranches`); the app keeps the loaded pipelines or runs and the continuation token, and every list load carries `pl.seq` so an outdated or wrong-mode response is dropped.
- Typing in the search bar filters the panel locally; `Enter` emits `SearchPipelinesMsg` and the app reloads with a name pattern (pipelines) or the text, whose qualifiers become `models.RunCriteria` inside the command (`pipeline:` is resolved with a name search there). A query error comes back as `searchErr` without calling the run list.
- Active runs schedule a silent reload every 15 seconds (`pipelinePollMsg` with `pollSeq`); for runs, `mergeRuns` keeps the pages loaded beyond the first.
- The preview pane loads the selected run's timeline and the selected pipeline's last runs debounced, like pull request policies.
- `ViewRun` shows `panels.PipelineRunView` (`internal/app/pipeline_run.go`). Opening it loads the run, its timeline and its changes. `reconcile` asks the view which log it needs (`WantedLog`), the same way the review asks for files. While the run is active a 5 second `runPollMsg` (guarded by `runSeq`) reloads the run and the timeline and marks logs stale, so the selected running step is tailed from its next line.
- The run dialog (`modals.RunPipelineForm`) never calls the API. It asks for the definition and refs (`RunDefinitionRequestMsg`) and for the YAML spec of a ref (`RunSpecRequestMsg`); the app answers with `RunDefinitionMsg` / `RunSpecMsg` only while the form is on top. `RunPipelineRequestMsg` starts the run; on success the app closes the form and opens the run's logs.

## Releases

- The Pipelines › Releases hub is `panels.HubReleases`; its state lives in `App.rl` (`internal/app/releases.go`). Three list panels share `panels.selectList` (cursor, headers, edge-to-edge highlight, reveal): `ReleaseDefinitionsPanel`, `ReleasesPanel` and `ApprovalsPanel`. `selectList` keeps the rendered lines of every row but the selected one until `setItems` or a width change, so a cursor move renders one row; a row's look must therefore depend only on what was passed to `setItems` (every panel setter rebuilds). The releases pane computes `models.CurrentDeployments` once per load, and its column layout once per width. The hub draws two panes itself (`renderReleasesBody`), so it does not use the right preview pane.
- The selected release pipeline's releases load debounced in `reconcile` (`syncReleaseSelection`), like pull request policies.
- Release pipeline search filters the loaded list in the panel. Every text change emits `ReleaseSearchTypedMsg`; when more pages exist the app schedules `releaseSearchDueMsg` (300 ms, `searchSeq`) and the panel shows the server's extra matches under *Found on Azure DevOps* (`SetFound`). `Enter` only accepts the selection (`FocusReleasesMsg`); it never reloads the list. Loads carry `listSeq`/`approvalsSeq`; deploying releases schedule a 15 second refresh.
- `ViewRelease` shows `panels.ReleaseView` (`internal/app/release_view.go`). Its Logs tab and the build run view both embed `panels.runLogs`, which owns the jobs, steps and log panes, log entries, tailing and follow. `reconcile` asks the view for the log it needs (`WantedLog` returns the task's log URL for releases).
- Approve, deploy, cancel and create are intents (`ApproveReleaseMsg`, `DeployStageMsg`, `CancelDeploymentMsg`, `CreateReleaseMsg`) that open dialogs; the dialogs return request messages, and after a change the app reloads the release, the list and the approvals.
- `ViewReleaseEditor` shows `panels.ReleaseEditor` (`internal/app/release_editor.go`) for a release pipeline or a single release. `models.ReleaseDoc` keeps the JSON Azure DevOps returned as a raw tree (numbers as `json.Number`) next to the typed view the panel reads, so a save sends back every field lazyado doesn't model. Changes are `models.DocEdit` values addressed by path (`environments/1/deployPhases/0/workflowTasks/2/inputs/scriptPath`, or `deployPhasesSnapshot` for a release) with the value before and after; `ReleaseDoc.Payload` applies them to a copy and `PUT`s the whole document.
- Task settings come from the task catalog (`distributedtask/tasks/{id}`, the version matching the task's major), cached per task and major in `App.rl.taskDefs`; `models.InputVisible` evaluates `visibleRule` against the current values, edits included. Service connections, agent pools and variable groups are resolved by ID in the same load, and a service connection picker lists the connections of the input's type once per type.
- Jobs are rows of the task list too: `ReleaseDoc.PhaseFields` maps a job's `deploymentInput` (pool, agent specification, demands, parallelism, timeouts, artifact download, OAuth token, condition) to the same `TaskField`s tasks use, so editing, undo and the review work unchanged.
- An input with a data source binding is `InputAzure`: it is never typed. `models.AzureRequestFor` resolves the binding's `$(input)` references against the task's current values (and refuses while one is empty), the app asks `serviceendpoint/endpointproxy` and caches the list per request key in `App.rl.azureLists`, and the field dialog shows loading, the values, Azure's error with retry, or the pipeline's variables. `models.DependentsOf` follows the bindings' references, so applying a change marks every setting that depends on it as stale (`models.StaleField`); the save dialog refuses to save while any is left and `JumpToStale` moves the cursor to the first.
- A definition save that fails is followed by a read of the definition: a newer revision becomes a conflict (`releaseDocConflictMsg`), and `r` reloads it and keeps the edits whose field still holds their *before* value and whose task (`Guard`, the task ID at that path) is still there (`models.ReapplyEdits`). Releases have no revision, so their save errors are shown as they are.

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
