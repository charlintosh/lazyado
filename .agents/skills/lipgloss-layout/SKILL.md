---
name: lipgloss-layout
description: Use this skill when working on TUI layout with lipgloss and bubbles/viewport in Go. Covers the Width/Height border-additive math, using viewport for overflow clipping instead of manual line counting, status bar single-line enforcement, and responsive breakpoint patterns. Trigger when panels overflow, content gets cut off, or layout dimensions seem wrong.
---

# Lipgloss Layout — Overflow, Clipping & Sizing

Patrones confirmados trabajando en `lazyado`. Aplican a cualquier TUI bubbletea + lipgloss.

---

## 1. Regla fundamental: Width/Height son aditivos con border

```
lipgloss.Style con Border:
  Width(W)  → outer width  = W + 2
  Height(H) → outer height = H + 2

lipgloss.Style sin Border:
  Width(W)  → outer width  = W  (exacto)
  Height(H) → outer height = H  (exacto, pero es mínimo — no clipea)
```

**Todos los `SetSize(width, height)` reciben dimensiones de contenido; el border se agrega externamente.**

Ejemplo de height total correcto para el layout principal:

```
1 (headerBar, sin border)
+ availableHeight (panels con border, Height = availableHeight-2)
+ 1 (statusBar, sin border)
= a.height ✓
```

---

## 2. Overflow clipping → usar `viewport.Model`, NO líneas manuales

### ❌ Anti-patrón: recorte manual de líneas

```go
// MAL — frágil, hardcodea offsets, se rompe al resize
lines := strings.Split(desc, "\n")
availableLines := d.height - 15
if len(lines) > availableLines {
    lines = lines[:availableLines]
    lines = append(lines, "...")
}
```

### ✅ Patrón correcto: viewport como clip region

```go
type DetailsPanel struct {
    viewport viewport.Model
    ready    bool
    // ...
}

func (d *DetailsPanel) SetSize(width, height int) {
    // PanelInactive: Border (+2 outer) + Padding(0,1) (1 char cada lado)
    d.viewport.Width  = width - 2   // inner text width
    d.viewport.Height = height      // no vertical padding → height exacto
    if d.item != nil {
        d.updateViewportContent()
    }
}

func (d *DetailsPanel) updateViewportContent() {
    // Construir TODO el contenido sin preocuparse por altura
    var b strings.Builder
    // ... escribir todo el contenido ...
    d.viewport.SetContent(b.String())
    d.ready = true
}

func (d DetailsPanel) View() string {
    // El viewport clipea automáticamente a viewport.Height líneas
    return d.styles.PanelInactive.
        Width(d.width).
        Height(d.height).
        Render(d.viewport.View())
}
```

**El viewport clipa automáticamente al `Height` asignado. No hay que contar líneas.**

### Math del viewport según el estilo del panel

| Estilo del panel | Padding horizontal | viewport.Width | viewport.Height |
|---|---|---|---|
| `Padding(0, 1)` + border | 1 char cada lado | `width - 2` | `height` |
| `Padding(0, 2)` + border | 2 chars cada lado | `width - 4` | `height` |
| Sin padding + border | 0 | `width` | `height` |

---

## 3. Status bar siempre 1 línea

`lipgloss.Height(1)` solo pone un **mínimo** — el contenido sigue wrapeando si es más ancho que el terminal. Esto empuja el header fuera del alt-screen.

### ❌ No funciona

```go
return style.Width(a.width).Height(1).Render(text) // Height(1) es mínimo, no clipea
```

### ✅ Truncación ANSI-aware con muesli/reflow

```go
import "github.com/muesli/reflow/truncate"

func (a *App) renderStatusBar() string {
    text := strings.Join(a.statusBarParts(), "  ")
    text = truncate.StringWithTail(text, uint(a.width), "…")
    return a.styles.StatusBar.Width(a.width).Render(text)
}
```

`truncate.StringWithTail` preserva secuencias ANSI (colores, bold) y trunca al ancho exacto en runes.

---

## 4. Layout responsivo — breakpoints recomendados

```go
// Ocultar panel de filtros en terminales angostas
showFilter := a.width >= 55

// Ocultar fila inferior (details/comments) cuando no hay altura suficiente
showBottomRow := rawBottomH >= 7 && contentWidth >= 40

// Ocultar panel de comments cuando el ancho es insuficiente
showComments := showBottomRow && contentWidth >= 55
```

Principio: **degradar gracefully** de 4 paneles → 3 → 2 → 1 en lugar de mostrar contenido roto.

---

## 5. Verificación de altura total

Antes de hacer cambios al layout, verificar que la suma de alturas no exceda `a.height`:

```
Single-row (sin bottom):
  workItemsHeight = availableHeight - 2  → outer = availableHeight ✓

Two-row (con bottom):
  workItemsHeight + bottomRowHeight = availableHeight - 4
  → outer combined = availableHeight ✓

Total: header(1) + panels(availableHeight) + statusBar(1) = a.height ✓
```

---

## 6. Cuándo agregar Update() al panel

Un panel con viewport necesita `Update(msg tea.Msg)` **solo si el usuario puede scrollear en él** (es decir, si el panel tiene foco):

```go
// En app.go Update(), solo cuando el panel tiene foco:
case PanelDetails:
    newDetails, cmd := a.detailsPanel.Update(msg)
    a.detailsPanel = newDetails
    // ...
```

Si el panel es solo display (sin foco de teclado), el viewport puede usarse **solo para clipping** sin wiring de Update — `viewport.View()` siempre muestra desde la posición actual (default: top).
