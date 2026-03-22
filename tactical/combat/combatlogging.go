package combat

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

func InitializeCombatLog(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) *CombatLog {
	return &CombatLog{
		AttackerSquadID:   attackerSquadID,
		DefenderSquadID:   defenderSquadID,
		AttackerSquadName: squads.GetSquadName(attackerSquadID, manager),
		DefenderSquadName: squads.GetSquadName(defenderSquadID, manager),
		SquadDistance:     squads.GetSquadDistance(attackerSquadID, defenderSquadID, manager),
		AttackEvents:      []AttackEvent{},
		AttackingUnits:    []UnitSnapshot{},
		DefendingUnits:    []UnitSnapshot{},
	}
}

func snapshotUnits(squadID ecs.EntityID, squadDistance int, filterByRange bool, manager *common.EntityManager) []UnitSnapshot {
	var snapshots []UnitSnapshot
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		if filterByRange && !CanUnitAttack(unitID, squadDistance, manager) {
			continue
		}

		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		name := common.GetComponentType[*common.Name](entity, common.NameComponent)
		gridPos := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
		rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
		roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent)

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

		snapshot := UnitSnapshot{
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

func SnapshotAttackingUnits(squadID ecs.EntityID, squadDistance int, manager *common.EntityManager) []UnitSnapshot {
	return snapshotUnits(squadID, squadDistance, true, manager)
}

func SnapshotAllUnits(squadID ecs.EntityID, manager *common.EntityManager) []UnitSnapshot {
	return snapshotUnits(squadID, -1, false, manager)
}

func FinalizeCombatLog(result *CombatResult, log *CombatLog, defenderSquadID, attackerSquadID ecs.EntityID, manager *common.EntityManager) {
	result.TotalDamage = sumDamageMap(result.DamageByUnit)
	log.TotalDamage = result.TotalDamage
	log.UnitsKilled = len(result.UnitsKilled)
	log.DefenderStatus = calculateSquadStatus(defenderSquadID, manager)
}
