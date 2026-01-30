package overworld

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CreateFaction creates a new NPC faction entity
func CreateFaction(
	manager *common.EntityManager,
	factionType FactionType,
	homePosition coords.LogicalPosition,
	initialStrength int,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	factionData := &OverworldFactionData{
		FactionID:     entity.GetID(),
		FactionType:   factionType,
		Strength:      initialStrength,
		TerritorySize: 1,
		Disposition:   -50, // Default hostile
		CurrentIntent: IntentExpand,
		GrowthRate:    1.0,
	}

	territoryData := &TerritoryData{
		OwnedTiles:  []coords.LogicalPosition{homePosition},
		BorderTiles: []coords.LogicalPosition{},
	}

	intentData := &StrategicIntentData{
		Intent:         FactionIntent(IntentExpand),
		TargetPosition: nil,
		TicksRemaining: DefaultIntentTickDuration,
		Priority:       0.5,
	}

	entity.AddComponent(OverworldFactionComponent, factionData)
	entity.AddComponent(TerritoryComponent, territoryData)
	entity.AddComponent(StrategicIntentComponent, intentData)

	// Spawn initial threat at home position
	SpawnThreatForFaction(manager, entity, homePosition, factionType)

	return entity.GetID()
}

// UpdateFactions executes faction AI for all factions
func UpdateFactions(manager *common.EntityManager, currentTick int64) error {
	for _, result := range manager.World.Query(OverworldFactionTag) {
		entity := result.Entity
		factionData := common.GetComponentType[*OverworldFactionData](entity, OverworldFactionComponent)
		intentData := common.GetComponentType[*StrategicIntentData](entity, StrategicIntentComponent)

		if factionData == nil || intentData == nil {
			continue
		}

		// Decrement intent timer
		intentData.TicksRemaining--

		// Re-evaluate intent if timer expired
		if intentData.TicksRemaining <= 0 {
			EvaluateFactionIntent(manager, entity, factionData, intentData)
			intentData.TicksRemaining = DefaultIntentTickDuration
		}

		// Execute current intent
		ExecuteFactionIntent(manager, entity, factionData, intentData)
	}

	return nil
}

// EvaluateFactionIntent determines what faction should do next
func EvaluateFactionIntent(
	manager *common.EntityManager,
	entity *ecs.Entity,
	factionData *OverworldFactionData,
	intentData *StrategicIntentData,
) {
	// Score possible actions
	expandScore := ScoreExpansion(manager, entity, factionData)
	fortifyScore := ScoreFortification(manager, entity, factionData)
	raidScore := ScoreRaiding(manager, entity, factionData)
	retreatScore := ScoreRetreat(manager, entity, factionData)

	// Choose highest-scoring action
	maxScore := expandScore
	newIntent := IntentExpand
	newPriority := 0.5

	if fortifyScore > maxScore {
		maxScore = fortifyScore
		newIntent = IntentFortify
	}
	if raidScore > maxScore {
		maxScore = raidScore
		newIntent = IntentRaid
	}
	if retreatScore > maxScore {
		maxScore = retreatScore
		newIntent = IntentRetreat
	}

	// If all scores are low, go idle
	if maxScore < 2.0 {
		newIntent = IntentIdle
	}

	intentData.Intent = newIntent
	intentData.Priority = newPriority
	factionData.CurrentIntent = newIntent
}

// ExecuteFactionIntent performs the chosen action
func ExecuteFactionIntent(
	manager *common.EntityManager,
	entity *ecs.Entity,
	factionData *OverworldFactionData,
	intentData *StrategicIntentData,
) {
	switch intentData.Intent {
	case IntentExpand:
		ExpandTerritory(manager, entity, factionData)
	case IntentFortify:
		FortifyTerritory(manager, entity, factionData)
	case IntentRaid:
		ExecuteRaid(manager, entity, factionData)
	case IntentRetreat:
		AbandonTerritory(manager, entity, factionData)
	case IntentIdle:
		// Do nothing
	}
}

// ExpandTerritory claims adjacent tiles
func ExpandTerritory(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) {
	territoryData := common.GetComponentType[*TerritoryData](entity, TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return
	}

	// Don't expand beyond limit
	if factionData.TerritorySize >= MaxTerritorySize {
		return
	}

	// Pick random owned tile
	randomTile := GetRandomTileFromSlice(territoryData.OwnedTiles)
	if randomTile == nil {
		return
	}

	// Try to claim adjacent tile
	adjacents := GetCardinalNeighbors(*randomTile)

	for _, adj := range adjacents {
		// Check bounds using configured map dimensions
		if adj.X < 0 || adj.X >= DefaultMapWidth || adj.Y < 0 || adj.Y >= DefaultMapHeight {
			continue
		}

		// Check if tile is unoccupied
		if !IsTileOwnedByAnyFaction(manager, adj) {
			territoryData.OwnedTiles = append(territoryData.OwnedTiles, adj)
			factionData.TerritorySize++

			// Log expansion event
			LogEvent(EventFactionExpanded, GetCurrentTick(manager), entity.GetID(),
				formatEventString("%s expanded to (%d, %d)",
					factionData.FactionType.String(), adj.X, adj.Y), nil)

			// Spawn threat on newly claimed tile
			if common.RandomInt(100) < ExpansionThreatSpawnChance {
				SpawnThreatForFaction(manager, entity, adj, factionData.FactionType)
			}

			return
		}
	}
}

// FortifyTerritory increases faction strength and spawns threats
func FortifyTerritory(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) {
	territoryData := common.GetComponentType[*TerritoryData](entity, TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return
	}

	// Increase strength
	factionData.Strength += FortificationStrengthGain

	// Spawn threat on random owned tile
	if common.RandomInt(100) < FortifyThreatSpawnChance {
		randomTile := GetRandomTileFromSlice(territoryData.OwnedTiles)
		if randomTile != nil {
			SpawnThreatForFaction(manager, entity, *randomTile, factionData.FactionType)
		}
	}
}

// ExecuteRaid attacks player or rival faction
func ExecuteRaid(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) {
	territoryData := common.GetComponentType[*TerritoryData](entity, TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return
	}

	// Spawn aggressive threat near faction border
	// Pick random border tile
	randomTile := GetRandomTileFromSlice(territoryData.OwnedTiles)
	if randomTile == nil {
		return
	}

	// Spawn higher-intensity threat for raids
	threatType := MapFactionToThreatType(factionData.FactionType)
	intensity := 3 + (factionData.Strength / 3) // Stronger factions spawn stronger raids
	currentTick := GetCurrentTick(manager)

	CreateThreatNode(manager, *randomTile, threatType, intensity, currentTick)

	// Log raid event
	LogEvent(EventFactionRaid, currentTick, entity.GetID(),
		formatEventString("%s launched raid! Spawned intensity %d %s at (%d, %d)",
			factionData.FactionType.String(), intensity, threatType.String(),
			randomTile.X, randomTile.Y), nil)
}

// AbandonTerritory shrinks faction
func AbandonTerritory(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) {
	territoryData := common.GetComponentType[*TerritoryData](entity, TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) <= 1 {
		return
	}

	// Abandon outermost tile (simplified - just remove last)
	territoryData.OwnedTiles = territoryData.OwnedTiles[:len(territoryData.OwnedTiles)-1]
	factionData.TerritorySize--
}

// IsTileOwnedByAnyFaction checks if a tile is controlled
func IsTileOwnedByAnyFaction(manager *common.EntityManager, pos coords.LogicalPosition) bool {
	for _, result := range manager.World.Query(OverworldFactionTag) {
		territoryData := common.GetComponentType[*TerritoryData](result.Entity, TerritoryComponent)
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

// SpawnThreatForFaction creates a threat matching faction type
func SpawnThreatForFaction(
	manager *common.EntityManager,
	factionEntity *ecs.Entity,
	position coords.LogicalPosition,
	factionType FactionType,
) ecs.EntityID {
	threatType := MapFactionToThreatType(factionType)
	intensity := 1 + common.RandomInt(3) // Random intensity 1-3

	return CreateThreatNode(manager, position, threatType, intensity, GetCurrentTick(manager))
}
