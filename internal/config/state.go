package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FilterState holds the persisted filter selections
type FilterState struct {
	Sprint   string `json:"sprint"`
	State    string `json:"state"`
	Assigned string `json:"assigned"`
	Area     string `json:"area"`
}

// getStatePath returns the path to the state file
func getStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazyado", "state.json"), nil
}

// LoadFilterState loads the persisted filter state
func LoadFilterState() (*FilterState, error) {
	statePath, err := getStatePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state if file doesn't exist
			return &FilterState{
				Sprint:   "current",
				State:    "all",
				Assigned: "me",
				Area:     "all",
			}, nil
		}
		return nil, err
	}

	var state FilterState
	if err := json.Unmarshal(data, &state); err != nil {
		// Return default state if file is corrupted
		return &FilterState{
			Sprint:   "current",
			State:    "all",
			Assigned: "me",
			Area:     "all",
		}, nil
	}

	return &state, nil
}

// SaveFilterState saves the filter state to disk
func SaveFilterState(state *FilterState) error {
	statePath, err := getStatePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0600)
}
