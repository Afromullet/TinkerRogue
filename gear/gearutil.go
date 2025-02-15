package gear

import (
	cryptorand "crypto/rand"
	"game_main/common"
	"math/big"

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

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
// Here to avoid circular inclusions of randgen
// Todo replace with stdlib random num generaiton
func GetRandomBetween(low int, high int) int {
	var randy int = -1
	for {
		randy = GetDiceRoll(high)
		if randy >= low {
			break
		}
	}
	return randy
}

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
// Here to avoid circular inclusions of randgen
func GetDiceRoll(num int) int {
	x, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(num)))
	return int(x.Int64()) + 1

}

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

func GetArmor(e *ecs.Entity) *Armor {
	return common.GetComponentType[*Armor](e, ArmorComponent)
}

func GetItem(e *ecs.Entity) *Item {
	return common.GetComponentType[*Item](e, ItemComponent)
}

// Gets the underlying type of an item. Todo onl used in PlayeData, so it coud be moved there,

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
