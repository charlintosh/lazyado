# Pipelines

The *Pipelines › Builds* hub (`D`) shows the pipelines of the project and their runs. You can open any run to see its stages, jobs and logs, and you can start a run from a branch or tag with the options the pipeline offers.

- [The pipeline lists](#the-pipeline-lists)
- [Searching](#searching)
- [A pipeline's runs](#a-pipelines-runs)
- [A run](#a-run)
- [Running a pipeline](#running-a-pipeline)

## The pipeline lists

| Key | View |
| --- | ---- |
| `1` | **Recent**: the 10 pipelines that ran most recently, with their latest run |
| `2` | **All**: every pipeline in the project, grouped by folder |
| `3` | **Runs**: the latest runs of every pipeline |

`Tab` / `Shift+Tab`, `h` / `l` and `[` / `]` also cycle through the views.

Each row shows the state of the run (or the pipeline's latest run), the run number and commit message, the pipeline, trigger, branch, commit and who started it, when it started and how long it took. *Runs* and a pipeline's runs are grouped under **Today**, **Yesterday**, **Last 7 days** and **Older**. On wider lists a **Stages** column shows one cell per stage, in run order: a coloured `█` for a stage that ran (green, red, blue while running, yellow if partially succeeded), `░` for one that hasn't run and `·` for one that was skipped. Past 12 stages it ends with `+N`. It fills in for the rows on screen and refreshes while a run is active.

On wide terminals a preview pane shows the selected run: its state, number, title, branch, commit and trigger, its **Stages**, **Where it failed** (the first error and where it came from) or **Running now**, and the last 10 runs of its pipeline with their pass rate and median duration.

The preview's **Stages** stays short however many stages a run has: a strip with one cell per stage (two columns each when they fit, one when they don't, and counts per state like `✓ 41  ○ 16` when even that doesn't fit), then up to four stages, picked in this order and shown in run order: failed ones (up to two), running ones, the last one that finished, and the next one to run. The rest are summed up, like `+ 17 more queued`.

Run, stage, job and step states are shown with one symbol. The run screen's header spells the state out (`✗ FAILED`).

| Symbol | Meaning |
| ------ | ------- |
| `✓` | Succeeded |
| `✗` | Failed |
| `!` | Partially succeeded |
| `◐` | Running |
| `○` | Queued, waiting for an agent or approval, or not run |
| `⊘` | Canceled |
| `–` | Skipped (stages, jobs and steps) |

| Key | Action |
| --- | ------ |
| `enter` | Open the pipeline's runs (in *Runs*: open the run) |
| `v` | Open the latest run |
| `e` | Open the run's logs at its first error |
| `n` | [Run the pipeline](#running-a-pipeline) |
| `y` | [Share](sharing.md) the pipeline (link, definition ID, name) or run (link, build number, run ID) |
| `o` | Open in the browser |

All and Runs load 200 pipelines or 50 runs at a time. The next page loads when you scroll to the end of the list. While a listed run is queued or running, the list refreshes every 15 seconds.

## Searching

Press `/` to open the search bar. What you type filters the loaded list right away, with matches highlighted. When you stop typing for a moment (0.4 seconds), the search goes to Azure DevOps too, so it finds pipelines and runs that aren't loaded yet. The bar shows *searching* while it loads, and the list you see stays until the results arrive. Typing on cancels the pending search, so a burst of keystrokes makes one request.

- In **Recent** and **All**, the search covers every pipeline name in the project and shows the matches in *All*. `*` works as a wildcard.
- In **Runs**, and on a pipeline's runs, these filters run on Azure DevOps:

| Filter | Finds runs |
| ------ | ---------- |
| `state:running`, `queued`, `succeeded`, `failed`, `partial`, `canceled` | in that state |
| `branch:main` | of a branch (`refs/tags/v1.0` for a tag) |
| `pipeline:name` | of one pipeline. The name can be partial if it matches only one pipeline |
| `by:@me` | that you started |

Other words filter the loaded runs by number, message, pipeline, branch, commit and requester. A filter that isn't valid yet (like `state:fa` while you type) shows its error under the bar and keeps the list. `Enter` closes the bar and keeps the search, `Esc` clears it, and a second `Esc` closes the bar.

## A pipeline's runs

`enter` on a pipeline opens its runs. The header shows the pipeline name and folder.

| Key | View |
| --- | ---- |
| `1` | **Runs**: every run, newest first |
| `2` | **Branches**: the latest run on each branch |

`enter` opens a run, `n` runs the pipeline, `/` searches, and `esc` goes back to the hub.

## A run

The header shows the run's state, number and title, then its pipeline, branch, commit, trigger, when it started and how long it took. The run screen has two tabs: `1` **Summary** and `2` **Logs**. `[` / `]` also switch tabs, `o` opens the run (or the selected job and step) in the browser, `n` runs the same pipeline again on the same branch, `y` [shares](sharing.md) the run (its link, build number, run ID, its pipeline, and on the Logs tab a link to the selected job's logs), and `esc` goes back.

While the run is queued or running, the screen refreshes every 5 seconds.

### Summary

- **Stages** (left pane): every stage, one line each, with its state, how many of its jobs are done and how long it took. Long names end in `…`, and the list scrolls to the selected stage. The title counts stages by state.
- **Jobs in** the selected stage (right pane): each job with its state and duration, where it failed or what it is running, and its agent.
- `j` / `k` move in the focused pane, `Tab` (or `h` / `l`) switches panes, and `enter` opens the logs of the stage's failed or running job, or of the selected job.
- **Errors and warnings** reported by the stages, jobs and steps, with where they came from. `e` opens the logs at the first error.
- **Details**: pipeline, state, trigger, who requested it, repository, branch, commit, agent pool, and when it was queued, started and finished.
- **Changes**: the commits included in the run.

### Logs

The left pane, **Jobs in run**, is a tree: stages, their jobs, and each job's steps. The right pane shows the log of the selected step, or the whole job's log when a job is selected. Its header shows the step's state and duration, where it is (`Stage › Job › Step`), the agent and when it started, and how many errors and warnings the log has. A failed or running job and step are selected and expanded for you when the logs open. The focused pane has a blue border, and the status bar says which one it is (`LOGS · JOBS` or `LOGS · LOG`).

| Key | Action |
| --- | ------ |
| `Tab` / `Shift+Tab` | Switch between the jobs tree and the log |
| `j` / `k` | Move to the next / previous job or step, or scroll the log |
| `l` / `h` | Expand / collapse a job, or go to its first step / back to the job. In the log, `h` goes back to the tree |
| `g` / `G` | First / last job or step, or top / bottom of the log |
| `enter` | Expand the job, or go to the log |
| `e` / `E` | Jump to the next / previous error in the log, or to the next failed step |
| `C` | Fold or unfold every group in the log |
| `F` | Follow the log: keep showing the newest lines of a running step |
| `esc` | Back to the tree, then out of the run |

Log lines keep Azure DevOps' markers: errors are red on a tinted row, warnings yellow, sections and commands highlighted. Groups (`##[group]`) start folded, showing how many lines they hide, unless they contain an error or warning or are still running. Very large logs are cut at the `diff.max_file_kb` size limit (see [Configuration](configuration.md#large-files-in-diffs)). `o` opens the full log in the browser.

## Running a pipeline

Press `n` on a pipeline, a run, a pipeline's runs or a run screen. The dialog loads the pipeline's settings and then shows:

1. **Branch or tag.** Branches are listed first, starting with the default branch, then tags, newest first. Type to filter. On a run screen it starts on that run's branch.
2. **Commit (optional).** Leave it empty to run the latest commit, or paste a commit ID from that branch.
3. **Parameters.** These are read from the pipeline's YAML at the chosen branch or tag, so they follow that version of the pipeline. A list of values is a picker, `true` / `false` is a checkbox, and text and numbers are fields. Parameters you already changed keep their values when you switch branches. Object and step-list parameters use their defaults.
4. **Stages to run.** Every stage starts ticked. `j` / `k` move and `enter` (or `space`) skips or includes a stage.
5. **Variables.** Only the variables the pipeline lets you set at queue time. Only the ones you change are sent. Secret variables start empty and are kept unless you type a new value.
6. **System diagnostics.** Turns on debug logging for this run.

`Ctrl+s` starts the run, then its logs open. If the branch changed and its parameters are still loading, the first `Ctrl+s` waits for them so you can check them. If Azure DevOps refuses the run, the error is shown over the dialog and everything you entered is kept.

Classic (non-YAML) pipelines have no parameters or stages: you choose the branch, an optional commit, and the variables. For YAML that lives outside Azure Repos (GitHub, Bitbucket), lazyado can't read the file, so you type the branch and the pipeline's defaults are used.
