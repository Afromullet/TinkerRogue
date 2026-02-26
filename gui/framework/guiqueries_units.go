package framework

import (
	"game_main/common"
	"game_main/tactical/squads"

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

	gridPos := common.GetComponentTypeByID[*squads.GridPositionData](
		manager, unitID, squads.GridPositionComponent)
	if gridPos == nil {
		return nil
	}

	name := common.GetEntityName(manager, unitID, "Unit")
	isLeader := manager.HasComponent(unitID, squads.LeaderComponent)

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
		info.MaxHP = attrs.MaxHealth
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
func (gq *GUIQueries) GetUnitTypeData(unitID ecs.EntityID) *squads.UnitTypeData {
	return common.GetComponentTypeByID[*squads.UnitTypeData](
		gq.ECSManager, unitID, squads.UnitTypeComponent)
}

// GetTargetRowData returns the target row data for a unit.
func (gq *GUIQueries) GetTargetRowData(unitID ecs.EntityID) *squads.TargetRowData {
	return common.GetComponentTypeByID[*squads.TargetRowData](
		gq.ECSManager, unitID, squads.TargetRowComponent)
}

// GetSquadName returns the name of a squad by ID.
func (gq *GUIQueries) GetSquadName(squadID ecs.EntityID) string {
	return squads.GetSquadName(squadID, gq.ECSManager)
}
