package framework

import (
	"game_main/common"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// UnitGridInfo bundles formation grid display data for a single unit.
type UnitGridInfo struct {
	UnitID    ecs.EntityID
	Name      string
	AnchorRow int
	AnchorCol int
	IsLeader  bool
	IsAlive   bool
	CurrentHP int
	MaxHP     int
}

// GetUnitGridInfo returns formation grid display data for a unit.
// Combines grid position, name, leader status, and health into one struct.
func (gq *GUIQueries) GetUnitGridInfo(unitID ecs.EntityID) *UnitGridInfo {
	manager := gq.ECSManager

	gridPos := common.GetComponentTypeByID[*squadcore.GridPositionData](
		manager, unitID, squadcore.GridPositionComponent)
	if gridPos == nil {
		return nil
	}

	name := common.GetEntityName(manager, unitID, "Unit")
	isLeader := manager.HasComponent(unitID, squadcore.LeaderComponent)

	attrs := common.GetComponentTypeByID[*common.Attributes](
		manager, unitID, common.AttributeComponent)

	info := &UnitGridInfo{
		UnitID:    unitID,
		Name:      name,
		AnchorRow: gridPos.AnchorRow,
		AnchorCol: gridPos.AnchorCol,
		IsLeader:  isLeader,
	}

	if attrs != nil {
		info.CurrentHP = attrs.CurrentHealth
		info.MaxHP = attrs.GetMaxHealth()
		info.IsAlive = attrs.CurrentHealth > 0
	}

	return info
}

// GetUnitAttributes returns the attributes component for a unit.
func (gq *GUIQueries) GetUnitAttributes(unitID ecs.EntityID) *common.Attributes {
	return common.GetComponentTypeByID[*common.Attributes](
		gq.ECSManager, unitID, common.AttributeComponent)
}

// GetUnitTypeData returns the unit type data for a unit.
func (gq *GUIQueries) GetUnitTypeData(unitID ecs.EntityID) *squadcore.UnitTypeData {
	return common.GetComponentTypeByID[*squadcore.UnitTypeData](
		gq.ECSManager, unitID, squadcore.UnitTypeComponent)
}

// GetTargetRowData returns the target row data for a unit.
func (gq *GUIQueries) GetTargetRowData(unitID ecs.EntityID) *squadcore.TargetRowData {
	return common.GetComponentTypeByID[*squadcore.TargetRowData](
		gq.ECSManager, unitID, squadcore.TargetRowComponent)
}

// GetSquadName returns the name of a squad by ID.
func (gq *GUIQueries) GetSquadName(squadID ecs.EntityID) string {
	return squadcore.GetSquadName(squadID, gq.ECSManager)
}
