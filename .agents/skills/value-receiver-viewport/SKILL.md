---
name: value-receiver-viewport
description: Use this skill when viewport scroll does not work inside a Bubble Tea panel despite correct key delegation. Covers the value-receiver trap where View() copies lose SetSize mutations, how to pre-size panels in the Update path, and safe string truncation inside content builders. Trigger when a viewport ignores j/k/arrows/mouse-wheel, when ready stays false, or when a slice-bounds panic occurs in a content builder.
---

# Value-Receiver Viewport — Sizing Panels in the Update Path

Proven pattern from fixing the `DetailsPanel` scroll in `lazyado`. Applies to any Bubble Tea app that embeds `viewport.Model` inside a sub-component rendered from a value-receiver `View()`.

---

## 1. The Problem

Bubble Tea's `View() string` is typically a **value receiver** on the root model:

```go
func (a App) View() string { ... }
```

Inside `View()`, calling `a.detailsPanel.SetSize(w, h)` modifies a **copy**. The real `a.detailsPanel` never gets its viewport dimensions set.

### Consequence chain

```
View() copy gets SetSize → viewport.Width = 40 → content builds → ready = true  ✓ (copy)
Real panel:               → viewport.Width = 0  → content skipped → ready = false ✗

Update() checks !d.ready → returns early → ALL scroll input silently dropped
```

The panel **appears** to render correctly (the copy builds content every frame), but scrolling never works because the real model's `ready` flag is permanently `false`.

---

## 2. The Fix — Pre-size in the Update path

Size panels inside `updateSizes()` (or equivalent), which runs on `tea.WindowSizeMsg` via a **pointer receiver** on the real model:

```go
func (a *App) updateSizes() {
    // ... modal sizing ...
    a.updateContentPanelSizes()   // ← NEW: size real panels
    a.updateFocus()
}

func (a *App) updateContentPanelSizes() {
    if a.width == 0 || a.height == 0 {
        return
    }

    // Replicate the same layout math used in renderMainView / renderMinimalView
    availableHeight := a.height - AppHeaderFooterSize

    // Minimal view: single panel fills everything
    if a.width < MinTerminalWidth || a.height < MinTerminalHeight {
        pw := a.width - PanelBorderOffset
        ph := availableHeight - PanelBorderOffset
        a.detailsPanel.SetSize(pw, ph)
        a.commentsPanel.SetSize(pw, ph)
        return
    }

    // Normal view: compute content width, bottom row height, etc.
    contentWidth := computeContentWidth(a.width)
    bottomH := computeBottomRowHeight(availableHeight)

    if !showBottomRow(bottomH, contentWidth) {
        return
    }

    detailsW, commentsW := computePanelWidths(contentWidth)
    a.detailsPanel.SetSize(detailsW, bottomH)
    a.commentsPanel.SetSize(commentsW, bottomH)
}
```

### Key rule

> **Every sub-component that needs valid dimensions in `Update()` must be sized inside a pointer-receiver method triggered by `WindowSizeMsg`, not only inside `View()`.**

`View()` may still call `SetSize` for cosmetic adjustments — use an early-return guard so it doesn't reset scroll position:

```go
func (d *DetailsPanel) SetSize(width, height int) {
    if d.width == width && d.height == height && d.ready {
        return
    }
    d.width = width
    d.height = height
    d.viewport.Width = vpWidth
    d.viewport.Height = d.height - headerLines
    if d.item != nil {
        d.updateViewportContent()
    }
}
```

The `&& d.ready` clause ensures the first real `SetSize` always runs through, while subsequent identical calls from `View()` are no-ops that preserve scroll state.

---

## 3. Safe String Truncation in Content Builders

When building viewport content, never use `d.viewport.Width` directly for slice bounds — the viewport may have width 0 during initialization.

### Bad — panics with negative slice index

```go
func (d *DetailsPanel) writeHeader(b *strings.Builder) {
    title := fmt.Sprintf("#%d %s", d.item.ID, d.item.Title)
    if len(title) > d.viewport.Width-2 {
        title = title[:d.viewport.Width-5] + "..."  // PANIC: slice [:-5]
    }
}
```

### Good — pass wrapWidth computed with a floor

```go
func (d *DetailsPanel) updateViewportContent() {
    if d.item == nil || d.viewport.Width <= 0 {
        return
    }
    wrapWidth := d.viewport.Width - 2
    if wrapWidth < 10 {
        wrapWidth = 10
    }
    d.viewport.SetContent(d.buildContent(wrapWidth))
    d.ready = true
}

func (d *DetailsPanel) writeHeader(b *strings.Builder, wrapWidth int) {
    title := fmt.Sprintf("#%d %s", d.item.ID, d.item.Title)
    if wrapWidth > 3 && len(title) > wrapWidth {
        title = title[:wrapWidth-3] + "..."
    }
    b.WriteString(d.styles.DetailTitle.Render(title))
    b.WriteString("\n\n")
}
```

### Rules

1. `updateViewportContent` guards on `viewport.Width <= 0` and returns early.
2. Compute `wrapWidth` once with a floor (e.g. 10) and pass it down to all helpers.
3. Never index a string with an expression derived from viewport dimensions without a bounds check.

---

## 4. Diagnostic Checklist

When a viewport-based panel doesn't scroll, check these in order:

| #   | Check                                     | How                                                                              |
| --- | ----------------------------------------- | -------------------------------------------------------------------------------- |
| 1   | Is `View()` a value receiver?             | `func (a App) View()` → copies lose mutations                                    |
| 2   | Is `SetSize` only called inside `View()`? | If yes, the real model never gets sized                                          |
| 3   | Is `ready` ever `true` on the real model? | Add `debug.Scope("panel").Debugf("ready=%v vp.W=%d", d.ready, d.viewport.Width)` |
| 4   | Does `Update()` early-return on `!ready`? | `if !d.focused \|\| !d.ready { return d, nil }` blocks everything                |
| 5   | Is `viewport.Height` > 0?                 | A zero-height viewport has `maxYOffset() = 0`, scroll is a no-op                 |
| 6   | Does content exceed viewport height?      | `TotalLineCount() > VisibleLineCount()` must be true for scroll to have effect   |

---

## 5. Viewport Key Handling Reference

The `viewport.DefaultKeyMap()` from `charmbracelet/bubbles` includes:

| Key                      | Action                                            |
| ------------------------ | ------------------------------------------------- |
| `j` / `down`             | Scroll down 1 line                                |
| `k` / `up`               | Scroll up 1 line                                  |
| `f` / `pgdown` / `space` | Page down                                         |
| `b` / `pgup`             | Page up                                           |
| `u` / `ctrl+u`           | Half page up                                      |
| `d` / `ctrl+d`           | Half page down                                    |
| Mouse wheel              | Scroll (when `MouseWheelEnabled = true`, default) |

These only work when `viewport.Update(msg)` is called AND the viewport has `Height > 0` and content taller than the viewport.

---

## 6. Full Pattern — Minimal Scrollable Panel

```go
type MyPanel struct {
    viewport viewport.Model
    content  string
    width    int
    height   int
    ready    bool
    focused  bool
}

func NewMyPanel() MyPanel {
    return MyPanel{
        viewport: viewport.New(0, 0),
    }
}

func (p *MyPanel) SetSize(width, height int) {
    if p.width == width && p.height == height && p.ready {
        return
    }
    p.width = width
    p.height = height
    p.viewport.Width = width - 2  // adjust for border/padding
    p.viewport.Height = height - 2
    if p.viewport.Height < 1 {
        p.viewport.Height = 1
    }
    p.rebuildContent()
}

func (p *MyPanel) SetContent(content string) {
    p.content = content
    p.viewport.GotoTop()
    p.rebuildContent()
}

func (p *MyPanel) rebuildContent() {
    if p.content == "" || p.viewport.Width <= 0 {
        return
    }
    p.viewport.SetContent(p.content)
    p.ready = true
}

func (p *MyPanel) SetFocused(f bool) { p.focused = f }

func (p MyPanel) Update(msg tea.Msg) (MyPanel, tea.Cmd) {
    if !p.focused || !p.ready {
        return p, nil
    }
    var cmd tea.Cmd
    p.viewport, cmd = p.viewport.Update(msg)
    return p, cmd
}

func (p MyPanel) View() string {
    if !p.ready {
        return "Loading..."
    }
    return p.viewport.View()
}
```

And in the root App:

```go
// In updateSizes (pointer receiver, called on WindowSizeMsg):
func (a *App) updateSizes() {
    a.myPanel.SetSize(computedWidth, computedHeight)
}

// In Update (delegates to panel when focused):
case PanelMine:
    newPanel, cmd := a.myPanel.Update(msg)
    a.myPanel = newPanel

// In View (value receiver — SetSize here is cosmetic/redundant):
a.myPanel.SetSize(w, h)  // safe: early-returns if already sized + ready
return a.myPanel.View()
```
