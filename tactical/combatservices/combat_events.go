package combatservices

import (
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Callback types for post-action hooks
type OnAttackCompleteFunc func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult)
type OnMoveCompleteFunc func(squadID ecs.EntityID)
type OnTurnEndFunc func(round int)
type PostResetHookFunc func(factionID ecs.EntityID, squadIDs []ecs.EntityID)

// RegisterOnAttackComplete adds a callback fired after each successful attack.
func (cs *CombatService) RegisterOnAttackComplete(fn OnAttackCompleteFunc) {
	cs.onAttackComplete = append(cs.onAttackComplete, fn)
}

// RegisterOnMoveComplete adds a callback fired after each successful squad move.
func (cs *CombatService) RegisterOnMoveComplete(fn OnMoveCompleteFunc) {
	cs.onMoveComplete = append(cs.onMoveComplete, fn)
}

// RegisterOnTurnEnd adds a callback fired after each turn end.
func (cs *CombatService) RegisterOnTurnEnd(fn OnTurnEndFunc) {
	cs.onTurnEnd = append(cs.onTurnEnd, fn)
}

// RegisterPostResetHook adds a callback fired after squad actions are reset for a faction.
func (cs *CombatService) RegisterPostResetHook(fn PostResetHookFunc) {
	cs.postResetHooks = append(cs.postResetHooks, fn)
}

// ClearCallbacks removes all registered callbacks.
func (cs *CombatService) ClearCallbacks() {
	cs.onAttackComplete = nil
	cs.onMoveComplete = nil
	cs.onTurnEnd = nil
	cs.postResetHooks = nil
}
