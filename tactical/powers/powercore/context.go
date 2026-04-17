// Package powercore provides the shared foundation for the artifacts and perks
// power systems: a common runtime context, a unified logger interface, and an
// ordered event pipeline used by CombatService to fan out lifecycle events.
//
// Both artifacts.BehaviorContext and perks.HookContext embed *PowerContext so
// that cross-cutting dependencies (entity manager, query cache, round number,
// logger) live in exactly one place. Package-specific fields (ChargeTracker on
// artifacts, RoundState on perks) remain on the embedding struct.
package powercore

import (
	"game_main/common"
	"game_main/tactical/combat/combatstate"
)

// PowerContext bundles the runtime dependencies that every artifact behavior
// and every perk hook needs. It is constructed once per dispatch and passed
// by pointer so helper mutations (logging, cache reads) are visible to all
// hooks in the chain.
type PowerContext struct {
	Manager     *common.EntityManager
	Cache       *combatstate.CombatQueryCache
	RoundNumber int
	Logger      PowerLogger
}

// NewPowerContext creates a PowerContext. Logger may be nil; logging calls
// through the context will silently no-op when no logger is set, which keeps
// tests and headless runs quiet.
func NewPowerContext(manager *common.EntityManager, cache *combatstate.CombatQueryCache, roundNumber int, logger PowerLogger) *PowerContext {
	return &PowerContext{
		Manager:     manager,
		Cache:       cache,
		RoundNumber: roundNumber,
		Logger:      logger,
	}
}
