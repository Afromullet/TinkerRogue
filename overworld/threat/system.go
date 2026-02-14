package threat

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/node"
	"game_main/templates"
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

// CreateThreatNode spawns a new threat at a position.
// Delegates to node.CreateNode with threat-specific setup (encounter selection, faction owner).
func CreateThreatNode(
	manager *common.EntityManager,
	pos coords.LogicalPosition,
	threatType core.ThreatType,
	initialIntensity int,
	currentTick int64,
) ecs.EntityID {
	// Determine faction owner from node definition
	nodeDef := core.GetNodeRegistry().GetNodeByType(threatType)
	ownerID := ""
	if nodeDef != nil {
		ownerID = nodeDef.FactionID
	}

	// Select random encounter variant
	encounterID := selectRandomEncounterForNode(threatType)

	// Cap intensity at max allowed
	if maxIntensity := core.GetMaxThreatIntensity(); initialIntensity > maxIntensity {
		initialIntensity = maxIntensity
	}

	entityID, err := node.CreateNode(manager, node.CreateNodeParams{
		Position:         pos,
		NodeTypeID:       string(threatType),
		OwnerID:          ownerID,
		InitialIntensity: initialIntensity,
		EncounterID:      encounterID,
		CurrentTick:      currentTick,
	})
	if err != nil {
		fmt.Printf("WARNING: CreateThreatNode failed: %v\n", err)
		return 0
	}

	// Log threat spawn event
	core.LogEvent(core.EventThreatSpawned, currentTick, entityID,
		fmt.Sprintf("%s spawned at (%d, %d) with intensity %d",
			threatType.String(), pos.X, pos.Y, initialIntensity), nil)

	return entityID
}

// UpdateThreatNodes evolves all threat nodes by one tick.
// Uses unified OverworldNodeComponent, filters by threat category.
func UpdateThreatNodes(manager *common.EntityManager, currentTick int64) error {
	for _, result := range core.OverworldNodeView.Get() {
		entity := result.Entity
		nodeData := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)

		if nodeData == nil || nodeData.Category != core.NodeCategoryThreat {
			continue
		}

		// Apply growth (scaled by difficulty)
		growthAmount := nodeData.GrowthRate * templates.GlobalDifficulty.Overworld().ThreatGrowthScale
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

	// Increase intensity (cap at difficulty-adjusted max)
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
			fmt.Sprintf("Threat evolved %d -> %d", oldIntensity, nodeData.Intensity), nil)

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
	case string(core.ThreatNecromancer):
		// Spawn child node at tier 3 (with max intensity 5, only spawns once)
		if nodeData.Intensity%templates.OverworldConfigTemplate.ThreatGrowth.ChildNodeSpawnThreshold == 0 {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			if pos != nil {
				if !SpawnChildThreatNode(manager, *pos, "necromancer", 1) {
					core.LogEvent(core.EventThreatEvolved, core.GetCurrentTick(manager), entity.GetID(),
						fmt.Sprintf("Failed to spawn child node (no valid positions)"), nil)
				}
			}
		}
	case string(core.ThreatCorruption):
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

		// Check if position is walkable and free (no other threat nodes)
		if core.IsTileWalkable(newPos) && !core.IsThreatAtPosition(manager, newPos) {
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

	// Check if walkable and not already corrupted
	if !core.IsTileWalkable(targetPos) || core.IsThreatAtPosition(manager, targetPos) {
		return
	}

	// Spawn new corruption
	CreateThreatNode(manager, targetPos, "corruption", 1, core.GetCurrentTick(manager))
}

// DestroyThreatNode removes a threat from the overworld.
// Logs the destruction event, then delegates to node.DestroyNode for cleanup.
func DestroyThreatNode(manager *common.EntityManager, threatEntity *ecs.Entity) {
	nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)

	// Log destruction event before disposal
	if nodeData != nil {
		pos := common.GetComponentType[*coords.LogicalPosition](threatEntity, common.PositionComponent)
		location := "unknown"
		if pos != nil {
			location = fmt.Sprintf("(%d, %d)", pos.X, pos.Y)
		}

		nodeDef := core.GetNodeRegistry().GetNodeByID(nodeData.NodeTypeID)
		displayName := nodeData.NodeTypeID
		if nodeDef != nil {
			displayName = nodeDef.DisplayName
		}

		core.LogEvent(core.EventThreatDestroyed, core.GetCurrentTick(manager), threatEntity.GetID(),
			fmt.Sprintf("%s destroyed at %s", displayName, location), nil)
	}

	node.DestroyNode(manager, threatEntity)
}
