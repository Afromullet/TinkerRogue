package faction

import (
	"fmt"

	"game_main/common"
	"game_main/config"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/overworld/threat"
	"game_main/templates"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CreateFaction creates a new NPC faction entity
func CreateFaction(
	manager *common.EntityManager,
	factionType core.FactionType,
	homePosition coords.LogicalPosition,
	initialStrength int,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	factionData := &core.OverworldFactionData{
		FactionID:     entity.GetID(),
		FactionType:   factionType,
		Strength:      initialStrength,
		TerritorySize: 1,
		Disposition:   -50, // Default hostile
		CurrentIntent: core.IntentExpand,
		GrowthRate:    1.0,
	}

	territoryData := &core.TerritoryData{
		OwnedTiles: []coords.LogicalPosition{homePosition},
	}

	intentData := &core.StrategicIntentData{
		Intent:         core.FactionIntent(core.IntentExpand),
		TargetPosition: nil,
		TicksRemaining: templates.OverworldConfigTemplate.FactionAI.DefaultIntentTickDuration,
		Priority:       0.5,
	}

	entity.AddComponent(core.OverworldFactionComponent, factionData)
	entity.AddComponent(core.TerritoryComponent, territoryData)
	entity.AddComponent(core.StrategicIntentComponent, intentData)
	entity.AddComponent(common.ResourceStockpileComponent, common.NewResourceStockpile(
		config.DefaultFactionStartingGold,
		config.DefaultFactionStartingIron,
		config.DefaultFactionStartingWood,
		config.DefaultFactionStartingStone,
	))

	// Spawn initial threat at home position
	SpawnThreatForFaction(manager, entity, homePosition, factionType)

	return entity.GetID()
}

// UpdateFactions executes faction AI for all factions.
// Returns a PendingRaid if a faction raids a garrisoned player node this tick.
func UpdateFactions(manager *common.EntityManager, currentTick int64) (*core.PendingRaid, error) {
	var pendingRaid *core.PendingRaid

	for _, result := range core.OverworldFactionView.Get() {
		entity := result.Entity
		factionData := common.GetComponentType[*core.OverworldFactionData](entity, core.OverworldFactionComponent)
		intentData := common.GetComponentType[*core.StrategicIntentData](entity, core.StrategicIntentComponent)

		if factionData == nil || intentData == nil {
			continue
		}

		// Decrement intent timer
		intentData.TicksRemaining--

		// Re-evaluate intent if timer expired
		if intentData.TicksRemaining <= 0 {
			EvaluateFactionIntent(manager, entity, factionData, intentData)
			intentData.TicksRemaining = templates.OverworldConfigTemplate.FactionAI.DefaultIntentTickDuration
		}

		// Execute current intent, collecting any raid result
		raid := ExecuteFactionIntent(manager, entity, factionData, intentData)
		if raid != nil && pendingRaid == nil {
			pendingRaid = raid // Only one raid per tick
		}
	}

	return pendingRaid, nil
}

// EvaluateFactionIntent determines what faction should do next
func EvaluateFactionIntent(
	manager *common.EntityManager,
	entity *ecs.Entity,
	factionData *core.OverworldFactionData,
	intentData *core.StrategicIntentData,
) {
	// Score possible actions
	expandScore := ScoreExpansion(factionData)
	fortifyScore := ScoreFortification(factionData)
	raidScore := ScoreRaiding(factionData)
	retreatScore := ScoreRetreat(factionData)

	// Choose highest-scoring action
	maxScore := expandScore
	newIntent := core.IntentExpand
	newPriority := 0.5

	if fortifyScore > maxScore {
		maxScore = fortifyScore
		newIntent = core.IntentFortify
	}
	if raidScore > maxScore {
		maxScore = raidScore
		newIntent = core.IntentRaid
	}
	if retreatScore > maxScore {
		maxScore = retreatScore
		newIntent = core.IntentRetreat
	}

	// If all scores are low, go idle (threshold from config)
	if maxScore < templates.OverworldConfigTemplate.FactionScoringControl.IdleScoreThreshold {
		newIntent = core.IntentIdle
	}

	intentData.Intent = newIntent
	intentData.Priority = newPriority
	factionData.CurrentIntent = newIntent
}

// TODO, consider using an interface for intent
// ExecuteFactionIntent performs the chosen action.
// Returns a PendingRaid if a raid targets a garrisoned player node.
func ExecuteFactionIntent(
	manager *common.EntityManager,
	entity *ecs.Entity,
	factionData *core.OverworldFactionData,
	intentData *core.StrategicIntentData,
) *core.PendingRaid {
	switch intentData.Intent {
	case core.IntentExpand:
		ExpandTerritory(manager, entity, factionData)
	case core.IntentFortify:
		FortifyTerritory(manager, entity, factionData)
	case core.IntentRaid:
		return ExecuteRaid(manager, entity, factionData)
	case core.IntentRetreat:
		AbandonTerritory(manager, entity, factionData)
	case core.IntentIdle:
		// Do nothing
	}
	return nil
}

// ExpandTerritory claims adjacent tiles
func ExpandTerritory(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) {
	territoryData := common.GetComponentType[*core.TerritoryData](entity, core.TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return
	}

	// Don't expand beyond limit
	if factionData.TerritorySize >= templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize {
		return
	}

	// Pick random owned tile
	randomTile := core.GetRandomTileFromSlice(territoryData.OwnedTiles)
	if randomTile == nil {
		return
	}

	// Try to claim adjacent tile
	adjacents := core.GetCardinalNeighbors(*randomTile)

	for _, adj := range adjacents {
		// Check bounds using configured map dimensions
		if adj.X < 0 || adj.X >= coords.CoordManager.GetDungeonWidth() || adj.Y < 0 || adj.Y >= coords.CoordManager.GetDungeonHeight() {
			continue
		}

		// Skip impassable terrain (mountains, water)
		if !core.IsTileWalkable(adj) {
			continue
		}

		// Check if tile is unoccupied
		if !IsTileOwnedByAnyFaction(manager, adj) {
			territoryData.OwnedTiles = append(territoryData.OwnedTiles, adj)
			factionData.TerritorySize++

			// Log expansion event
			core.LogEvent(core.EventFactionExpanded, core.GetCurrentTick(manager), entity.GetID(),
				fmt.Sprintf("%s expanded to (%d, %d)",
					factionData.FactionType.String(), adj.X, adj.Y), nil)

			// Spawn threat on newly claimed tile
			if common.RandomInt(100) < core.GetExpansionThreatSpawnChance() {
				SpawnThreatForFaction(manager, entity, adj, factionData.FactionType)
			}

			return
		}
	}
}

// FortifyTerritory increases faction strength, spawns threats, and garrisons nodes
func FortifyTerritory(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) {
	territoryData := common.GetComponentType[*core.TerritoryData](entity, core.TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return
	}

	// Increase strength
	factionData.Strength += core.GetFortificationStrengthGain()

	// Spawn threat on random owned tile
	if common.RandomInt(100) < core.GetFortifyThreatSpawnChance() {
		randomTile := core.GetRandomTileFromSlice(territoryData.OwnedTiles)
		if randomTile != nil {
			SpawnThreatForFaction(manager, entity, *randomTile, factionData.FactionType)
		}
	}

	// Create NPC garrisons on ungarrisoned hostile nodes owned by this faction
	for _, result := range core.OverworldNodeView.Get() {
		nodeEntity := result.Entity
		nodeData := common.GetComponentType[*core.OverworldNodeData](nodeEntity, core.OverworldNodeComponent)
		if nodeData == nil {
			continue
		}
		// Only garrison nodes owned by this faction
		if nodeData.OwnerID != factionData.FactionType.String() {
			continue
		}
		// Only garrison if not already garrisoned (30% chance per tick)
		if !garrison.IsNodeGarrisoned(manager, nodeEntity.GetID()) && common.RandomInt(100) < 30 {
			garrison.CreateNPCGarrison(manager, nodeEntity, factionData.FactionType, factionData.Strength)
		}
	}
}

// ExecuteRaid attacks player or rival faction.
// Returns a PendingRaid if a garrisoned player node is targeted.
func ExecuteRaid(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) *core.PendingRaid {
	territoryData := common.GetComponentType[*core.TerritoryData](entity, core.TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return nil
	}

	// Check for nearby garrisoned player nodes to target
	playerNodes := garrison.FindPlayerNodesNearFaction(manager, territoryData.OwnedTiles)
	for _, nodeID := range playerNodes {
		if garrison.IsNodeGarrisoned(manager, nodeID) {
			// Target this garrisoned node with a raid
			nodeEntity := manager.FindEntityByID(nodeID)
			if nodeEntity == nil {
				continue
			}
			pos := common.GetComponentType[*coords.LogicalPosition](nodeEntity, common.PositionComponent)
			if pos == nil {
				continue
			}

			currentTick := core.GetCurrentTick(manager)
			core.LogEvent(core.EventGarrisonAttacked, currentTick, entity.GetID(),
				fmt.Sprintf("%s raiding garrisoned node %d at (%d, %d)",
					factionData.FactionType.String(), nodeID, pos.X, pos.Y), nil)

			return &core.PendingRaid{
				AttackingFactionType: factionData.FactionType,
				AttackingStrength:    factionData.Strength,
				TargetNodeID:         nodeID,
				TargetNodePosition:   *pos,
			}
		}
	}

	// No garrisoned target found - existing behavior: spawn threat near border
	randomTile := core.GetRandomTileFromSlice(territoryData.OwnedTiles)
	if randomTile == nil {
		return nil
	}

	// Spawn higher-intensity threat for raids (formula from config, difficulty-adjusted)
	threatType := core.MapFactionToThreatType(factionData.FactionType)
	baseIntensity := core.GetRaidBaseIntensity()
	intensityScale := templates.OverworldConfigTemplate.FactionScoringControl.RaidIntensityScale
	intensity := baseIntensity + int(float64(factionData.Strength)*intensityScale)
	currentTick := core.GetCurrentTick(manager)

	threat.CreateThreatNode(manager, *randomTile, threatType, intensity, currentTick)

	// Log raid event
	core.LogEvent(core.EventFactionRaid, currentTick, entity.GetID(),
		fmt.Sprintf("%s launched raid! Spawned intensity %d %s at (%d, %d)",
			factionData.FactionType.String(), intensity, threatType.String(),
			randomTile.X, randomTile.Y), nil)

	return nil
}

// AbandonTerritory shrinks faction
func AbandonTerritory(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) {
	territoryData := common.GetComponentType[*core.TerritoryData](entity, core.TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) <= 1 {
		return
	}

	// Abandon outermost tile (simplified - just remove last)
	territoryData.OwnedTiles = territoryData.OwnedTiles[:len(territoryData.OwnedTiles)-1]
	factionData.TerritorySize--
}

// IsTileOwnedByAnyFaction checks if a tile is controlled
func IsTileOwnedByAnyFaction(manager *common.EntityManager, pos coords.LogicalPosition) bool {
	for _, result := range core.OverworldFactionView.Get() {
		territoryData := common.GetComponentType[*core.TerritoryData](result.Entity, core.TerritoryComponent)
		if territoryData != nil {
			for _, tile := range territoryData.OwnedTiles {
				if tile.X == pos.X && tile.Y == pos.Y {
					return true
				}
			}
		}
	}
	return false
}

// SpawnThreatForFaction creates a threat matching faction type.
// Checks and deducts resource cost from the faction's stockpile. Returns 0 if unaffordable.
func SpawnThreatForFaction(
	manager *common.EntityManager,
	factionEntity *ecs.Entity,
	position coords.LogicalPosition,
	factionType core.FactionType,
) ecs.EntityID {
	threatType := core.MapFactionToThreatType(factionType)

	// Check resource cost
	nodeDef := core.GetNodeRegistry().GetNodeByID(string(threatType))
	if nodeDef != nil {
		stockpile := common.GetComponentType[*common.ResourceStockpile](factionEntity, common.ResourceStockpileComponent)
		if stockpile != nil && !core.CanAfford(stockpile, nodeDef.Cost) {
			return 0 // Cannot afford, skip spawn
		}
		if stockpile != nil {
			core.SpendResources(stockpile, nodeDef.Cost)
		}
	}

	intensity := 1 + common.RandomInt(3) // Random intensity 1-3
	return threat.CreateThreatNode(manager, position, threatType, intensity, core.GetCurrentTick(manager))
}
