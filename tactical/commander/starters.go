package commander

import (
	"game_main/core/common"
	"game_main/tactical/powers/progression"

	"github.com/bytearena/ecs"
)

// StartingUnlockedPerks returns the perk IDs every new Commander starts with.
// Covers Tank + DPS + Support roles so any starter squad has something to equip.
func StartingUnlockedPerks() []string {
	return []string{
		"brace_for_impact",
		"reckless_assault",
		"shieldwall_discipline",
		"field_medic",
	}
}

// StartingUnlockedSpells returns the spell IDs every new Commander starts with.
// Low-cost damage spells so a starter mage leader is functional turn one.
func StartingUnlockedSpells() []string {
	return []string{
		"spark",
		"singe",
		"frost_snap",
	}
}

// SeedStarters appends the starter perk and spell lists to the commander's
// already-attached ProgressionData. Call immediately after CreateCommander
// when the caller wants the default starter library; skip it to leave the
// commander's library empty. No-op if the commander has no ProgressionComponent.
func SeedStarters(commanderID ecs.EntityID, manager *common.EntityManager) {
	data := progression.GetProgression(commanderID, manager)
	if data == nil {
		return
	}
	data.UnlockedPerkIDs = append(data.UnlockedPerkIDs, StartingUnlockedPerks()...)
	data.UnlockedSpellIDs = append(data.UnlockedSpellIDs, StartingUnlockedSpells()...)
}
