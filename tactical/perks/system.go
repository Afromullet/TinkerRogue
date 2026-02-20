package perks

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/effects"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// ApplyStatPerks creates permanent ActiveEffects from a perk's stat modifiers.
// Uses RemainingTurns=-1 (permanent) so they persist until explicitly removed.
func ApplyStatPerks(entityID ecs.EntityID, perkIDs []string, manager *common.EntityManager) {
	for _, perkID := range perkIDs {
		def := GetPerkDefinition(perkID)
		if def == nil || !def.HasStatModifiers() {
			continue
		}

		entity := manager.FindEntityByID(entityID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}

		for _, mod := range def.StatModifiers {
			modifier := mod.Modifier
			if mod.Percent != 0 {
				baseStat := getBaseStat(attr, mod.Stat)
				modifier = int(float64(baseStat) * mod.Percent)
				if modifier == 0 && mod.Percent > 0 {
					modifier = 1 // Minimum +1 for percentage buffs
				}
			}

			effect := effects.ActiveEffect{
				Name:           def.Name,
				Source:         effects.SourcePerk,
				Stat:           effects.ParseStatType(mod.Stat),
				Modifier:       modifier,
				RemainingTurns: -1, // Permanent
			}
			effects.ApplyEffect(entityID, effect, manager)
		}
	}
}

// RemoveStatPerks removes all perk-sourced effects from an entity.
func RemoveStatPerks(entityID ecs.EntityID, manager *common.EntityManager) {
	effects.RemoveEffectsBySource(entityID, effects.SourcePerk, manager)
}

// EquipPerk equips a perk to the given entity at the specified slot index.
// Validates role gate and exclusivity before equipping.
// Applies stat effects immediately on equip.
func EquipPerk(entityID ecs.EntityID, perkID string, slotIndex int, manager *common.EntityManager) error {
	def := GetPerkDefinition(perkID)
	if def == nil {
		return fmt.Errorf("unknown perk: %s", perkID)
	}

	reason := CanEquipPerk(entityID, perkID, slotIndex, manager)
	if reason != "" {
		return fmt.Errorf("%s", reason)
	}

	// Set the perk ID in the appropriate slot
	switch def.Level {
	case PerkLevelSquad:
		data := common.GetComponentTypeByID[*SquadPerkData](manager, entityID, SquadPerkComponent)
		if data == nil {
			return fmt.Errorf("entity %d has no squad perk component", entityID)
		}
		if slotIndex < 0 || slotIndex >= len(data.EquippedPerks) {
			return fmt.Errorf("invalid slot index %d for squad perk", slotIndex)
		}
		data.EquippedPerks[slotIndex] = perkID

		// Apply stat perks to all units in the squad
		if def.HasStatModifiers() {
			unitIDs := squads.GetUnitIDsInSquad(entityID, manager)
			for _, uid := range unitIDs {
				ApplyStatPerks(uid, []string{perkID}, manager)
			}
		}

	case PerkLevelUnit:
		data := common.GetComponentTypeByID[*UnitPerkData](manager, entityID, UnitPerkComponent)
		if data == nil {
			return fmt.Errorf("entity %d has no unit perk component", entityID)
		}
		if slotIndex < 0 || slotIndex >= len(data.EquippedPerks) {
			return fmt.Errorf("invalid slot index %d for unit perk", slotIndex)
		}
		data.EquippedPerks[slotIndex] = perkID

		if def.HasStatModifiers() {
			ApplyStatPerks(entityID, []string{perkID}, manager)
		}

	case PerkLevelCommander:
		data := common.GetComponentTypeByID[*CommanderPerkData](manager, entityID, CommanderPerkComponent)
		if data == nil {
			return fmt.Errorf("entity %d has no commander perk component", entityID)
		}
		if slotIndex < 0 || slotIndex >= len(data.EquippedPerks) {
			return fmt.Errorf("invalid slot index %d for commander perk", slotIndex)
		}
		data.EquippedPerks[slotIndex] = perkID
	}

	fmt.Printf("[PERK] Equipped '%s' on entity %d (slot %d)\n", perkID, entityID, slotIndex)
	return nil
}

// UnequipPerk removes a perk from the given slot.
// Removes all perk-sourced effects, then re-applies remaining perks.
func UnequipPerk(entityID ecs.EntityID, level PerkLevel, slotIndex int, manager *common.EntityManager) error {
	switch level {
	case PerkLevelSquad:
		data := common.GetComponentTypeByID[*SquadPerkData](manager, entityID, SquadPerkComponent)
		if data == nil {
			return fmt.Errorf("entity %d has no squad perk component", entityID)
		}
		if slotIndex < 0 || slotIndex >= len(data.EquippedPerks) {
			return fmt.Errorf("invalid slot index %d", slotIndex)
		}
		removedID := data.EquippedPerks[slotIndex]
		data.EquippedPerks[slotIndex] = ""

		// Remove and re-apply perk effects on all units
		unitIDs := squads.GetUnitIDsInSquad(entityID, manager)
		for _, uid := range unitIDs {
			RemoveStatPerks(uid, manager)
		}
		// Re-apply remaining squad perk stat effects
		remaining := collectNonEmpty(data.EquippedPerks[:])
		for _, uid := range unitIDs {
			ApplyStatPerks(uid, remaining, manager)
			// Also re-apply unit perks
			unitData := common.GetComponentTypeByID[*UnitPerkData](manager, uid, UnitPerkComponent)
			if unitData != nil {
				ApplyStatPerks(uid, collectNonEmpty(unitData.EquippedPerks[:]), manager)
			}
		}

		fmt.Printf("[PERK] Unequipped '%s' from entity %d (slot %d)\n", removedID, entityID, slotIndex)

	case PerkLevelUnit:
		data := common.GetComponentTypeByID[*UnitPerkData](manager, entityID, UnitPerkComponent)
		if data == nil {
			return fmt.Errorf("entity %d has no unit perk component", entityID)
		}
		if slotIndex < 0 || slotIndex >= len(data.EquippedPerks) {
			return fmt.Errorf("invalid slot index %d", slotIndex)
		}
		removedID := data.EquippedPerks[slotIndex]
		data.EquippedPerks[slotIndex] = ""

		// Remove all perk effects and re-apply remaining
		RemoveStatPerks(entityID, manager)
		remaining := collectNonEmpty(data.EquippedPerks[:])
		ApplyStatPerks(entityID, remaining, manager)

		// Re-apply squad perks too
		memberData := common.GetComponentTypeByID[*squads.SquadMemberData](manager, entityID, squads.SquadMemberComponent)
		if memberData != nil {
			squadPerkData := common.GetComponentTypeByID[*SquadPerkData](manager, memberData.SquadID, SquadPerkComponent)
			if squadPerkData != nil {
				ApplyStatPerks(entityID, collectNonEmpty(squadPerkData.EquippedPerks[:]), manager)
			}
		}

		fmt.Printf("[PERK] Unequipped '%s' from entity %d (slot %d)\n", removedID, entityID, slotIndex)

	case PerkLevelCommander:
		data := common.GetComponentTypeByID[*CommanderPerkData](manager, entityID, CommanderPerkComponent)
		if data == nil {
			return fmt.Errorf("entity %d has no commander perk component", entityID)
		}
		if slotIndex < 0 || slotIndex >= len(data.EquippedPerks) {
			return fmt.Errorf("invalid slot index %d", slotIndex)
		}
		removedID := data.EquippedPerks[slotIndex]
		data.EquippedPerks[slotIndex] = ""
		fmt.Printf("[PERK] Unequipped '%s' from entity %d (slot %d)\n", removedID, entityID, slotIndex)
	}

	return nil
}

// CanEquipPerk checks if a perk can be equipped. Returns "" if valid, or a reason string.
func CanEquipPerk(entityID ecs.EntityID, perkID string, slotIndex int, manager *common.EntityManager) string {
	def := GetPerkDefinition(perkID)
	if def == nil {
		return "unknown perk"
	}

	// Check role gate
	if def.RoleGate != "" {
		roleData := common.GetComponentTypeByID[*squads.UnitRoleData](manager, entityID, squads.UnitRoleComponent)
		if roleData != nil && roleData.Role.String() != def.RoleGate {
			return fmt.Sprintf("requires role %s", def.RoleGate)
		}
	}

	// Check exclusivity against currently equipped perks
	equipped := GetEquippedPerks(entityID, manager)
	for _, equippedID := range equipped {
		for _, exID := range def.ExclusiveWith {
			if equippedID == exID {
				return fmt.Sprintf("exclusive with %s", exID)
			}
		}
	}

	return ""
}

// getBaseStat retrieves the base value of a stat from Attributes for percentage calculations.
func getBaseStat(attr *common.Attributes, statName string) int {
	switch statName {
	case "strength":
		return attr.Strength
	case "dexterity":
		return attr.Dexterity
	case "magic":
		return attr.Magic
	case "leadership":
		return attr.Leadership
	case "armor":
		return attr.Armor
	case "weapon":
		return attr.Weapon
	case "movementspeed":
		return attr.MovementSpeed
	case "attackrange":
		return attr.AttackRange
	default:
		return 0
	}
}

// collectNonEmpty returns non-empty strings from a slice.
func collectNonEmpty(ids []string) []string {
	var result []string
	for _, id := range ids {
		if id != "" {
			result = append(result, id)
		}
	}
	return result
}
