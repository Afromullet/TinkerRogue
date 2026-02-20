package perks

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// getCommanderPerkIDs returns equipped commander perk IDs for an entity.
func getCommanderPerkIDs(commanderID ecs.EntityID, manager *common.EntityManager) []string {
	var ids []string
	if data := common.GetComponentTypeByID[*CommanderPerkData](manager, commanderID, CommanderPerkComponent); data != nil {
		for _, id := range data.EquippedPerks {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// ModifySpellCost adjusts mana cost based on commander perks.
// Returns modified cost (always >= 1).
func ModifySpellCost(casterID ecs.EntityID, baseCost int, manager *common.EntityManager) int {
	cost := baseCost
	for _, perkID := range getCommanderPerkIDs(casterID, manager) {
		switch perkID {
		case "efficient_casting":
			cost = int(float64(baseCost) * 0.8)
		case "overcharge":
			cost = int(float64(baseCost) * 1.5)
		}
	}
	if cost < 1 {
		cost = 1
	}
	return cost
}

// ModifySpellDamage adjusts spell damage based on commander perks.
func ModifySpellDamage(casterID ecs.EntityID, baseDamage int, manager *common.EntityManager) int {
	damage := baseDamage
	for _, perkID := range getCommanderPerkIDs(casterID, manager) {
		switch perkID {
		case "spell_mastery":
			damage = int(float64(baseDamage) * 1.25)
		case "overcharge":
			damage = int(float64(baseDamage) * 1.75)
		}
	}
	return damage
}

// ModifySpellDuration adjusts buff/debuff duration based on commander perks.
func ModifySpellDuration(casterID ecs.EntityID, baseDuration int, manager *common.EntityManager) int {
	duration := baseDuration
	for _, perkID := range getCommanderPerkIDs(casterID, manager) {
		if perkID == "lingering_magic" {
			duration += 2
		}
	}
	return duration
}

// ModifySpellModifier adjusts buff/debuff stat modifier values.
// Preserves sign (negative debuffs stay negative, rounded toward larger magnitude).
func ModifySpellModifier(casterID ecs.EntityID, baseModifier int, manager *common.EntityManager) int {
	modifier := baseModifier
	for _, perkID := range getCommanderPerkIDs(casterID, manager) {
		if perkID == "potent_enchantment" {
			if modifier >= 0 {
				modifier = int(float64(baseModifier) * 1.5)
			} else {
				// For negative values, multiply absolute value and re-negate
				abs := -baseModifier
				scaled := int(float64(abs) * 1.5)
				if scaled == abs {
					scaled = abs + 1 // Ensure at least +1 magnitude increase
				}
				modifier = -scaled
			}
		}
	}
	return modifier
}
