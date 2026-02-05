package threat

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// selectRandomEncounterForNode randomly selects an encounter variant for a threat node.
// This is called once when the node is created to determine which encounter variant it uses.
func selectRandomEncounterForNode(threatType core.ThreatType) string {
	// Get the threat definition to find the faction
	threatDef := core.GetThreatRegistry().GetByEnum(threatType)
	if threatDef == nil || threatDef.FactionID == "" {
		// Fallback to default encounter for this threat type
		return threatType.EncounterTypeID()
	}

	// Get all encounters for this faction
	encounters := core.GetNodeRegistry().GetEncountersByFaction(threatDef.FactionID)
	if len(encounters) == 0 {
		// Fallback if no encounters found
		return threatType.EncounterTypeID()
	}

	// Randomly select one encounter from the faction's pool
	selectedEncounter := encounters[common.RandomInt(len(encounters))]
	return selectedEncounter.ID
}

// CreateThreatNode spawns a new threat at a position
func CreateThreatNode(
	manager *common.EntityManager,
	pos coords.LogicalPosition,
	threatType core.ThreatType,
	initialIntensity int,
	currentTick int64,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	// Add position component
	entity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: pos.X,
		Y: pos.Y,
	})

	// Randomly select an encounter variant for this specific node
	encounterID := selectRandomEncounterForNode(threatType)

	// Add threat node data (use config)
	params := core.GetThreatTypeParamsFromConfig(threatType)
	entity.AddComponent(core.ThreatNodeComponent, &core.ThreatNodeData{
		ThreatID:       entity.GetID(),
		ThreatType:     threatType,
		EncounterID:    encounterID,
		Intensity:      initialIntensity,
		GrowthProgress: 0.0,
		GrowthRate:     params.BaseGrowthRate,
		IsContained:    false,
		SpawnedTick:    currentTick,
	})

	// Add influence component
	entity.AddComponent(core.InfluenceComponent, &core.InfluenceData{
		Radius:         params.BaseRadius + initialIntensity,
		EffectType:     params.PrimaryEffect,
		EffectStrength: float64(initialIntensity) * 0.1,
	})

	// Register in position system
	common.GlobalPositionSystem.AddEntity(entity.GetID(), pos)

	// Log threat spawn event
	core.LogEvent(core.EventThreatSpawned, currentTick, entity.GetID(),
		core.FormatEventString("%s spawned at (%d, %d) with intensity %d",
			threatType.String(), pos.X, pos.Y, initialIntensity), nil)

	return entity.GetID()
}

// UpdateThreatNodes evolves all threat nodes by one tick
func UpdateThreatNodes(manager *common.EntityManager, currentTick int64) error {
	for _, result := range manager.World.Query(core.ThreatNodeTag) {
		entity := result.Entity
		threatData := common.GetComponentType[*core.ThreatNodeData](entity, core.ThreatNodeComponent)

		if threatData == nil {
			continue
		}

		// Apply growth
		growthAmount := threatData.GrowthRate
		if threatData.IsContained {
			growthAmount *= core.GetContainmentSlowdown() // Player presence slows growth
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
func EvolveThreatNode(manager *common.EntityManager, entity *ecs.Entity, threatData *core.ThreatNodeData) {
	params := core.GetThreatTypeParamsFromConfig(threatData.ThreatType)

	// Increase intensity (cap at global max)
	if threatData.Intensity < core.GetMaxThreatIntensity() {
		oldIntensity := threatData.Intensity
		threatData.Intensity++

		// Update influence radius
		influenceData := common.GetComponentType[*core.InfluenceData](entity, core.InfluenceComponent)
		if influenceData != nil {
			influenceData.Radius = params.BaseRadius + threatData.Intensity
			influenceData.EffectStrength = float64(threatData.Intensity) * 0.1
		}

		// Log evolution event
		core.LogEvent(core.EventThreatEvolved, core.GetCurrentTick(manager), entity.GetID(),
			core.FormatEventString("Threat evolved %d -> %d", oldIntensity, threatData.Intensity), nil)

		// Trigger evolution effect (spawn child nodes, etc.)
		ExecuteThreatEvolutionEffect(manager, entity, threatData)
	}
}

// ExecuteThreatEvolutionEffect applies type-specific evolution behavior
// TODO, need to expand on this
func ExecuteThreatEvolutionEffect(manager *common.EntityManager, entity *ecs.Entity, threatData *core.ThreatNodeData) {
	params := core.GetThreatTypeParamsFromConfig(threatData.ThreatType)

	if !params.CanSpawnChildren {
		return
	}

	switch threatData.ThreatType {
	case core.ThreatNecromancer:
		// Spawn child node at tier 3 (with max intensity 5, only spawns once)
		if threatData.Intensity%core.GetChildNodeSpawnThreshold() == 0 {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			if pos != nil {
				if !SpawnChildThreatNode(manager, *pos, core.ThreatNecromancer, 1) {
					// Log warning if spawn failed (no valid positions available)
					core.LogEvent(core.EventThreatEvolved, core.GetCurrentTick(manager), entity.GetID(),
						core.FormatEventString("Failed to spawn child node (no valid positions)"), nil)
				}
			}
		}
	case core.ThreatCorruption:
		// Spread to adjacent tiles
		SpreadCorruption(manager, entity, threatData)
	}
}

// SpawnChildThreatNode creates a nearby threat node.
// Returns true if spawn succeeded, false if no valid position found.
func SpawnChildThreatNode(manager *common.EntityManager, parentPos coords.LogicalPosition, threatType core.ThreatType, intensity int) bool {
	// Find nearby unoccupied position (within radius 3)
	maxAttempts := core.GetMaxChildNodeSpawnAttempts()
	for attempts := 0; attempts < maxAttempts; attempts++ {
		offsetX := common.RandomInt(7) - 3 // -3 to 3
		offsetY := common.RandomInt(7) - 3
		newPos := coords.LogicalPosition{
			X: parentPos.X + offsetX,
			Y: parentPos.Y + offsetY,
		}

		// Check if position is free (no other threat nodes)
		if !core.IsThreatAtPosition(manager, newPos) {
			CreateThreatNode(manager, newPos, threatType, intensity, core.GetCurrentTick(manager))
			return true
		}
	}
	// Failed to find valid position after max attempts
	return false
}

// SpreadCorruption spreads corruption to adjacent tiles
func SpreadCorruption(manager *common.EntityManager, entity *ecs.Entity, threatData *core.ThreatNodeData) {
	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
	if pos == nil {
		return
	}

	// Try to spawn corruption on adjacent tile
	adjacents := core.GetCardinalNeighbors(*pos)

	// Pick random adjacent
	targetPos := adjacents[common.RandomInt(len(adjacents))]

	// Check if already corrupted
	if core.IsThreatAtPosition(manager, targetPos) {
		return // Already has a threat
	}

	// Spawn new corruption
	CreateThreatNode(manager, targetPos, core.ThreatCorruption, 1, core.GetCurrentTick(manager))
}

// DestroyThreatNode removes a threat from the overworld
func DestroyThreatNode(manager *common.EntityManager, threatEntity *ecs.Entity) {
	// Get threat data for logging before destruction
	threatData := common.GetComponentType[*core.ThreatNodeData](threatEntity, core.ThreatNodeComponent)
	pos := common.GetComponentType[*coords.LogicalPosition](threatEntity, common.PositionComponent)

	// Log destruction event
	if threatData != nil {
		location := "unknown"
		if pos != nil {
			location = core.FormatEventString("(%d, %d)", pos.X, pos.Y)
		}

		core.LogEvent(core.EventThreatDestroyed, core.GetCurrentTick(manager), threatEntity.GetID(),
			core.FormatEventString("%s destroyed at %s",
				threatData.ThreatType.String(), location), nil)
	}

	// Remove entity
	if pos != nil {
		manager.CleanDisposeEntity(threatEntity, pos)
	} else {
		manager.World.DisposeEntities(threatEntity)
	}
}
