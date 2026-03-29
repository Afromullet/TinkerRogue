package perks

import (
	"github.com/bytearena/ecs"
)

// PerkSlotData stores equipped perks on a squad entity.
// Number of available slots scales with squad progression.
type PerkSlotData struct {
	PerkIDs []string // Equipped perk IDs (max based on squad level)
}

// PerkRoundState tracks per-round state needed by conditional perks.
// Reset at round start, except fields marked as per-battle.
type PerkRoundState struct {
	// Per-turn state (resets each turn)
	MovedThisTurn      bool // For Stalwart, Fortify
	AttackedThisTurn   bool // For Reckless Assault vulnerability window
	RecklessVulnerable bool // For Reckless Assault (+20% damage received)

	// Per-round state (resets each round)
	AttackedBy        map[ecs.EntityID]int  // For Adaptive Armor (attacker -> hit count)
	KillsThisRound    int                   // For Bloodlust
	DisruptionTargets map[ecs.EntityID]bool // For Disruption (squads debuffed this round)
	OverwatchActive   bool                  // For Overwatch

	// Per-round but persists across rounds
	TurnsStationary int // For Fortify (resets on movement)

	// Per-battle state (persists entire combat)
	HasAttackedThisCombat bool                  // For Opening Salvo (one-time bonus)
	ResoluteUsed          map[ecs.EntityID]bool // For Resolute (unit -> used flag)
	RoundStartHP          map[ecs.EntityID]int  // For Resolute (updated each round, not reset)
	GrudgeStacks          map[ecs.EntityID]int  // For Grudge Bearer (enemy squad -> damage count, persists)
	WasAttackedLastTurn   bool                  // For Counterpunch (was attacked previous turn)
	DidNotAttackLastTurn  bool                  // For Counterpunch (did not attack previous turn)
	CounterpunchReady     bool                  // For Counterpunch (both conditions met)
	WasIdleLastTurn       bool                  // For Deadshot's Patience (no move AND no attack last turn)
	DeadshotReady         bool                  // For Deadshot's Patience (ready to fire)
	MarkedSquad           ecs.EntityID          // For Marked for Death (which enemy is marked, 0 = none)
}

// ResetPerTurn resets fields that should clear at the start of each turn.
// Called before TurnStartHooks run.
func (s *PerkRoundState) ResetPerTurn() {
	// Snapshot previous turn state for Counterpunch/Deadshot before clearing
	s.WasAttackedLastTurn = s.RecklessVulnerable || s.WasAttackedLastTurn
	s.DidNotAttackLastTurn = !s.AttackedThisTurn
	s.WasIdleLastTurn = !s.MovedThisTurn && !s.AttackedThisTurn

	s.MovedThisTurn = false
	s.AttackedThisTurn = false
	s.RecklessVulnerable = false
}

// ResetPerRound resets fields that should clear at the start of each round.
func (s *PerkRoundState) ResetPerRound() {
	s.AttackedBy = nil
	s.KillsThisRound = 0
	s.DisruptionTargets = nil
	s.OverwatchActive = false
}

// PerkUnlockData tracks which perks have been unlocked for a commander/roster.
type PerkUnlockData struct {
	UnlockedPerks map[string]bool // Perk IDs that have been unlocked
	PerkPoints    int             // Available points to spend
}

// ECS component variables
var (
	PerkSlotComponent       *ecs.Component
	PerkRoundStateComponent *ecs.Component
	PerkUnlockComponent     *ecs.Component

	PerkSlotTag ecs.Tag
)
