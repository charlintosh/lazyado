package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charlintosh/lazyado/internal/models"
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Organization string   `mapstructure:"organization"`
	Project      string   `mapstructure:"project"`
	Team         string   `mapstructure:"team"`
	PAT          string   `mapstructure:"pat"`
	Theme        string   `mapstructure:"theme"`
	Defaults     Defaults `mapstructure:"defaults"`
	Diff         Diff     `mapstructure:"diff"`
	Boards       Boards   `mapstructure:"boards"`
	UI           UI       `mapstructure:"ui"`
}

type UI struct {
	Splash string `mapstructure:"splash"`
}

type Boards struct {
	MaxWorkItems int `mapstructure:"max_work_items"`
}

func (c *Config) MaxWorkItems() int {
	if c.Boards.MaxWorkItems <= 0 {
		return models.DefaultMaxWorkItems
	}
	return c.Boards.MaxWorkItems
}

// Diff holds the limits for showing pull request diffs
type Diff struct {
	MaxChangedLines int      `mapstructure:"max_changed_lines"`
	MaxFileLines    int      `mapstructure:"max_file_lines"`
	MaxFileKB       int      `mapstructure:"max_file_kb"`
	Collapse        []string `mapstructure:"collapse"`
}

// DiffLimits returns the diff limits with defaults applied
func (c *Config) DiffLimits() models.DiffLimits {
	return models.DiffLimits{MaxChangedLines: c.Diff.MaxChangedLines, MaxFileLines: c.Diff.MaxFileLines, MaxFileKB: c.Diff.MaxFileKB, Collapse: c.Diff.Collapse}.WithDefaults()
}

// Defaults holds default filter settings
type Defaults struct {
	Sprint   string `mapstructure:"sprint"`
	State    string `mapstructure:"state"`
	Assigned string `mapstructure:"assigned"`
}

// Load loads the configuration from file and environment
func Load() (*Config, error) {
	v := viper.New()

	// Set config file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locating home directory: %w", err)
	}
	v.AddConfigPath(filepath.Join(home, ".config", "lazyado"))

	// Set defaults
	v.SetDefault("theme", "default")
	v.SetDefault("defaults.sprint", "current")
	v.SetDefault("defaults.state", "all")
	v.SetDefault("defaults.assigned", "me")
	v.SetDefault("diff.max_changed_lines", models.DefaultMaxChangedLines)
	v.SetDefault("diff.max_file_lines", models.DefaultMaxFileLines)
	v.SetDefault("diff.max_file_kb", models.DefaultMaxFileKB)
	v.SetDefault("boards.max_work_items", models.DefaultMaxWorkItems)
	v.SetDefault("ui.splash", "animated")

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Bind environment variables
	v.SetEnvPrefix("")
	v.AutomaticEnv()

	// Map environment variables
	v.BindEnv("pat", "AZURE_DEVOPS_PAT")
	v.BindEnv("organization", "AZURE_DEVOPS_ORG")
	v.BindEnv("project", "AZURE_DEVOPS_PROJECT")
	v.BindEnv("team", "AZURE_DEVOPS_TEAM")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate required fields
	if cfg.Organization == "" {
		return nil, fmt.Errorf("organization is required (set in config or AZURE_DEVOPS_ORG)")
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("project is required (set in config or AZURE_DEVOPS_PROJECT)")
	}
	if cfg.Team == "" {
		return nil, fmt.Errorf("team is required (set in config or AZURE_DEVOPS_TEAM)")
	}
	if cfg.PAT == "" {
		return nil, fmt.Errorf("PAT is required (set in config or AZURE_DEVOPS_PAT)")
	}

	return &cfg, nil
}

// BaseURL returns the Azure DevOps API base URL
func (c *Config) BaseURL() string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s/_apis", c.Organization, c.Project)
}

func (c *Config) ReleaseURL() string {
	return fmt.Sprintf("https://vsrm.dev.azure.com/%s/%s/_apis", c.Organization, c.Project)
}

// TeamURL returns the Azure DevOps API URL for team-specific endpoints
func (c *Config) TeamURL() string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s/%s/_apis", c.Organization, c.Project, c.Team)
}

// WebURL returns the Azure DevOps web URL for the project
func (c *Config) WebURL() string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s", c.Organization, c.Project)
}

// CreateDefaultConfig creates a default config file
func CreateDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "lazyado")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	content := `# Azure DevOps connection
organization: "my-organization"
project: "my-project"
team: "my-team"

# Authentication
# PAT can be set here or via environment variable AZURE_DEVOPS_PAT
pat: ""

# UI settings
theme: "default"  # default, dark, light
ui:
  splash: "animated"  # animated, static, off

# Default filters at startup
defaults:
  sprint: "current"      # "current", "all", or specific name
  state: "all"           # "all", "new", "active", "resolved", "closed"
  assigned: "me"         # "all", "me"
`

	return os.WriteFile(configPath, []byte(content), 0600)
}
