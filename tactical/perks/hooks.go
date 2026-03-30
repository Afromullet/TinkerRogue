package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"

	"github.com/bytearena/ecs"
)

// HookContext bundles common parameters passed to combat-inline perk hooks.
type HookContext struct {
	AttackerID      ecs.EntityID
	DefenderID      ecs.EntityID
	AttackerSquadID ecs.EntityID
	DefenderSquadID ecs.EntityID
	RoundState      *PerkRoundState
	Manager         *common.EntityManager
}

// DamageModHook modifies DamageModifiers before damage calculation.
type DamageModHook func(ctx *HookContext, modifiers *combatcore.DamageModifiers)

// TargetOverrideHook overrides target selection. Returns modified target list.
type TargetOverrideHook func(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID

// CounterModHook modifies or suppresses counterattack.
// Returns true if counter should be skipped entirely.
type CounterModHook func(
	defenderID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers,
	roundState *PerkRoundState,
	manager *common.EntityManager,
) (skipCounter bool)

// PostDamageHook runs after damage is recorded.
type PostDamageHook func(ctx *HookContext, damageDealt int, wasKill bool)

// TurnStartHook runs at start of a squad's turn.
type TurnStartHook func(
	squadID ecs.EntityID,
	roundNumber int,
	roundState *PerkRoundState,
	manager *common.EntityManager,
)

// CoverModHook modifies cover calculation.
type CoverModHook func(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown)

// DamageRedirectHook intercepts damage before it is recorded.
// Returns reduced damage for original target, redirect target ID, and redirect amount.
type DamageRedirectHook func(
	defenderID ecs.EntityID,
	defenderSquadID ecs.EntityID,
	damageAmount int,
	manager *common.EntityManager,
) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)

// DeathOverrideHook prevents lethal damage. Returns true to prevent death.
type DeathOverrideHook func(
	unitID ecs.EntityID,
	squadID ecs.EntityID,
	roundState *PerkRoundState,
	manager *common.EntityManager,
) (preventDeath bool)
