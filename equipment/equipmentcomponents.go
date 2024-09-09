package equipment

import (
	"game_main/common"
	"game_main/randgen"

	"github.com/bytearena/ecs"
)

var (
	ArmorComponent        *ecs.Component
	WeaponComponent       *ecs.Component
	InventoryComponent    *ecs.Component
	RangedWeaponComponent *ecs.Component
)

// This gets called so often that it might as well be a function
func GetItem(e *ecs.Entity) *Item {
	return common.GetComponentType[*Item](e, ItemComponent)
}

type Armor struct {
	ArmorClass  int
	Protection  int
	DodgeChance float32
}

// This gets called so often that it might as well be a function
func GetArmor(e *ecs.Entity) *Armor {
	return common.GetComponentType[*Armor](e, ArmorComponent)
}

type MeleeWeapon struct {
	MinDamage int
	MaxDamage int
}

func (w MeleeWeapon) CalculateDamage() int {

	return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)

}
