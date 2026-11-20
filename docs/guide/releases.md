# Releases

The *Pipelines › Releases* hub (`R`) shows the classic release pipelines of the project. You can follow a release through its stages, approve or reject deployments, deploy a stage, create a release with the artifact versions, stages and variables you choose, and see or [edit a release pipeline](#editing-a-release-pipeline) or a single release: its stages, tasks, task settings and variables.

- [Release pipelines and releases](#release-pipelines-and-releases)
- [Searching release pipelines](#searching-release-pipelines)
- [Approvals](#approvals)
- [A release](#a-release)
- [Deploying a stage](#deploying-a-stage)
- [Canceling a deployment](#canceling-a-deployment)
- [Approving or rejecting](#approving-or-rejecting)
- [Creating a release](#creating-a-release)
- [Editing a release pipeline](#editing-a-release-pipeline)

## Release pipelines and releases

| Key | View |
| --- | ---- |
| `1` | **Releases**: release pipelines on the left, the selected pipeline's releases on the right |
| `2` | **Approvals**: pending approvals in the project |

`[` / `]` also switch views.

The left pane lists the release pipelines, with **Starred** ones first: the ones you star here with `Ctrl+t` (saved in `state.json`), plus any release pipelines Azure DevOps reports as favorites for your account. The group is always shown; while it's empty it says how to star one. Each row shows the pipeline's latest release. `Tab` (or `h` / `l`) moves between the two panes, `enter` on a pipeline moves to its releases, and `v` opens its latest release directly.

Each release shows its stages as chips:

| Chip | Meaning |
| ---- | ------- |
| `OK` | Deployed |
| `RUN` | Deploying |
| `APPR` | Waiting for approval |
| `WAIT` | Queued or scheduled |
| `FAIL` | The deployment failed |
| `REJ` | An approver rejected it |
| `PART` | Partially deployed |
| `STOP` | Canceled |
| name only | Not deployed |

The row's second line shows the primary artifact's version and branch, who created the release and when. 25 releases load at a time, and more load as you scroll. While a release is deploying, the list refreshes every 15 seconds.

| Key | Action |
| --- | ------ |
| `/` | [Search](#searching-release-pipelines) release pipelines by name |
| `Ctrl+t` | Star / unstar the selected release pipeline |
| `enter` | On a pipeline: go to its releases. On a release: open it |
| `v` | On a pipeline: open its latest release. On a release: open it |
| `a` | Approve or reject the release's pending approval |
| `e` | On a pipeline: [see or edit it](#editing-a-release-pipeline). On a release: [edit that release](#editing-a-single-release) |
| `n` | [Create a release](#creating-a-release) of the selected pipeline |
| `y` | [Share](sharing.md) the release pipeline or release |
| `o` | Open in the browser |

## Searching release pipelines

Press `/` and type. The list filters as you type, with matches highlighted, and the releases pane follows the highlighted pipeline.

| Key | Action |
| --- | ------ |
| `↑` / `↓` (or `Ctrl+p` / `Ctrl+n`) | Move between matches |
| `Enter` | Close the bar and go into the highlighted pipeline's releases, with the newest selected |
| `Esc` | Clear the text; a second `Esc` closes the bar |

After `Enter` the filter stays: the pane title shows it (`· /pay`). `Esc` in the releases pane goes back to the filtered pipelines, `Esc` in the pipelines pane clears the filter and keeps your selection, and `/` reopens the bar with the text so you can refine it. So `/ pay Enter Enter` opens the newest release of the first match.

When the project has more release pipelines than are loaded, lazyado also asks Azure DevOps 300 ms after you stop typing. Its extra results appear under **Found on Azure DevOps** and work the same way. When everything is loaded, nothing is sent.

## Approvals

The **Approvals** view lists the pending pre- and post-deployment approvals of the release pipeline selected on the left: first the ones **waiting on you** (including approvals assigned to a group you belong to), then the ones **waiting on others**. Move to another pipeline with `h` or `Tab` to see its approvals. Each pipeline in the list shows how many approvals wait on you (orange) and on others (grey), and the pivot shows how many wait on you in the whole project. On wide terminals a preview shows the approvers, when it was requested, the artifact and the release's stages.

`A` switches between the selected pipeline's approvals and every pipeline's; with every pipeline, each approval also names its release pipeline. `a` approves or rejects the selected approval, `enter` opens its release, `y` [shares](sharing.md) it, and `o` opens it in the browser.

## A release

The release screen has two tabs: `1` **Pipeline** and `2` **Logs**. While a stage is deploying, the screen refreshes every 5 seconds.

**Pipeline** shows the stages as boxes, with the selected one highlighted. `h` / `l` move between stages. For the selected stage you see:

- how it's triggered (after the release, after another stage, or manual);
- what it's waiting on: the approval, its approvers, and whether you can act on it;
- the approvers that will be asked before and after it deploys;
- its last deployment attempt: who started it, its jobs and tasks, and any errors.

A sidebar lists the release's artifacts, each stage's trigger, who created the release and its description.

| Key | Action |
| --- | ------ |
| `a` | [Approve or reject](#approving-or-rejecting) the stage's pending approval |
| `d` | [Deploy](#deploying-a-stage) the stage, or redeploy it |
| `x` | [Cancel](#canceling-a-deployment) the stage's deployment |
| `e` | [Edit this release](#editing-a-single-release) |
| `enter` | Open the stage's logs |
| `n` | Create another release of this pipeline |
| `y` | [Share](sharing.md) the release: link, name, its stages, its release pipeline |
| `o` | Open the release in the browser |
| `esc` | Back to the hub |

**Logs** shows the logs of the stage selected in **Pipeline**, its latest attempt. A strip above the logs shows the stages; `<` / `>` switch to the previous or next stage without leaving the tab, and **Pipeline** keeps the stage you chose. Below it, the jobs and tasks of that stage are on the left and the selected task's log is on the right, as in a build's [logs](pipelines.md#logs). `Tab` switches between them, `e` jumps to errors in that stage, and `F` follows a running task. A stage that hasn't deployed yet says so instead of showing jobs.

## Deploying a stage

Press `d` on a stage that isn't deploying or waiting for approval. The dialog shows what will be deployed and warns you when:

- the stage has approvers, who will be asked once you deploy;
- the last attempt failed, with the first error.

Add an optional comment and press `Ctrl+s`. Only that stage deploys; later stages follow their own triggers.

## Canceling a deployment

Press `x` on a stage that is deploying, queued, scheduled or waiting for approval. The dialog says what will happen: running tasks stop, or a pending approval is withdrawn, and the stage ends as canceled. Stages that wait on it don't run. Add an optional comment and press `Ctrl+s`, or `Esc` to keep it running. You can deploy the stage again later with `d`.

## Approving or rejecting

Press `a` on a pending approval. Choose **Approve** or **Reject** with `h` / `l`, `Tab` to the comment, and press `Ctrl+s`. A comment is required to reject. Approving lets the deployment continue; rejecting stops the stage, and stages that wait on it don't run.

Azure DevOps decides whether you're allowed to approve. If you aren't, the error is shown and nothing changes.

## Creating a release

Press `n` on a release pipeline, a release, or the release screen. The dialog loads the pipeline and shows:

1. **Artifacts.** For a build artifact, pick a **branch** (or *Any branch*) and a **version**: the succeeded and partially succeeded builds of that branch, newest first. For an Azure Repos artifact, pick a branch; its latest commit is used. Other artifact types use their default version.
2. **Deploy automatically.** The stages that start on their own are ticked. Untick one to make it wait for a manual deploy in this release. Manual stages are listed underneath.
3. **Variables.** Only the ones the pipeline lets you set at release time. Only the ones you change are sent, and secret variables keep their value unless you type a new one.
4. **Description.**

`Ctrl+s` creates the release and opens it. If you changed a branch and its builds are still loading, the first `Ctrl+s` waits so you can check the version.

## Editing a release pipeline

Press `e` on a release pipeline to see how it's set up, as in the Azure DevOps release pipeline editor. It opens read-only; press `e` again to edit.

| Tab | Shows |
| --- | ----- |
| `1` **Pipeline** | The artifacts and triggers, the stages as boxes (jobs and tasks, trigger, approvers) and the selected stage: trigger, approvals before and after, gates, jobs and agent pool, variables, retention |
| `2` **Tasks** | A strip of stages on top (`<` / `>` switch), the selected stage's jobs with their tasks as a tree on the left, and the selected job's or task's settings on the right |
| `3` **Variables** | The pipeline's variables with their value, scope and whether they can be set at release time, and the linked variable groups. `h` / `l` switch between all, the release and each stage |
| `4` **History** | The pipeline's revisions: who changed it, when, and their comment |

Select a job (`▾ Agent job`) to see its settings, as in Azure DevOps: display name, **agent pool**, **agent specification** (the Microsoft-hosted image, shown for hosted pools), demands, parallelism, timeouts, whether artifacts download, OAuth token access and when the job runs.

Task settings show with the names Azure DevOps uses ("Script Path", not `scriptPath`), grouped as in Azure DevOps under colored tags (the task's own groups, then **Control options**). Settings that a task hides because of another setting (an inline script while *Script Location* is a path) are hidden here too, and change as you edit. Service connections and agent pools show by name. A task whose details can't be loaded, and task groups, show their inputs by their raw names.

`y` copies the selected task as YAML (in **Variables**, the selected value). `o` opens the pipeline in the browser.

What the glyphs and colors mean:

| Mark | Meaning |
| ---- | ------- |
| `●` / `○` | A task that's on / off (off tasks are struck through); a yes / no setting; a variable you can / can't set at release time |
| `├ └ │` | A job's tasks |
| `▾` | A setting picked from a list |
| `›` | A setting you type |
| `from Azure` | A setting whose values come from Azure: [never typed](#settings-azure-fills) |
| `▎` orange | A changed row; under a changed setting, `−` and its saved value |
| `!` red | A setting that must be picked again before saving |
| `⊘` | Read-only: a deployed stage of a release, a variable group, a setting lazyado can't edit |
| `◌` | Settings hidden because of another setting's value |
| `■` | A variable's scope, one color per scope |
| `$(Name)` violet | A pipeline variable inside a value |

### Changing it

Press `e` to start editing. The badge and the status bar turn orange.

| Key | Action |
| --- | ------ |
| `enter` | Change the selected setting or variable. Yes/no settings switch at once; lists, agent pools, agent images, service connections and [settings Azure fills](#settings-azure-fills) open a picker; multi-line settings and demands (one per line) open an editor (`Ctrl+s` applies) |
| `space` | Turn the selected task on or off |
| `u` | Undo the selected setting, or on the task list, every change to that job or task |
| `Ctrl+s` | Review and save |
| `esc` / `e` | Stop editing. If you changed something, lazyado asks before discarding it |

Changed settings are orange and show their saved value underneath. The task list counts each job's and task's changes, the stage strip and the tabs count theirs, and the header counts them all.

Secret variables never come back from Azure DevOps, so they show as `●●●●●●●●` with a `secret` tag. Typing a new value replaces the secret; leaving it alone keeps it. Variable groups are edited in *Library* on Azure DevOps.

Agent images are the Microsoft-hosted ones (`ubuntu-latest`, `windows-latest`, `macos-latest` and their pinned versions); an image that's no longer listed stays choosable as the saved value. Adding, removing or reordering jobs, tasks, stages and variables still happens in Azure DevOps.

### Settings Azure fills

Service connections, subscriptions, resource groups, locations, and anything else a task fills from Azure (app names, slots, clusters, key vaults, storage accounts) carry a `from Azure` tag, and **are never typed**. Azure DevOps' own editor fills them the same way:

1. The task definition says which list feeds which setting (its *data source bindings*), and which other settings it needs, usually the service connection and the subscription.
2. lazyado asks Azure DevOps for that list with the task's own values (`serviceendpoint/endpointproxy`).
3. Azure DevOps asks Azure as the service connection's identity, and returns what that identity can see.

Typing these values is how most of these deployments break:

- What you see isn't what's saved: subscriptions and connections are saved as IDs, locations as codes like `eastus2`.
- Each list depends on the one above it. A different connection may see other subscriptions, so the old resource group and location may not exist for it.
- A typo saves fine and fails only when the stage deploys.
- When a list can't load, the connection's identity usually lacks access in Azure, so a typed value would fail at deploy for the same reason.

So in the picker:

- The list comes from Azure, with the saved value marked. Lists are kept for the session per connection and parameters; `Ctrl+r` asks again.
- `Tab` switches to the pipeline's variables: a `$(Name)` value is resolved when the stage deploys.
- If Azure can't list the values, the dialog shows Azure's own error. `r` tries again, `v` uses a pipeline variable, `o` opens the pipeline in the browser, and `esc` keeps the saved value. There's no way to type one.
- With no service connection of the right type to choose from, the picker says so; a connection can't be typed either.

When you change a setting another one depends on (the connection, then the subscription), the settings that depend on it are marked `!` and say *pick again*: their values were listed for the old one. The task, the stage strip, the tab and the header count them, and `Ctrl+s` won't save until each is picked again or the change is undone. In the save dialog, `enter` jumps to the first one.

### Saving

`Ctrl+s` shows every change as before and after, grouped by stage and task, with a comment for the **History** tab. `Ctrl+s` again saves it as the next revision. Releases already created keep their own copy of the pipeline.

If someone else saved the pipeline while you were editing, nothing is overwritten: the dialog says who saved it. Press `r` to load their revision and put your changes back on top. A change to a setting they also changed is dropped, and the notice names it, so you can look at it again before you save.

### Editing a single release

Press `e` on a release in the hub, or on the release screen, to edit only that release, as *Edit release* does in Azure DevOps. It works like editing a pipeline, with a banner saying that only this release changes. Stages that have already deployed are read-only; the ones still to deploy use your changes when they run. The release pipeline itself stays as it is.

