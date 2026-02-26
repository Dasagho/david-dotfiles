package catalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Load parses catalog.toml at path and returns a validated, sorted slice of Programs.
func Load(path string) ([]Program, error) {
	var raw struct {
		Programs map[string]Program `toml:"programs"`
	}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}

	var errs []string
	var programs []Program

	for name, p := range raw.Programs {
		p.Name = name
		var fieldErrs []string
		if p.Repo == "" {
			fieldErrs = append(fieldErrs, "repo is required")
		}
		if p.AssetPattern == "" {
			fieldErrs = append(fieldErrs, "asset_pattern is required")
		}
		// bin is optional — if empty, the user picks binaries interactively at install time
		if len(fieldErrs) > 0 {
			errs = append(errs, fmt.Sprintf("[%s]: %s", name, strings.Join(fieldErrs, ", ")))
			continue
		}
		programs = append(programs, p)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("catalog validation errors:\n%s", strings.Join(errs, "\n"))
	}

	sort.Slice(programs, func(i, j int) bool {
		return programs[i].Name < programs[j].Name
	})

	return programs, nil
}
