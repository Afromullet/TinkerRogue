package behavior

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"

	"github.com/bytearena/ecs"
)

// Used when spawning a creature.
// Determines what kind of attack behavior the creature has
// If the creature has both a ranged and a melee weapon, it defaults to using the ranged weapon
func BehaviorSelector(ent *ecs.Entity, pl *avatar.PlayerData) {

	meleeWep := common.GetComponentType[*gear.MeleeWeapon](ent, gear.MeleeWeaponComponent)
	rangedWep := common.GetComponentType[*gear.RangedWeapon](ent, gear.RangedWeaponComponent)

	//If creature has both, use ranged attack
	if meleeWep != nil && rangedWep != nil {
		ent.AddComponent(RangeAttackBehaviorComp, &AttackBehavior{})

	} else if rangedWep != nil {
		ent.AddComponent(RangeAttackBehaviorComp, &AttackBehavior{})

	} else if meleeWep != nil {
		ent.AddComponent(ChargeAttackComp, &AttackBehavior{})

	}

}
