package equipment

import (
	"game_main/ecshelper"
	"math/big"

	cryptorand "crypto/rand"

	"github.com/bytearena/ecs"
)

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
func GetDiceRoll(num int) int {
	x, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(num)))
	return int(x.Int64()) + 1

}

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
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

var (
	ArmorComponent        *ecs.Component
	WeaponComponent       *ecs.Component
	InventoryComponent    *ecs.Component
	RangedWeaponComponent *ecs.Component
)

// This gets called so often that it might as well be a function
func GetItem(e *ecs.Entity) *Item {
	return ecshelper.GetComponentType[*Item](e, ItemComponent)
}

type Armor struct {
	ArmorClass  int
	Protection  int
	DodgeChance float32
}

// This gets called so often that it might as well be a function
func GetArmor(e *ecs.Entity) *Armor {
	return ecshelper.GetComponentType[*Armor](e, ArmorComponent)
}

type MeleeWeapon struct {
	MinDamage int
	MaxDamage int
}

func (w MeleeWeapon) CalculateDamage() int {

	return GetRandomBetween(w.MinDamage, w.MaxDamage)

}
