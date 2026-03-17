package combat

import (
	"fmt"
	"image/color"

	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// factionColorPalette provides distinct colors for factions.
// Index 0 = player (blue), then enemy factions get red, green, yellow, purple, etc.
var factionColorPalette = []color.RGBA{
	{R: 100, G: 149, B: 237, A: 255}, // Cornflower blue (player)
	{R: 220, G: 50, B: 50, A: 255},   // Red (enemy 1)
	{R: 50, G: 180, B: 50, A: 255},   // Green (enemy 2)
	{R: 230, G: 200, B: 50, A: 255},  // Yellow (enemy 3)
	{R: 160, G: 50, B: 200, A: 255},  // Purple (enemy 4)
	{R: 255, G: 140, B: 0, A: 255},   // Orange (enemy 5)
	{R: 0, G: 200, B: 200, A: 255},   // Cyan (enemy 6)
}

type CombatFactionManager struct {
	manager       *common.EntityManager
	combatCache   *CombatQueryCache
	factionCount  int // Tracks how many factions created (for color assignment)
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

	// Assign color from palette based on creation order
	colorIdx := fm.factionCount % len(factionColorPalette)
	fm.factionCount++

	faction.AddComponent(CombatFactionComponent, &FactionData{
		FactionID:          factionID,
		Name:               name,
		Mana:               100,
		MaxMana:            100,
		IsPlayerControlled: isPlayerControlled,
		PlayerID:           playerID,
		PlayerName:         playerName,
		EncounterID:        encounterID,
		Color:              factionColorPalette[colorIdx],
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

