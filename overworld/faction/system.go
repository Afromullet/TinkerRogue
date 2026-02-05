package faction

import (
	"game_main/common"
	"game_main/config"
	"game_main/overworld/core"
	"game_main/overworld/threat"
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
		OwnedTiles:  []coords.LogicalPosition{homePosition},
		BorderTiles: []coords.LogicalPosition{},
	}

	intentData := &core.StrategicIntentData{
		Intent:         core.FactionIntent(core.IntentExpand),
		TargetPosition: nil,
		TicksRemaining: core.GetDefaultIntentTickDuration(),
		Priority:       0.5,
	}

	entity.AddComponent(core.OverworldFactionComponent, factionData)
	entity.AddComponent(core.TerritoryComponent, territoryData)
	entity.AddComponent(core.StrategicIntentComponent, intentData)

	// Spawn initial threat at home position
	SpawnThreatForFaction(manager, entity, homePosition, factionType)

	return entity.GetID()
}

// UpdateFactions executes faction AI for all factions
func UpdateFactions(manager *common.EntityManager, currentTick int64) error {
	for _, result := range manager.World.Query(core.OverworldFactionTag) {
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
			intentData.TicksRemaining = core.GetDefaultIntentTickDuration()
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
	factionData *core.OverworldFactionData,
	intentData *core.StrategicIntentData,
) {
	// Score possible actions
	expandScore := ScoreExpansion(manager, entity, factionData)
	fortifyScore := ScoreFortification(manager, entity, factionData)
	raidScore := ScoreRaiding(manager, entity, factionData)
	retreatScore := ScoreRetreat(manager, entity, factionData)

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
	if maxScore < core.GetIdleScoreThreshold() {
		newIntent = core.IntentIdle
	}

	intentData.Intent = newIntent
	intentData.Priority = newPriority
	factionData.CurrentIntent = newIntent
}

// TODO, consider using an interface for intent
// ExecuteFactionIntent performs the chosen action
func ExecuteFactionIntent(
	manager *common.EntityManager,
	entity *ecs.Entity,
	factionData *core.OverworldFactionData,
	intentData *core.StrategicIntentData,
) {
	switch intentData.Intent {
	case core.IntentExpand:
		ExpandTerritory(manager, entity, factionData)
	case core.IntentFortify:
		FortifyTerritory(manager, entity, factionData)
	case core.IntentRaid:
		ExecuteRaid(manager, entity, factionData)
	case core.IntentRetreat:
		AbandonTerritory(manager, entity, factionData)
	case core.IntentIdle:
		// Do nothing
	}
}

// ExpandTerritory claims adjacent tiles
func ExpandTerritory(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) {
	territoryData := common.GetComponentType[*core.TerritoryData](entity, core.TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return
	}

	// Don't expand beyond limit
	if factionData.TerritorySize >= core.GetMaxTerritorySize() {
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
		if adj.X < 0 || adj.X >= config.DefaultMapWidth || adj.Y < 0 || adj.Y >= config.DefaultMapHeight {
			continue
		}

		// Check if tile is unoccupied
		if !IsTileOwnedByAnyFaction(manager, adj) {
			territoryData.OwnedTiles = append(territoryData.OwnedTiles, adj)
			factionData.TerritorySize++

			// Log expansion event
			core.LogEvent(core.EventFactionExpanded, core.GetCurrentTick(manager), entity.GetID(),
				core.FormatEventString("%s expanded to (%d, %d)",
					factionData.FactionType.String(), adj.X, adj.Y), nil)

			// Spawn threat on newly claimed tile
			if common.RandomInt(100) < core.GetExpansionThreatSpawnChance() {
				SpawnThreatForFaction(manager, entity, adj, factionData.FactionType)
			}

			return
		}
	}
}

// FortifyTerritory increases faction strength and spawns threats
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
}

// ExecuteRaid attacks player or rival faction
func ExecuteRaid(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) {
	territoryData := common.GetComponentType[*core.TerritoryData](entity, core.TerritoryComponent)
	if territoryData == nil || len(territoryData.OwnedTiles) == 0 {
		return
	}

	// Spawn aggressive threat near faction border
	// Pick random border tile
	randomTile := core.GetRandomTileFromSlice(territoryData.OwnedTiles)
	if randomTile == nil {
		return
	}

	// Spawn higher-intensity threat for raids (formula from config)
	threatType := core.MapFactionToThreatType(factionData.FactionType)
	baseIntensity := core.GetRaidBaseIntensity()
	intensityScale := core.GetRaidIntensityScale()
	intensity := baseIntensity + int(float64(factionData.Strength)*intensityScale)
	currentTick := core.GetCurrentTick(manager)

	threat.CreateThreatNode(manager, *randomTile, threatType, intensity, currentTick)

	// Log raid event
	core.LogEvent(core.EventFactionRaid, currentTick, entity.GetID(),
		core.FormatEventString("%s launched raid! Spawned intensity %d %s at (%d, %d)",
			factionData.FactionType.String(), intensity, threatType.String(),
			randomTile.X, randomTile.Y), nil)
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
	for _, result := range manager.World.Query(core.OverworldFactionTag) {
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

// SpawnThreatForFaction creates a threat matching faction type
func SpawnThreatForFaction(
	manager *common.EntityManager,
	factionEntity *ecs.Entity,
	position coords.LogicalPosition,
	factionType core.FactionType,
) ecs.EntityID {
	threatType := core.MapFactionToThreatType(factionType)
	intensity := 1 + common.RandomInt(3) // Random intensity 1-3

	return threat.CreateThreatNode(manager, position, threatType, intensity, core.GetCurrentTick(manager))
}
