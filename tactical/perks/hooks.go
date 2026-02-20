package perks

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// DamageModHook modifies damage modifiers before damage calculation.
// Called for the attacking unit's perks inside calculateDamage().
type DamageModHook func(
	attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers,
	manager *common.EntityManager,
)

// TargetOverrideHook overrides target selection.
// Returns modified target list. Return defaultTargets to keep defaults.
type TargetOverrideHook func(
	attackerID, defenderSquadID ecs.EntityID,
	defaultTargets []ecs.EntityID,
	manager *common.EntityManager,
) []ecs.EntityID

// CounterModHook modifies counterattack behavior.
// Return skipCounter=true to suppress counterattack entirely.
type CounterModHook func(
	defenderID, attackerID ecs.EntityID,
	modifiers *squads.DamageModifiers,
	manager *common.EntityManager,
) (skipCounter bool)

// PostDamageHook runs after damage is recorded for a single attack.
// Note: runs with pre-damage HP. Use damageDealt/wasKill params, not entity HP.
type PostDamageHook func(
	attackerID, defenderID ecs.EntityID,
	damageDealt int, wasKill bool,
	manager *common.EntityManager,
)

// TurnStartHook runs at the start of a squad's turn.
type TurnStartHook func(
	squadID ecs.EntityID,
	manager *common.EntityManager,
)

// CoverModHook modifies cover calculation for a defender.
type CoverModHook func(
	attackerID, defenderID ecs.EntityID,
	coverBreakdown *squads.CoverBreakdown,
	manager *common.EntityManager,
)

// DamageRedirectHook intercepts damage before recording.
// Returns reduced damage for original target, plus redirect target and amount.
// Deferred to v2 (Guardian perk).
type DamageRedirectHook func(
	defenderID ecs.EntityID,
	damageAmount int,
	manager *common.EntityManager,
) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
