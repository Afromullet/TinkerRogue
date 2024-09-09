package main

import (
	"fmt"
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"
	"game_main/randgen"
	"game_main/worldmap"
	"log"

	"github.com/bytearena/ecs"
)

// Rolls 1d20+AttackBonus and compares it to defenders armorclass. Has to be greater than or equal to the armor class to hit
// Then the defender does a dodge roll. If the dodge roll is greater than or equal to its dodge value, the attack hits
// If the attacker hits, subtract the Defenders protection value from the damage
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *PlayerData, gm *worldmap.GameMap, attackerPos *common.Position, defenderPos *common.Position) {

	var attacker *ecs.Entity = nil
	var defender *ecs.Entity = nil

	//var weaponComponent any
	var weapon *equipment.MeleeWeapon = nil

	if pl.position.IsEqual(attackerPos) {
		fmt.Println("Player is attacking")
		attacker = pl.PlayerEntity
		defender = GetCreatureAtPosition(ecsmanager, defenderPos)
		weapon = pl.GetPlayerWeapon()

	} else {
		attacker = GetCreatureAtPosition(ecsmanager, attackerPos)
		defender = pl.PlayerEntity
		fmt.Println("Monster is attacking")
		weapon = common.GetComponentType[*equipment.MeleeWeapon](attacker, equipment.WeaponComponent)

	}

	if weapon != nil {

		PerformAttack(ecsmanager, pl, gm, weapon.CalculateDamage(), attacker, defender)

	} else {
		log.Print("Failed to attack. No weapon")
	}

}

// Passing the damage rather than the weapon so that Melee and Ranged Attacks can use the same function
// Currently Melee and Ranged Weapons are different types without a common interface
func PerformAttack(ecsmanagr *common.EntityManager, pl *PlayerData, gm *worldmap.GameMap, damage int, attacker *ecs.Entity, defender *ecs.Entity) {

	attAttr := common.GetAttributes(attacker)
	defAttr := common.GetAttributes(defender)

	attackRoll := randgen.GetDiceRoll(20) + attAttr.AttackBonus

	if attackRoll >= defAttr.TotalArmorClass {

		dodgeRoll := randgen.GetRandomBetween(0, 100)

		if dodgeRoll >= int(defAttr.TotalDodgeChance) {

			fmt.Println("Hit")
			totalDamage := damage - defAttr.TotalProtection

			if totalDamage < 0 {
				totalDamage = 0
			}

			defAttr.CurrentHealth -= totalDamage
			fmt.Println("Remaining health ", defAttr.CurrentHealth)

		} else {
			fmt.Println("Dodged")
		}

	} else {
		fmt.Println("Missed")
	}

	RemoveDeadEntity(ecsmanagr, pl, gm, defender)
}

// A monster doing a ranged attack is simple right now.
// It ignores the weapons AOE and selects only the player as the target
func RangedAttackSystem(ecsmanager *common.EntityManager, pl *PlayerData, gm *worldmap.GameMap, attackerPos *common.Position) {

	var attacker *ecs.Entity = nil

	var weapon *equipment.RangedWeapon = nil

	var targets []*ecs.Entity

	if pl.position.IsEqual(attackerPos) {
		attacker = pl.PlayerEntity
		weapon = pl.GetPlayerRangedWeapon()
		if weapon != nil {
			targets = weapon.GetTargets(ecsmanager)
		}
	} else {
		attacker = GetCreatureAtPosition(ecsmanager, attackerPos) //todo I think this will cause an issue. Should be attackerPos. Worry about this when allowing monsters to attack

		fmt.Println("Monster is shooting")

		weapon = common.GetComponentType[*equipment.RangedWeapon](attacker, equipment.RangedWeaponComponent)
		targets = append(targets, pl.PlayerEntity)
	}

	// Todo I could return from the function when checking if weapon is not nill above
	if weapon != nil {

		for _, t := range targets {

			defenderPos := common.GetPosition(t)
			if attackerPos.InRange(defenderPos, weapon.ShootingRange) {
				fmt.Println("Shooting")

				PerformAttack(ecsmanager, pl, gm, weapon.CalculateDamage(), attacker, t)
				weapon.DisplayShootingVX(attackerPos, defenderPos)

			} else {
				fmt.Println("Out of range")
			}

		}

	} else {
		log.Print("Failed to attack. No ranged weapon")
	}

}

// Todo need to handle player death differently
// Todo if it attacks the player, it removes the attacking creature
// TOdo can also just call GetPosition instead of passing defenderPos
func RemoveDeadEntity(ecsmnager *common.EntityManager, pl *PlayerData, gm *worldmap.GameMap, defender *ecs.Entity) {

	defenderPos := common.GetPosition(defender)
	defAttr := common.GetAttributes(defender)
	if pl.position.IsEqual(defenderPos) {
		fmt.Println("Player dead")
	} else if defAttr.CurrentHealth <= 0 {
		//Todo removing an entity is really closely coupled to teh map right now.
		//Do it differently in the future
		index := graphics.IndexFromXY(defenderPos.X, defenderPos.Y)

		gm.Tiles[index].Blocked = false
		ecsmnager.World.DisposeEntity(defender)
	}

}

func GetCreatureAtPosition(ecsmnager *common.EntityManager, pos *common.Position) *ecs.Entity {

	var e *ecs.Entity = nil
	for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {

		curPos := c.Components[common.PositionComponent].(*common.Position)

		if pos.IsEqual(curPos) {
			e = c.Entity
			break
		}

	}

	return e

}
