package templates

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Loader[T] loads, unmarshals, and validates a single JSON config file.
//
// Optional=true means a missing file is non-fatal: caller gets a zero T and a
// log line. Optional=false means missing file returns a wrapped error.
//
// Validate runs after successful unmarshal; nil skips validation.
type Loader[T any] struct {
	Name     string          // for logs and error wrapping, e.g., "monsters"
	Path     string          // relative to assets root; use the *Path constants
	Optional bool            // true: missing file logs warning, returns zero T
	Validate func(*T) error  // post-load validation; nil = none
}

// Load reads and validates the configured JSON file. On success it returns the
// populated T and nil. On error it returns the zero T and a wrapped error.
// Missing-file errors are suppressed only when Optional=true.
func (l Loader[T]) Load() (T, error) {
	var target T
	data, err := os.ReadFile(AssetPath(l.Path))
	if err != nil {
		if os.IsNotExist(err) && l.Optional {
			log.Printf("[templates] %s: file not found at %s, using defaults", l.Name, l.Path)
			return target, nil
		}
		return target, fmt.Errorf("templates: read %s (%s): %w", l.Name, l.Path, err)
	}
	if err := json.Unmarshal(data, &target); err != nil {
		return target, fmt.Errorf("templates: parse %s (%s): %w", l.Name, l.Path, err)
	}
	if l.Validate != nil {
		if err := l.Validate(&target); err != nil {
			return target, fmt.Errorf("templates: validate %s: %w", l.Name, err)
		}
	}
	return target, nil
}
