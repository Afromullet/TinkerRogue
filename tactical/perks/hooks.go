package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"

	"github.com/bytearena/ecs"
)

// DamageModHook modifies damage modifiers before calculation.
// Called inside calculateDamage() for the attacking or defending unit.
type DamageModHook func(
	attackerID, defenderID ecs.EntityID,
	attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers,
	roundState *PerkRoundState,
	manager *common.EntityManager,
)

// TargetOverrideHook overrides target selection.
// Returns modified target list; return defaultTargets unchanged for no override.
type TargetOverrideHook func(
	attackerID, defenderSquadID ecs.EntityID,
	defaultTargets []ecs.EntityID,
	manager *common.EntityManager,
) []ecs.EntityID

// CounterModHook modifies counterattack behavior.
// Return skipCounter=true to suppress counterattack entirely.
type CounterModHook func(
	defenderID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers,
	roundState *PerkRoundState,
	manager *common.EntityManager,
) (skipCounter bool)

// PostDamageHook runs after damage is recorded for a single attack.
type PostDamageHook func(
	attackerID, defenderID ecs.EntityID,
	attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool,
	roundState *PerkRoundState,
	manager *common.EntityManager,
)

// TurnStartHook runs at the start of a squad's turn.
type TurnStartHook func(
	squadID ecs.EntityID,
	roundNumber int,
	roundState *PerkRoundState,
	manager *common.EntityManager,
)

// CoverModHook modifies cover calculation for a defender.
type CoverModHook func(
	attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown,
	roundState *PerkRoundState,
	manager *common.EntityManager,
)

// DamageRedirectHook intercepts damage before recordDamageToUnit.
// Returns reduced damage for original target, plus a redirect target and amount.
type DamageRedirectHook func(
	defenderID ecs.EntityID,
	defenderSquadID ecs.EntityID,
	damageAmount int,
	manager *common.EntityManager,
) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)

// DeathOverrideHook intercepts lethal damage.
// Returns true if death should be prevented (unit survives at 1 HP).
type DeathOverrideHook func(
	unitID ecs.EntityID,
	squadID ecs.EntityID,
	roundState *PerkRoundState,
	manager *common.EntityManager,
) (preventDeath bool)
