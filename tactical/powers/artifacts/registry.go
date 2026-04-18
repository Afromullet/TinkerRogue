package artifacts

import (
	"fmt"
	"game_main/templates"
	"sort"

	"github.com/bytearena/ecs"
)

var behaviorRegistry = map[string]ArtifactBehavior{}

// RegisterBehavior adds a behavior to the global registry.
func RegisterBehavior(b ArtifactBehavior) {
	behaviorRegistry[b.BehaviorKey()] = b
}

// GetBehavior returns the behavior for the given key, or nil.
func GetBehavior(key string) ArtifactBehavior {
	return behaviorRegistry[key]
}

// IsRegisteredBehavior reports whether the given key names a registered
// artifact behavior. Used by the combat log to route messages to the
// [GEAR] prefix vs. the default perk prefix.
func IsRegisteredBehavior(key string) bool {
	_, ok := behaviorRegistry[key]
	return ok
}

// AllBehaviors returns all registered behaviors in deterministic order.
func AllBehaviors() []ArtifactBehavior {
	keys := make([]string, 0, len(behaviorRegistry))
	for k := range behaviorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]ArtifactBehavior, 0, len(keys))
	for _, k := range keys {
		result = append(result, behaviorRegistry[k])
	}
	return result
}

// ActivateArtifact dispatches activation to the registered behavior for the given key.
// Only player-activated behaviors can be triggered this way.
func ActivateArtifact(behavior string, targetSquadID ecs.EntityID, ctx *BehaviorContext) error {
	b := GetBehavior(behavior)
	if b == nil {
		return fmt.Errorf("unknown behavior %q", behavior)
	}
	if !b.IsPlayerActivated() {
		return fmt.Errorf("behavior %q is not player-activated", behavior)
	}
	return b.Activate(ctx, targetSquadID)
}

// CanActivateArtifact returns true if the given artifact behavior's charge is available.
func CanActivateArtifact(behavior string, charges *ArtifactChargeTracker) bool {
	if charges == nil {
		return false
	}
	return charges.IsAvailable(behavior)
}

// ValidateBehaviorCoverage checks that JSON definitions and behavior registrations
// are in sync. Returns the slice of problems found; empty means everything matches.
// Callers decide whether mismatches are fatal (e.g. fail startup in debug builds).
func ValidateBehaviorCoverage() []error {
	var errs []error
	for id, def := range templates.ArtifactRegistry {
		if def.Behavior == "" {
			continue // minor artifact, no behavior expected
		}
		if GetBehavior(def.Behavior) == nil {
			errs = append(errs, fmt.Errorf("artifact %q declares behavior %q but no implementation is registered", id, def.Behavior))
		}
	}
	for key := range behaviorRegistry {
		found := false
		for _, def := range templates.ArtifactRegistry {
			if def.Behavior == key {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("behavior %q is registered but no artifact definition references it", key))
		}
	}
	return errs
}
