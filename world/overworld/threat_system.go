package overworld

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CreateThreatNode spawns a new threat at a position
func CreateThreatNode(
	manager *common.EntityManager,
	pos coords.LogicalPosition,
	threatType ThreatType,
	initialIntensity int,
	currentTick int64,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	// Add position component
	entity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: pos.X,
		Y: pos.Y,
	})

	// Add threat node data
	params := GetThreatTypeParams(threatType)
	entity.AddComponent(ThreatNodeComponent, &ThreatNodeData{
		ThreatID:       entity.GetID(),
		ThreatType:     threatType,
		Intensity:      initialIntensity,
		GrowthProgress: 0.0,
		GrowthRate:     params.BaseGrowthRate,
		IsContained:    false,
		SpawnedTick:    currentTick,
	})

	// Add influence component
	entity.AddComponent(InfluenceComponent, &InfluenceData{
		Radius:         params.BaseRadius + initialIntensity,
		EffectType:     params.PrimaryEffect,
		EffectStrength: float64(initialIntensity) * 0.1,
	})

	// Register in position system
	common.GlobalPositionSystem.AddEntity(entity.GetID(), pos)

	// Log threat spawn event
	LogEvent(EventThreatSpawned, currentTick, entity.GetID(),
		formatEventString("%s spawned at (%d, %d) with intensity %d",
			threatType.String(), pos.X, pos.Y, initialIntensity))

	return entity.GetID()
}

// UpdateThreatNodes evolves all threat nodes by one tick
func UpdateThreatNodes(manager *common.EntityManager, currentTick int64) error {
	for _, result := range manager.World.Query(ThreatNodeTag) {
		entity := result.Entity
		threatData := common.GetComponentType[*ThreatNodeData](entity, ThreatNodeComponent)

		if threatData == nil {
			continue
		}

		// Apply growth
		growthAmount := threatData.GrowthRate
		if threatData.IsContained {
			growthAmount *= ContainmentSlowdown // Player presence slows growth
		}

		threatData.GrowthProgress += growthAmount

		// Check for evolution
		if threatData.GrowthProgress >= 1.0 {
			EvolveThreatNode(manager, entity, threatData)
			threatData.GrowthProgress = 0.0
		}
	}
	return nil
}

// EvolveThreatNode increases threat intensity
func EvolveThreatNode(manager *common.EntityManager, entity *ecs.Entity, threatData *ThreatNodeData) {
	params := GetThreatTypeParams(threatData.ThreatType)

	// Increase intensity (cap at threat-specific max)
	if threatData.Intensity < params.MaxIntensity {
		oldIntensity := threatData.Intensity
		threatData.Intensity++

		// Update influence radius
		influenceData := common.GetComponentType[*InfluenceData](entity, InfluenceComponent)
		if influenceData != nil {
			influenceData.Radius = params.BaseRadius + threatData.Intensity
			influenceData.EffectStrength = float64(threatData.Intensity) * 0.1
		}

		// Log evolution event
		LogEvent(EventThreatEvolved, GetCurrentTick(manager), entity.GetID(),
			formatEventString("Threat evolved %d -> %d", oldIntensity, threatData.Intensity))

		// Trigger evolution effect (spawn child nodes, etc.)
		ExecuteThreatEvolutionEffect(manager, entity, threatData)
	}
}

// ExecuteThreatEvolutionEffect applies type-specific evolution behavior
func ExecuteThreatEvolutionEffect(manager *common.EntityManager, entity *ecs.Entity, threatData *ThreatNodeData) {
	params := GetThreatTypeParams(threatData.ThreatType)

	if !params.CanSpawnChildren {
		return
	}

	switch threatData.ThreatType {
	case ThreatNecromancer:
		// Spawn child node at tier 3, 6, 9
		if threatData.Intensity%ChildNodeSpawnThreshold == 0 {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			if pos != nil {
				SpawnChildThreatNode(manager, *pos, ThreatNecromancer, 1)
			}
		}
	case ThreatCorruption:
		// Spread to adjacent tiles
		SpreadCorruption(manager, entity, threatData)
	}
}

// SpawnChildThreatNode creates a nearby threat node
func SpawnChildThreatNode(manager *common.EntityManager, parentPos coords.LogicalPosition, threatType ThreatType, intensity int) {
	// Find nearby unoccupied position (within radius 3)
	for attempts := 0; attempts < 10; attempts++ {
		offsetX := common.RandomInt(7) - 3 // -3 to 3
		offsetY := common.RandomInt(7) - 3
		newPos := coords.LogicalPosition{
			X: parentPos.X + offsetX,
			Y: parentPos.Y + offsetY,
		}

		// Check if position is free (no other threat nodes)
		entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(newPos)
		occupied := false
		for _, entityID := range entityIDs {
			if manager.HasComponent(entityID, ThreatNodeComponent) {
				occupied = true
				break
			}
		}

		if !occupied {
			CreateThreatNode(manager, newPos, threatType, intensity, GetCurrentTick(manager))
			return
		}
	}
}

// SpreadCorruption spreads corruption to adjacent tiles
func SpreadCorruption(manager *common.EntityManager, entity *ecs.Entity, threatData *ThreatNodeData) {
	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
	if pos == nil {
		return
	}

	// Try to spawn corruption on adjacent tile
	adjacents := []coords.LogicalPosition{
		{X: pos.X + 1, Y: pos.Y},
		{X: pos.X - 1, Y: pos.Y},
		{X: pos.X, Y: pos.Y + 1},
		{X: pos.X, Y: pos.Y - 1},
	}

	// Pick random adjacent
	if len(adjacents) > 0 {
		targetPos := adjacents[common.RandomInt(len(adjacents))]

		// Check if already corrupted
		entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(targetPos)
		for _, entityID := range entityIDs {
			if manager.HasComponent(entityID, ThreatNodeComponent) {
				return // Already has a threat
			}
		}

		// Spawn new corruption
		CreateThreatNode(manager, targetPos, ThreatCorruption, 1, GetCurrentTick(manager))
	}
}

// DestroyThreatNode removes a threat from the overworld
func DestroyThreatNode(manager *common.EntityManager, threatEntity *ecs.Entity) {
	// Get threat data for logging before destruction
	threatData := common.GetComponentType[*ThreatNodeData](threatEntity, ThreatNodeComponent)
	pos := common.GetComponentType[*coords.LogicalPosition](threatEntity, common.PositionComponent)

	// Log destruction event
	if threatData != nil {
		location := "unknown"
		if pos != nil {
			location = formatEventString("(%d, %d)", pos.X, pos.Y)
		}

		LogEvent(EventThreatDestroyed, GetCurrentTick(manager), threatEntity.GetID(),
			formatEventString("%s destroyed at %s",
				threatData.ThreatType.String(), location))
	}

	// Remove entity
	if pos != nil {
		manager.CleanDisposeEntity(threatEntity, pos)
	} else {
		manager.World.DisposeEntities(threatEntity)
	}
}
