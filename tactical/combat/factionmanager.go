package combat

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

type FactionManager struct {
	manager     *common.EntityManager
	combatCache *CombatQueryCache
}

func NewFactionManager(manager *common.EntityManager) *FactionManager {
	return &FactionManager{
		manager:     manager,
		combatCache: NewCombatQueryCache(manager),
	}
}

func (fm *FactionManager) CreateFaction(name string, isPlayer bool) ecs.EntityID {
	playerID := 0
	playerName := ""
	if isPlayer {
		playerID = 1 // Default to Player 1 for single-player
		playerName = "Player 1"
	}
	return fm.CreateFactionWithPlayer(name, playerID, playerName)
}

// CreateFactionWithPlayer creates a faction with specific player assignment
func (fm *FactionManager) CreateFactionWithPlayer(name string, playerID int, playerName string) ecs.EntityID {
	faction := fm.manager.World.NewEntity()
	factionID := faction.GetID()

	isPlayerControlled := playerID > 0 // Derive from PlayerID

	faction.AddComponent(FactionComponent, &FactionData{
		FactionID:          factionID,
		Name:               name,
		Mana:               100,
		MaxMana:            100,
		IsPlayerControlled: isPlayerControlled,
		PlayerID:           playerID,
		PlayerName:         playerName,
	})

	return factionID
}

func (fm *FactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID, position coords.LogicalPosition) error {

	faction := fm.combatCache.FindFactionByID(factionID, fm.manager)
	if faction == nil {
		return fmt.Errorf("faction %d not found", factionID)
	}

	// Verify squad exists by checking for SquadComponent
	squad := common.FindEntityByIDWithTag(fm.manager, squadID, squads.SquadTag)
	if squad == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Add CombatFactionComponent directly to squad entity (NEW: ECS-idiomatic approach)
	// Replaces creating a separate MapPosition entity
	squad.AddComponent(CombatFactionComponent, &CombatFactionData{
		FactionID: factionID,
	})

	// Add or update PositionComponent on squad entity
	if !fm.manager.HasComponentByIDWithTag(squadID, squads.SquadTag, common.PositionComponent) {
		// Squad has no position yet - add it
		posPtr := new(coords.LogicalPosition)
		*posPtr = position
		squad.AddComponent(common.PositionComponent, posPtr)
		// Register in PositionSystem (canonical position source)
		common.GlobalPositionSystem.AddEntity(squadID, position)
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

func (fm *FactionManager) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID {
	var squadIDs []ecs.EntityID

	// Query all squads and filter by CombatFactionComponent
	for _, result := range fm.manager.World.Query(squads.SquadTag) {
		combatFaction := common.GetComponentType[*CombatFactionData](result.Entity, CombatFactionComponent)
		if combatFaction != nil && combatFaction.FactionID == factionID {
			squadIDs = append(squadIDs, result.Entity.GetID())
		}
	}

	return squadIDs
}

func (fm *FactionManager) RemoveSquadFromFaction(factionID, squadID ecs.EntityID) error {
	// Find squad entity
	squad := common.FindEntityByIDWithTag(fm.manager, squadID, squads.SquadTag)
	if squad == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Verify squad has CombatFactionComponent
	combatFaction := common.GetComponentType[*CombatFactionData](squad, CombatFactionComponent)
	if combatFaction == nil {
		return fmt.Errorf("squad %d is not in combat", squadID)
	}

	// Verify squad belongs to this faction
	if combatFaction.FactionID != factionID {
		return fmt.Errorf("squad %d does not belong to faction %d", squadID, factionID)
	}

	// Get position before removal for PositionSystem cleanup
	position := common.GetComponentType[*coords.LogicalPosition](squad, common.PositionComponent)
	if position != nil {
		// Remove from PositionSystem spatial grid
		common.GlobalPositionSystem.RemoveEntity(squadID, *position)
	}

	// Remove CombatFactionComponent from squad (squad exits combat)
	squad.RemoveComponent(CombatFactionComponent)

	return nil
}

func (fm *FactionManager) GetFactionMana(factionID ecs.EntityID) (current, max int) {
	// Find faction entity (using cached query for performance)
	faction := fm.combatCache.FindFactionByID(factionID, fm.manager)
	if faction == nil {
		return 0, 0 // Faction not found
	}

	// Get faction data
	factionData := common.GetComponentType[*FactionData](faction, FactionComponent)
	return factionData.Mana, factionData.MaxMana
}

func (fm *FactionManager) GetFactionName(factionID ecs.EntityID) string {
	// Find faction entity (using cached query for performance)
	faction := fm.combatCache.FindFactionByID(factionID, fm.manager)
	if faction == nil {
		return "Unknown"
	}

	// Get faction data
	factionData := common.GetComponentType[*FactionData](faction, FactionComponent)
	if factionData != nil {
		return factionData.Name
	}
	return "Unknown"
}

// GetPlayerFactions returns all factions controlled by human players
func (fm *FactionManager) GetPlayerFactions() []ecs.EntityID {
	var playerFactions []ecs.EntityID
	for _, result := range fm.manager.World.Query(FactionTag) {
		factionData := common.GetComponentType[*FactionData](result.Entity, FactionComponent)
		if factionData != nil && factionData.PlayerID > 0 {
			playerFactions = append(playerFactions, result.Entity.GetID())
		}
	}
	return playerFactions
}
