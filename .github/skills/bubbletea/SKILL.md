---
name: bubbletea
description: Browse Bubbletea TUI framework documentation and examples. Use when working with Bubbletea components, models, commands, or building terminal user interfaces in Go.
---

# Bubble Tea - TUI Framework

> **The foundation for building terminal user interfaces in Go**

## Overview

Bubble Tea is a complete TUI (Terminal User Interface) framework based on **The Elm Architecture**. It provides the foundational runtime, event loop, and architectural patterns needed to build interactive terminal applications.

## Core Concept

Bubble Tea is **always** the base of your TUI application. Every other library in the Charm ecosystem builds on top of it or integrates with it.

```
┌─────────────────────────────────────┐
│         Bubble Tea (Framework)       │  ← The Foundation
│  ┌───────────┐  ┌────────────────┐  │
│  │ Lip Gloss │  │    Bubbles     │  │  ← Optional Add-ons
│  │ (Styling) │  │ (Components)   │  │
│  └───────────┘  └────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │       Huh (Forms)             │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

## What is Bubble Tea?

Bubble Tea is:
- **An event-driven framework** - handles keyboard, mouse, and terminal events
- **State management** - manages your application's state with a clear pattern
- **The runtime** - executes your application's lifecycle
- **Based on The Elm Architecture** - functional, predictable state updates

Bubble Tea is **NOT**:
- A styling library (use [Lip Gloss](./references/lipgloss.md))
- A collection of components (use [Bubbles](./references/bubbles.md))
- A form builder (use [Huh](./references/huh.md))

## The Elm Architecture (TEA)

Bubble Tea applications follow a simple, three-part pattern:

```go
┌──────────────────────────────────────────┐
│  Model (State)                           │
│  ↓                                       │
│  Init (Initialize)                       │
│  ↓                                       │
│  ┌─────────────────────────────────┐    │
│  │  Event Loop                      │    │
│  │  ┌────────────────────────────┐ │    │
│  │  │ 1. Message arrives (Msg)   │ │    │
│  │  │ 2. Update (handle event)   │ │    │
│  │  │ 3. View (render UI)        │ │    │
│  │  └────────────────────────────┘ │    │
│  └─────────────────────────────────┘    │
└──────────────────────────────────────────┘
```

### 1. Model - Your Application State

The Model holds **all** of your application's state.

```go
type model struct {
    items    []string
    cursor   int
    selected map[int]struct{}
    width    int
    height   int
    // Any data your app needs
}
```

**Rules:**
- Should be a simple data structure (usually a struct)
- Contains **all** state - no global variables
- Immutable - never modified directly, only through Update

---

### 2. Init - Initialization

Returns an optional command to execute when your app starts.

```go
func (m model) Init() tea.Cmd {
    // Return a command to run at startup
    // Or return nil for no initial command
    return nil
}
```

**Common uses:**
- Fetching initial data
- Starting timers
- Reading configuration files
- Initializing sub-components

**Example:**
```go
func (m model) Init() tea.Cmd {
    return tea.Batch(
        fetchInitialData,
        startTimer,
        m.spinner.Tick,
    )
}
```

---

### 3. Update - Event Handling

This is where **all** state changes happen. It receives messages and returns an updated model and optional command.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            if m.cursor < len(m.items)-1 {
                m.cursor++
            }
        case "enter", " ":
            m.selected[m.cursor] = struct{}{}
        }
    
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    
    case myCustomMsg:
        // Handle custom messages
        m.data = msg.data
    }
    
    return m, nil
}
```

**Built-in Message Types:**

| Message Type | Description | Example |
|-------------|-------------|---------|
| `tea.KeyMsg` | Keyboard input | Arrow keys, letters, ctrl+c |
| `tea.MouseMsg` | Mouse events | Clicks, scrolling, movement |
| `tea.WindowSizeMsg` | Terminal resize | Width and height changes |

**Custom Messages:**
```go
type dataLoadedMsg struct {
    data []string
}

type tickMsg time.Time

type errorMsg struct {
    err error
}
```

**Rules:**
- **Pure function** - no side effects
- Returns new model, doesn't mutate existing one
- Can return commands for async operations
- Must handle all message types gracefully

---

### 4. View - Rendering

Returns a string that represents your entire UI.

```go
func (m model) View() string {
    s := "What should we buy?\n\n"
    
    for i, item := range m.items {
        cursor := " "
        if m.cursor == i {
            cursor = ">"
        }
        
        checked := " "
        if _, ok := m.selected[i]; ok {
            checked = "x"
        }
        
        s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, item)
    }
    
    s += "\nPress q to quit.\n"
    return s
}
```

**Bubble Tea handles:**
- When to call View()
- Efficient terminal updates
- Rendering to the screen

**You handle:**
- Building the UI string
- Formatting and layout (often with [Lip Gloss](./references/lipgloss.md))

**Rules:**
- **Pure function** - no side effects
- Returns a string (use [Lip Gloss](./references/lipgloss.md) for styling)
- Called frequently - should be fast
- Never perform I/O or mutations

---

## Commands - Handling Side Effects

Commands are functions that return messages. They enable async operations without blocking Update().

### Basic Command

```go
func fetchData() tea.Msg {
    time.Sleep(2 * time.Second)
    data := getData() // Expensive operation
    return dataLoadedMsg{data: data}
}

// In Update:
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "f" {
            return m, fetchData // Return command
        }
    
    case dataLoadedMsg:
        m.items = msg.data
        m.loading = false
    }
    
    return m, nil
}
```

### Tick Command

For periodic updates (timers, animations):

```go
type tickMsg time.Time

func tick() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tickMsg:
        m.counter++
        return m, tick() // Schedule next tick
    }
    return m, nil
}
```

### Batch Commands

Run multiple commands simultaneously:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, tea.Batch(
        fetchData,
        startTimer,
        m.spinner.Tick,
    )
}
```

### Sequence Commands

Run commands one after another:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, tea.Sequence(
        loadConfig,
        connectToServer,
        fetchInitialData,
    )
}
```

---

## The Runtime - tea.Program

The Program is what executes your application.

### Basic Usage

```go
func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### With Options

```go
func main() {
    p := tea.NewProgram(
        initialModel(),
        tea.WithAltScreen(),       // Use alternate screen buffer
        tea.WithMouseCellMotion(), // Enable mouse support
    )
    
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Common Options

| Option | Description | Use Case |
|--------|-------------|----------|
| `tea.WithAltScreen()` | Use alternate screen buffer | Full-screen apps (like vim, less) |
| `tea.WithMouseCellMotion()` | Track mouse clicks and drags | Interactive elements |
| `tea.WithMouseAllMotion()` | Track all mouse movement | Hover effects |
| `tea.WithInput(io.Reader)` | Custom input source | Testing, SSH |
| `tea.WithOutput(io.Writer)` | Custom output destination | Testing, logging |

---

## Integration with Other Libraries

### With Lip Gloss (Styling)

```go
import "github.com/charmbracelet/lipgloss"

var titleStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA"))

func (m model) View() string {
    title := titleStyle.Render("My App")
    // ... rest of view
    return title + "\n" + content
}
```

See [Lip Gloss reference](./references/lipgloss.md) for more.

---

### With Bubbles (Components)

```go
import "github.com/charmbracelet/bubbles/textinput"

type model struct {
    textInput textinput.Model
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
    return m.textInput.View()
}
```

See [Bubbles reference](./references/bubbles.md) for more.

---

### With Huh (Forms)

```go
import "github.com/charmbracelet/huh"

type model struct {
    form *huh.Form
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    form, cmd := m.form.Update(msg)
    if f, ok := form.(*huh.Form); ok {
        m.form = f
    }
    return m, cmd
}

func (m model) View() string {
    return m.form.View()
}
```

See [Huh reference](./references/huh.md) for more.

---

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    
    tea "github.com/charmbracelet/bubbletea"
)

// 1. Define the model (state)
type model struct {
    choices  []string
    cursor   int
    selected map[int]struct{}
}

// 2. Initial state
func initialModel() model {
    return model{
        choices:  []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},
        selected: make(map[int]struct{}),
    }
}

// 3. Initialize
func (m model) Init() tea.Cmd {
    return nil
}

// 4. Update (handle events)
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
            
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
            
        case "down", "j":
            if m.cursor < len(m.choices)-1 {
                m.cursor++
            }
            
        case "enter", " ":
            _, ok := m.selected[m.cursor]
            if ok {
                delete(m.selected, m.cursor)
            } else {
                m.selected[m.cursor] = struct{}{}
            }
        }
    }
    
    return m, nil
}

// 5. View (render)
func (m model) View() string {
    s := "What should we buy at the market?\n\n"
    
    for i, choice := range m.choices {
        cursor := " "
        if m.cursor == i {
            cursor = ">"
        }
        
        checked := " "
        if _, ok := m.selected[i]; ok {
            checked = "x"
        }
        
        s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
    }
    
    s += "\nPress q to quit.\n"
    
    return s
}

// 6. Run the program
func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

---

## Best Practices

### 1. Keep Update Pure

```go
// ❌ Bad - side effects in Update
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    fmt.Println("Processing message") // Side effect!
    file.Write(data)                  // Side effect!
    return m, nil
}

// ✅ Good - return command for side effects
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, func() tea.Msg {
        file.Write(data) // Side effect in command
        return dataWrittenMsg{}
    }
}
```

### 2. Handle All Message Types

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle keys
    case tea.WindowSizeMsg:
        // Handle resize
    case tea.MouseMsg:
        // Handle mouse
    default:
        // Unknown messages are ok - just ignore
    }
    return m, nil
}
```

### 3. Compose Sub-Models

```go
type model struct {
    header headerModel
    body   bodyModel
    footer footerModel
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd
    
    // Update all sub-models
    m.header, cmd = m.header.Update(msg)
    cmds = append(cmds, cmd)
    
    m.body, cmd = m.body.Update(msg)
    cmds = append(cmds, cmd)
    
    m.footer, cmd = m.footer.Update(msg)
    cmds = append(cmds, cmd)
    
    return m, tea.Batch(cmds...)
}

func (m model) View() string {
    return m.header.View() + "\n" +
           m.body.View() + "\n" +
           m.footer.View()
}
```

### 4. Use Type Switches for Messages

```go
// ✅ Good - type switch
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKey(msg)
    case dataMsg:
        return m.handleData(msg)
    }
    return m, nil
}

// ❌ Bad - type assertions
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if keyMsg, ok := msg.(tea.KeyMsg); ok {
        return m.handleKey(keyMsg)
    }
    if dataMsg, ok := msg.(dataMsg); ok {
        return m.handleData(dataMsg)
    }
    return m, nil
}
```

---

## Common Patterns

### Pattern: Loading State

```go
type model struct {
    loading bool
    data    []string
}

type dataLoadedMsg struct {
    data []string
}

func fetchData() tea.Msg {
    time.Sleep(2 * time.Second)
    return dataLoadedMsg{data: []string{"item1", "item2"}}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "r" {
            m.loading = true
            return m, fetchData
        }
    
    case dataLoadedMsg:
        m.loading = false
        m.data = msg.data
    }
    return m, nil
}

func (m model) View() string {
    if m.loading {
        return "Loading..."
    }
    return fmt.Sprintf("Data: %v", m.data)
}
```

### Pattern: Error Handling

```go
type errorMsg struct {
    err error
}

func fetchData() tea.Msg {
    data, err := loadData()
    if err != nil {
        return errorMsg{err}
    }
    return dataLoadedMsg{data}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case errorMsg:
        m.err = msg.err
        m.loading = false
    case dataLoadedMsg:
        m.data = msg.data
        m.loading = false
        m.err = nil
    }
    return m, nil
}
```

### Pattern: Multiple Views

```go
type view int

const (
    mainView view = iota
    settingsView
    helpView
)

type model struct {
    currentView view
}

func (m model) View() string {
    switch m.currentView {
    case mainView:
        return m.renderMain()
    case settingsView:
        return m.renderSettings()
    case helpView:
        return m.renderHelp()
    default:
        return ""
    }
}
```

---

## Resources

- **Official Tutorial**: [Bubble Tea Basics](https://github.com/charmbracelet/bubbletea/tree/main/tutorials/basics)
- **Examples**: [Official Examples](https://github.com/charmbracelet/bubbletea/tree/main/examples)
- **API Docs**: [pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbletea)
- **Video Tutorial**: [Charm YouTube Channel](https://charm.sh/yt)

## Related Skills

- [Lip Gloss](./references/lipgloss.md) - Styling and layouts
- [Bubbles](./references/bubbles.md) - Pre-built components
- [Huh](./references/huh.md) - Forms and prompts

---

## Installation

```bash
go get github.com/charmbracelet/bubbletea
```

## Quick Start

```go
package main

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
)

type model struct{}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if _, ok := msg.(tea.KeyMsg); ok {
        return m, tea.Quit
    }
    return m, nil
}

func (m model) View() string {
    return "Hello, Bubble Tea! (press any key to exit)"
}

func main() {
    tea.NewProgram(model{}).Run()
}
```