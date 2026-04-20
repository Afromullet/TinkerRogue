package raid

import (
	"fmt"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/tactical/combat/combatstate"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// Position offsets for squad placement in raid encounters.
const (
	playerOffsetX = -3 // Player squads start left of combat position
	playerOffsetY = -2 // Player squads start above combat position
	enemyOffsetX  = 3  // Enemy squads start right of combat position
	enemyOffsetY  = 2  // Enemy squads start below combat position
	squadSpreadX  = 2  // Horizontal spacing between squads
)

// SetupRaidFactions sets up a combat encounter for a garrison room.
// Creates factions, positions squads, and initializes combat state.
// This works with pre-composed garrison squads instead of power-budget generation.
//
// Parameters:
//   - encounterID: a pre-created encounter entity ID
//   - garrisonSquadIDs: enemy garrison squads defending the room
//   - playerDeployedIDs: player squads selected for this encounter
//   - combatPos: world position where combat takes place
func SetupRaidFactions(
	manager *common.EntityManager,
	encounterID ecs.EntityID,
	garrisonSquadIDs []ecs.EntityID,
	playerDeployedIDs []ecs.EntityID,
	combatPos coords.LogicalPosition,
) (playerFactionID, enemyFactionID ecs.EntityID, err error) {
	if len(playerDeployedIDs) == 0 {
		return 0, 0, fmt.Errorf("no player squads deployed")
	}
	if len(garrisonSquadIDs) == 0 {
		return 0, 0, fmt.Errorf("no garrison squads in room")
	}

	// Create factions (player attacks, garrison defends)
	var fm *combatstate.CombatFactionManager
	fm, playerFactionID, enemyFactionID = combatlifecycle.CreateFactionPair(manager, "Raid Attackers", "Garrison Defenders", encounterID)

	// Pre-compute player squad positions
	playerPositions := make([]coords.LogicalPosition, len(playerDeployedIDs))
	for i := range playerDeployedIDs {
		playerPositions[i] = coords.LogicalPosition{
			X: combatPos.X + playerOffsetX + (i * squadSpreadX),
			Y: combatPos.Y + playerOffsetY,
		}
	}
	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, playerFactionID, playerDeployedIDs, playerPositions, true); err != nil {
		return 0, 0, fmt.Errorf("failed to add player squads: %w", err)
	}

	// Pre-compute garrison squad positions
	garrisonPositions := make([]coords.LogicalPosition, len(garrisonSquadIDs))
	for i := range garrisonSquadIDs {
		garrisonPositions[i] = coords.LogicalPosition{
			X: combatPos.X + enemyOffsetX + (i * squadSpreadX),
			Y: combatPos.Y + enemyOffsetY,
		}
	}
	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, enemyFactionID, garrisonSquadIDs, garrisonPositions, false); err != nil {
		return 0, 0, fmt.Errorf("failed to add garrison squads: %w", err)
	}

	return playerFactionID, enemyFactionID, nil
}
