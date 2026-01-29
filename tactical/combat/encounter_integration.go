package combat

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"math"

	"github.com/bytearena/ecs"
)

// EncounterCombatSetup holds the result of setting up combat from an encounter.
type EncounterCombatSetup struct {
	PlayerFactionID  ecs.EntityID
	EnemyFactionID   ecs.EntityID
	FactionManager   *CombatFactionManager
	Cache            *CombatQueryCache
	EnemySquadIDs    []ecs.EntityID
}

// SetupCombatFromEncounter sets up combat infrastructure from encounter data.
// This is the combat-side of encounter setup - it creates factions and action states.
// The encounter package generates the squads; this function adds them to combat.
//
// Parameters:
//   - manager: Entity manager
//   - playerEntityID: Player entity owning squads
//   - playerStartPos: Starting position for player squads
//   - enemySquads: Pre-generated enemy squads with positions
//
// Returns EncounterCombatSetup with faction IDs and manager references.
func SetupCombatFromEncounter(
	manager *common.EntityManager,
	playerEntityID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	enemySquads []EnemySquadInput,
) (*EncounterCombatSetup, error) {
	// Create combat infrastructure
	cache := NewCombatQueryCache(manager)
	fm := NewCombatFactionManager(manager, cache)

	// Create factions
	playerFactionID := fm.CreateFactionWithPlayer("Player Forces", 1, "Player 1")
	enemyFactionID := fm.CreateFactionWithPlayer("Enemy Forces", 0, "")

	// Add player's deployed squads to faction
	if err := addPlayerSquadsToFaction(fm, playerEntityID, manager, playerFactionID, playerStartPos); err != nil {
		return nil, fmt.Errorf("failed to assign player squads: %w", err)
	}

	// Track enemy squad IDs for cleanup
	createdEnemySquadIDs := make([]ecs.EntityID, 0, len(enemySquads))

	// Add enemy squads to faction
	for i, squadInfo := range enemySquads {
		if err := fm.AddSquadToFaction(enemyFactionID, squadInfo.SquadID, squadInfo.Position); err != nil {
			return nil, fmt.Errorf("failed to add enemy squad %d to faction: %w", i, err)
		}
		if err := CreateActionStateForSquad(manager, squadInfo.SquadID); err != nil {
			return nil, fmt.Errorf("failed to create action state for enemy squad %d: %w", i, err)
		}
		createdEnemySquadIDs = append(createdEnemySquadIDs, squadInfo.SquadID)
	}

	return &EncounterCombatSetup{
		PlayerFactionID:  playerFactionID,
		EnemyFactionID:   enemyFactionID,
		FactionManager:   fm,
		Cache:            cache,
		EnemySquadIDs:    createdEnemySquadIDs,
	}, nil
}

// EnemySquadInput is input for adding enemy squads to combat.
type EnemySquadInput struct {
	SquadID  ecs.EntityID
	Position coords.LogicalPosition
}

// addPlayerSquadsToFaction adds all deployed player squads to the player faction.
// Internal helper for SetupCombatFromEncounter.
func addPlayerSquadsToFaction(
	fm *CombatFactionManager,
	playerID ecs.EntityID,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
) error {
	roster := squads.GetPlayerSquadRoster(playerID, manager)
	if roster == nil {
		return fmt.Errorf("player has no squad roster")
	}

	deployedSquads := roster.GetDeployedSquads(manager)
	if len(deployedSquads) == 0 {
		return fmt.Errorf("player has no deployed squads")
	}

	fmt.Printf("Adding %d player squads to faction\n", len(deployedSquads))

	// Generate positions for player squads
	squadPositions := generatePlayerSquadPositions(playerStartPos, len(deployedSquads))

	// Add each deployed squad to faction
	for i, squadID := range deployedSquads {
		pos := squadPositions[i]

		// Update squad's position component for combat
		squadEntity := manager.FindEntityByID(squadID)
		if squadEntity != nil {
			squadPos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
			if squadPos != nil {
				oldPos := *squadPos
				manager.MoveEntity(squadID, squadEntity, oldPos, pos)
			}
		}

		// Add to faction
		if err := fm.AddSquadToFaction(factionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add squad %d to faction: %w", squadID, err)
		}

		// Create action state
		if err := CreateActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for squad %d: %w", squadID, err)
		}
	}

	return nil
}

// generatePlayerSquadPositions creates positions for player squads around starting point.
// Positions squads in a defensive arc formation.
func generatePlayerSquadPositions(startPos coords.LogicalPosition, count int) []coords.LogicalPosition {
	positions := make([]coords.LogicalPosition, count)

	for i := 0; i < count; i++ {
		angle := (float64(i) / float64(count)) * math.Pi
		angle = angle - math.Pi/2
		distance := 3 + (i % 2)

		offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
		offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

		positions[i] = coords.LogicalPosition{
			X: clampPositionCombat(startPos.X+offsetX, 0, 99),
			Y: clampPositionCombat(startPos.Y+offsetY, 0, 79),
		}
	}

	return positions
}

// clampPositionCombat ensures a position stays within bounds.
func clampPositionCombat(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// CreateActionStateForSquad creates the ActionStateData component for a squad.
// Exported for use by encounter package during legacy setup.
func CreateActionStateForSquad(manager *common.EntityManager, squadID ecs.EntityID) error {
	actionEntity := manager.World.NewEntity()

	movementSpeed := squads.GetSquadMovementSpeed(squadID, manager)
	if movementSpeed == 0 {
		movementSpeed = 3 // Default if no valid units found
	}

	actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
		SquadID:           squadID,
		HasMoved:          false,
		HasActed:          false,
		MovementRemaining: movementSpeed,
	})

	return nil
}
