---
name: html-to-markdown
description: Use this skill when working with Azure DevOps rich-text HTML fields (Description, Acceptance Criteria, Repro Steps, etc.) that need to be rendered in the terminal UI. Covers the HTML → Markdown → glamour pipeline, model field conventions, and how to extend it to new fields. Trigger when descriptions look like raw HTML, formatting is lost, or a new rich-text field is being added.
---

# HTML-to-Markdown — Azure DevOps Rich-Text Rendering

Azure DevOps returns rich-text fields (`System.Description`, `Microsoft.VSTS.Common.AcceptanceCriteria`, `Microsoft.VSTS.TCM.ReproSteps`, etc.) as **HTML**. This skill documents the pipeline that converts that HTML into styled terminal output.

---

## 1. Pipeline overview

```
Azure DevOps API (HTML)
        │
        ▼
  WorkItem model
  ├── DescriptionHTML        (raw HTML, for rendering)
  ├── Description            (plain text via stripHTML, for search/truncation)
  ├── AcceptanceCriteriaHTML (raw HTML)
  └── AcceptanceCriteria     (plain text)
        │
        ▼
  components.HTMLToMarkdown(html) → Markdown string
        │
        ▼
  components.RenderMarkdown(md, width) → glamour ANSI output (cached)
        │
        ▼
  Terminal (styled headings, bold, lists, links, tables)
```

## 2. Key files

| File                                        | Role                                                                      |
| ------------------------------------------- | ------------------------------------------------------------------------- |
| `internal/components/html_markdown.go`      | Shared `HTMLToMarkdown()` converter using `html-to-markdown/v2`, plus the @mention rule |
| `internal/components/markdown.go`           | `RenderMarkdown()`, `RenderHTML()`, `CommentMarkdown()`; theme-matched glamour style and render cache |
| `internal/models/work_item.go`              | `WorkItem` struct with `*HTML` fields                                     |
| `internal/api/work_items.go`                | `convertWorkItem()` populates both HTML and plain-text fields             |
| `internal/components/panels/details.go`     | Side-panel rendering via `components.RenderHTML()`                        |
| `internal/components/panels/comment_list.go` | Comment bodies via `components.CommentMarkdown()` → `RenderMarkdown()`    |
| `internal/components/panels/detail_view.go` | Fullscreen view rendering, same pipeline                                  |

## 3. Adding a new rich-text field

Example: adding `ReproSteps` for Bug work items.

### Step 1 — API struct

In `workItemFields` (work_items.go):

```go
ReproSteps string `json:"Microsoft.VSTS.TCM.ReproSteps"`
```

### Step 2 — Model

In `WorkItem` (work_item.go):

```go
ReproSteps     string `json:"reproSteps"`
ReproStepsHTML string `json:"reproStepsHTML,omitempty"`
```

### Step 3 — Conversion

In `convertWorkItem()`:

```go
ReproSteps:     stripHTML(item.Fields.ReproSteps),
ReproStepsHTML: item.Fields.ReproSteps,
```

### Step 4 — Rendering

Use the shared helper in any panel:

```go
components.RenderHTML(d.item.ReproStepsHTML, d.item.ReproSteps, wrapWidth)
```

## 4. The converter

`components.HTMLToMarkdown()` uses `JohannesKaufmann/html-to-markdown/v2` with three plugins:

- **base** — fundamental tag handling
- **commonmark** — headings, bold, italic, links, lists, code blocks, blockquotes
- **table** — HTML `<table>` → Markdown table

The converter is initialized once (package-level `init()`) and is safe for concurrent use.

If conversion fails, the function returns the escaped HTML string as a safe fallback.

## 5. Why two fields (plain text + HTML)?

- **Plain text** (`Description`, `AcceptanceCriteria`): used for list-panel truncation, search filtering, and contexts where Markdown rendering is inappropriate.
- **HTML** (`DescriptionHTML`, `AcceptanceCriteriaHTML`): used exclusively for the rendering pipeline in detail panels.

This mirrors the `Comment` model pattern (`Text` + `RenderedText`).

## 6. The rendering helpers

`components.RenderHTML(htmlContent, fallback, width)` in `markdown.go`:

1. If `htmlContent` is non-empty → convert via `HTMLToMarkdown()`.
2. Otherwise → use the plain-text `fallback`.
3. Pass result through `RenderMarkdown()`.

`RenderMarkdown(md, width)` keeps one glamour renderer per width and memoises rendered output per `(width, source)`, so calling it from `Update` on every selection change is cheap. The glamour style is built from the `styles.Color*` palette with no document margin, so output needs no dedenting.

Comments use the same pipeline: `CommentMarkdown(comment)` returns the comment text as Markdown (as-is when `Format` is `markdown`, otherwise `RenderedText`/`Text` converted from HTML). Azure DevOps mentions (`<a data-vss-mention>`) are converted to bold `@Name` by a converter rule registered for `<a>` with `PriorityEarly`.

Images (`<img>`, an `<a>` wrapping only an `<img>`, and `![alt](url)` in Markdown-format comments) are normalised to `![image: <name>](<url>)`, where the name comes from the `fileName` query parameter, the URL path, or the alt text. The glamour style renders them as a short `[image: image.png]` label that is an OSC 8 hyperlink to the full URL (`ImageText.Format = "[{{.text}}]"`, `Image.Format = `{{""}}`` hides the printed URL). Do not restore glamour's default `Image: alt → url` output: the long wrapped URL next to the ambiguous-width `→` corrupts the layout in terminals behind ConPTY (WSL, Windows Terminal, VS Code).

## 7. Troubleshooting

| Symptom                            | Cause                                           | Fix                                                                                          |
| ---------------------------------- | ----------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Raw `<div>`, `<br>` tags visible   | HTML field not wired through `HTMLToMarkdown()` | Check rendering call uses `components.RenderHTML()`                                              |
| No formatting (flat text)          | `*HTML` field is empty                          | Verify `convertWorkItem()` sets the `*HTML` field                                            |
| Tables render as plain text        | Table plugin not registered                     | Ensure `table.NewTablePlugin()` is in the converter                                          |
| Garbled entities (`&amp;`, `&lt;`) | Double-escaping                                 | `HTMLToMarkdown()` handles entities; do not call `stripHTML()` on the HTML before converting |

## 8. Reference links

- [Azure DevOps REST API — Get Work Item (v7.1)](https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/work-items/get-work-item?view=azure-devops-rest-7.1&tabs=HTTP)
- [html-to-markdown v2 (GitHub)](https://github.com/JohannesKaufmann/html-to-markdown)
- [glamour — Markdown terminal renderer](https://github.com/charmbracelet/glamour)
