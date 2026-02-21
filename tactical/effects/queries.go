package effects

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// GetActiveEffects returns the ActiveEffectsData for an entity, or nil if none.
func GetActiveEffects(entityID ecs.EntityID, manager *common.EntityManager) *ActiveEffectsData {
	return common.GetComponentTypeByID[*ActiveEffectsData](manager, entityID, ActiveEffectsComponent)
}

// HasActiveEffects returns true if the entity has any active effects.
func HasActiveEffects(entityID ecs.EntityID, manager *common.EntityManager) bool {
	data := GetActiveEffects(entityID, manager)
	return data != nil && len(data.Effects) > 0
}
