package combatcore

// Re-exports from combatmath for backward compatibility.

import (
	"game_main/common"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/combat/combattypes"

	"github.com/bytearena/ecs"
)

func ApplyRecordedDamage(result *combattypes.CombatResult, squadmanager *common.EntityManager) {
	combatmath.ApplyRecordedDamage(result, squadmanager)
}

func ApplyRecordedHealing(result *combattypes.CombatResult, manager *common.EntityManager) {
	combatmath.ApplyRecordedHealing(result, manager)
}

func CalculateCoverBreakdown(defenderID ecs.EntityID, squadmanager *common.EntityManager) combattypes.CoverBreakdown {
	return combatmath.CalculateCoverBreakdown(defenderID, squadmanager)
}
