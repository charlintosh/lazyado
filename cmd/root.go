package cmd

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	ui "github.com/charlintosh/lazyado/internal/app"
	"github.com/charlintosh/lazyado/internal/config"
)

var ErrConfigCreated = errors.New("default config created; please edit ~/.config/lazyado/config.yaml")

func Execute() error {

	cfg, err := config.Load()
	if err != nil {
		if err := config.CreateDefaultConfig(); err == nil {
			fmt.Fprintln(os.Stderr, "Created default config file at ~/.config/lazyado/config.yaml")
			fmt.Fprintln(os.Stderr, "Please edit the config file with your Azure DevOps settings.")
			return ErrConfigCreated
		}
		return fmt.Errorf("configuration error: %w", err)
	}

	client := api.NewClient(cfg)

	app := ui.NewApp(client)

	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running application: %w", err)
	}

	return nil
}
