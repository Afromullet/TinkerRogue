package bootstrap

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/commander"
	"game_main/tactical/squads/roster"
	"game_main/templates"
	"sort"

	"github.com/bytearena/ecs"
)

// SeedAllArtifacts adds `count` copies of every artifact in the registry to the player's inventory.
// It bumps MaxArtifacts if needed to fit all copies.
func SeedAllArtifacts(playerID ecs.EntityID, count int, manager *common.EntityManager) error {
	inv := artifacts.GetPlayerArtifactInventory(playerID, manager)
	if inv == nil {
		return fmt.Errorf("player %d has no artifact inventory", playerID)
	}

	needed := len(templates.ArtifactRegistry) * count
	current, _ := artifacts.GetArtifactCount(inv)
	if current+needed > inv.MaxArtifacts {
		inv.MaxArtifacts = current + needed
	}

	for id := range templates.ArtifactRegistry {
		for i := 0; i < count; i++ {
			if err := artifacts.AddArtifactToInventory(inv, id); err != nil {
				return fmt.Errorf("failed to seed artifact %q copy %d: %w", id, i+1, err)
			}
		}
	}
	return nil
}

// EquipPlayerActivatedArtifacts equips all major artifacts from the registry,
// round-robin distributing them across every commander's squads.
// Must be called after SeedAllArtifacts so enough artifact copies exist in inventory.
func EquipPlayerActivatedArtifacts(playerID ecs.EntityID, manager *common.EntityManager) {
	// Collect all major artifact IDs from the registry
	var majorIDs []string
	for id, def := range templates.ArtifactRegistry {
		if def.Tier == "major" {
			majorIDs = append(majorIDs, id)
		}
	}
	sort.Strings(majorIDs) // deterministic ordering

	if len(majorIDs) == 0 {
		fmt.Println("[EquipArtifacts] No major artifacts in registry, skipping")
		return
	}

	// Get commander roster and equip each commander's squads
	rosterData := commander.GetPlayerCommanderRoster(playerID, manager)
	if rosterData == nil || len(rosterData.CommanderIDs) == 0 {
		fmt.Println("[EquipArtifacts] No commanders found, skipping artifact equip")
		return
	}

	for _, commanderID := range rosterData.CommanderIDs {
		squadRoster := roster.GetPlayerSquadRoster(commanderID, manager)
		if squadRoster == nil || len(squadRoster.OwnedSquads) == 0 {
			fmt.Printf("[EquipArtifacts] Commander %d has no squads, skipping\n", commanderID)
			continue
		}

		// Round-robin artifacts across this commander's squads
		squadList := squadRoster.OwnedSquads
		for i, id := range majorIDs {
			squadID := squadList[i%len(squadList)]
			if err := artifacts.EquipArtifact(playerID, squadID, id, manager); err != nil {
				fmt.Printf("[EquipArtifacts] Failed to equip %s on squad %d (commander %d): %v\n", id, squadID, commanderID, err)
			} else {
				fmt.Printf("[EquipArtifacts] Equipped %s on squad %d (commander %d)\n", id, squadID, commanderID)
			}
		}
	}
}
