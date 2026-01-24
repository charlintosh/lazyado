# Lip Gloss - Styling and Layouts

> **CSS-like styling for terminal applications**

[← Back to Bubble Tea](../SKILL.md)

## Overview

Lip Gloss is a styling library that provides declarative, CSS-like styling for terminal applications. Think of it as the presentation layer for your TUI - it handles **how things look**, not **what they do**.

```
Bubble Tea (Framework)
    └── Lip Gloss (Styling) ← You are here
```

## What is Lip Gloss?

Lip Gloss provides:
- **Text styling** (colors, bold, italic, underline)
- **Block-level formatting** (padding, margin, borders, alignment)
- **Layout utilities** (joining, positioning, sizing)
- **Pre-built components** (tables, lists, trees)
- **Adaptive colors** (automatic color profile detection)

Lip Gloss is **NOT**:
- An interactive component library (use [Bubbles](./bubbles.md))
- A form builder (use [Huh](./huh.md))
- A framework (use [Bubble Tea](../SKILL.md))

## When to Use Lip Gloss

Use Lip Gloss when you need to:
- ✅ Add colors to text
- ✅ Format text blocks with padding/margins
- ✅ Create borders around content
- ✅ Align text or blocks
- ✅ Build layouts (columns, rows)
- ✅ Render tables, lists, or trees
- ✅ Make your TUI visually appealing

Don't use Lip Gloss for:
- ❌ Handling user input
- ❌ Managing application state
- ❌ Interactive components

---

## Core Concepts

### Styles

A `Style` is an immutable set of formatting rules.

```go
import "github.com/charmbracelet/lipgloss"

style := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    Padding(1, 2)

// Apply the style
output := style.Render("Hello, World!")
fmt.Println(output)
```

**Key points:**
- Styles are **immutable** - each method returns a new style
- Use `Render()` to apply the style to text
- Chain methods for readability

---

## Text Styling

### Colors

```go
// True Color (24-bit)
lipgloss.Color("#FF00FF")  // Hex
lipgloss.Color("#F0F")     // Short hex

// ANSI 256 (8-bit)
lipgloss.Color("86")
lipgloss.Color("201")

// ANSI 16 (4-bit)
lipgloss.Color("5")   // Magenta
lipgloss.Color("12")  // Light blue
```

### Foreground and Background

```go
style := lipgloss.NewStyle().
    Foreground(lipgloss.Color("#FAFAFA")).  // Text color
    Background(lipgloss.Color("#7D56F4"))   // Background color
```

### Adaptive Colors

Colors that change based on terminal background:

```go
adaptiveColor := lipgloss.AdaptiveColor{
    Light: "#000000",  // Dark text on light background
    Dark:  "#FFFFFF",  // Light text on dark background
}

style := lipgloss.NewStyle().
    Foreground(adaptiveColor)
```

### Complete Colors

Specify exact values for each color profile:

```go
completeColor := lipgloss.CompleteColor{
    TrueColor: "#0000FF",
    ANSI256:   "86",
    ANSI:      "5",
}

style := lipgloss.NewStyle().
    Foreground(completeColor)
```

### Text Formatting

```go
style := lipgloss.NewStyle().
    Bold(true).
    Italic(true).
    Faint(true).
    Blink(true).
    Strikethrough(true).
    Underline(true).
    Reverse(true)
```

---

## Block-Level Formatting

### Padding

Add space inside the element:

```go
// All sides
style := lipgloss.NewStyle().Padding(2)

// Vertical and horizontal
style := lipgloss.NewStyle().Padding(2, 4)  // 2 top/bottom, 4 left/right

// Top, horizontal, bottom
style := lipgloss.NewStyle().Padding(1, 4, 2)

// All sides individually (clockwise from top)
style := lipgloss.NewStyle().Padding(1, 2, 3, 4)

// Individual sides
style := lipgloss.NewStyle().
    PaddingTop(1).
    PaddingRight(2).
    PaddingBottom(3).
    PaddingLeft(4)
```

### Margin

Add space outside the element:

```go
// Same syntax as padding
style := lipgloss.NewStyle().Margin(2)
style := lipgloss.NewStyle().Margin(2, 4)
style := lipgloss.NewStyle().Margin(1, 4, 2)
style := lipgloss.NewStyle().Margin(1, 2, 3, 4)

// Individual sides
style := lipgloss.NewStyle().
    MarginTop(1).
    MarginRight(2).
    MarginBottom(3).
    MarginLeft(4)
```

### Width and Height

```go
style := lipgloss.NewStyle().
    Width(40).
    Height(10)

// Max dimensions
style := lipgloss.NewStyle().
    MaxWidth(80).
    MaxHeight(24)
```

### Alignment

```go
// Horizontal alignment
style := lipgloss.NewStyle().
    Width(40).
    Align(lipgloss.Left)    // or lipgloss.Center, lipgloss.Right

// Vertical alignment
style := lipgloss.NewStyle().
    Height(10).
    AlignVertical(lipgloss.Top)  // or lipgloss.Center, lipgloss.Bottom

// Numeric alignment (0.0 = left/top, 0.5 = center, 1.0 = right/bottom)
style := lipgloss.NewStyle().
    Align(lipgloss.Position(0.5))
```

---

## Borders

### Border Styles

```go
// Pre-defined border styles
lipgloss.NormalBorder()
lipgloss.RoundedBorder()
lipgloss.ThickBorder()
lipgloss.DoubleBorder()
lipgloss.HiddenBorder()

// Apply border
style := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder())
```

### Border Sides

```go
// All sides
style := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder())

// Specific sides (top, right, bottom, left)
style := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder(), true, false, true, false)  // Top and bottom only

// Individual sides
style := lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderTop(true).
    BorderRight(false).
    BorderBottom(true).
    BorderLeft(false)
```

### Border Colors

```go
style := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("#FAFAFA")).
    BorderBackground(lipgloss.Color("#7D56F4"))

// Individual side colors
style := lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderTopForeground(lipgloss.Color("86")).
    BorderRightForeground(lipgloss.Color("201"))
```

### Custom Borders

```go
customBorder := lipgloss.Border{
    Top:         "─",
    Bottom:      "─",
    Left:        "│",
    Right:       "│",
    TopLeft:     "╭",
    TopRight:    "╮",
    BottomLeft:  "╰",
    BottomRight: "╯",
}

style := lipgloss.NewStyle().
    Border(customBorder)
```

---

## Layout Utilities

### Joining Blocks Horizontally

```go
col1 := lipgloss.NewStyle().Width(20).Render("Column 1")
col2 := lipgloss.NewStyle().Width(20).Render("Column 2")
col3 := lipgloss.NewStyle().Width(20).Render("Column 3")

// Join at bottom edge
row := lipgloss.JoinHorizontal(lipgloss.Bottom, col1, col2, col3)

// Join at center
row := lipgloss.JoinHorizontal(lipgloss.Center, col1, col2, col3)

// Join at top
row := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

// Join at custom position (0.0 = top, 0.5 = center, 1.0 = bottom)
row := lipgloss.JoinHorizontal(0.3, col1, col2, col3)
```

### Joining Blocks Vertically

```go
header := lipgloss.NewStyle().Render("Header")
body := lipgloss.NewStyle().Render("Body")
footer := lipgloss.NewStyle().Render("Footer")

// Join at left edge
page := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

// Join at center
page := lipgloss.JoinVertical(lipgloss.Center, header, body, footer)

// Join at right
page := lipgloss.JoinVertical(lipgloss.Right, header, body, footer)

// Join at custom position (0.0 = left, 0.5 = center, 1.0 = right)
page := lipgloss.JoinVertical(0.2, header, body, footer)
```

### Placing Text in Space

```go
// Center horizontally in 80 columns
centered := lipgloss.PlaceHorizontal(80, lipgloss.Center, text)

// Place at bottom of 30 rows
bottom := lipgloss.PlaceVertical(30, lipgloss.Bottom, text)

// Place in bottom-right of 30x80 space
corner := lipgloss.Place(30, 80, lipgloss.Right, lipgloss.Bottom, text)
```

### Measuring Blocks

```go
rendered := style.Render("Some text")

width := lipgloss.Width(rendered)
height := lipgloss.Height(rendered)

// Or both at once
w, h := lipgloss.Size(rendered)
```

---

## Tables

```go
import "github.com/charmbracelet/lipgloss/table"

rows := [][]string{
    {"Name", "Age", "City"},
    {"Alice", "30", "New York"},
    {"Bob", "25", "Los Angeles"},
    {"Charlie", "35", "Chicago"},
}

t := table.New().
    Border(lipgloss.NormalBorder()).
    BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
    Headers("NAME", "AGE", "CITY").
    Rows(rows[1:]...)

fmt.Println(t)
```

### Styling Tables

```go
var (
    headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
    evenRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
    oddRowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

t := table.New().
    Border(lipgloss.NormalBorder()).
    Headers("NAME", "AGE", "CITY").
    Rows(rows...).
    StyleFunc(func(row, col int) lipgloss.Style {
        switch {
        case row == table.HeaderRow:
            return headerStyle
        case row%2 == 0:
            return evenRowStyle
        default:
            return oddRowStyle
        }
    })
```

---

## Lists

```go
import "github.com/charmbracelet/lipgloss/list"

l := list.New("Apples", "Oranges", "Bananas")
fmt.Println(l)
// • Apples
// • Oranges
// • Bananas
```

### Nested Lists

```go
l := list.New(
    "Fruits",
    list.New("Apples", "Oranges", "Bananas"),
    "Vegetables",
    list.New("Carrots", "Celery", "Kale"),
)
```

### List Enumerators

```go
// Bullet (default)
l := list.New("A", "B", "C").Enumerator(list.Bullet)
// • A
// • B
// • C

// Alphabet
l := list.New("A", "B", "C").Enumerator(list.Alphabet)
// A. A
// B. B
// C. C

// Roman numerals
l := list.New("A", "B", "C").Enumerator(list.Roman)
// I. A
// II. B
// III. C

// Arabic numbers
l := list.New("A", "B", "C").Enumerator(list.Arabic)
// 1. A
// 2. B
// 3. C
```

### Styling Lists

```go
enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

l := list.New("Apples", "Oranges", "Bananas").
    Enumerator(list.Arabic).
    EnumeratorStyle(enumeratorStyle).
    ItemStyle(itemStyle)
```

---

## Trees

```go
import "github.com/charmbracelet/lipgloss/tree"

t := tree.Root(".").
    Child("src").
    Child("main.go").
    Child("utils.go")

fmt.Println(t)
// .
// ├── src
// ├── main.go
// └── utils.go
```

### Nested Trees

```go
t := tree.Root("Project").
    Child("src",
        tree.New().
            Root("frontend").
            Child("index.js").
            Child("app.js"),
        tree.New().
            Root("backend").
            Child("server.go").
            Child("db.go"),
    ).
    Child("README.md")
```

### Tree Enumerators

```go
// Default
t := tree.Root("Root").Enumerator(tree.DefaultEnumerator)

// Rounded
t := tree.Root("Root").Enumerator(tree.RoundedEnumerator)
```

### Styling Trees

```go
rootStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("35"))
itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

t := tree.Root("Project").
    Child("src", "main.go").
    RootStyle(rootStyle).
    ItemStyle(itemStyle).
    EnumeratorStyle(enumeratorStyle)
```

---

## Integration with Bubble Tea

Lip Gloss works seamlessly with [Bubble Tea](../SKILL.md):

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    title string
    items []string
}

var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FAFAFA")).
        Background(lipgloss.Color("#7D56F4")).
        Padding(0, 1)
    
    boxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("63")).
        Padding(1, 2)
)

func (m model) View() string {
    title := titleStyle.Render(m.title)
    
    content := ""
    for _, item := range m.items {
        content += item + "\n"
    }
    
    box := boxStyle.Render(content)
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        "",
        box,
    )
}
```

---

## Common Patterns

### Pattern: Card Component

```go
func renderCard(title, content string) string {
    titleStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("86")).
        Padding(0, 1)
    
    cardStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("63")).
        Padding(1, 2).
        Width(40)
    
    renderedTitle := titleStyle.Render(title)
    renderedContent := cardStyle.Render(content)
    
    return lipgloss.JoinVertical(lipgloss.Left, renderedTitle, renderedContent)
}
```

### Pattern: Two-Column Layout

```go
func renderTwoColumns(left, right string, width int) string {
    colWidth := (width - 4) / 2
    
    leftStyle := lipgloss.NewStyle().
        Width(colWidth).
        Border(lipgloss.RoundedBorder()).
        Padding(1)
    
    rightStyle := lipgloss.NewStyle().
        Width(colWidth).
        Border(lipgloss.RoundedBorder()).
        Padding(1)
    
    leftCol := leftStyle.Render(left)
    rightCol := rightStyle.Render(right)
    
    return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
}
```

### Pattern: Status Bar

```go
func renderStatusBar(left, center, right string, width int) string {
    leftStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("86"))
    
    centerStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("212"))
    
    rightStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("241"))
    
    leftText := leftStyle.Render(left)
    centerText := centerStyle.Render(center)
    rightText := rightStyle.Render(right)
    
    // Calculate spacing
    leftWidth := lipgloss.Width(leftText)
    centerWidth := lipgloss.Width(centerText)
    rightWidth := lipgloss.Width(rightText)
    
    spacing := (width - leftWidth - centerWidth - rightWidth) / 2
    spacer := strings.Repeat(" ", spacing)
    
    return leftText + spacer + centerText + spacer + rightText
}
```

### Pattern: Help Text

```go
func renderHelp(bindings map[string]string) string {
    helpStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("241")).
        Italic(true)
    
    keyStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("212")).
        Bold(true)
    
    items := []string{}
    for key, desc := range bindings {
        item := keyStyle.Render(key) + " " + helpStyle.Render(desc)
        items = append(items, item)
    }
    
    return strings.Join(items, " • ")
}
```

---

## Best Practices

### 1. Define Styles as Constants

```go
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FAFAFA"))
    
    errorStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("9"))
    
    successStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("10"))
)
```

### 2. Use Adaptive Colors

```go
// Good - adapts to terminal background
style := lipgloss.NewStyle().
    Foreground(lipgloss.AdaptiveColor{Light: "#000", Dark: "#FFF"})

// Less good - fixed color
style := lipgloss.NewStyle().
    Foreground(lipgloss.Color("#000"))
```

### 3. Compose Styles

```go
baseStyle := lipgloss.NewStyle().
    Padding(1, 2).
    Border(lipgloss.RoundedBorder())

errorStyle := baseStyle.Copy().
    BorderForeground(lipgloss.Color("9"))

successStyle := baseStyle.Copy().
    BorderForeground(lipgloss.Color("10"))
```

### 4. Measure Before Rendering

```go
// Measure content first
contentWidth := lipgloss.Width(content)
contentHeight := lipgloss.Height(content)

// Then apply appropriate sizing
style := lipgloss.NewStyle().
    Width(contentWidth + 4).  // Add padding
    Height(contentHeight + 2).
    Padding(1, 2)
```

---

## Troubleshooting

### Colors Not Showing

```go
// Check color profile
profile := lipgloss.ColorProfile()
fmt.Println(profile) // TrueColor, ANSI256, ANSI, or Ascii

// Force a profile (for testing)
lipgloss.SetColorProfile(termenv.TrueColor)
```

### Misaligned Text (CJK Characters)

Set the `RUNEWIDTH_EASTASIAN` environment variable:

```bash
export RUNEWIDTH_EASTASIAN=0
```

Or in Go:
```go
import "os"
os.Setenv("RUNEWIDTH_EASTASIAN", "0")
```

---

## Resources

- **Official Docs**: [pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/lipgloss)
- **Examples**: [Lip Gloss Examples](https://github.com/charmbracelet/lipgloss/tree/master/examples)
- **Table Docs**: [Table Package](https://pkg.go.dev/github.com/charmbracelet/lipgloss/table)
- **List Docs**: [List Package](https://pkg.go.dev/github.com/charmbracelet/lipgloss/list)
- **Tree Docs**: [Tree Package](https://pkg.go.dev/github.com/charmbracelet/lipgloss/tree)

## Related Skills

- [← Bubble Tea](../SKILL.md) - The framework foundation
- [Bubbles](./bubbles.md) - Interactive components
- [Huh](./huh.md) - Forms and prompts

---

## Installation

```bash
go get github.com/charmbracelet/lipgloss
```

## Quick Example

```go
import "github.com/charmbracelet/lipgloss"

style := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    Padding(1, 4).
    Border(lipgloss.RoundedBorder())

fmt.Println(style.Render("Hello, Lip Gloss!"))
```