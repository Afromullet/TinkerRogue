package bootstrap

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
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
	current, _ := inv.GetArtifactCount()
	if current+needed > inv.MaxArtifacts {
		inv.MaxArtifacts = current + needed
	}

	for id := range templates.ArtifactRegistry {
		for i := 0; i < count; i++ {
			if err := inv.AddArtifact(id); err != nil {
				return fmt.Errorf("failed to seed artifact %q copy %d: %w", id, i+1, err)
			}
		}
	}
	return nil
}
