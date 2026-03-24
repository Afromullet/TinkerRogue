package combatcore

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

type CombatFactionManager struct {
	manager     *common.EntityManager
	combatCache *CombatQueryCache
}

func NewCombatFactionManager(manager *common.EntityManager, cache *CombatQueryCache) *CombatFactionManager {
	return &CombatFactionManager{
		manager:     manager,
		combatCache: cache,
	}
}

func (fm *CombatFactionManager) CreateCombatFaction(name string, isPlayer bool) ecs.EntityID {
	playerID := 0
	playerName := ""
	if isPlayer {
		playerID = 1 // Default to Player 1 for single-player
		playerName = "Player 1"
	}
	return fm.CreateFactionWithPlayer(name, playerID, playerName, 0) // 0 = no encounter
}

// CreateFactionWithPlayer creates a faction with specific player assignment and encounter tracking
func (fm *CombatFactionManager) CreateFactionWithPlayer(name string, playerID int, playerName string, encounterID ecs.EntityID) ecs.EntityID {
	faction := fm.manager.World.NewEntity()
	factionID := faction.GetID()

	isPlayerControlled := playerID > 0 // Derive from PlayerID

	faction.AddComponent(CombatFactionComponent, &FactionData{
		FactionID:          factionID,
		Name:               name,
		Mana:               100,
		MaxMana:            100,
		IsPlayerControlled: isPlayerControlled,
		PlayerID:           playerID,
		PlayerName:         playerName,
		EncounterID:        encounterID,
	})

	return factionID
}

func (fm *CombatFactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID, position coords.LogicalPosition) error {

	faction := fm.combatCache.FindFactionByID(factionID)
	if faction == nil {
		return fmt.Errorf("faction %d not found", factionID)
	}

	// Verify squad exists by checking for SquadComponent
	squad := fm.manager.FindEntityByID(squadID)
	if squad == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Add FactionMembershipComponent directly to squad entity (NEW: ECS-idiomatic approach)
	// Replaces creating a separate MapPosition entity
	squad.AddComponent(FactionMembershipComponent, &CombatFactionData{
		FactionID: factionID,
	})

	// Add or update PositionComponent on squad entity
	if !fm.manager.HasComponent(squadID, common.PositionComponent) {
		// Squad has no position yet - atomically add component and register
		fm.manager.RegisterEntityPosition(squad, position)
	} else {
		// Squad already has position - move it atomically
		oldPos := common.GetComponentTypeByID[*coords.LogicalPosition](fm.manager, squadID, common.PositionComponent)
		if oldPos != nil {
			// Use MoveEntity to synchronize position component and position system
			err := fm.manager.MoveEntity(squadID, squad, *oldPos, position)
			if err != nil {
				return fmt.Errorf("failed to update squad position: %w", err)
			}
		}
	}

	return nil
}

// CreateStandardFactions creates a player faction and an enemy faction for an encounter.
// Returns (playerFactionID, enemyFactionID).
func (fm *CombatFactionManager) CreateStandardFactions(playerFactionName, enemyFactionName string, encounterID ecs.EntityID) (ecs.EntityID, ecs.EntityID) {
	playerFactionID := fm.CreateFactionWithPlayer(playerFactionName, 1, "Player 1", encounterID)
	enemyFactionID := fm.CreateFactionWithPlayer(enemyFactionName, 0, "", encounterID)
	return playerFactionID, enemyFactionID
}

func (fm *CombatFactionManager) GetFactionMana(factionID ecs.EntityID) (current, max int) {
	// Find faction entity (using cached query for performance)
	faction := fm.combatCache.FindFactionByID(factionID)
	if faction == nil {
		return 0, 0 // Faction not found
	}

	// Get faction data
	factionData := common.GetComponentType[*FactionData](faction, CombatFactionComponent)
	return factionData.Mana, factionData.MaxMana
}

func (fm *CombatFactionManager) GetFactionName(factionID ecs.EntityID) string {
	// Find faction entity (using cached query for performance)
	faction := fm.combatCache.FindFactionByID(factionID)
	if faction == nil {
		return "Unknown"
	}

	// Get faction data
	factionData := common.GetComponentType[*FactionData](faction, CombatFactionComponent)
	if factionData != nil {
		return factionData.Name
	}
	return "Unknown"
}
