package templates

import "fmt"

// requiredRoles is the single source of truth for "every combat role that must
// have an entry in both aiconfig.json (RoleBehaviors) and powerconfig.json
// (RoleMultipliers)". The node/encounter "required" set is derived dynamically
// from JSON via requiredNodeIDsFrom in validate_node_encounter.go.
var requiredRoles = []string{"Tank", "DPS", "Support"}

// makeRequiredMap returns a map[id]false built from the given id slice, for use
// with markFound / checkRequired.
func makeRequiredMap(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = false
	}
	return m
}

// checkRequired returns an error naming any required entry still missing.
func checkRequired(label string, required map[string]bool) error {
	for key, found := range required {
		if !found {
			return fmt.Errorf("missing required %s: %s", label, key)
		}
	}
	return nil
}

// markFound marks a key as found in a required map, if it exists.
func markFound(required map[string]bool, key string) {
	if _, exists := required[key]; exists {
		required[key] = true
	}
}
