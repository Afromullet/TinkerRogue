package main

import (
	"fmt"
	"game_main/ecshelper"
	"game_main/graphics"
	"log"

	"github.com/bytearena/ecs"
)

// Rolls 1d20+AttackBonus and compares it to defenders armorclass. Has to be greater than or equal to the armor class to hit
// Then the defender does a dodge roll. If the dodge roll is greater than or equal to its dodge value, the attack hits
// If the attacker hits, subtract the Defenders protection value from the damage
func MeleeAttackSystem(g *Game, attackerPos *ecshelper.Position, defenderPos *ecshelper.Position) {

	var attacker *ecs.Entity = nil
	var defender *ecs.Entity = nil

	//var weaponComponent any
	var weapon *Weapon = nil

	if g.playerData.position.IsEqual(attackerPos) {
		fmt.Println("Player is attacking")
		attacker = g.playerData.PlayerEntity
		defender = GetCreatureAtPosition(g, defenderPos)
		weapon = g.playerData.GetPlayerWeapon()

	} else {
		attacker = GetCreatureAtPosition(g, attackerPos)
		defender = g.playerData.PlayerEntity
		fmt.Println("Monster is attacking")
		weapon = ecshelper.GetComponentType[*Weapon](attacker, WeaponComponent)

	}

	if weapon != nil {

		PerformAttack(g, weapon.CalculateDamage(), attacker, defender)

	} else {
		log.Print("Failed to attack. No weapon")
	}

}

// Passing the damage rather than the weapon so that Melee and Ranged Attacks can use the same function
// Currently Melee and Ranged Weapons are different types without a common interface
func PerformAttack(g *Game, damage int, attacker *ecs.Entity, defender *ecs.Entity) {

	attAttr := GetAttributes(attacker)
	defAttr := GetAttributes(defender)

	attackRoll := GetDiceRoll(20) + attAttr.AttackBonus

	if attackRoll >= defAttr.TotalArmorClass {

		dodgeRoll := GetRandomBetween(0, 100)

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

	RemoveDeadEntity(g, defender)

}

// A monster doing a ranged attack is simple right now.
// It ignores the weapons AOE and selects only the player as the target
func RangedAttackSystem(g *Game, attackerPos *ecshelper.Position) {

	var attacker *ecs.Entity = nil

	var weapon *RangedWeapon = nil

	var targets []*ecs.Entity

	if g.playerData.position.IsEqual(attackerPos) {
		attacker = g.playerData.PlayerEntity
		weapon = g.playerData.GetPlayerRangedWeapon()
		if weapon != nil {
			targets = weapon.GetTargets(g)
		}
	} else {
		attacker = GetCreatureAtPosition(g, attackerPos) //todo I think this will cause an issue. Should be attackerPos. Worry about this when allowing monsters to attack

		fmt.Println("Monster is shooting")

		weapon = ecshelper.GetComponentType[*RangedWeapon](attacker, RangedWeaponComponent)
		targets = append(targets, g.playerData.PlayerEntity)
	}

	// Todo I could return from the function when checking if weapon is not nill above
	if weapon != nil {

		for _, t := range targets {

			defenderPos := GetPosition(t)
			if attackerPos.InRange(defenderPos, weapon.ShootingRange) {
				fmt.Println("Shooting")

				PerformAttack(g, weapon.CalculateDamage(), attacker, t)
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
func RemoveDeadEntity(g *Game, defender *ecs.Entity) {

	defenderPos := GetPosition(defender)
	defAttr := GetAttributes(defender)
	if g.playerData.position.IsEqual(defenderPos) {
		fmt.Println("Player dead")
	} else if defAttr.CurrentHealth <= 0 {
		//Todo removing an entity is really closely coupled to teh map right now.
		//Do it differently in the future
		index := graphics.IndexFromXY(defenderPos.X, defenderPos.Y)

		g.gameMap.Tiles[index].Blocked = false
		g.World.DisposeEntity(defender)
	}

}

func GetCreatureAtPosition(g *Game, pos *ecshelper.Position) *ecs.Entity {

	var e *ecs.Entity = nil
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[ecshelper.PositionComponent].(*ecshelper.Position)

		if pos.IsEqual(curPos) {
			e = c.Entity
			break
		}

	}

	return e

}
