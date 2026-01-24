# Huh - Forms and Prompts

> **Build terminal forms and prompts with ease**

[← Back to Bubble Tea](../SKILL.md)

## Overview

Huh is a library for building interactive forms and prompts in the terminal. It provides a high-level API for creating multi-field forms with built-in validation, navigation, and accessibility support.

```
Bubble Tea (Framework)
    └── Huh (Forms) ← You are here
```

## What is Huh?

Huh provides:
- **Multi-field forms** with automatic navigation
- **Built-in validation** for all field types
- **Two usage modes**: standalone (blocking) or integrated with Bubble Tea
- **Accessible mode** for screen readers
- **Field grouping** for multi-page forms
- **Dynamic forms** that change based on user input

Huh is **NOT**:
- A styling library (use [Lip Gloss](./lipgloss.md))
- A collection of individual components (use [Bubbles](./bubbles.md))
- A framework (use [Bubble Tea](../SKILL.md))

## When to Use Huh

Use Huh when you need:
- ✅ Multiple related input fields
- ✅ Form validation
- ✅ Wizard-style interfaces
- ✅ Quick user prompts
- ✅ Structured data collection
- ✅ Multi-page forms

Don't use Huh for:
- ❌ Single input fields (use [Bubbles](./bubbles.md) textinput)
- ❌ Complex custom interactions
- ❌ Non-form UI elements

---

## Two Usage Modes

### Standalone Mode (Blocking)

Quick and simple - blocks until the form is complete:

```go
var name string
var age int

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Title("Name").Value(&name),
        huh.NewSelect[int]().Title("Age").Value(&age).
            Options(
                huh.NewOption("Under 18", 0),
                huh.NewOption("18-30", 1),
                huh.NewOption("30+", 2),
            ),
    ),
)

err := form.Run() // Blocks here
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Hello, %s (age group %d)\n", name, age)
```

### Integrated Mode (Bubble Tea)

Full control - integrate as a `tea.Model`:

```go
type model struct {
    form *huh.Form
}

func initialModel() model {
    var name string
    
    form := huh.NewForm(
        huh.NewGroup(
            huh.NewInput().Title("Name").Value(&name),
        ),
    )
    
    return model{form: form}
}

func (m model) Init() tea.Cmd {
    return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    form, cmd := m.form.Update(msg)
    if f, ok := form.(*huh.Form); ok {
        m.form = f
    }
    
    if m.form.State == huh.StateCompleted {
        return m, tea.Quit
    }
    
    return m, cmd
}

func (m model) View() string {
    if m.form.State == huh.StateCompleted {
        return "Form complete!"
    }
    return m.form.View()
}
```

---

## Field Types

### Input - Single-line Text

```go
var name string

huh.NewInput().
    Title("What's your name?").
    Placeholder("John Doe").
    Prompt("? ").
    Value(&name).
    Validate(func(s string) error {
        if len(s) < 2 {
            return errors.New("name too short")
        }
        return nil
    })
```

**Options:**
```go
input := huh.NewInput()

// Appearance
input.Title("Question")
input.Description("Help text")
input.Placeholder("Type here...")
input.Prompt("> ")

// Behavior
input.Value(&stringVar)
input.CharLimit(100)

// Password mode
input.EchoMode(huh.EchoModePassword)
input.EchoCharacter('•')

// Validation
input.Validate(func(s string) error {
    if !isValid(s) {
        return errors.New("invalid input")
    }
    return nil
})
```

---

### Text - Multi-line Text

```go
var story string

huh.NewText().
    Title("Tell me a story").
    Placeholder("Once upon a time...").
    CharLimit(500).
    Value(&story).
    Validate(func(s string) error {
        if len(s) < 10 {
            return errors.New("story too short")
        }
        return nil
    })
```

**Options:**
```go
text := huh.NewText()

// Appearance
text.Title("Question")
text.Description("Help text")
text.Placeholder("Type here...")

// Behavior
text.Value(&stringVar)
text.CharLimit(1000)

// Editor
text.Editor("vim") // Use external editor

// Validation
text.Validate(validationFunc)
```

---

### Select - Choose One Option

```go
var country string

huh.NewSelect[string]().
    Title("Pick a country").
    Options(
        huh.NewOption("United States", "US"),
        huh.NewOption("Canada", "CA"),
        huh.NewOption("Mexico", "MX"),
        huh.NewOption("Brazil", "BR"),
    ).
    Value(&country)
```

**With Different Types:**
```go
// Integer values
var age int
huh.NewSelect[int]().
    Title("Age group").
    Options(
        huh.NewOption("Under 18", 0),
        huh.NewOption("18-30", 1),
        huh.NewOption("30-50", 2),
        huh.NewOption("50+", 3),
    ).
    Value(&age)

// Struct values
type User struct {
    Name string
    ID   int
}

var user User
huh.NewSelect[User]().
    Options(
        huh.NewOption("Alice", User{"Alice", 1}),
        huh.NewOption("Bob", User{"Bob", 2}),
    ).
    Value(&user)
```

**Options:**
```go
select := huh.NewSelect[string]()

// Appearance
select.Title("Choose")
select.Description("Pick one")

// Options
select.Options(
    huh.NewOption("Display", "value"),
    huh.NewOption("Another", "value2").Selected(true), // Pre-select
)

// Behavior
select.Value(&stringVar)
select.Height(10) // Visible options

// Filtering
select.Filterable(true)
```

---

### MultiSelect - Choose Multiple Options

```go
var toppings []string

huh.NewMultiSelect[string]().
    Title("Choose toppings").
    Options(
        huh.NewOption("Lettuce", "lettuce").Selected(true),
        huh.NewOption("Tomatoes", "tomatoes").Selected(true),
        huh.NewOption("Cheese", "cheese"),
        huh.NewOption("Onions", "onions"),
    ).
    Limit(3). // Max 3 selections
    Value(&toppings)
```

**Options:**
```go
multi := huh.NewMultiSelect[string]()

// Appearance
multi.Title("Choose multiple")
multi.Description("Select all that apply")

// Options
multi.Options(
    huh.NewOption("Option 1", "opt1").Selected(true),
    huh.NewOption("Option 2", "opt2"),
)

// Behavior
multi.Value(&sliceVar)
multi.Limit(5) // Maximum selections
multi.Height(10)

// Filtering
multi.Filterable(true)
```

---

### Confirm - Yes/No

```go
var accept bool

huh.NewConfirm().
    Title("Do you accept?").
    Description("This is a binding agreement").
    Affirmative("Yes!").
    Negative("No.").
    Value(&accept)
```

**Options:**
```go
confirm := huh.NewConfirm()

// Appearance
confirm.Title("Question?")
confirm.Description("Additional info")
confirm.Affirmative("Yes")  // Default: "Yes"
confirm.Negative("No")       // Default: "No"

// Behavior
confirm.Value(&boolVar)
```

---

### FilePicker - Select Files

```go
var filepath string

huh.NewFilePicker().
    Title("Choose a file").
    CurrentDirectory(".").
    AllowedTypes([]string{".go", ".md"}).
    ShowHidden(false).
    Value(&filepath)
```

**Options:**
```go
picker := huh.NewFilePicker()

// Appearance
picker.Title("Select file")
picker.Description("Choose wisely")

// Behavior
picker.Value(&pathVar)
picker.CurrentDirectory("/home/user")
picker.AllowedTypes([]string{".txt", ".md"})
picker.ShowHidden(true)
picker.DirAllowed(true)  // Allow selecting directories
picker.FileAllowed(true) // Allow selecting files
```

---

## Form Structure

### Groups - Multi-page Forms

Forms are divided into groups (pages):

```go
form := huh.NewForm(
    // Page 1
    huh.NewGroup(
        huh.NewInput().Title("Name").Value(&name),
        huh.NewInput().Title("Email").Value(&email),
    ),
    
    // Page 2
    huh.NewGroup(
        huh.NewSelect[string]().Title("Country").Value(&country),
        huh.NewConfirm().Title("Subscribe?").Value(&subscribe),
    ),
    
    // Page 3
    huh.NewGroup(
        huh.NewText().Title("Bio").Value(&bio),
    ),
)
```

**Group Options:**
```go
group := huh.NewGroup(fields...)

// Conditional display
group.WithHide(func() bool {
    return !showGroup // Hide if true
})

// Group title
group.Title("Personal Information")

// Width control
group.WithWidth(80)
```

---

## Validation

### Field Validation

```go
huh.NewInput().
    Title("Email").
    Value(&email).
    Validate(func(s string) error {
        if !strings.Contains(s, "@") {
            return errors.New("must be a valid email")
        }
        return nil
    })
```

### Built-in Validators

```go
import "github.com/charmbracelet/huh"

// Use built-in validators (if available)
huh.NewInput().
    Validate(huh.ValidateEmail())

// Custom validators
func validateAge(s string) error {
    age, err := strconv.Atoi(s)
    if err != nil {
        return errors.New("must be a number")
    }
    if age < 18 {
        return errors.New("must be 18 or older")
    }
    return nil
}

huh.NewInput().
    Title("Age").
    Validate(validateAge)
```

---

## Dynamic Forms

Forms that change based on user input:

```go
var country string
var state string

form := huh.NewForm(
    huh.NewGroup(
        huh.NewSelect[string]().
            Title("Country").
            Options(huh.NewOptions("USA", "Canada", "Mexico")...).
            Value(&country),
        
        // Dynamic field based on country
        huh.NewSelect[string]().
            TitleFunc(func() string {
                switch country {
                case "USA":
                    return "State"
                case "Canada":
                    return "Province"
                default:
                    return "Region"
                }
            }, &country). // Recompute when country changes
            OptionsFunc(func() []huh.Option[string] {
                states := getStatesForCountry(country)
                return huh.NewOptions(states...)
            }, &country). // Recompute when country changes
            Value(&state),
    ),
)
```

### Dynamic Functions

```go
// TitleFunc - dynamic title
field.TitleFunc(func() string {
    return computeTitle()
}, &dependency)

// DescriptionFunc - dynamic description
field.DescriptionFunc(func() string {
    return computeDescription()
}, &dependency)

// OptionsFunc - dynamic options (Select/MultiSelect)
select.OptionsFunc(func() []huh.Option[T] {
    return computeOptions()
}, &dependency)

// WithHideFunc - dynamic visibility
group.WithHideFunc(func() bool {
    return shouldHide()
}, &dependency)
```

**Rules:**
- Function recomputes when dependency changes
- Pass `&variable` as dependency
- Can have multiple dependencies: `func() T {...}, &dep1, &dep2`

---

## Themes

Huh comes with built-in themes:

```go
form := huh.NewForm(groups...).
    WithTheme(huh.ThemeCharm())     // Default
    WithTheme(huh.ThemeDracula())   // Dracula
    WithTheme(huh.ThemeCatppuccin()) // Catppuccin
    WithTheme(huh.ThemeBase16())    // Base16
    WithTheme(huh.ThemeDefault())   // Plain
```

### Custom Theme

```go
import "github.com/charmbracelet/lipgloss"

customTheme := huh.ThemeCharm()

// Customize colors
customTheme.Focused.Base = lipgloss.NewStyle().
    BorderForeground(lipgloss.Color("86"))

customTheme.Focused.Title = lipgloss.NewStyle().
    Foreground(lipgloss.Color("212")).
    Bold(true)

form.WithTheme(customTheme)
```

---

## Accessibility

Huh supports an accessible mode for screen readers:

```go
form := huh.NewForm(groups...).
    WithAccessible(true)
```

**Best Practice:**
```go
accessibleMode := os.Getenv("ACCESSIBLE") != ""

form := huh.NewForm(groups...).
    WithAccessible(accessibleMode)
```

In accessible mode:
- Forms use standard prompts instead of TUI
- Better for screen readers
- Simpler visual output
- Same functionality

---

## Form Options

```go
form := huh.NewForm(groups...)

// Theme
form.WithTheme(huh.ThemeCharm())

// Accessibility
form.WithAccessible(true)

// Width
form.WithWidth(80)

// Show help
form.WithShowHelp(true)

// Show errors
form.WithShowErrors(true)

// Custom key map
form.WithKeyMap(&customKeyMap)
```

---

## Getting Form Values

### Using Value Pointers (Recommended)

```go
var name string
var age int

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Value(&name).Title("Name"),
        huh.NewSelect[int]().Value(&age).Title("Age"),
    ),
)

form.Run()

// Values are automatically updated
fmt.Printf("Name: %s, Age: %d\n", name, age)
```

### Using GetString/GetInt (Alternative)

```go
form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Key("name").Title("Name"),
        huh.NewSelect[int]().Key("age").Title("Age"),
    ),
)

form.Run()

name := form.GetString("name")
age := form.GetInt("age")
```

---

## Form State

```go
form := huh.NewForm(groups...)

// Check state
switch form.State {
case huh.StateNormal:
    // Form is running
case huh.StateCompleted:
    // Form is complete
case huh.StateAborted:
    // Form was cancelled
}

// In Bubble Tea
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ...
    
    if m.form.State == huh.StateCompleted {
        return m, tea.Quit
    }
    
    if m.form.State == huh.StateAborted {
        return m, tea.Quit
    }
    
    return m, nil
}
```

---

## Common Patterns

### Pattern: Login Form

```go
var username string
var password string

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("Username").
            Value(&username).
            Validate(func(s string) error {
                if len(s) < 3 {
                    return errors.New("username too short")
                }
                return nil
            }),
        
        huh.NewInput().
            Title("Password").
            EchoMode(huh.EchoModePassword).
            Value(&password).
            Validate(func(s string) error {
                if len(s) < 8 {
                    return errors.New("password too short")
                }
                return nil
            }),
    ),
)

if err := form.Run(); err != nil {
    log.Fatal(err)
}

// Authenticate with username and password
```

---

### Pattern: Multi-step Wizard

```go
var (
    name     string
    email    string
    country  string
    agreeToS bool
)

form := huh.NewForm(
    // Step 1: Personal Info
    huh.NewGroup(
        huh.NewNote().
            Title("Step 1: Personal Information").
            Description("Tell us about yourself"),
        huh.NewInput().
            Title("Full Name").
            Value(&name),
        huh.NewInput().
            Title("Email").
            Value(&email).
            Validate(validateEmail),
    ).Title("Personal Information"),
    
    // Step 2: Location
    huh.NewGroup(
        huh.NewNote().
            Title("Step 2: Location").
            Description("Where are you located?"),
        huh.NewSelect[string]().
            Title("Country").
            Options(countryOptions...).
            Value(&country),
    ).Title("Location"),
    
    // Step 3: Confirmation
    huh.NewGroup(
        huh.NewNote().
            Title("Step 3: Terms & Conditions").
            Description("Please review and accept"),
        huh.NewConfirm().
            Title("I agree to the Terms of Service").
            Value(&agreeToS).
            Validate(func(b bool) error {
                if !b {
                    return errors.New("must accept terms")
                }
                return nil
            }),
    ).Title("Confirmation"),
)

form.Run()
```

---

### Pattern: Conditional Fields

```go
var (
    hasAccount bool
    username   string
    password   string
    email      string
)

form := huh.NewForm(
    huh.NewGroup(
        huh.NewConfirm().
            Title("Do you have an account?").
            Value(&hasAccount),
    ),
    
    // Login group - only show if hasAccount
    huh.NewGroup(
        huh.NewInput().
            Title("Username").
            Value(&username),
        huh.NewInput().
            Title("Password").
            EchoMode(huh.EchoModePassword).
            Value(&password),
    ).WithHideFunc(func() bool {
        return !hasAccount
    }, &hasAccount),
    
    // Signup group - only show if !hasAccount
    huh.NewGroup(
        huh.NewInput().
            Title("Email").
            Value(&email),
        huh.NewInput().
            Title("Choose Username").
            Value(&username),
        huh.NewInput().
            Title("Choose Password").
            EchoMode(huh.EchoModePassword).
            Value(&password),
    ).WithHideFunc(func() bool {
        return hasAccount
    }, &hasAccount),
)

form.Run()
```

---

## Integration with Other Libraries

### With Bubble Tea

Full integration example:

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/huh"
)

type model struct {
    form         *huh.Form
    formComplete bool
    name         string
}

func initialModel() model {
    var name string
    
    form := huh.NewForm(
        huh.NewGroup(
            huh.NewInput().
                Title("What's your name?").
                Value(&name),
        ),
    )
    
    return model{
        form: form,
        name: name,
    }
}

func (m model) Init() tea.Cmd {
    return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c":
            return m, tea.Quit
        }
    }
    
    // Update form
    form, cmd := m.form.Update(msg)
    if f, ok := form.(*huh.Form); ok {
        m.form = f
    }
    
    // Check if complete
    if m.form.State == huh.StateCompleted {
        m.formComplete = true
    }
    
    return m, cmd
}

func (m model) View() string {
    if m.formComplete {
        return fmt.Sprintf("Hello, %s!\n", m.name)
    }
    
    return m.form.View()
}
```

### With Lip Gloss

Wrap form in styled container:

```go
import (
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/huh"
)

containerStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("86")).
    Padding(1, 2)

func (m model) View() string {
    formView := m.form.View()
    return containerStyle.Render(formView)
}
```

---

## Best Practices

### 1. Use Value Pointers

```go
// ✅ Good - values automatically updated
var name string
huh.NewInput().Value(&name)

// ❌ Less good - requires manual retrieval
huh.NewInput().Key("name")
name := form.GetString("name")
```

### 2. Validate Early

```go
huh.NewInput().
    Title("Email").
    Validate(func(s string) error {
        if !isValidEmail(s) {
            return errors.New("invalid email")
        }
        return nil
    })
```

### 3. Group Related Fields

```go
// ✅ Good - logical grouping
huh.NewForm(
    huh.NewGroup( /* Personal info */ ),
    huh.NewGroup( /* Contact info */ ),
    huh.NewGroup( /* Preferences */ ),
)

// ❌ Bad - everything in one group
huh.NewForm(
    huh.NewGroup( /* 20 fields */ ),
)
```

### 4. Use Dynamic Forms Wisely

```go
// Only use dynamic functions when values actually change
huh.NewSelect[string]().
    OptionsFunc(func() []huh.Option[string] {
        return getOptionsFor(country)
    }, &country) // Recompute when country changes
```

---

## Resources

- **Official Docs**: [pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/huh)
- **Examples**: [Huh Examples](https://github.com/charmbracelet/huh/tree/main/examples)
- **Source Code**: [GitHub Repository](https://github.com/charmbracelet/huh)

## Related Skills

- [← Bubble Tea](../SKILL.md) - The framework foundation
- [Lip Gloss](./lipgloss.md) - Styling and layouts
- [Bubbles](./bubbles.md) - Individual components

---

## Installation

```bash
go get github.com/charmbracelet/huh
```

## Quick Example

```go
import "github.com/charmbracelet/huh"

func main() {
    var name string
    
    huh.NewInput().
        Title("What's your name?").
        Value(&name).
        Run()
    
    fmt.Printf("Hello, %s!\n", name)
}
```