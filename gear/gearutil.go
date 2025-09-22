package gear

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// TypeOfItem is a helper type for equipping and unequpping items
type TypeOfItem int

const (
	ArmorType = iota
	MeleeWeaponType
	RangedWeaponType
	InvalidItemType
)


// ItemStats returns a display string for an item's statistics.
// It checks the entity for armor, melee weapon, or ranged weapon components and returns their display info.
func ItemStats(e *ecs.Entity) string {

	if item := common.GetComponentType[*Armor](e, ArmorComponent); item != nil {
		return item.DisplayString()
	} else if item := common.GetComponentType[*MeleeWeapon](e, MeleeWeaponComponent); item != nil {
		return item.DisplayString()
	} else if item := common.GetComponentType[*RangedWeapon](e, RangedWeaponComponent); item != nil {
		return item.DisplayString()
	} else {
		return ""
	}

}

// GetArmor retrieves the Armor component from an entity.
// Returns nil if the entity doesn't have an armor component.
func GetArmor(e *ecs.Entity) *Armor {
	return common.GetComponentType[*Armor](e, ArmorComponent)
}

// GetItem retrieves the Item component from an entity.
// Returns nil if the entity doesn't have an item component.
func GetItem(e *ecs.Entity) *Item {
	return common.GetComponentType[*Item](e, ItemComponent)
}

// KindOfItem determines the type of item (armor, melee weapon, ranged weapon) for an entity.
// Returns InvalidItemType if the entity doesn't match any known item types.
// TODO: Only used in PlayerData, consider moving there.
func KindOfItem(e *ecs.Entity) TypeOfItem {

	if item := common.GetComponentType[*Armor](e, ArmorComponent); item != nil {
		return ArmorType
	} else if item := common.GetComponentType[*MeleeWeapon](e, MeleeWeaponComponent); item != nil {
		return MeleeWeaponType
	} else if item := common.GetComponentType[*RangedWeapon](e, RangedWeaponComponent); item != nil {
		return RangedWeaponType
	} else {
		return InvalidItemType
	}

}
