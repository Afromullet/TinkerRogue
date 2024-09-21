package gear

import (
	cryptorand "crypto/rand"
	"game_main/common"
	"math/big"

	"github.com/bytearena/ecs"
)

type TypeOfItem int

const (
	ArmorType = iota
	MeleeWeaponType
	RangedWeaponType
	InvalidItemType
)

// Anything that implements this returns a string that dispalys item information to the user
type ItemStatDisplayer interface {
	Stats() string
}

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
// Here to avoid circular inclusions of randgen
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
		return item.Stats()
	} else if item := common.GetComponentType[*MeleeWeapon](e, MeleeWeaponComponent); item != nil {
		return item.Stats()
	} else if item := common.GetComponentType[*RangedWeapon](e, RangedWeaponComponent); item != nil {
		return item.Stats()
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

// Gets the underlying type of an item
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
