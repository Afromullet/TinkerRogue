package combatcore

// Re-exports from combatmath for backward compatibility.

import (
	"game_main/common"
	"game_main/tactical/combat/combatmath"

	"github.com/bytearena/ecs"
)

func CanUnitAttack(attackerID ecs.EntityID, squadDistance int, manager *common.EntityManager) bool {
	return combatmath.CanUnitAttack(attackerID, squadDistance, manager)
}

func SelectTargetUnits(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	return combatmath.SelectTargetUnits(attackerID, defenderSquadID, manager)
}

func IsHealUnit(unitID ecs.EntityID, manager *common.EntityManager) bool {
	return combatmath.IsHealUnit(unitID, manager)
}

func SelectHealTargets(healerID, healerSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	return combatmath.SelectHealTargets(healerID, healerSquadID, manager)
}
