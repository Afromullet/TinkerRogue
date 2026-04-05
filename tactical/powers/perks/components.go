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
// dispatch layer (perk_dispatch.go) and read by multiple perks.
//
// Per-perk state lives in the PerkState map, keyed by perk ID. Each perk
// defines its own small state struct. This prevents the struct from growing
// a new field for every stateful perk and makes reset automatic.
//
// Shared field lifecycle:
//
//	Field                 Writer                               Reader                          Reset
//	MovedThisTurn         perk_dispatch.go OnMoveComplete       stalwart, fortify               ResetPerTurn
//	AttackedThisTurn      perk_dispatch.go OnAttackComplete     ResetPerTurn (snapshot)          ResetPerTurn
//	WasAttackedThisTurn   perk_dispatch.go OnAttackComplete     ResetPerTurn (snapshot)          ResetPerTurn
//	RecklessVulnerable    reckless_assault (AttackerDamageMod)  reckless_assault (DefenderMod)   ResetPerTurn
//	TurnsStationary       fortify (TurnStart), OnMoveComplete   fortify (CoverMod)              OnMoveComplete (set to 0)
//	WasAttackedLastTurn   ResetPerTurn (snapshot from WasAttackedThisTurn)  counterpunch (TurnStart)  Overwritten each turn
//	DidNotAttackLastTurn  ResetPerTurn (snapshot)               counterpunch (TurnStart)         Overwritten each turn
//	WasIdleLastTurn       ResetPerTurn (snapshot)               deadshots_patience (TurnStart)   Overwritten each turn
type PerkRoundState struct {
	// ---- Shared tracking (set by dispatch layer, read by multiple perks) ----

	// Per-turn (cleared by ResetPerTurn)
	MovedThisTurn       bool // Set by OnMoveComplete. Read by Stalwart, Fortify.
	AttackedThisTurn    bool // Set by OnAttackComplete. Read by ResetPerTurn snapshots.
	WasAttackedThisTurn bool // Set by OnAttackComplete (defender). Snapshotted into WasAttackedLastTurn.
	RecklessVulnerable  bool // Set by Reckless Assault attacker. Read by Reckless Assault defender.

	// Turn-boundary snapshots (computed by ResetPerTurn from previous turn)
	WasAttackedLastTurn  bool // Snapshot of previous turn. Read by Counterpunch.
	DidNotAttackLastTurn bool // Snapshot of previous turn. Read by Counterpunch.
	WasIdleLastTurn      bool // Snapshot of previous turn. Read by Deadshot's Patience.

	// Movement-gated (accumulates across rounds, resets on movement)
	TurnsStationary int // Set by Fortify TurnStart + OnMoveComplete. Read by Fortify CoverMod.

	// ---- Per-perk isolated state ----

	// PerkState holds per-perk state structs, keyed by perk ID.
	// Each perk defines its own state struct and accesses it via GetPerkState/SetPerkState.
	// Cleared entirely by ResetPerRound; per-battle state uses PerkBattleState instead.
	PerkState map[string]interface{}

	// PerkBattleState holds per-perk state that persists the entire combat.
	// Never reset during combat; cleaned up by CleanupRoundState.
	PerkBattleState map[string]interface{}
}

// GetPerkState returns the per-perk round state for the given perk ID, or nil.
func GetPerkState[T any](s *PerkRoundState, perkID string) T {
	var zero T
	if s.PerkState == nil {
		return zero
	}
	v, ok := s.PerkState[perkID]
	if !ok {
		return zero
	}
	typed, ok := v.(T)
	if !ok {
		return zero
	}
	return typed
}

// GetOrInitPerkState returns existing per-perk round state, or initializes it via initFn.
func GetOrInitPerkState[T any](s *PerkRoundState, perkID string, initFn func() T) T {
	if s.PerkState != nil {
		if v, ok := s.PerkState[perkID]; ok {
			if typed, ok := v.(T); ok {
				return typed
			}
		}
	}
	state := initFn()
	SetPerkState(s, perkID, state)
	return state
}

// SetPerkState stores per-perk round state for the given perk ID.
func SetPerkState(s *PerkRoundState, perkID string, state interface{}) {
	if s.PerkState == nil {
		s.PerkState = make(map[string]interface{})
	}
	s.PerkState[perkID] = state
}

// GetBattleState returns the per-perk battle state for the given perk ID, or nil.
func GetBattleState[T any](s *PerkRoundState, perkID string) T {
	var zero T
	if s.PerkBattleState == nil {
		return zero
	}
	v, ok := s.PerkBattleState[perkID]
	if !ok {
		return zero
	}
	typed, ok := v.(T)
	if !ok {
		return zero
	}
	return typed
}

// GetOrInitBattleState returns existing per-perk battle state, or initializes it via initFn.
func GetOrInitBattleState[T any](s *PerkRoundState, perkID string, initFn func() T) T {
	if s.PerkBattleState != nil {
		if v, ok := s.PerkBattleState[perkID]; ok {
			if typed, ok := v.(T); ok {
				return typed
			}
		}
	}
	state := initFn()
	SetBattleState(s, perkID, state)
	return state
}

// SetBattleState stores per-perk battle state for the given perk ID.
func SetBattleState(s *PerkRoundState, perkID string, state interface{}) {
	if s.PerkBattleState == nil {
		s.PerkBattleState = make(map[string]interface{})
	}
	s.PerkBattleState[perkID] = state
}

// ---- Per-perk state structs ----
// Each stateful perk defines a small struct here. Stateless perks need nothing.

// AdaptiveArmorState tracks hits from each attacker this round.
type AdaptiveArmorState struct {
	AttackedBy map[ecs.EntityID]int
}

// BloodlustState tracks kills this round.
type BloodlustState struct {
	KillsThisRound int
}

// DisruptionState tracks which squads have been disrupted this round.
type DisruptionState struct {
	Targets map[ecs.EntityID]bool
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

// MarkedForDeathState tracks which enemy squad is marked.
type MarkedForDeathState struct {
	MarkedSquad ecs.EntityID
}

// OverwatchState is a placeholder for the overwatch perk.
type OverwatchState struct {
	Active bool
}

// ResetPerTurn resets shared tracking fields at the start of each turn.
// Called before TurnStartHooks run. Per-perk state is NOT reset here —
// perks manage their own per-turn state in their TurnStart hooks.
func (s *PerkRoundState) ResetPerTurn() {
	// Snapshot previous turn state for Counterpunch/Deadshot before clearing
	s.WasAttackedLastTurn = s.WasAttackedThisTurn
	s.DidNotAttackLastTurn = !s.AttackedThisTurn
	s.WasIdleLastTurn = !s.MovedThisTurn && !s.AttackedThisTurn

	s.MovedThisTurn = false
	s.AttackedThisTurn = false
	s.WasAttackedThisTurn = false
	s.RecklessVulnerable = false
}

// ResetPerRound clears all per-perk round state at the start of each round.
// Per-battle state (PerkBattleState) is preserved.
func (s *PerkRoundState) ResetPerRound() {
	s.PerkState = nil
}

// ECS component variables
var (
	PerkSlotComponent       *ecs.Component
	PerkRoundStateComponent *ecs.Component

	PerkSlotTag ecs.Tag
)
