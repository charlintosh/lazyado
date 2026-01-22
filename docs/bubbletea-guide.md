# The Complete Bubble Tea Architecture Guide

A comprehensive reference for building robust Terminal User Interface applications in Go using Bubble Tea and The Elm Architecture pattern.

## Table of Contents

1. [The Elm Architecture in Bubble Tea](#1-the-elm-architecture-in-bubble-tea)
2. [Project Structure](#2-project-structure)
3. [The Model](#3-the-model)
4. [Messages and Type System](#4-messages-and-type-system)
5. [The Update Function](#5-the-update-function)
6. [Commands: Async Operations](#6-commands-async-operations)
7. [The View Function](#7-the-view-function)
8. [Keybindings](#8-keybindings)
9. [Styling with Lipgloss](#9-styling-with-lipgloss)
10. [Reusable Components](#10-reusable-components)
11. [Layout Management](#11-layout-management)
12. [Testing and Debugging](#12-testing-and-debugging)
13. [Best Practices](#13-best-practices)
14. [Anti-Patterns](#14-anti-patterns)
15. [Advanced Patterns](#15-advanced-patterns)

---

## 1. The Elm Architecture in Bubble Tea

Bubble Tea implements The Elm Architecture, a functional pattern for building user interfaces that originated in the Elm language. This architecture provides a predictable, unidirectional data flow that makes applications easy to reason about and test.

### The MVU Cycle

The Model-View-Update cycle is the core of every Bubble Tea application. The Model holds all application state. The View renders the current state into a string representation. The Update processes messages and returns a new model along with optional commands. Commands perform asynchronous operations and return messages.

```
    ┌──────────┐
    │  Model   │  ← Application State
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │   View   │  ← Renders UI
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │   User   │  ← Interacts
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │ Message  │  ← Event Occurs
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │  Update  │  ← Processes Event
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │ Command  │  ← Async Operations
    └────┬─────┘
         │
         └──────→  Back to Model
```

### Core Principles

The architecture enforces several critical principles that ensure application reliability. State immutability means the model is never modified in place. Instead, Update returns a new model with the desired changes. Unidirectional data flow ensures that data always moves in one direction through the cycle, making state changes predictable. Pure functions are used for Update, meaning it should not perform side effects directly. All I/O operations must go through Commands. The single source of truth principle means all application state lives in the model, with no hidden state in closures or global variables.

### The Three Required Methods

Every Bubble Tea application must implement these three methods on its model:

```go
type Model interface {
    // Init returns the initial command to run
    // Called once when the program starts
    Init() tea.Cmd
    
    // Update handles messages and returns updated state
    // Called for every event in the system
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    
    // View renders the current state to a string
    // Called after every Update
    View() string
}
```

Understanding this cycle is fundamental. When your program starts, Bubble Tea calls Init to get an initial command. The runtime then enters a loop where it processes messages by calling Update, which returns a new model and optional command. After each Update, View is called to render the current state. Commands run asynchronously and send their results back as messages to Update. This cycle continues until the program receives a quit command.

---

## 2. Project Structure

A well-organized project structure is essential for maintainability as your application grows.

### Recommended Directory Layout

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go                 # Entry point only
│
├── internal/
│   ├── app/
│   │   └── app.go                  # Application bootstrap
│   │
│   ├── model/
│   │   ├── model.go                # Root model
│   │   ├── state.go                # State definitions
│   │   └── messages.go             # Custom messages
│   │
│   ├── components/
│   │   ├── list/
│   │   │   ├── list.go
│   │   │   └── delegate.go
│   │   ├── input/
│   │   │   └── input.go
│   │   └── table/
│   │       └── table.go
│   │
│   ├── views/
│   │   ├── main.go
│   │   ├── detail.go
│   │   └── help.go
│   │
│   ├── styles/
│   │   ├── theme.go
│   │   └── colors.go
│   │
│   ├── keys/
│   │   └── keymap.go
│   │
│   └── commands/
│       ├── io.go
│       └── async.go
│
├── pkg/
│   └── domain/                     # Business logic
│       ├── entities.go
│       └── repository.go
│
└── go.mod
```

### Organization Principles

The cmd directory should contain only the main function that initializes and starts the application. Keep this file minimal, typically under twenty lines. The internal directory contains all application-specific code that should not be imported by other projects. Within internal, separate concerns clearly. The model package contains the root application model and state definitions. The components package holds reusable UI components. The views package contains rendering logic for different screens. The styles package centralizes all styling definitions. The keys package manages keyboard bindings. The commands package contains all async operations.

The pkg directory is for code that could potentially be reused by other projects. Keep domain logic separate from UI concerns. Each package should have a single, well-defined responsibility.

### File Size Guidelines

Maintain readable file sizes by keeping individual files under 300 lines when possible. If a file grows beyond this, consider splitting it into multiple files with related functionality. For example, split Update handlers by message type or view state.

---

## 3. The Model

The Model is the heart of your application, containing all state and configuration.

### Basic Model Structure

```go
package model

import (
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    
    "myapp/internal/keys"
    "myapp/internal/styles"
    "myapp/pkg/domain"
)

type Model struct {
    // Application state
    state       ViewState
    err         error
    
    // Terminal dimensions
    width       int
    height      int
    ready       bool
    
    // Bubble components
    list        list.Model
    viewport    viewport.Model
    input       textinput.Model
    
    // Domain data
    items       []domain.Item
    selected    *domain.Item
    
    // Configuration
    keys        keys.KeyMap
    styles      styles.Styles
}
```

### Initialization

```go
func New() Model {
    return Model{
        state:  ViewStateLoading,
        keys:   keys.Default(),
        styles: styles.Default(),
    }
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        fetchInitialData,
        textinput.Blink,
    )
}
```

### Model with Composition

For complex applications, compose multiple models together:

```go
type RootModel struct {
    // Global state
    width  int
    height int
    
    // Sub-models
    list   ListModel
    detail DetailModel
    edit   EditModel
    
    // Active view
    activeView ViewState
}

type ListModel struct {
    list   list.Model
    items  []domain.Item
}

type DetailModel struct {
    viewport viewport.Model
    item     *domain.Item
}
```

---

## 4. Messages and Type System

Messages are the sole mechanism for communicating state changes in Bubble Tea. Strong typing ensures reliability and maintainability.

### Built-in Messages

Bubble Tea provides several standard messages that every application receives:

```go
// Terminal events
tea.WindowSizeMsg  // Terminal resized
tea.KeyMsg         // Key pressed
tea.MouseMsg       // Mouse event

// Focus events
tea.FocusMsg      // Window gained focus
tea.BlurMsg       // Window lost focus

// Control messages
tea.QuitMsg       // Quit requested
```

### Custom Message Types

Define custom messages for application-specific events:

```go
package model

// Data messages
type (
    DataLoadedMsg struct {
        Items []domain.Item
        Count int
    }
    
    DataSavedMsg struct {
        ID      string
        Success bool
    }
    
    ErrorMsg struct {
        Err     error
        Context string
    }
)

// Navigation messages
type (
    NavigateMsg struct {
        To   ViewState
        Data interface{}
    }
    
    BackMsg struct{}
)

// UI messages
type (
    SpinnerTickMsg struct{}
    
    StatusUpdateMsg struct {
        Text string
        Type StatusType
    }
)
```

### Enum-based State

Use typed constants for state management:

```go
type ViewState int

const (
    ViewStateList ViewState = iota
    ViewStateDetail
    ViewStateEdit
    ViewStateHelp
)

func (v ViewState) String() string {
    return [...]string{
        "List",
        "Detail",
        "Edit",
        "Help",
    }[v]
}

type StatusType int

const (
    StatusInfo StatusType = iota
    StatusSuccess
    StatusWarning
    StatusError
)
```

---

## 5. The Update Function

Update is where all state transitions occur. It must be a pure function that processes messages and returns a new model.

### Basic Update Pattern

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd
    
    // Handle global messages first
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true
        
        // Resize components
        m.list.SetSize(msg.Width-4, msg.Height-8)
        m.viewport.Width = msg.Width - 4
        m.viewport.Height = msg.Height - 8
        
        return m, nil
        
    case tea.KeyMsg:
        // Global keybindings
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }
        
        // Delegate to state-specific handler
        return m.handleKeyPress(msg)
        
    case ErrorMsg:
        m.err = msg.Err
        return m, nil
        
    case DataLoadedMsg:
        m.items = msg.Items
        m.list.SetItems(toListItems(msg.Items))
        return m, nil
    }
    
    // Update active components
    switch m.state {
    case ViewStateList:
        m.list, cmd = m.list.Update(msg)
        cmds = append(cmds, cmd)
    case ViewStateDetail:
        m.viewport, cmd = m.viewport.Update(msg)
        cmds = append(cmds, cmd)
    }
    
    return m, tea.Batch(cmds...)
}
```

### Message Routing

Route messages to specific handlers based on application state:

```go
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch m.state {
    case ViewStateList:
        return m.handleListKeys(msg)
    case ViewStateDetail:
        return m.handleDetailKeys(msg)
    case ViewStateEdit:
        return m.handleEditKeys(msg)
    default:
        return m, nil
    }
}

func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        if item, ok := m.list.SelectedItem().(ListItem); ok {
            m.selected = &item.Item
            m.state = ViewStateDetail
            m.viewport.SetContent(formatItem(item.Item))
        }
        return m, nil
        
    case "n":
        m.state = ViewStateEdit
        m.selected = nil
        return m, initEditForm(nil)
        
    case "d":
        if item, ok := m.list.SelectedItem().(ListItem); ok {
            return m, deleteItem(item.Item.ID)
        }
        return m, nil
    }
    
    return m, nil
}
```

### Batching Commands

When multiple commands need to execute, use tea.Batch:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case RefreshMsg:
        return m, tea.Batch(
            fetchData,
            m.spinner.Tick,
            showStatusMsg("Refreshing..."),
        )
    }
    return m, nil
}
```

---

## 6. Commands: Async Operations

Commands are the exclusive mechanism for performing I/O and asynchronous operations in Bubble Tea.

### Command Fundamentals

A command is simply a function with the signature `func() tea.Msg`. Commands run in goroutines managed by the Bubble Tea runtime and return messages when complete:

```go
type tea.Cmd func() tea.Msg
```

### Basic Command Pattern

```go
func fetchData() tea.Msg {
    // Perform I/O operation
    items, err := api.GetItems()
    if err != nil {
        return ErrorMsg{Err: err, Context: "fetching items"}
    }
    
    return DataLoadedMsg{Items: items, Count: len(items)}
}

// In Update:
case RefreshMsg:
    return m, fetchData
```

### Commands with Arguments

To pass arguments to commands, return a closure:

```go
func fetchUser(id int) tea.Cmd {
    return func() tea.Msg {
        user, err := api.GetUser(id)
        if err != nil {
            return ErrorMsg{Err: err, Context: "fetching user"}
        }
        return UserLoadedMsg{User: user}
    }
}

// Usage:
return m, fetchUser(123)
```

### HTTP Requests

Commands are ideal for HTTP operations:

```go
func fetchItems(filter string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(
            context.Background(),
            10*time.Second,
        )
        defer cancel()
        
        url := fmt.Sprintf("https://api.example.com/items?filter=%s", filter)
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            return ErrorMsg{Err: err, Context: "creating request"}
        }
        
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return ErrorMsg{Err: err, Context: "making request"}
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            return ErrorMsg{
                Err:     fmt.Errorf("status %d", resp.StatusCode),
                Context: "unexpected status code",
            }
        }
        
        var items []Item
        if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
            return ErrorMsg{Err: err, Context: "decoding response"}
        }
        
        return ItemsLoadedMsg{Items: items}
    }
}
```

### Timer Commands

For periodic updates, use timer commands:

```go
func tick() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}

// In Update:
case TickMsg:
    m.elapsed++
    return m, tick() // Continue ticking
```

### File I/O Commands

```go
func loadConfig() tea.Cmd {
    return func() tea.Msg {
        data, err := os.ReadFile("config.json")
        if err != nil {
            return ErrorMsg{Err: err, Context: "reading config"}
        }
        
        var config Config
        if err := json.Unmarshal(data, &config); err != nil {
            return ErrorMsg{Err: err, Context: "parsing config"}
        }
        
        return ConfigLoadedMsg{Config: config}
    }
}

func saveData(items []Item) tea.Cmd {
    return func() tea.Msg {
        data, err := json.Marshal(items)
        if err != nil {
            return ErrorMsg{Err: err, Context: "marshaling data"}
        }
        
        if err := os.WriteFile("data.json", data, 0644); err != nil {
            return ErrorMsg{Err: err, Context: "writing file"}
        }
        
        return DataSavedMsg{Success: true}
    }
}
```

### Critical Command Rules

Never use goroutines directly in your application code. Always use commands instead. Commands run in goroutines managed by Bubble Tea with proper panic recovery. Never block in Update. All blocking operations must be in commands. Commands should be pure in the sense that they do not modify the model directly. Always include timeouts for network operations. Handle all errors and return them as error messages.

---

## 7. The View Function

The View function transforms your model into a string that Bubble Tea renders to the terminal.

### Basic View Pattern

```go
func (m Model) View() string {
    if !m.ready {
        return "Loading..."
    }
    
    var content string
    
    switch m.state {
    case ViewStateList:
        content = m.renderList()
    case ViewStateDetail:
        content = m.renderDetail()
    case ViewStateEdit:
        content = m.renderEdit()
    }
    
    header := m.renderHeader()
    footer := m.renderFooter()
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        content,
        footer,
    )
}
```

### Component-based Rendering

Break down complex views into smaller functions:

```go
func (m Model) renderList() string {
    if len(m.items) == 0 {
        return m.styles.EmptyState.Render(
            "No items found\n\nPress 'n' to create one",
        )
    }
    
    return m.styles.Container.Render(
        m.list.View(),
    )
}

func (m Model) renderDetail() string {
    if m.selected == nil {
        return "No item selected"
    }
    
    title := m.styles.Title.Render(m.selected.Title)
    content := m.viewport.View()
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        "",
        content,
    )
}

func (m Model) renderHeader() string {
    left := m.styles.HeaderLeft.Render(
        fmt.Sprintf(" %s ", m.state),
    )
    
    right := m.styles.HeaderRight.Render(
        fmt.Sprintf(" %d items ", len(m.items)),
    )
    
    gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
    if gap < 0 {
        gap = 0
    }
    
    return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderFooter() string {
    keys := m.renderKeyHelp()
    status := m.renderStatus()
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        keys,
        status,
    )
}
```

### Responsive Layout

Calculate sizes based on terminal dimensions:

```go
func (m Model) renderSplitView() string {
    leftWidth := m.width / 3
    rightWidth := m.width - leftWidth - 2
    
    left := m.styles.Panel.
        Width(leftWidth).
        Height(m.height - 4).
        Render(m.sidebar.View())
    
    right := m.styles.Panel.
        Width(rightWidth).
        Height(m.height - 4).
        Render(m.content.View())
    
    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        left,
        m.styles.Divider.Render("│"),
        right,
    )
}
```

---

## 8. Keybindings

Proper keybinding management is essential for good user experience.

### KeyMap Pattern

Define a structured keybinding map:

```go
package keys

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
    // Navigation
    Up    key.Binding
    Down  key.Binding
    Left  key.Binding
    Right key.Binding
    
    // Actions
    Enter  key.Binding
    Back   key.Binding
    Quit   key.Binding
    Help   key.Binding
    
    // Context-specific
    New     key.Binding
    Edit    key.Binding
    Delete  key.Binding
    Refresh key.Binding
}

func Default() KeyMap {
    return KeyMap{
        Up: key.NewBinding(
            key.WithKeys("up", "k"),
            key.WithHelp("↑/k", "up"),
        ),
        Down: key.NewBinding(
            key.WithKeys("down", "j"),
            key.WithHelp("↓/j", "down"),
        ),
        Enter: key.NewBinding(
            key.WithKeys("enter"),
            key.WithHelp("enter", "select"),
        ),
        Back: key.NewBinding(
            key.WithKeys("esc"),
            key.WithHelp("esc", "back"),
        ),
        Quit: key.NewBinding(
            key.WithKeys("q", "ctrl+c"),
            key.WithHelp("q", "quit"),
        ),
        Help: key.NewBinding(
            key.WithKeys("?"),
            key.WithHelp("?", "help"),
        ),
        New: key.NewBinding(
            key.WithKeys("n"),
            key.WithHelp("n", "new"),
        ),
        Edit: key.NewBinding(
            key.WithKeys("e"),
            key.WithHelp("e", "edit"),
        ),
        Delete: key.NewBinding(
            key.WithKeys("d"),
            key.WithHelp("d", "delete"),
        ),
        Refresh: key.NewBinding(
            key.WithKeys("r"),
            key.WithHelp("r", "refresh"),
        ),
    }
}
```

### Context-aware Bindings

Enable or disable keys based on application state:

```go
func (k *KeyMap) SetContext(state ViewState) {
    switch state {
    case ViewStateList:
        k.New.SetEnabled(true)
        k.Edit.SetEnabled(false)
        k.Back.SetEnabled(false)
        
    case ViewStateDetail:
        k.New.SetEnabled(false)
        k.Edit.SetEnabled(true)
        k.Back.SetEnabled(true)
        
    case ViewStateEdit:
        k.New.SetEnabled(false)
        k.Edit.SetEnabled(false)
        k.Delete.SetEnabled(false)
        k.Back.SetEnabled(true)
    }
}
```

### Key Handling

Use the key package for reliable key matching:

```go
import "github.com/charmbracelet/bubbles/key"

func (m Model) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Use key.Matches for complex bindings
    if key.Matches(msg, m.keys.Quit) {
        return m, tea.Quit
    }
    
    if key.Matches(msg, m.keys.Enter) {
        return m.handleEnter()
    }
    
    // String matching for simple cases
    switch msg.String() {
    case "1", "2", "3", "4", "5":
        index, _ := strconv.Atoi(msg.String())
        return m.selectTab(index - 1)
    }
    
    return m, nil
}
```

---

## 9. Styling with Lipgloss

Lipgloss provides CSS-like styling for terminal applications.

### Theme Structure

```go
package styles

import "github.com/charmbracelet/lipgloss"

type Styles struct {
    // Layout
    App       lipgloss.Style
    Container lipgloss.Style
    Panel     lipgloss.Style
    
    // Text
    Title    lipgloss.Style
    Subtitle lipgloss.Style
    Text     lipgloss.Style
    Dimmed   lipgloss.Style
    
    // States
    Selected   lipgloss.Style
    Unselected lipgloss.Style
    Focused    lipgloss.Style
    Blurred    lipgloss.Style
    
    // Feedback
    Success lipgloss.Style
    Error   lipgloss.Style
    Warning lipgloss.Style
    Info    lipgloss.Style
    
    // Components
    StatusBar lipgloss.Style
    Keys      lipgloss.Style
}

func Default() Styles {
    // Colors
    primary := lipgloss.Color("#7D56F4")
    success := lipgloss.Color("#04B575")
    error := lipgloss.Color("#EF4444")
    warning := lipgloss.Color("#F59E0B")
    dimmed := lipgloss.Color("#6B7280")
    
    return Styles{
        App: lipgloss.NewStyle().
            Padding(1, 2),
        
        Container: lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(primary).
            Padding(1, 2),
        
        Title: lipgloss.NewStyle().
            Foreground(lipgloss.Color("#FAFAFA")).
            Background(primary).
            Bold(true).
            Padding(0, 1),
        
        Selected: lipgloss.NewStyle().
            Foreground(primary).
            Bold(true).
            PaddingLeft(2),
        
        Unselected: lipgloss.NewStyle().
            Foreground(dimmed).
            PaddingLeft(2),
        
        Success: lipgloss.NewStyle().
            Foreground(success).
            Bold(true),
        
        Error: lipgloss.NewStyle().
            Foreground(error).
            Bold(true),
        
        StatusBar: lipgloss.NewStyle().
            Foreground(dimmed).
            Background(lipgloss.Color("#1F2937")).
            Padding(0, 1),
    }
}
```

### Responsive Styles

Adapt styles based on terminal size:

```go
func (s *Styles) Resize(width, height int) {
    s.Container = s.Container.
        Width(width - 4).
        Height(height - 8)
    
    s.Panel = s.Panel.
        Width((width / 2) - 4).
        Height(height - 10)
}
```

### Layout Composition

```go
func renderCard(title, content string, styles Styles) string {
    titleBar := styles.Title.
        Width(40).
        Render(title)
    
    contentBox := styles.Container.
        Width(40).
        Height(10).
        Render(content)
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        titleBar,
        contentBox,
    )
}

func renderGrid(items []string, cols int, styles Styles) string {
    var rows []string
    var currentRow []string
    
    for i, item := range items {
        box := styles.Panel.
            Width(20).
            Height(5).
            Render(item)
        
        currentRow = append(currentRow, box)
        
        if (i+1)%cols == 0 || i == len(items)-1 {
            rows = append(rows, lipgloss.JoinHorizontal(
                lipgloss.Top,
                currentRow...,
            ))
            currentRow = nil
        }
    }
    
    return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
```

---

## 10. Reusable Components

Build reusable components by composing Bubbles or creating custom ones.

### Using Bubbles Components

Bubbles provides common components like lists, text inputs, and viewports:

```go
import (
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
)

// Initialize components
func (m Model) initComponents() Model {
    // List
    items := []list.Item{/* ... */}
    m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
    m.list.Title = "Items"
    
    // Text input
    m.input = textinput.New()
    m.input.Placeholder = "Enter text..."
    m.input.Focus()
    
    // Viewport
    m.viewport = viewport.New(80, 20)
    m.viewport.SetContent("Scrollable content...")
    
    return m
}
```

### Custom List Delegate

```go
type ItemDelegate struct {
    styles Styles
}

func (d ItemDelegate) Height() int  { return 2 }
func (d ItemDelegate) Spacing() int { return 1 }
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
    return nil
}

func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    i, ok := item.(Item)
    if !ok {
        return
    }
    
    style := d.styles.Unselected
    if index == m.Index() {
        style = d.styles.Selected
    }
    
    title := style.Render(i.Title())
    desc := d.styles.Dimmed.Render(i.Description())
    
    fmt.Fprintf(w, "%s\n%s", title, desc)
}
```

### Custom Component Pattern

```go
type TabsComponent struct {
    tabs     []string
    active   int
    width    int
    styles   Styles
}

func NewTabs(tabs []string, styles Styles) TabsComponent {
    return TabsComponent{
        tabs:   tabs,
        active: 0,
        styles: styles,
    }
}

func (t TabsComponent) Update(msg tea.Msg) (TabsComponent, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "left", "h":
            if t.active > 0 {
                t.active--
            }
        case "right", "l":
            if t.active < len(t.tabs)-1 {
                t.active++
            }
        }
    }
    return t, nil
}

func (t TabsComponent) View() string {
    var tabs []string
    
    for i, tab := range t.tabs {
        style := t.styles.Unselected
        if i == t.active {
            style = t.styles.Selected
        }
        
        tabs = append(tabs, style.Render(
            fmt.Sprintf(" %s ", tab),
        ))
    }
    
    return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (t TabsComponent) Active() int {
    return t.active
}
```

---

## 11. Layout Management

Managing layouts responsively is critical for TUI applications.

### Responsive Width Calculation

```go
func (m Model) calculateLayout() (leftWidth, rightWidth, contentHeight int) {
    // Account for borders and padding
    const (
        borderWidth = 2
        padding     = 4
    )
    
    availableWidth := m.width - borderWidth - padding
    availableHeight := m.height - 6 // Header + footer + borders
    
    // Split view: 1/3 sidebar, 2/3 content
    leftWidth = availableWidth / 3
    rightWidth = availableWidth - leftWidth - 2 // Divider
    contentHeight = availableHeight
    
    return
}
```

### Centering Content

```go
func centerContent(content string, width, height int) string {
    return lipgloss.Place(
        width,
        height,
        lipgloss.Center,
        lipgloss.Center,
        content,
    )
}

func centerHorizontal(content string, width int) string {
    return lipgloss.Place(
        width,
        1,
        lipgloss.Center,
        lipgloss.Center,
        content,
    )
}
```

### Flexible Grids

```go
func renderFlexGrid(items []string, width int) string {
    const (
        itemWidth  = 20
        itemHeight = 5
        spacing    = 2
    )
    
    cols := (width + spacing) / (itemWidth + spacing)
    if cols < 1 {
        cols = 1
    }
    
    var rows []string
    var row []string
    
    for i, item := range items {
        box := lipgloss.NewStyle().
            Width(itemWidth).
            Height(itemHeight).
            Border(lipgloss.RoundedBorder()).
            Render(item)
        
        row = append(row, box)
        
        if (i+1)%cols == 0 || i == len(items)-1 {
            rowStr := lipgloss.JoinHorizontal(
                lipgloss.Top,
                row...,
            )
            rows = append(rows, rowStr)
            row = nil
        }
    }
    
    return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
```

---

## 12. Testing and Debugging

Testing Bubble Tea applications requires specific strategies.

### Testing Update Logic

```go
func TestUpdate_DataLoaded(t *testing.T) {
    m := New()
    
    msg := DataLoadedMsg{
        Items: []Item{
            {ID: "1", Title: "Test"},
        },
        Count: 1,
    }
    
    newModel, cmd := m.Update(msg)
    m = newModel.(Model)
    
    if len(m.items) != 1 {
        t.Errorf("expected 1 item, got %d", len(m.items))
    }
    
    if cmd != nil {
        t.Error("expected no command")
    }
}

func TestUpdate_Navigation(t *testing.T) {
    m := New()
    m.state = ViewStateList
    
    msg := NavigateMsg{To: ViewStateDetail}
    newModel, _ := m.Update(msg)
    m = newModel.(Model)
    
    if m.state != ViewStateDetail {
        t.Errorf("expected state %v, got %v", ViewStateDetail, m.state)
    }
}
```

### Testing Commands

```go
func TestFetchData_Success(t *testing.T) {
    // Mock the API
    originalFetch := api.GetItems
    defer func() { api.GetItems = originalFetch }()
    
    expectedItems := []Item{{ID: "1", Title: "Test"}}
    api.GetItems = func() ([]Item, error) {
        return expectedItems, nil
    }
    
    // Execute command
    cmd := fetchData()
    msg := cmd()
    
    // Verify result
    dataMsg, ok := msg.(DataLoadedMsg)
    if !ok {
        t.Fatal("expected DataLoadedMsg")
    }
    
    if len(dataMsg.Items) != 1 {
        t.Errorf("expected 1 item, got %d", len(dataMsg.Items))
    }
}

func TestFetchData_Error(t *testing.T) {
    originalFetch := api.GetItems
    defer func() { api.GetItems = originalFetch }()
    
    expectedErr := errors.New("network error")
    api.GetItems = func() ([]Item, error) {
        return nil, expectedErr
    }
    
    cmd := fetchData()
    msg := cmd()
    
    errMsg, ok := msg.(ErrorMsg)
    if !ok {
        t.Fatal("expected ErrorMsg")
    }
    
    if errMsg.Err != expectedErr {
        t.Errorf("expected error %v, got %v", expectedErr, errMsg.Err)
    }
}
```

### Debug Mode

```go
type Model struct {
    // ... other fields
    debug     bool
    debugLog  []string
}

func (m *Model) log(format string, args ...interface{}) {
    if m.debug {
        msg := fmt.Sprintf(format, args...)
        m.debugLog = append(m.debugLog, msg)
        
        // Keep only last 100 entries
        if len(m.debugLog) > 100 {
            m.debugLog = m.debugLog[1:]
        }
    }
}

func (m Model) renderDebugPanel() string {
    if !m.debug {
        return ""
    }
    
    logs := strings.Join(m.debugLog, "\n")
    
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#FF0000")).
        Width(40).
        Height(10).
        Render(logs)
}
```

### Testing with TeaTest

```go
import "github.com/charmbracelet/x/exp/teatest"

func TestApp(t *testing.T) {
    m := New()
    
    tm := teatest.NewTestModel(
        t,
        m,
        teatest.WithInitialTermSize(80, 24),
    )
    
    // Send keypress
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    
    // Wait for expected output
    teatest.WaitFor(
        t, tm.Output(),
        func(bts []byte) bool {
            return strings.Contains(string(bts), "Expected Text")
        },
        teatest.WithCheckInterval(time.Millisecond*100),
        teatest.WithDuration(time.Second*3),
    )
    
    // Verify final state
    finalModel := tm.FinalModel(t).(Model)
    if finalModel.state != ViewStateDetail {
        t.Errorf("expected detail view")
    }
}
```

---

## 13. Best Practices

### State Management

Keep state immutable. Never modify the model in place. Return a new model from Update with the desired changes. Keep all state in the model. Avoid hidden state in closures, global variables, or component-local state. Use typed states with enums rather than strings. Define clear state transitions and document them.

### Message Design

Create specific message types for each event. Avoid using generic messages with type fields. Include all necessary context in messages. Messages should be self-contained. Name messages as past-tense events like DataLoadedMsg, not DataLoadMsg. Use clear, descriptive names that indicate what happened.

### Component Composition

Build small, focused components. Each component should have a single responsibility. Compose components together to build complex UIs. Pass styles and configuration as parameters, not global state. Make components reusable by avoiding hard-coded values.

### Error Handling

Always handle errors from commands. Never panic in Update. Return error messages instead. Provide context with errors so users understand what went wrong. Consider whether errors are fatal or recoverable. Display errors in a user-friendly way.

### Performance

Avoid expensive computations in View. Pre-compute values in Update if possible. Use lazy rendering for large lists. Only render visible items. Cache rendered strings when appropriate. Profile your application to find bottlenecks.

### Code Organization

Keep files small and focused. Group related functionality together. Separate view logic from update logic. Use clear package boundaries. Document public APIs and complex logic.

---

## 14. Anti-Patterns

### Never Do These Things

Never perform I/O operations directly in Update. This blocks the UI and violates the architecture:

```go
// ❌ NEVER DO THIS
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // This blocks the entire application
    data, err := http.Get("https://api.example.com/data")
    m.data = data
    return m, nil
}

// ✅ DO THIS INSTEAD
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, fetchData() // Return a command
}

func fetchData() tea.Cmd {
    return func() tea.Msg {
        data, err := http.Get("https://api.example.com/data")
        if err != nil {
            return ErrorMsg{Err: err}
        }
        return DataLoadedMsg{Data: data}
    }
}
```

Never use goroutines directly:

```go
// ❌ NEVER DO THIS
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    go func() {
        // This can cause race conditions and panics
        m.data = fetchData()
    }()
    return m, nil
}

// ✅ DO THIS INSTEAD
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, fetchDataCmd()
}
```

Never modify the model outside of Update:

```go
// ❌ NEVER DO THIS
func (m *Model) SetState(s ViewState) {
    m.state = s
}

// ✅ DO THIS INSTEAD
type SetStateMsg struct {
    State ViewState
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case SetStateMsg:
        m.state = msg.State
    }
    return m, nil
}
```

Never ignore WindowSizeMsg:

```go
// ❌ NEVER DO THIS - Your UI will break
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tea.WindowSizeMsg:
        // Ignoring this
    }
    return m, nil
}

// ✅ ALWAYS DO THIS
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.resizeComponents()
    }
    return m, nil
}
```

Never hard-code dimensions in View:

```go
// ❌ AVOID THIS
func (m Model) View() string {
    return lipgloss.NewStyle().
        Width(80).  // Hard-coded!
        Height(24). // Hard-coded!
        Render(content)
}

// ✅ DO THIS
func (m Model) View() string {
    return lipgloss.NewStyle().
        Width(m.width - 4).
        Height(m.height - 6).
        Render(content)
}
```

---

## 15. Advanced Patterns

### State Machine Pattern

```go
type State interface {
    Enter(m *Model) tea.Cmd
    Exit(m *Model) tea.Cmd
    Update(msg tea.Msg, m *Model) (tea.Model, tea.Cmd)
    View(m Model) string
}

type ListState struct{}
type DetailState struct{}
type EditState struct{}

func (s ListState) Enter(m *Model) tea.Cmd {
    m.keys.SetContext(ViewStateList)
    return loadData()
}

func (s ListState) Update(msg tea.Msg, m *Model) (tea.Model, tea.Cmd) {
    // Handle list-specific logic
    return *m, nil
}

func (m Model) transitionTo(newState ViewState) (Model, tea.Cmd) {
    // Exit current state
    if cmd := m.currentState.Exit(&m); cmd != nil {
        // Handle exit command
    }
    
    // Enter new state
    m.state = newState
    m.currentState = m.getState(newState)
    cmd := m.currentState.Enter(&m)
    
    return m, cmd
}
```

### Middleware Pattern

```go
type Middleware func(tea.Msg) (tea.Msg, tea.Cmd)

func (m Model) applyMiddleware(msg tea.Msg) (tea.Msg, tea.Cmd) {
    // Logging middleware
    if m.debug {
        m.log("Message: %T", msg)
    }
    
    // Error tracking middleware
    if errMsg, ok := msg.(ErrorMsg); ok {
        m.errorCount++
        m.lastError = errMsg
    }
    
    // Rate limiting middleware
    if _, ok := msg.(RefreshMsg); ok {
        if time.Since(m.lastRefresh) < time.Second {
            return nil, nil // Debounce
        }
        m.lastRefresh = time.Now()
    }
    
    return msg, nil
}
```

### Undo/Redo Pattern

```go
type Model struct {
    // ... other fields
    history    []ModelSnapshot
    historyPos int
}

type ModelSnapshot struct {
    Items    []Item
    Selected *Item
    State    ViewState
}

func (m Model) snapshot() ModelSnapshot {
    return ModelSnapshot{
        Items:    append([]Item{}, m.items...),
        Selected: m.selected,
        State:    m.state,
    }
}

func (m Model) undo() (Model, tea.Cmd) {
    if m.historyPos > 0 {
        m.historyPos--
        snapshot := m.history[m.historyPos]
        m.items = snapshot.Items
        m.selected = snapshot.Selected
        m.state = snapshot.State
    }
    return m, nil
}

func (m Model) redo() (Model, tea.Cmd) {
    if m.historyPos < len(m.history)-1 {
        m.historyPos++
        snapshot := m.history[m.historyPos]
        m.items = snapshot.Items
        m.selected = snapshot.Selected
        m.state = snapshot.State
    }
    return m, nil
}

func (m Model) recordHistory() Model {
    // Truncate future history
    m.history = m.history[:m.historyPos+1]
    
    // Add new snapshot
    m.history = append(m.history, m.snapshot())
    m.historyPos = len(m.history) - 1
    
    // Limit history size
    if len(m.history) > 50 {
        m.history = m.history[1:]
        m.historyPos--
    }
    
    return m
}
```

### Plugin/Extension Pattern

```go
type Plugin interface {
    Name() string
    Init(m *Model) tea.Cmd
    Update(msg tea.Msg, m *Model) tea.Cmd
    View(m Model) string
}

type Model struct {
    // ... other fields
    plugins []Plugin
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    // Let plugins process message
    for _, plugin := range m.plugins {
        if cmd := plugin.Update(msg, &m); cmd != nil {
            cmds = append(cmds, cmd)
        }
    }
    
    // Regular update logic
    // ...
    
    return m, tea.Batch(cmds...)
}
```

### Form Validation Pattern

```go
type Validator interface {
    Validate(value string) error
}

type RequiredValidator struct{}

func (v RequiredValidator) Validate(value string) error {
    if strings.TrimSpace(value) == "" {
        return errors.New("this field is required")
    }
    return nil
}

type EmailValidator struct{}

func (v EmailValidator) Validate(value string) error {
    if !strings.Contains(value, "@") {
        return errors.New("invalid email format")
    }
    return nil
}

type FormField struct {
    Input      textinput.Model
    Validators []Validator
    Error      error
}

func (f *FormField) Validate() bool {
    f.Error = nil
    
    for _, validator := range f.Validators {
        if err := validator.Validate(f.Input.Value()); err != nil {
            f.Error = err
            return false
        }
    }
    
    return true
}

type Form struct {
    Fields map[string]*FormField
}

func (f Form) ValidateAll() bool {
    valid := true
    
    for _, field := range f.Fields {
        if !field.Validate() {
            valid = false
        }
    }
    
    return valid
}
```

---

## Conclusion

Bubble Tea's architecture, based on The Elm Architecture, provides a robust foundation for building terminal applications. By following these patterns and practices, you can create maintainable, testable, and user-friendly TUIs.

Key takeaways:
- Embrace immutability and pure functions
- Use commands for all I/O and async operations
- Keep components small and focused
- Handle all messages explicitly
- Make your UI responsive to terminal size changes
- Test update logic independently from commands
- Build reusable components and compose them together

The unidirectional data flow ensures predictability, while the separation of concerns makes your code easy to understand and maintain. Start simple, follow the patterns, and your applications will scale gracefully as they grow in complexity.