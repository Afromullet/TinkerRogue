package bootstrap

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// SeedAllArtifacts adds `count` copies of every artifact in the registry to the player's inventory.
// It bumps MaxArtifacts if needed to fit all copies.
func SeedAllArtifacts(playerID ecs.EntityID, count int, manager *common.EntityManager) error {
	inv := gear.GetPlayerArtifactInventory(playerID, manager)
	if inv == nil {
		return fmt.Errorf("player %d has no artifact inventory", playerID)
	}

	needed := len(templates.ArtifactRegistry) * count
	current, _ := gear.GetArtifactCount(inv)
	if current+needed > inv.MaxArtifacts {
		inv.MaxArtifacts = current + needed
	}

	for id := range templates.ArtifactRegistry {
		for i := 0; i < count; i++ {
			if err := gear.AddArtifactToInventory(inv, id); err != nil {
				return fmt.Errorf("failed to seed artifact %q copy %d: %w", id, i+1, err)
			}
		}
	}
	return nil
}

// EquipPlayerActivatedArtifacts equips the 6 player-activated artifacts onto the first 2 squads.
// Must be called after SeedAllArtifacts so the artifacts exist in inventory.
func EquipPlayerActivatedArtifacts(playerID ecs.EntityID, manager *common.EntityManager) {
	// Player-activated artifacts to equip (3 per squad)
	batch1 := []string{"double_time_drums", "stand_down_orders", "chain_of_command_scepter"}
	batch2 := []string{"saboteurs_hourglass", "anthem_of_perseverance", "deadlock_shackles"}

	// Get commander roster → first commander → squad roster
	rosterData := commander.GetPlayerCommanderRoster(playerID, manager)
	if rosterData == nil || len(rosterData.CommanderIDs) == 0 {
		fmt.Println("[EquipArtifacts] No commanders found, skipping artifact equip")
		return
	}

	commanderID := rosterData.CommanderIDs[0]
	squadRoster := squads.GetPlayerSquadRoster(commanderID, manager)
	if squadRoster == nil || len(squadRoster.OwnedSquads) < 2 {
		fmt.Printf("[EquipArtifacts] Need at least 2 squads, have %d, skipping\n", len(squadRoster.OwnedSquads))
		return
	}

	squad1 := squadRoster.OwnedSquads[0]
	squad2 := squadRoster.OwnedSquads[1]

	for _, id := range batch1 {
		if err := gear.EquipArtifact(playerID, squad1, id, manager); err != nil {
			fmt.Printf("[EquipArtifacts] Failed to equip %s on squad %d: %v\n", id, squad1, err)
		} else {
			fmt.Printf("[EquipArtifacts] Equipped %s on squad %d\n", id, squad1)
		}
	}

	for _, id := range batch2 {
		if err := gear.EquipArtifact(playerID, squad2, id, manager); err != nil {
			fmt.Printf("[EquipArtifacts] Failed to equip %s on squad %d: %v\n", id, squad2, err)
		} else {
			fmt.Printf("[EquipArtifacts] Equipped %s on squad %d\n", id, squad2)
		}
	}
}
