package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Installation struct {
	Version     string    `json:"version"`
	Method      string    `json:"method"`
	Path        string    `json:"path,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
}

type State struct {
	Tools map[string]Installation `json:"tools"`
}

func Load(path string) (*State, error) {
	s := &State{Tools: map[string]Installation{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, s); err != nil {
		return nil, err
	}
	if s.Tools == nil {
		s.Tools = map[string]Installation{}
	}
	return s, nil
}

func (s *State) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
