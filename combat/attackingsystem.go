package combat

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"game_main/randgen"
	"game_main/worldmap"
	"log"

	"github.com/bytearena/ecs"
)

// Rolls 1d20+AttackBonus and compares it to defenders armorclass. Has to be greater than or equal to the armor class to hit
// Then the defender does a dodge roll. If the dodge roll is greater than or equal to its dodge value, the attack hits
// If the attacker hits, subtract the Defenders protection value from the damage
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, attackerPos *common.Position, defenderPos *common.Position) {

	var attacker *ecs.Entity = nil
	var defender *ecs.Entity = nil

	//var weaponComponent any
	var weapon *gear.MeleeWeapon = nil

	if pl.Pos.IsEqual(attackerPos) {
		attacker = pl.PlayerEntity
		defender = GetCreatureAtPosition(ecsmanager, defenderPos)
		weapon = pl.GetPlayerWeapon()

	} else {
		attacker = GetCreatureAtPosition(ecsmanager, attackerPos)
		defender = pl.PlayerEntity
		weapon = common.GetComponentType[*gear.MeleeWeapon](attacker, gear.MeleeWeaponComponent)

	}

	if weapon != nil {

		PerformAttack(ecsmanager, pl, gm, weapon.CalculateDamage(), attacker, defender)

	} else {
		log.Print("Failed to attack. No weapon")
	}

}

// Passing the damage rather than the weapon so that Melee and Ranged Attacks can use the same function
// Currently Melee and Ranged Weapons are different types without a common interface
func PerformAttack(ecsmanagr *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, damage int, attacker *ecs.Entity, defender *ecs.Entity) {

	attAttr := common.GetAttributes(attacker)
	defAttr := common.GetAttributes(defender)

	attackRoll := randgen.GetDiceRoll(20) + attAttr.AttackBonus

	if attackRoll >= defAttr.TotalArmorClass {

		dodgeRoll := randgen.GetRandomBetween(0, 100)

		if dodgeRoll >= int(defAttr.TotalDodgeChance) {

			totalDamage := damage - defAttr.TotalProtection

			if totalDamage < 0 {
				totalDamage = 0
			}

			defAttr.CurrentHealth -= totalDamage

		} else {
			fmt.Println("Dodged")
		}

	} else {
		fmt.Println("Missed")
	}

	//RemoveDeadEntity(ecsmanagr, pl, gm, defender)
}

// A monster performing a ranged attack is simple right now.
// It ignores the weapons AOE and selects only the player as the target
// Todo add nill check for when there is no weapon for a player or monster attacker
func RangedAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, attackerPos *common.Position) {

	var attacker *ecs.Entity = nil

	var weapon *gear.RangedWeapon = nil
	var targets []*ecs.Entity

	if pl.Pos.IsEqual(attackerPos) {
		attacker = pl.PlayerEntity
		weapon = pl.GetPlayerRangedWeapon()
		if weapon != nil {
			targets = weapon.GetTargets(ecsmanager)
		}
	} else {
		attacker = GetCreatureAtPosition(ecsmanager, attackerPos)
		weapon = common.GetComponentType[*gear.RangedWeapon](attacker, gear.RangedWeaponComponent)
		targets = append(targets, pl.PlayerEntity)
	}

	if weapon != nil {

		for _, t := range targets {

			defenderPos := common.GetPosition(t)
			if attackerPos.InRange(defenderPos, weapon.ShootingRange) {

				PerformAttack(ecsmanager, pl, gm, weapon.CalculateDamage(), attacker, t)
				weapon.DisplayShootingVX(attackerPos, defenderPos)

			}
		}

	} else {
		log.Print("Failed to attack. No ranged weapon")
	}

}

// Does not remove the player if they die.
func RemoveDeadEntity(ecsmnager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, defender *ecs.Entity) {

	defenderPos := common.GetPosition(defender)
	defAttr := common.GetAttributes(defender)
	if pl.Pos.IsEqual(defenderPos) {
		graphics.IndexFromXY(defenderPos.X, defenderPos.Y) //Just here as a placeholder. Does nothing.
	} else if defAttr.CurrentHealth <= 0 {
		index := graphics.IndexFromXY(defenderPos.X, defenderPos.Y)

		gm.Tiles[index].Blocked = false
		ecsmnager.World.DisposeEntity(defender)
	}

}

func GetCreatureAtPosition(ecsmnager *common.EntityManager, pos *common.Position) *ecs.Entity {

	var e *ecs.Entity = nil
	for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {

		curPos := common.GetPosition(c.Entity)

		if pos.IsEqual(curPos) {
			e = c.Entity
			break
		}

	}

	return e

}
