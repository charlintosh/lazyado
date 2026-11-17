# Releases

The *Pipelines › Releases* hub (`R`) shows the classic release pipelines of the project. You can follow a release through its stages, approve or reject deployments, deploy a stage, and create a release with the artifact versions, stages and variables you choose. Editing release pipelines isn't supported; use Azure DevOps for that.

- [Release pipelines and releases](#release-pipelines-and-releases)
- [Searching release pipelines](#searching-release-pipelines)
- [Approvals](#approvals)
- [A release](#a-release)
- [Deploying a stage](#deploying-a-stage)
- [Canceling a deployment](#canceling-a-deployment)
- [Approving or rejecting](#approving-or-rejecting)
- [Creating a release](#creating-a-release)

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

The **Approvals** view lists pending pre- and post-deployment approvals: first the ones **waiting on you** (including approvals assigned to a group you belong to), then the ones **waiting on others**. The pivot shows how many are waiting on you. On wide terminals a preview shows the approvers, when it was requested, the artifact and the release's stages.

`a` approves or rejects the selected approval, `enter` opens its release, `y` [shares](sharing.md) it, and `o` opens it in the browser.

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
| `enter` | Open the stage's logs |
| `n` | Create another release of this pipeline |
| `y` | [Share](sharing.md) the release: link, name, its stages, its release pipeline |
| `o` | Open the release in the browser |
| `esc` | Back to the hub |

**Logs** works like a build's [logs](pipelines.md#logs): jobs grouped by stage on the left, the selected job's tasks and the task's log on the right. `Tab` moves between them, and `F` follows a running task.

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
