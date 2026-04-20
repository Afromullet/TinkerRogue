package battlelog

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

func InitializeCombatLog(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) *combattypes.CombatLog {
	return &combattypes.CombatLog{
		AttackerSquadID:   attackerSquadID,
		DefenderSquadID:   defenderSquadID,
		AttackerSquadName: squadcore.GetSquadName(attackerSquadID, manager),
		DefenderSquadName: squadcore.GetSquadName(defenderSquadID, manager),
		SquadDistance:     squadcore.GetSquadDistance(attackerSquadID, defenderSquadID, manager),
		AttackEvents:      []combattypes.AttackEvent{},
		AttackingUnits:    []combattypes.UnitSnapshot{},
		DefendingUnits:    []combattypes.UnitSnapshot{},
	}
}

func snapshotUnits(squadID ecs.EntityID, squadDistance int, filterByRange bool, manager *common.EntityManager) []combattypes.UnitSnapshot {
	var snapshots []combattypes.UnitSnapshot
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		if filterByRange && !combatmath.CanUnitAttack(unitID, squadDistance, manager) {
			continue
		}

		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		name := common.GetComponentType[*common.Name](entity, common.NameComponent)
		gridPos := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent)
		rangeData := common.GetComponentType[*squadcore.AttackRangeData](entity, squadcore.AttackRangeComponent)
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)

		unitName := "Unknown"
		if name != nil {
			unitName = name.NameStr
		}

		row, col := 0, 0
		if gridPos != nil {
			row, col = gridPos.AnchorRow, gridPos.AnchorCol
		}

		attackRange := 0
		if rangeData != nil {
			attackRange = rangeData.Range
		}

		roleName := "Unknown"
		if roleData != nil {
			roleName = roleData.Role.String()
		}

		snapshot := combattypes.UnitSnapshot{
			UnitID:      unitID,
			UnitName:    unitName,
			GridRow:     row,
			GridCol:     col,
			AttackRange: attackRange,
			RoleName:    roleName,
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots
}

func SnapshotAttackingUnits(squadID ecs.EntityID, squadDistance int, manager *common.EntityManager) []combattypes.UnitSnapshot {
	return snapshotUnits(squadID, squadDistance, true, manager)
}

func SnapshotAllUnits(squadID ecs.EntityID, manager *common.EntityManager) []combattypes.UnitSnapshot {
	return snapshotUnits(squadID, -1, false, manager)
}

func FinalizeCombatLog(result *combattypes.CombatResult, log *combattypes.CombatLog, defenderSquadID, attackerSquadID ecs.EntityID, manager *common.EntityManager) {
	result.TotalDamage = combatmath.SumDamageMap(result.DamageByUnit)
	log.TotalDamage = result.TotalDamage
	log.UnitsKilled = len(result.UnitsKilled)
	log.DefenderStatus = combatmath.CalculateSquadStatus(defenderSquadID, manager)
}
