package behavior

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"

	"github.com/bytearena/ecs"
)

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
