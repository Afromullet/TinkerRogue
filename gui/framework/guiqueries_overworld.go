package framework

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/visual/rendering"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetEntityPosition returns the logical position for an entity by ID.
func (gq *GUIQueries) GetEntityPosition(entityID ecs.EntityID) *coords.LogicalPosition {
	return common.GetComponentTypeByID[*coords.LogicalPosition](
		gq.ECSManager, entityID, common.PositionComponent)
}

// GetNodeData returns the overworld node data for an entity by ID.
func (gq *GUIQueries) GetNodeData(entityID ecs.EntityID) *core.OverworldNodeData {
	return common.GetComponentTypeByID[*core.OverworldNodeData](
		gq.ECSManager, entityID, core.OverworldNodeComponent)
}

// GetNodeDataFromEntity returns overworld node data from an entity pointer.
// Use when the caller already has the entity from View iteration.
func (gq *GUIQueries) GetNodeDataFromEntity(entity *ecs.Entity) *core.OverworldNodeData {
	return common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)
}

// GetInfluenceDataFromEntity returns influence data from an entity pointer.
// Use when the caller already has the entity from View iteration.
func (gq *GUIQueries) GetInfluenceDataFromEntity(entity *ecs.Entity) *core.InfluenceData {
	return common.GetComponentType[*core.InfluenceData](entity, core.InfluenceComponent)
}

// GetEntityPositionFromEntity returns logical position from an entity pointer.
// Use when the caller already has the entity from View iteration.
func (gq *GUIQueries) GetEntityPositionFromEntity(entity *ecs.Entity) *coords.LogicalPosition {
	return common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
}

// GetRenderableFromEntity returns the renderable component from an entity pointer.
// Use when the caller already has the entity from View iteration.
func (gq *GUIQueries) GetRenderableFromEntity(entity *ecs.Entity) *rendering.Renderable {
	return common.GetComponentType[*rendering.Renderable](entity, rendering.RenderableComponent)
}
