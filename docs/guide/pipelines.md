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

Each pipeline row shows the state of its latest run, the run number and commit message, the trigger and branch, when it started and how long it took. On wide terminals a preview pane shows the selected pipeline's last 10 runs, its pass rate and median duration, and the stages of the latest run.

The run states are shown as labels:

| Label | Meaning |
| ----- | ------- |
| `OK` | Succeeded |
| `FAIL` | Failed |
| `PART` | Partially succeeded |
| `RUN` | Running |
| `WAIT` | Queued, or waiting for an agent or approval |
| `STOP` | Canceled |
| `SKIP` | Skipped (stages, jobs and steps) |

| Key | Action |
| --- | ------ |
| `enter` | Open the pipeline's runs (in *Runs*: open the run) |
| `v` | Open the latest run |
| `n` | [Run the pipeline](#running-a-pipeline) |
| `o` | Open in the browser |

All and Runs load 200 pipelines or 50 runs at a time. The next page loads when you scroll to the end of the list. While a listed run is queued or running, the list refreshes every 15 seconds.

## Searching

Press `/` to open the search bar. What you type filters the loaded list as you type, with matches highlighted. `Enter` sends the search to Azure DevOps, so it finds pipelines and runs that aren't loaded yet.

- In **Recent** and **All**, `Enter` searches every pipeline name in the project and shows the matches in *All*. `*` works as a wildcard.
- In **Runs**, and on a pipeline's runs, these filters run on Azure DevOps:

| Filter | Finds runs |
| ------ | ---------- |
| `state:running`, `queued`, `succeeded`, `failed`, `partial`, `canceled` | in that state |
| `branch:main` | of a branch (`refs/tags/v1.0` for a tag) |
| `pipeline:name` | of one pipeline. The name can be partial if it matches only one pipeline |
| `by:@me` | that you started |

Other words filter the loaded runs by number, message, pipeline, branch, commit and requester. `Esc` clears the search, and a second `Esc` closes the bar.

## A pipeline's runs

`enter` on a pipeline opens its runs. The header shows the pipeline name and folder.

| Key | View |
| --- | ---- |
| `1` | **Runs**: every run, newest first |
| `2` | **Branches**: the latest run on each branch |

`enter` opens a run, `n` runs the pipeline, `/` searches, and `esc` goes back to the hub.

## A run

The run screen has two tabs: `1` **Summary** and `2` **Logs**. `[` / `]` also switch tabs, `o` opens the run (or the selected job and step) in the browser, `n` runs the same pipeline again on the same branch, and `esc` goes back.

While the run is queued or running, the screen refreshes every 5 seconds.

### Summary

- **Stages and jobs**: each stage with how many of its jobs are done and how long it took, and its jobs under it. `j` / `k` move between jobs and `enter` opens the job's logs.
- **Errors and warnings** reported by the stages, jobs and steps, with where they came from.
- **Details**: pipeline, state, trigger, who requested it, repository, branch, commit, agent pool, and when it was queued, started and finished.
- **Changes**: the commits included in the run.

### Logs

The left pane lists the jobs, grouped by stage. The right side shows the selected job's agent, its steps with their durations, and the log of the selected step. A failed or running job and step are selected for you when the logs open.

| Key | Action |
| --- | ------ |
| `Tab` / `Shift+Tab` | Move between the jobs, the steps and the log |
| `j` / `k` | Move to the next / previous job or step, or scroll the log |
| `g` / `G` | First / last job or step, or top / bottom of the log |
| `enter` | Go to the next pane (jobs, then steps, then the log) |
| `F` | Follow the log: keep showing the newest lines of a running step |
| `esc` | Back to the previous pane, then out of the run |

Log lines keep Azure DevOps' markers: errors are red, warnings yellow, sections and commands highlighted. Very large logs are cut at the `diff.max_file_kb` size limit (see [Configuration](configuration.md#large-files-in-diffs)). `o` opens the full log in the browser.

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
