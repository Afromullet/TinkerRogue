package perks

import (
	"github.com/bytearena/ecs"
)

// PerkSlotData stores equipped perks on a squad entity.
// Number of available slots scales with squad progression.
type PerkSlotData struct {
	PerkIDs []string // Equipped perk IDs (max based on squad level)
}

// PerkRoundState tracks combat state needed by perks.
//
// Shared tracking fields live directly on this struct — they are set by the
// dispatch layer (combat_power_dispatch.go) and read by multiple perks.
//
// Per-perk state lives in the PerkState map, keyed by perk ID. Each perk
// defines its own small state struct. This prevents the struct from growing
// a new field for every stateful perk and makes reset automatic.
//
// Shared field lifecycle:
//
//	Field                 Writer                               Reader                          Reset
//	MovedThisTurn         combat_power_dispatch.go OnMoveComplete       stalwart, fortify               ResetPerkRoundStateTurn
//	AttackedThisTurn      combat_power_dispatch.go OnAttackComplete     ResetPerkRoundStateTurn (snapshot)          ResetPerkRoundStateTurn
//	WasAttackedThisTurn   combat_power_dispatch.go OnAttackComplete     ResetPerkRoundStateTurn (snapshot)          ResetPerkRoundStateTurn
//	TurnsStationary       fortify (TurnStart), OnMoveComplete   fortify (CoverMod)              OnMoveComplete (set to 0)
//	WasAttackedLastTurn   ResetPerkRoundStateTurn (snapshot from WasAttackedThisTurn)  counterpunch (TurnStart)  Overwritten each turn
//	DidNotAttackLastTurn  ResetPerkRoundStateTurn (snapshot)               counterpunch (TurnStart)         Overwritten each turn
//	WasIdleLastTurn       ResetPerkRoundStateTurn (snapshot)               deadshots_patience (TurnStart)   Overwritten each turn
type PerkRoundState struct {
	// ---- Shared tracking (set by dispatch layer, read by multiple perks) ----

	// Per-turn (cleared by ResetPerkRoundStateTurn)
	MovedThisTurn       bool // Set by OnMoveComplete. Read by Stalwart, Fortify.
	AttackedThisTurn    bool // Set by OnAttackComplete. Read by ResetPerkRoundStateTurn snapshots.
	WasAttackedThisTurn bool // Set by OnAttackComplete (defender). Snapshotted into WasAttackedLastTurn.

	// Turn-boundary snapshots (computed by ResetPerkRoundStateTurn from previous turn)
	WasAttackedLastTurn  bool // Snapshot of previous turn. Read by Counterpunch.
	DidNotAttackLastTurn bool // Snapshot of previous turn. Read by Counterpunch.
	WasIdleLastTurn      bool // Snapshot of previous turn. Read by Deadshot's Patience.

	// Movement-gated (accumulates across rounds, resets on movement)
	TurnsStationary int // Set by Fortify TurnStart + OnMoveComplete. Read by Fortify CoverMod.

	// ---- Per-perk isolated state ----

	// PerkState holds per-perk state structs, keyed by perk ID.
	// Each perk defines its own state struct and accesses it via GetPerkState/SetPerkState.
	// Cleared entirely by ResetPerRound; per-battle state uses PerkBattleState instead.
	PerkState map[string]any

	// PerkBattleState holds per-perk state that persists the entire combat.
	// Never reset during combat; cleaned up by CleanupRoundState.
	PerkBattleState map[string]any
}

// ---- Shared state map helpers ----

// getFromMap retrieves a typed value from a string-keyed map, returning zero value if missing or wrong type.
func getFromMap[T any](m map[string]any, key string) T {
	var zero T
	if m == nil {
		return zero
	}
	v, ok := m[key]
	if !ok {
		return zero
	}
	typed, ok := v.(T)
	if !ok {
		return zero
	}
	return typed
}

// setInMap stores a value in a string-keyed map, lazily initializing the map if nil.
func setInMap(m *map[string]any, key string, val any) {
	if *m == nil {
		*m = make(map[string]any)
	}
	(*m)[key] = val
}

// getOrInitFromMap retrieves a typed value, or initializes it via initFn and stores it.
func getOrInitFromMap[T any](m *map[string]any, key string, initFn func() T) T {
	if *m != nil {
		if v, ok := (*m)[key]; ok {
			if typed, ok := v.(T); ok {
				return typed
			}
		}
	}
	state := initFn()
	setInMap(m, key, state)
	return state
}

// ---- Per-perk round state accessors ----

// GetPerkState returns the per-perk round state for the given perk ID, or zero value.
func GetPerkState[T any](s *PerkRoundState, perkID string) T {
	return getFromMap[T](s.PerkState, perkID)
}

// GetOrInitPerkState returns existing per-perk round state, or initializes it via initFn.
func GetOrInitPerkState[T any](s *PerkRoundState, perkID string, initFn func() T) T {
	return getOrInitFromMap[T](&s.PerkState, perkID, initFn)
}

// SetPerkState stores per-perk round state for the given perk ID.
func SetPerkState(s *PerkRoundState, perkID string, state any) {
	setInMap(&s.PerkState, perkID, state)
}

// ---- Per-perk battle state accessors ----

// GetBattleState returns the per-perk battle state for the given perk ID, or zero value.
func GetBattleState[T any](s *PerkRoundState, perkID string) T {
	return getFromMap[T](s.PerkBattleState, perkID)
}

// GetOrInitBattleState returns existing per-perk battle state, or initializes it via initFn.
func GetOrInitBattleState[T any](s *PerkRoundState, perkID string, initFn func() T) T {
	return getOrInitFromMap[T](&s.PerkBattleState, perkID, initFn)
}

// SetBattleState stores per-perk battle state for the given perk ID.
func SetBattleState(s *PerkRoundState, perkID string, state any) {
	setInMap(&s.PerkBattleState, perkID, state)
}

// ---- Per-perk state structs ----
// Each stateful perk defines a small struct here. Stateless perks need nothing.

// RecklessAssaultState tracks whether the squad is vulnerable after attacking.
type RecklessAssaultState struct {
	Vulnerable bool
}

// AdaptiveArmorState tracks hits from each attacker this round.
type AdaptiveArmorState struct {
	AttackedBy map[ecs.EntityID]int
}

// BloodlustState tracks kills this round.
type BloodlustState struct {
	KillsThisRound int
}

// OpeningSalvoState tracks whether the squad has attacked this combat.
type OpeningSalvoState struct {
	HasAttackedThisCombat bool
}

// ResoluteState tracks which units have used their resolute save and HP snapshots.
type ResoluteState struct {
	Used         map[ecs.EntityID]bool
	RoundStartHP map[ecs.EntityID]int
}

// GrudgeBearerState tracks grudge stacks against enemy squads.
type GrudgeBearerState struct {
	Stacks map[ecs.EntityID]int
}

// CounterpunchState tracks whether the perk is armed.
type CounterpunchState struct {
	Ready bool
}

// DeadshotState tracks whether the perk is armed.
type DeadshotState struct {
	Ready bool
}


// ECS component variables
var (
	PerkSlotComponent       *ecs.Component
	PerkRoundStateComponent *ecs.Component

	PerkSlotTag ecs.Tag
)
