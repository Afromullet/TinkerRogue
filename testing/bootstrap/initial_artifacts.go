package bootstrap

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
	"game_main/tactical/commander"
	"game_main/tactical/perks"
	"game_main/tactical/squads"
	"game_main/templates"
	"sort"

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
		squadRoster := squads.GetPlayerSquadRoster(commanderID, manager)
		if squadRoster == nil || len(squadRoster.OwnedSquads) == 0 {
			fmt.Printf("[EquipArtifacts] Commander %d has no squads, skipping\n", commanderID)
			continue
		}

		// Round-robin artifacts across this commander's squads
		squadList := squadRoster.OwnedSquads
		for i, id := range majorIDs {
			squadID := squadList[i%len(squadList)]
			if err := gear.EquipArtifact(playerID, squadID, id, manager); err != nil {
				fmt.Printf("[EquipArtifacts] Failed to equip %s on squad %d (commander %d): %v\n", id, squadID, commanderID, err)
			} else {
				fmt.Printf("[EquipArtifacts] Equipped %s on squad %d (commander %d)\n", id, squadID, commanderID)
			}
		}
	}
}

// EquipAllPerks equips perks from the registry onto all player entities for testing.
// Commander perks round-robin into 3 commander slots, squad perks into 3 squad slots,
// and unit perks into 2 unit slots.
func EquipAllPerks(playerID ecs.EntityID, manager *common.EntityManager) {
	// Partition perks by level
	var commanderIDs, squadIDs, unitIDs []string
	for _, id := range perks.GetAllPerkIDs() {
		def := perks.GetPerkDefinition(id)
		if def == nil {
			continue
		}
		switch def.Level {
		case perks.PerkLevelCommander:
			commanderIDs = append(commanderIDs, id)
		case perks.PerkLevelSquad:
			squadIDs = append(squadIDs, id)
		case perks.PerkLevelUnit:
			unitIDs = append(unitIDs, id)
		}
	}
	sort.Strings(commanderIDs)
	sort.Strings(squadIDs)
	sort.Strings(unitIDs)

	rosterData := commander.GetPlayerCommanderRoster(playerID, manager)
	if rosterData == nil || len(rosterData.CommanderIDs) == 0 {
		fmt.Println("[EquipPerks] No commanders found, skipping")
		return
	}

	for _, cmdID := range rosterData.CommanderIDs {
		// Equip commander-level perks round-robin into 3 slots
		for i, perkID := range commanderIDs {
			slot := i % 3
			if err := perks.EquipPerk(cmdID, perkID, slot, manager); err != nil {
				fmt.Printf("[EquipPerks] Commander %d slot %d perk %s: %v\n", cmdID, slot, perkID, err)
			}
		}

		squadRoster := squads.GetPlayerSquadRoster(cmdID, manager)
		if squadRoster == nil || len(squadRoster.OwnedSquads) == 0 {
			continue
		}

		for _, squadID := range squadRoster.OwnedSquads {
			// Ensure squad has perk component
			squadEntity := manager.FindEntityByID(squadID)
			if squadEntity == nil {
				continue
			}
			if !manager.HasComponent(squadID, perks.SquadPerkComponent) {
				perks.AttachSquadPerkComponent(squadEntity)
			}

			// Equip squad-level perks round-robin into 3 slots
			for i, perkID := range squadIDs {
				slot := i % 3
				if err := perks.EquipPerk(squadID, perkID, slot, manager); err != nil {
					fmt.Printf("[EquipPerks] Squad %d slot %d perk %s: %v\n", squadID, slot, perkID, err)
				}
			}

			// Equip unit-level perks on each unit in the squad
			for _, uid := range squads.GetUnitIDsInSquad(squadID, manager) {
				unitEntity := manager.FindEntityByID(uid)
				if unitEntity == nil {
					continue
				}
				if !manager.HasComponent(uid, perks.UnitPerkComponent) {
					perks.AttachUnitPerkComponent(unitEntity)
				}

				for i, perkID := range unitIDs {
					slot := i % 2
					if err := perks.EquipPerk(uid, perkID, slot, manager); err != nil {
						fmt.Printf("[EquipPerks] Unit %d slot %d perk %s: %v\n", uid, slot, perkID, err)
					}
				}
			}
		}
	}

	fmt.Printf("[EquipPerks] Done: %d commander, %d squad, %d unit perks distributed\n",
		len(commanderIDs), len(squadIDs), len(unitIDs))
}
