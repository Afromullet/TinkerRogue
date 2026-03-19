package raid

import (
	"fmt"

	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/tactical/combat"
	"game_main/world/coords"

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

	// Create combat query cache and faction manager
	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)

	// Create factions (player attacks, garrison defends)
	playerFactionID, enemyFactionID = fm.CreateStandardFactions("Raid Attackers", "Garrison Defenders", encounterID)

	// Position and add player squads
	for i, squadID := range playerDeployedIDs {
		pos := coords.LogicalPosition{
			X: combatPos.X + playerOffsetX + (i * squadSpreadX),
			Y: combatPos.Y + playerOffsetY,
		}

		if err := combatlifecycle.EnrollSquadInFaction(fm, manager, playerFactionID, squadID, pos, true); err != nil {
			return 0, 0, fmt.Errorf("failed to add player squad %d: %w", squadID, err)
		}
	}

	// Position and add garrison squads (defenders)
	for i, squadID := range garrisonSquadIDs {
		pos := coords.LogicalPosition{
			X: combatPos.X + enemyOffsetX + (i * squadSpreadX),
			Y: combatPos.Y + enemyOffsetY,
		}

		if err := combatlifecycle.EnrollSquadInFaction(fm, manager, enemyFactionID, squadID, pos, false); err != nil {
			return 0, 0, fmt.Errorf("failed to add garrison squad %d: %w", squadID, err)
		}
	}

	return playerFactionID, enemyFactionID, nil
}
