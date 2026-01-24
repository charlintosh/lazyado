# Bubbles - TUI Components

> **Pre-built interactive components for Bubble Tea**

[← Back to Bubble Tea](../SKILL.md)

## Overview

Bubbles is a collection of ready-to-use TUI components (widgets) that work with [Bubble Tea](../SKILL.md). Each component is a complete `tea.Model` with its own state, update logic, and view rendering.

```
Bubble Tea (Framework)
    └── Bubbles (Components) ← You are here
```

## What are Bubbles?

Bubbles provides:
- **Interactive components** ready to use
- Each bubble is a **complete `tea.Model`**
- Built-in **state management** for each component
- **Event handling** for specific interactions
- **Reusable** across different applications

Bubbles is **NOT**:
- A styling library (use [Lip Gloss](./lipgloss.md))
- A form builder (use [Huh](./huh.md))
- A framework (use [Bubble Tea](../SKILL.md))

## When to Use Bubbles

Use Bubbles when you need:
- ✅ A pre-built interactive component
- ✅ Text input or text area fields
- ✅ Selectable lists with filtering
- ✅ Progress indicators or spinners
- ✅ Scrollable content areas
- ✅ Data tables
- ✅ Timers or stopwatches

Don't use Bubbles for:
- ❌ Styling (use [Lip Gloss](./lipgloss.md))
- ❌ Complete forms with validation (use [Huh](./huh.md))
- ❌ Application architecture (use [Bubble Tea](../SKILL.md))

---

## Available Components

| Component | Description | Use Case |
|-----------|-------------|----------|
| [Text Input](#text-input) | Single-line text input | Usernames, search queries |
| [Text Area](#text-area) | Multi-line text input | Comments, descriptions, code |
| [List](#list) | Selectable list with filtering | Menus, file browsers |
| [Table](#table) | Data table with navigation | Data display, selection |
| [Spinner](#spinner) | Loading indicator | Background operations |
| [Progress](#progress-bar) | Progress bar | Downloads, processing |
| [Viewport](#viewport) | Scrollable content area | Long text, logs |
| [Paginator](#paginator) | Page navigation | Multi-page content |
| [Timer](#timer) | Countdown timer | Time limits, deadlines |
| [Stopwatch](#stopwatch) | Count-up timer | Tracking duration |
| [File Picker](#file-picker) | File system navigator | File selection |
| [Help](#help) | Help text viewer | Key bindings display |

---

## Text Input

Single-line text input field with cursor, placeholder, and validation.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/textinput"

type model struct {
    textInput textinput.Model
}

func initialModel() model {
    ti := textinput.New()
    ti.Placeholder = "Enter your name..."
    ti.Focus()
    ti.CharLimit = 50
    ti.Width = 30
    
    return model{
        textInput: ti,
    }
}

func (m model) Init() tea.Cmd {
    return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.textInput, cmd = m.textInput.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return fmt.Sprintf(
        "What's your name?\n\n%s\n\n%s",
        m.textInput.View(),
        "(esc to quit)",
    )
}
```

### Common Options

```go
ti := textinput.New()

// Appearance
ti.Placeholder = "Type here..."
ti.Width = 40
ti.CharLimit = 100

// Behavior
ti.Focus()           // Give focus
ti.Blur()            // Remove focus
ti.SetValue("text")  // Set value programmatically

// Styling
ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

// Password mode
ti.EchoMode = textinput.EchoPassword
ti.EchoCharacter = '•'

// Validation
ti.Validate = func(s string) error {
    if len(s) < 3 {
        return errors.New("too short")
    }
    return nil
}

// Get value
value := ti.Value()
```

---

## Text Area

Multi-line text editor with word wrapping and scrolling.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/textarea"

type model struct {
    textarea textarea.Model
}

func initialModel() model {
    ta := textarea.New()
    ta.Placeholder = "Type your message..."
    ta.Focus()
    
    return model{
        textarea: ta,
    }
}

func (m model) Init() tea.Cmd {
    return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.textarea, cmd = m.textarea.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return fmt.Sprintf(
        "Tell me a story:\n\n%s\n\n%s",
        m.textarea.View(),
        "(ctrl+s to save, esc to quit)",
    )
}
```

### Common Options

```go
ta := textarea.New()

// Size
ta.SetWidth(80)
ta.SetHeight(10)

// Behavior
ta.Focus()
ta.Blur()
ta.SetValue("Initial text\nLine 2")

// Limits
ta.CharLimit = 500
ta.MaxHeight = 20
ta.MaxWidth = 100

// Appearance
ta.Placeholder = "Start typing..."
ta.ShowLineNumbers = true

// Styling
ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("235"))
ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

// Get value
value := ta.Value()
lines := ta.Line() // Current line number
```

---

## List

Interactive list with selection, filtering, and pagination.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/list"

type item struct {
    title       string
    description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

type model struct {
    list list.Model
}

func initialModel() model {
    items := []list.Item{
        item{title: "Raspberry Pi", description: "A small computer"},
        item{title: "Arduino", description: "A microcontroller"},
        item{title: "ESP32", description: "A wireless microcontroller"},
    }
    
    l := list.New(items, list.NewDefaultDelegate(), 0, 0)
    l.Title = "Choose a device"
    
    return model{list: l}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.list.SetWidth(msg.Width)
        m.list.SetHeight(msg.Height)
    
    case tea.KeyMsg:
        if msg.String() == "enter" {
            selected := m.list.SelectedItem().(item)
            fmt.Printf("Selected: %s\n", selected.title)
        }
    }
    
    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.list.View()
}
```

### Common Options

```go
l := list.New(items, delegate, width, height)

// Appearance
l.Title = "My List"
l.SetShowTitle(true)
l.SetShowStatusBar(true)
l.SetShowPagination(true)
l.SetShowHelp(true)
l.SetFilteringEnabled(true)

// Size
l.SetWidth(80)
l.SetHeight(20)

// Selection
selected := l.SelectedItem()
l.Select(3) // Select by index

// Items
l.SetItems(newItems)
l.InsertItem(0, newItem)
l.RemoveItem(2)
items := l.Items()

// Filtering
l.SetFilteringEnabled(true)
l.Filter("search term")
```

### Custom Delegate

```go
type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    i, ok := item.(item)
    if !ok {
        return
    }
    
    str := fmt.Sprintf("%d. %s", index+1, i.Title())
    
    if index == m.Index() {
        str = lipgloss.NewStyle().
            Foreground(lipgloss.Color("170")).
            Render("> " + str)
    }
    
    fmt.Fprint(w, str)
}
```

---

## Table

Data table with navigation and selection.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/table"

type model struct {
    table table.Model
}

func initialModel() model {
    columns := []table.Column{
        {Title: "Name", Width: 20},
        {Title: "Age", Width: 10},
        {Title: "City", Width: 20},
    }
    
    rows := []table.Row{
        {"Alice", "30", "New York"},
        {"Bob", "25", "Los Angeles"},
        {"Charlie", "35", "Chicago"},
    }
    
    t := table.New(
        table.WithColumns(columns),
        table.WithRows(rows),
        table.WithFocused(true),
        table.WithHeight(7),
    )
    
    return model{table: t}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.table, cmd = m.table.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.table.View() + "\n"
}
```

### Common Options

```go
t := table.New(
    table.WithColumns(columns),
    table.WithRows(rows),
    table.WithFocused(true),
    table.WithHeight(10),
)

// Styling
s := table.DefaultStyles()
s.Header = s.Header.
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("240")).
    Bold(true)
s.Selected = s.Selected.
    Foreground(lipgloss.Color("229")).
    Background(lipgloss.Color("57")).
    Bold(false)
t.SetStyles(s)

// Navigation
t.Focus()
t.Blur()
cursor := t.Cursor()
t.SetCursor(5)

// Data
t.SetRows(newRows)
t.SetColumns(newColumns)
selectedRow := t.SelectedRow()

// Size
t.SetWidth(80)
t.SetHeight(20)
```

---

## Spinner

Animated loading indicator.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/spinner"

type model struct {
    spinner spinner.Model
}

func initialModel() model {
    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
    
    return model{spinner: s}
}

func (m model) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return fmt.Sprintf("%s Loading...\n", m.spinner.View())
}
```

### Spinner Types

```go
s := spinner.New()

// Built-in spinners
s.Spinner = spinner.Line      // ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏
s.Spinner = spinner.Dot       // ⣾ ⣽ ⣻ ⢿ ⡿ ⣟ ⣯ ⣷
s.Spinner = spinner.MiniDot   // ⠋ ⠙ ⠚ ⠞ ⠖ ⠦ ⠴ ⠲ ⠳ ⠓
s.Spinner = spinner.Jump      // ⢄ ⢂ ⢁ ⡁ ⡈ ⡐ ⡠
s.Spinner = spinner.Pulse     // █ ▓ ▒ ░
s.Spinner = spinner.Points    // ∙∙∙ ●∙∙ ∙●∙ ∙∙●
s.Spinner = spinner.Globe     // 🌍 🌎 🌏
s.Spinner = spinner.Moon      // 🌑 🌒 🌓 🌔 🌕 🌖 🌗 🌘

// Custom spinner
s.Spinner = spinner.Spinner{
    Frames: []string{"◐", "◓", "◑", "◒"},
    FPS:    time.Second / 10,
}

// Styling
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
```

---

## Progress Bar

Visual progress indicator.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/progress"

type model struct {
    progress progress.Model
    percent  float64
}

func initialModel() model {
    return model{
        progress: progress.New(progress.WithDefaultGradient()),
        percent:  0.0,
    }
}

func (m model) View() string {
    return fmt.Sprintf(
        "\n\n   %s %3.0f%%\n\n",
        m.progress.ViewAs(m.percent),
        m.percent*100,
    )
}
```

### Common Options

```go
prog := progress.New(
    progress.WithDefaultGradient(),
    progress.WithWidth(80),
    progress.WithoutPercentage(), // Hide percentage
)

// Solid color
prog := progress.New(
    progress.WithSolidFill("205"),
)

// Custom gradient
prog := progress.New(
    progress.WithGradient("#FF00FF", "#00FFFF"),
)

// Update progress
prog.SetPercent(0.75) // 75%

// View at specific percentage
view := prog.ViewAs(0.5) // Show at 50%
```

---

## Viewport

Scrollable content area for long text.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/viewport"

type model struct {
    viewport viewport.Model
    ready    bool
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        if !m.ready {
            m.viewport = viewport.New(msg.Width, msg.Height)
            m.viewport.SetContent(longContent)
            m.ready = true
        } else {
            m.viewport.Width = msg.Width
            m.viewport.Height = msg.Height
        }
    }
    
    var cmd tea.Cmd
    m.viewport, cmd = m.viewport.Update(msg)
    return m, cmd
}

func (m model) View() string {
    if !m.ready {
        return "Initializing..."
    }
    return m.viewport.View()
}
```

### Common Options

```go
vp := viewport.New(width, height)

// Content
vp.SetContent("Very long text...")

// Navigation
vp.GotoTop()
vp.GotoBottom()
vp.HalfViewDown()
vp.HalfViewUp()
vp.LineDown(5)
vp.LineUp(3)

// Mouse support
vp.MouseWheelEnabled = true

// High performance mode (for large content)
vp.HighPerformanceRendering = true

// Style
vp.Style = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(1)

// Info
atTop := vp.AtTop()
atBottom := vp.AtBottom()
scrollPercent := vp.ScrollPercent()
```

---

## Paginator

Page navigation for multi-page content.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/paginator"

type model struct {
    paginator paginator.Model
    items     []string
}

func initialModel() model {
    items := make([]string, 100)
    for i := range items {
        items[i] = fmt.Sprintf("Item %d", i+1)
    }
    
    p := paginator.New()
    p.Type = paginator.Dots
    p.PerPage = 10
    p.SetTotalPages(len(items) / p.PerPage)
    
    return model{
        paginator: p,
        items:     items,
    }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.paginator, cmd = m.paginator.Update(msg)
    return m, cmd
}

func (m model) View() string {
    start := m.paginator.Page * m.paginator.PerPage
    end := start + m.paginator.PerPage
    
    s := strings.Join(m.items[start:end], "\n")
    s += "\n\n" + m.paginator.View()
    
    return s
}
```

### Paginator Types

```go
p := paginator.New()

// Dot style (iOS-like)
p.Type = paginator.Dots
// ⚪ ⚫ ⚪

// Arabic numbers
p.Type = paginator.Arabic
// 1 [2] 3 4 5

// Options
p.PerPage = 10
p.SetTotalPages(50)
p.Page = 0 // Current page
p.ActiveDot = "●"
p.InactiveDot = "○"
```

---

## Timer

Countdown timer.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/timer"

type model struct {
    timer timer.Model
}

func initialModel() model {
    return model{
        timer: timer.NewWithInterval(10*time.Second, time.Second),
    }
}

func (m model) Init() tea.Cmd {
    return m.timer.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case timer.TickMsg:
        var cmd tea.Cmd
        m.timer, cmd = m.timer.Update(msg)
        return m, cmd
    }
    
    return m, nil
}

func (m model) View() string {
    return fmt.Sprintf("Time remaining: %s\n", m.timer.View())
}
```

### Common Options

```go
// Create timer
t := timer.NewWithInterval(
    30*time.Second,  // Duration
    time.Second,     // Tick interval
)

// Control
t.Init()   // Start
t.Start()  // Start/Resume
t.Stop()   // Pause
t.Toggle() // Toggle start/stop

// Info
t.Timeout() // Returns true when time is up
t.Running() // Returns true if running
t.Timedout  // Boolean field
```

---

## Stopwatch

Count-up timer.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/stopwatch"

type model struct {
    stopwatch stopwatch.Model
}

func initialModel() model {
    return model{
        stopwatch: stopwatch.NewWithInterval(time.Millisecond * 100),
    }
}

func (m model) Init() tea.Cmd {
    return m.stopwatch.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.stopwatch, cmd = m.stopwatch.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return fmt.Sprintf("Elapsed: %s\n", m.stopwatch.View())
}
```

---

## File Picker

Navigate and select files from the file system.

### Basic Usage

```go
import "github.com/charmbracelet/bubbles/filepicker"

type model struct {
    filepicker filepicker.Model
}

func initialModel() model {
    fp := filepicker.New()
    fp.CurrentDirectory = "."
    
    return model{
        filepicker: fp,
    }
}

func (m model) Init() tea.Cmd {
    return m.filepicker.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" && m.filepicker.DidSelectFile(msg) {
            return m, tea.Quit
        }
    }
    
    var cmd tea.Cmd
    m.filepicker, cmd = m.filepicker.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.filepicker.View()
}
```

### Common Options

```go
fp := filepicker.New()

// Filters
fp.AllowedTypes = []string{".go", ".md"}
fp.ShowHidden = true

// Directory
fp.CurrentDirectory = "/home/user"

// Get selection
if fp.DidSelectFile(msg) {
    selectedPath := fp.SelectedFile
}
```

---

## Help

Display help text for key bindings.

### Basic Usage

```go
import (
    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
    Up   key.Binding
    Down key.Binding
    Quit key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down},
        {k.Quit},
    }
}

var keys = keyMap{
    Up: key.NewBinding(
        key.WithKeys("up", "k"),
        key.WithHelp("↑/k", "move up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("down", "j"),
        key.WithHelp("↓/j", "move down"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
}

type model struct {
    help help.Model
    keys keyMap
}

func initialModel() model {
    return model{
        help: help.New(),
        keys: keys,
    }
}

func (m model) View() string {
    return m.help.View(m.keys)
}
```

---

## Integration with Other Libraries

### With Bubble Tea

Bubbles are designed to work seamlessly with [Bubble Tea](../SKILL.md):

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textinput"
)

type model struct {
    textInput textinput.Model
}

// Each bubble handles its own Init, Update, and View
func (m model) Init() tea.Cmd {
    return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.textInput, cmd = m.textInput.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.textInput.View()
}
```

### With Lip Gloss

Style bubbles with [Lip Gloss](./lipgloss.md):

```go
import (
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/textinput"
)

ti := textinput.New()
ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

// Or wrap the entire view
boxStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(1, 2)

view := boxStyle.Render(ti.View())
```

---

## Best Practices

### 1. Initialize in Init()

```go
func (m model) Init() tea.Cmd {
    return tea.Batch(
        textinput.Blink,
        m.spinner.Tick,
        m.timer.Init(),
    )
}
```

### 2. Handle Window Resize

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.list.SetWidth(msg.Width)
        m.list.SetHeight(msg.Height - 4)
        m.viewport.Width = msg.Width
        m.viewport.Height = msg.Height - 4
    }
    // ... rest of update
}
```

### 3. Compose Multiple Bubbles

```go
type model struct {
    textInput textinput.Model
    list      list.Model
    spinner   spinner.Model
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd
    
    m.textInput, cmd = m.textInput.Update(msg)
    cmds = append(cmds, cmd)
    
    m.list, cmd = m.list.Update(msg)
    cmds = append(cmds, cmd)
    
    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)
    
    return m, tea.Batch(cmds...)
}
```

### 4. Conditional Rendering

```go
func (m model) View() string {
    if m.loading {
        return m.spinner.View() + " Loading..."
    }
    
    if m.showList {
        return m.list.View()
    }
    
    return m.textInput.View()
}
```

---

## Resources

- **Official Docs**: [pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles)
- **Examples**: [Bubbles Examples](https://github.com/charmbracelet/bubbletea/tree/main/examples)
- **Source Code**: [GitHub Repository](https://github.com/charmbracelet/bubbles)

## Related Skills

- [← Bubble Tea](../SKILL.md) - The framework foundation
- [Lip Gloss](./lipgloss.md) - Styling and layouts
- [Huh](./huh.md) - Forms and prompts

---

## Installation

```bash
go get github.com/charmbracelet/bubbles
```

## Quick Example

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textinput"
)

type model struct {
    textInput textinput.Model
}

func main() {
    ti := textinput.New()
    ti.Placeholder = "Enter your name"
    ti.Focus()
    
    p := tea.NewProgram(model{textInput: ti})
    p.Run()
}
```