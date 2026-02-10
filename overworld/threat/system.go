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
	// Get the node definition to find the faction
	node := core.GetNodeRegistry().GetNodeByType(threatType)
	if node == nil || node.FactionID == "" {
		return threatType.EncounterTypeID()
	}

	// Get all encounters for this faction
	encounters := core.GetNodeRegistry().GetEncountersByFaction(node.FactionID)
	if len(encounters) == 0 {
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

	// Get config for this threat type
	params := core.GetThreatTypeParamsFromConfig(threatType)

	// Add influence component
	entity.AddComponent(core.InfluenceComponent, &core.InfluenceData{
		Radius:        params.BaseRadius + initialIntensity,
		BaseMagnitude: core.CalculateBaseMagnitude(initialIntensity),
	})

	// Add unified OverworldNodeComponent
	nodeDef := core.GetNodeRegistry().GetNodeByType(threatType)
	ownerID := ""
	if nodeDef != nil {
		ownerID = nodeDef.FactionID
	}
	entity.AddComponent(core.OverworldNodeComponent, &core.OverworldNodeData{
		NodeID:         entity.GetID(),
		NodeTypeID:     string(threatType),
		Category:       core.NodeCategoryThreat,
		OwnerID:        ownerID,
		EncounterID:    encounterID,
		Intensity:      initialIntensity,
		GrowthProgress: 0.0,
		GrowthRate:     params.BaseGrowthRate,
		IsContained:    false,
		CreatedTick:    currentTick,
	})

	// Register in position system
	common.GlobalPositionSystem.AddEntity(entity.GetID(), pos)

	// Log threat spawn event
	core.LogEvent(core.EventThreatSpawned, currentTick, entity.GetID(),
		core.FormatEventString("%s spawned at (%d, %d) with intensity %d",
			threatType.String(), pos.X, pos.Y, initialIntensity), nil)

	return entity.GetID()
}

// UpdateThreatNodes evolves all threat nodes by one tick.
// Uses unified OverworldNodeComponent, filters by threat category.
func UpdateThreatNodes(manager *common.EntityManager, currentTick int64) error {
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		entity := result.Entity
		nodeData := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)

		if nodeData == nil || nodeData.Category != core.NodeCategoryThreat {
			continue
		}

		// Apply growth
		growthAmount := nodeData.GrowthRate
		if nodeData.IsContained {
			growthAmount *= core.GetContainmentSlowdown() // Player presence slows growth
		}

		// Apply influence interaction modifier (synergy boosts, competition/suppression slows)
		if manager.HasComponent(entity.GetID(), core.InteractionComponent) {
			interactionData := common.GetComponentType[*core.InteractionData](entity, core.InteractionComponent)
			if interactionData != nil {
				growthAmount *= interactionData.NetModifier
			}
		}

		nodeData.GrowthProgress += growthAmount

		// Check for evolution
		if nodeData.GrowthProgress >= 1.0 {
			EvolveThreatNode(manager, entity, nodeData)
			nodeData.GrowthProgress = 0.0
		}
	}
	return nil
}

// EvolveThreatNode increases threat intensity.
// Uses unified OverworldNodeData.
func EvolveThreatNode(manager *common.EntityManager, entity *ecs.Entity, nodeData *core.OverworldNodeData) {
	params := core.GetThreatTypeParamsFromConfig(core.ThreatType(nodeData.NodeTypeID))

	// Increase intensity (cap at global max)
	if nodeData.Intensity < core.GetMaxThreatIntensity() {
		oldIntensity := nodeData.Intensity
		nodeData.Intensity++

		// Update influence radius
		influenceData := common.GetComponentType[*core.InfluenceData](entity, core.InfluenceComponent)
		if influenceData != nil {
			influenceData.Radius = params.BaseRadius + nodeData.Intensity
			influenceData.BaseMagnitude = core.CalculateBaseMagnitude(nodeData.Intensity)
		}

		// Log evolution event
		core.LogEvent(core.EventThreatEvolved, core.GetCurrentTick(manager), entity.GetID(),
			core.FormatEventString("Threat evolved %d -> %d", oldIntensity, nodeData.Intensity), nil)

		// Trigger evolution effect (spawn child nodes, etc.)
		ExecuteThreatEvolutionEffect(manager, entity, nodeData)
	}
}

// ExecuteThreatEvolutionEffect applies type-specific evolution behavior
func ExecuteThreatEvolutionEffect(manager *common.EntityManager, entity *ecs.Entity, nodeData *core.OverworldNodeData) {
	params := core.GetThreatTypeParamsFromConfig(core.ThreatType(nodeData.NodeTypeID))

	if !params.CanSpawnChildren {
		return
	}

	switch nodeData.NodeTypeID {
	case "necromancer":
		// Spawn child node at tier 3 (with max intensity 5, only spawns once)
		if nodeData.Intensity%core.GetChildNodeSpawnThreshold() == 0 {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			if pos != nil {
				if !SpawnChildThreatNode(manager, *pos, "necromancer", 1) {
					core.LogEvent(core.EventThreatEvolved, core.GetCurrentTick(manager), entity.GetID(),
						core.FormatEventString("Failed to spawn child node (no valid positions)"), nil)
				}
			}
		}
	case "corruption":
		// Spread to adjacent tiles
		SpreadCorruption(manager, entity)
	}
}

// SpawnChildThreatNode creates a nearby threat node.
// Returns true if spawn succeeded, false if no valid position found.
func SpawnChildThreatNode(manager *common.EntityManager, parentPos coords.LogicalPosition, threatType core.ThreatType, intensity int) bool {
	// Find nearby unoccupied position (within radius 3)
	const maxAttempts = 10
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
func SpreadCorruption(manager *common.EntityManager, entity *ecs.Entity) {
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
	CreateThreatNode(manager, targetPos, "corruption", 1, core.GetCurrentTick(manager))
}

// DestroyThreatNode removes a threat from the overworld.
// Uses unified OverworldNodeData for logging.
func DestroyThreatNode(manager *common.EntityManager, threatEntity *ecs.Entity) {
	nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
	pos := common.GetComponentType[*coords.LogicalPosition](threatEntity, common.PositionComponent)

	// Log destruction event
	if nodeData != nil {
		location := "unknown"
		if pos != nil {
			location = core.FormatEventString("(%d, %d)", pos.X, pos.Y)
		}

		nodeDef := core.GetNodeRegistry().GetNodeByID(nodeData.NodeTypeID)
		displayName := nodeData.NodeTypeID
		if nodeDef != nil {
			displayName = nodeDef.DisplayName
		}

		core.LogEvent(core.EventThreatDestroyed, core.GetCurrentTick(manager), threatEntity.GetID(),
			core.FormatEventString("%s destroyed at %s", displayName, location), nil)
	}

	// Remove entity
	if pos != nil {
		manager.CleanDisposeEntity(threatEntity, pos)
	} else {
		manager.World.DisposeEntities(threatEntity)
	}
}
