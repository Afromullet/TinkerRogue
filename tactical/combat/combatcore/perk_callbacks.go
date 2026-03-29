package combatcore

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// Perk callback function types.
// These are defined in combatcore to avoid circular imports with the perks package.
// The perks package provides implementations; combatservices wires them together.

// DamageHookRunner runs perk DamageMod hooks for an attacker's perks.
// Called inside calculateDamage before multiplier application.
type DamageHookRunner func(
	attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *DamageModifiers,
	manager *common.EntityManager,
)

// CoverHookRunner runs perk CoverMod hooks.
// Called inside calculateDamage after cover calculation.
type CoverHookRunner func(
	attackerID, defenderID ecs.EntityID,
	cover *CoverBreakdown,
	manager *common.EntityManager,
)

// TargetHookRunner runs perk TargetOverride hooks.
// Called inside processAttackWithModifiers before the target loop.
type TargetHookRunner func(
	attackerID, defenderSquadID ecs.EntityID,
	targets []ecs.EntityID,
	manager *common.EntityManager,
) []ecs.EntityID

// PostDamageRunner runs perk PostDamage hooks after damage is recorded.
type PostDamageRunner func(
	attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damage int, wasKill bool,
	manager *common.EntityManager,
)

// DeathOverrideRunner checks if lethal damage should be prevented.
type DeathOverrideRunner func(
	unitID, squadID ecs.EntityID,
	manager *common.EntityManager,
) bool

// CounterModRunner modifies counterattack modifiers.
// Returns true if counter should be skipped.
type CounterModRunner func(
	defenderSquadID, attackerID ecs.EntityID,
	modifiers *DamageModifiers,
	manager *common.EntityManager,
) bool

// PerkCallbacks holds all perk callback functions injected from the perks package.
// Set on CombatActionSystem before combat begins.
type PerkCallbacks struct {
	AttackerDamageMod  DamageHookRunner
	DefenderDamageMod  DamageHookRunner
	CoverMod           CoverHookRunner
	TargetOverride     TargetHookRunner
	PostDamage         PostDamageRunner
	DefenderPostDamage PostDamageRunner
	DeathOverride      DeathOverrideRunner
	CounterMod         CounterModRunner
}
