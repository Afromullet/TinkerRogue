package framework

import (
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/visual/rendering"

	"github.com/bytearena/ecs"
)

// GetAllSquadIDs returns all squad entity IDs.
// Satisfies rendering.SquadInfoProvider.
func (gq *GUIQueries) GetAllSquadIDs() []ecs.EntityID {
	return gq.SquadCache.FindAllSquads()
}

// GetSquadRenderInfo returns minimal squad data for rendering.
// Satisfies rendering.SquadInfoProvider.
func (gq *GUIQueries) GetSquadRenderInfo(squadID ecs.EntityID) *rendering.SquadRenderInfo {
	info := gq.GetSquadInfo(squadID)
	if info == nil {
		return nil
	}
	return &rendering.SquadRenderInfo{
		ID:          info.ID,
		Position:    info.Position,
		FactionID:   info.FactionID,
		IsDestroyed: info.IsDestroyed,
		CurrentHP:   info.CurrentHP,
		MaxHP:       info.MaxHP,
	}
}

// GetUnitIDsInSquad returns all unit entity IDs in a squad.
// Satisfies rendering.UnitInfoProvider.
func (gq *GUIQueries) GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID {
	return gq.SquadCache.GetUnitIDsInSquad(squadID)
}

// GetUnitRenderInfo returns minimal unit data for combat animation rendering.
// Satisfies rendering.UnitInfoProvider.
func (gq *GUIQueries) GetUnitRenderInfo(unitID ecs.EntityID) *rendering.UnitRenderInfo {
	entity := gq.ECSManager.FindEntityByID(unitID)
	if entity == nil {
		return nil
	}

	gridPos := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
	if gridPos == nil {
		return nil
	}

	renderable := common.GetComponentType[*rendering.Renderable](entity, rendering.RenderableComponent)
	if renderable == nil || renderable.Image == nil {
		return nil
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	isAlive := attr != nil && attr.CurrentHealth > 0

	return &rendering.UnitRenderInfo{
		AnchorRow: gridPos.AnchorRow,
		AnchorCol: gridPos.AnchorCol,
		Width:     gridPos.Width,
		Height:    gridPos.Height,
		Image:     renderable.Image,
		IsAlive:   isAlive,
	}
}
