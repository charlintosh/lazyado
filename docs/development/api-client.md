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
- Policy evaluations are a preview API (`getPreview`) keyed by `vstfs:///CodeReview/CodeReviewId/{projectId}/{pullRequestId}`; a status-check policy and a raw pull request status with the same `genre/name` are shown once.

## Error handling

- Wrap errors with context using `fmt.Errorf("context: %w", err)` to preserve the original cause.

## References

- Project guidelines: [AGENTS.md](../../AGENTS.md)
- Examples: [internal/api/work_items.go](../../internal/api/work_items.go), [internal/api/client.go](../../internal/api/client.go)
