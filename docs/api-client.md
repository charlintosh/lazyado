# API Client Patterns

## Authentication & Requests

- Use Basic auth with an empty username and the PAT as the password for Azure DevOps.
- Use the centralized request helpers in [internal/api](internal/api) (see [internal/api/client.go](internal/api/client.go)) so authentication, headers and common error wrapping are consistent.
- Ensure `api-version=7.1` is included on calls (helpers like `getWithBase()` add this automatically).

## Data & WIQL

- Return strongly-typed Go structs from API helpers in [internal/api](internal/api).
- Escape WIQL user values with `escapeWIQL()` (double single quotes) before embedding into WIQL strings.
- See `QueryWorkItems()` in [internal/api/work_items.go](internal/api/work_items.go) for an example of secure query construction.

## Error handling

- Wrap errors with context using `fmt.Errorf("context: %w", err)` to preserve the original cause.

## References

- Project guidelines: [AGENTS.md](AGENTS.md)
- Examples: [internal/api/work_items.go](internal/api/work_items.go), [internal/api/client.go](internal/api/client.go)
