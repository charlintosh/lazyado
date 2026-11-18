# API Client Patterns

## Authentication & Requests

- Use Basic auth with an empty username and the PAT as the password for Azure DevOps.
- Use the centralized request helpers in [internal/api](../../internal/api) (see [internal/api/client.go](../../internal/api/client.go)) so authentication, headers and common error wrapping are consistent.
- Ensure `api-version=7.1` is included on calls (helpers like `getWithBase()` add this automatically).

## Data & WIQL

- Return strongly-typed Go structs from API helpers in [internal/api](../../internal/api).
- Escape WIQL user values with `escapeWIQL()` (double single quotes) before embedding into WIQL strings.
- See `QueryWorkItems()` in [internal/api/work_items.go](../../internal/api/work_items.go) for an example of secure query construction.

## Pull requests

- [internal/api/pull_requests.go](../../internal/api/pull_requests.go) covers listing, threads, votes, policy evaluations, statuses, iteration changes and file contents.
- JSON writes that are not work item patches go through `sendJSON(method, endpoint, payload)`, which adds `api-version` and sends `application/json`.
- Work item comment likes use the preview reactions endpoint `.../wit/workItems/{id}/comments/{commentId}/reactions/like` (`PUT` to like, `DELETE` to unlike); `GetComments` reads them from `reactions` (`$expand=all`).
- Pull request comment likes use the preview `.../threads/{threadId}/comments/{commentId}/likes` endpoint (`POST` to like, `DELETE` to unlike, `api-version=7.1-preview.1`); `usersLiked` on each comment says who liked it.
- Creating a pull request uses `refs?filter=heads/` for branches, the repository's `defaultBranch`, `diffs/commits` (ahead/behind and change counts), `commits` with `itemVersion` / `compareVersion` for the commit messages, the items API on the target branch for `pull_request_template.md`, and `POST .../pullrequests` ([internal/api/pull_request_create.go](../../internal/api/pull_request_create.go)).
- The pull requests API has no text search; `ListPullRequests` sends only `searchCriteria` (`creatorId`, `reviewerId`, `sourceRefName`, `targetRefName`), and text filtering happens on the loaded list.
- Quick search looks pull requests up by ID with the project-scoped `/git/pullrequests/{id}`, so no repository is needed.
- A check's `BuildID` comes from the policy evaluation's `context.buildId` (build validation) or from a status's `targetUrl` when it points to `_build/results?buildId=`. A status that only adds a build to a policy with the same name fills the policy's `BuildID` instead of adding a row.
- Policy evaluations are a preview API (`getPreview`) keyed by `vstfs:///CodeReview/CodeReviewId/{projectId}/{pullRequestId}`; a status-check policy and a raw pull request status with the same `genre/name` are shown once.

## Repos

- [internal/api/git_files.go](../../internal/api/git_files.go). Repositories come from `git/repositories`, which already carries `defaultBranch`, `remoteUrl`, `sshUrl` and `webUrl`, so cloning needs no extra call.
- A folder: `items?scopePath=&recursionLevel=OneLevel&latestProcessedChange=true` plus `versionDescriptor.versionType` (`branch`, `tag` or `commit`) and `versionDescriptor.version`. The response includes the folder itself (dropped) and submodules (`gitObjectType: commit`, dropped); `latestProcessedChange` is the last commit that touched each item.
- Go to file: one `items?scopePath=/&recursionLevel=Full` with the version descriptor and **without** `latestProcessedChange` (the slow part), decoded as a stream and cut at `models.MaxIndexedItems` (`ListAllItems`). It is cached per repository and version (last three), so the finder makes no request while typing; previews reuse `GetItemText` and the Files pane's cache. Keep it that way: the finder must never call the API per keystroke.
- A file: `items?path=&$format=text` with the same version descriptor, read up to `diff.max_file_kb` (`GetItemText`).
- Branch and tag names: `refs?filter=heads/` and `refs?filter=tags/`.
- Branches: `stats/branches?baseVersionDescriptor.version=<default>` gives each branch's last commit and ahead/behind counts in one call; `refs?filter=heads/` adds `objectId`, `isLocked` and `creator`, and `refs?includeMyBranches=true` marks *Mine*. That call accepts neither `filter` nor `$top` (it answers 400 "Pagination is not supported"), and a failure there only leaves *Mine* with the default branch ([internal/api/git_refs.go](../../internal/api/git_refs.go)).
- Creating and deleting a branch are both `POST refs` with one `{name, oldObjectId, newObjectId}` update: a zero `oldObjectId` creates, a zero `newObjectId` deletes. The reply is `200` even when the update is refused, so `updateRef` checks `success` and reports `customMessage` or `updateStatus`. `ResolveCommit` turns a branch or tag into its commit with `commits?searchCriteria.$top=1`.
- Tags: `refs?filter=tags/&peelTags=true` — an annotated tag's `objectId` is the tag object and `peeledObjectId` its commit; a lightweight tag has no `peeledObjectId`. `annotatedtags/{objectId}` gives the message and `taggedBy`. Create with `POST annotatedtags` (`name`, `taggedObject.objectId`, `message`), which is what the web UI does; lightweight tags would be a `POST refs` like branches ([internal/api/git_tags.go](../../internal/api/git_tags.go)).
- Commits: `commits?searchCriteria.itemVersion.version=&searchCriteria.itemVersion.versionType=&searchCriteria.$top=100&searchCriteria.$skip=`, plus `searchCriteria.author` and `searchCriteria.itemPath` ([internal/api/git_commits.go](../../internal/api/git_commits.go)). The list doesn't include parents; `commits/{id}` does, and `commits/{id}/changes` lists the changed items. There is no commit graph in the API.
- Azure DevOps has no REST endpoint for blame or for comparing two files on screen, so the Files hub leaves those to the browser (`o`).

## Pipelines

- [internal/api/pipelines.go](../../internal/api/pipelines.go) uses the Build API (`_apis/build/...`) for reading, because it is the one that filters runs by state, branch, pipeline and requester on the server and searches pipeline names. Starting a YAML run uses the Pipelines API.
- Lists page with the `x-ms-continuationtoken` response header, passed back as `continuationToken` (`getPage`).
- *Recent* reads the latest 100 runs (`builds?queryOrder=queueTimeDescending`) and keeps the first run of each pipeline (`models.LatestPerPipeline`).
- *All* is `build/definitions?includeLatestBuilds=true&queryOrder=definitionNameAscending`; a search sends `name=*text*`. The list response has no repository or process, so the run dialog loads `build/definitions/{id}` for the repository, the YAML path and the variables with `allowOverride`.
- Stages, jobs, steps and issues come from `build/builds/{id}/timeline`, which is empty until the run starts. `models.BuildStages` walks Stage → Phase → Job → Task; classic pipelines have no Stage records and get one synthetic stage.
- Logs are plain text from `build/builds/{id}/logs/{logId}`, read up to `diff.max_file_kb`. A running step is tailed with `startLine`. `CleanLogLine` strips the timestamp prefix and ANSI escapes and keeps the `##[error]` style markers for rendering.
- The run dialog reads the YAML at the chosen branch or tag through the Items API (`versionDescriptor.versionType` `branch` or `tag`) and `models.ParsePipelineSpec` keeps top-level scalar `parameters:` and `stages:`. Parameters and stages that only exist in templates are not seen.
- YAML pipelines start with `POST pipelines/{id}/runs` (`resources.repositories.self.refName` and `version`, `templateParameters`, `variables`, `stagesToSkip`). Classic pipelines queue with `POST build/builds` (`sourceBranch`, `sourceVersion`, and the variables JSON-encoded in `parameters`). *System diagnostics* sets `system.debug`.

## Releases

- Classic releases live on a different host: `https://vsrm.dev.azure.com/{org}/{project}/_apis` (`Config.ReleaseURL`). [internal/api/releases.go](../../internal/api/releases.go) builds those URLs with `releaseEndpoint` and adds `api-version` itself.
- Release pipelines: `release/definitions?searchText=&$expand=lastRelease&queryOrder=nameAscending`, paged with the continuation token. Stars come from the organization favorites API (artifact types containing `ReleaseDefinition`, the ID is the trailing number of the artifact ID) plus local stars in `state.json`.
- Releases: `release/releases?definitionId=&$expand=environments,artifacts,approvals`. A stage's state comes from `models.StageStateOf`: a rejected approval wins, then a pending manual approval, then the environment status (`rejected` without a rejected approval means the deployment failed).
- A stage's last attempt is the `deploySteps` entry with the highest `attempt`; its phases, deployment jobs and tasks become `RunJob`s so the build log viewer can show them. Task logs are read from each task's `logUrl`, and their log ID is a hash of that URL so it stays the same across refreshes.
- Approvals: `release/approvals?statusFilter=pending` for everything, and again with `assignedToFilter` (unique name, or ID) and `includeMyGroupApprovals=true` for the ones waiting on you. Decide with `PATCH release/approvals/{id}` (`status`, `comments`).
- Deploy a stage with `PATCH release/releases/{id}/environments/{envId}` and `status: inProgress` (`7.1-preview.7`); cancel its deployment with the same call and `status: canceled`.
- Create with `POST release/releases`: `artifacts[].instanceReference` (`id`, `name`, `sourceBranch`), `manualEnvironments` and `variables`. Build artifact versions come from the Build API (`resultFilter=succeeded,partiallySucceeded`); Git artifacts use the branch's head commit from the refs API.

## Azure DevOps REST API reference

Official Microsoft Learn docs. Use them when you need a route, method, parameter or response schema that isn't already used in [internal/api](../../internal/api).

- [REST API reference](https://learn.microsoft.com/en-us/rest/api/azure/devops/?view=azure-devops-rest-7.1): every API area (Work Item Tracking, Git, Policy and so on). The page has a version picker; lazyado pins `7.1`.
- [Calling the REST API](https://learn.microsoft.com/en-us/azure/devops/integrate/how-to/call-rest-api?view=azure-devops): URL structure, headers and `api-version`.
- [REST API samples](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/rest/samples?view=azure-devops)
- [Authentication guidance](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/authentication-guidance?view=azure-devops)
- [Rate limits and throttling](https://learn.microsoft.com/en-us/azure/devops/integrate/concepts/rate-limits?view=azure-devops)

How to look something up:

1. Open the reference, pick the API area, then pick the operation. The operation page has the exact route, method, parameters, request body, response and examples.
2. Check that the operation exists in `7.1`. If it's only in preview (`7.1-preview.N`), call it through `getPreview` or pass the explicit version, as the likes and policy evaluation calls above do.
3. Treat the Microsoft Learn pages as the source of truth. Preview endpoints and versions can change without notice.
4. Authentication stays the same: a PAT over Basic auth through the [internal/api](../../internal/api) helpers. Never hard-code, log or print credentials.

## Error handling

- Wrap errors with context using `fmt.Errorf("context: %w", err)` to preserve the original cause.

## References

- Project guidelines: [AGENTS.md](../../AGENTS.md)
- Examples: [internal/api/work_items.go](../../internal/api/work_items.go), [internal/api/client.go](../../internal/api/client.go)
