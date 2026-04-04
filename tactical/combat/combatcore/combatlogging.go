package combatcore

// Re-exports from battlelog for backward compatibility.

import (
	"game_main/common"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combat/combattypes"

	"github.com/bytearena/ecs"
)

func InitializeCombatLog(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) *combattypes.CombatLog {
	return battlelog.InitializeCombatLog(attackerSquadID, defenderSquadID, manager)
}

func SnapshotAttackingUnits(squadID ecs.EntityID, squadDistance int, manager *common.EntityManager) []combattypes.UnitSnapshot {
	return battlelog.SnapshotAttackingUnits(squadID, squadDistance, manager)
}

func SnapshotAllUnits(squadID ecs.EntityID, manager *common.EntityManager) []combattypes.UnitSnapshot {
	return battlelog.SnapshotAllUnits(squadID, manager)
}

func FinalizeCombatLog(result *combattypes.CombatResult, log *combattypes.CombatLog, defenderSquadID, attackerSquadID ecs.EntityID, manager *common.EntityManager) {
	battlelog.FinalizeCombatLog(result, log, defenderSquadID, attackerSquadID, manager)
}
