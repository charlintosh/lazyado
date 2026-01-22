package config

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Add config paths
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "lazyado"))
	}
	v.AddConfigPath(".")

	// Set defaults
	v.SetDefault("theme", "default")
	v.SetDefault("defaults.sprint", "current")
	v.SetDefault("defaults.state", "all")
	v.SetDefault("defaults.assigned", "me")

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
	if err := os.MkdirAll(configDir, 0755); err != nil {
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

# Default filters at startup
defaults:
  sprint: "current"      # "current", "all", or specific name
  state: "all"           # "all", "new", "active", "resolved", "closed"
  assigned: "me"         # "all", "me"
`

	return os.WriteFile(configPath, []byte(content), 0600)
}
