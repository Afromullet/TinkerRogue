package main

import (
	"fmt"
	"log"

	"github.com/bytearena/ecs"
)

// Rolls 1d20+AttackBonus and compares it to defenders armorclass. Has to be greater than or equal to the armor class to hit
// Then the defender does a dodge roll. If the dodge roll is greater than or equal to its dodge value, the attack hits
// If the attacker hits, subtract the Defenders protection value from the damage
func MeleeAttackSystem(g *Game, attackerPos *Position, defenderPos *Position) {

	//var attacker *ecs.QueryResult = nil
	//log.Print(attacker)

	//Determine if the player is the attacker or defender

	var attacker *ecs.Entity = nil
	var defender *ecs.Entity = nil
	//var attackerMessage *UserMessage = nil

	//var weaponComponent any
	var weapon *Weapon = nil

	if g.playerData.position.IsEqual(attackerPos) {
		fmt.Println("Player is attacking")
		attacker = g.playerData.PlayerEntity
		defender = GetCreatureAtPosition(g, defenderPos)
		weapon = g.playerData.GetPlayerWeapon()

	} else {
		attacker = GetCreatureAtPosition(g, attackerPos) //todo I think this will cause an issue. Should be attackerPos. Worry about this when allowing monsters to attack
		defender = g.playerData.PlayerEntity
		fmt.Println("Monster is attacking")

		weapon = GetComponentType[*Weapon](attacker, WeaponComponent)

	}

	attAttr := GetComponentType[*Attributes](attacker, AttributeComponent)
	defAttr := GetComponentType[*Attributes](defender, AttributeComponent)

	if weapon != nil && attAttr != nil && defAttr != nil {

		PerformAttack(g, weapon.CalculateDamage(), defender, defenderPos, attAttr, defAttr)

	} else {
		log.Print("Failed to attack. No weapon")
	}

}

func PerformAttack(g *Game, damage int, defender *ecs.Entity, defenderPos *Position, attAttr *Attributes, defAttr *Attributes) {

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

	RemoveDeadEntity(g, defender, defAttr, defenderPos)

}

func RangedAttackSystem(g *Game, attackerPos *Position) {

	//var attacker *ecs.QueryResult = nil
	//log.Print(attacker)

	//Determine if the player is the attacker or defender

	var attacker *ecs.Entity = nil
	//var defender *ecs.Entity = nil
	//var attackerMessage *UserMessage = nil

	//var weaponComponent any
	var weapon *RangedWeapon = nil

	if g.playerData.position.IsEqual(attackerPos) {
		attacker = g.playerData.PlayerEntity
		weapon = g.playerData.GetPlayerRangedWeapon()
	}

	attAttr := GetComponentType[*Attributes](attacker, AttributeComponent)

	if weapon != nil {

		targets := weapon.GetTargets(g)

		var defAttr *Attributes
		for _, t := range targets {

			defenderPos := GetPosition(t)
			if attackerPos.InRange(defenderPos, weapon.ShootingRange) {
				fmt.Println("Shooting")

				defAttr = GetComponentType[*Attributes](t, AttributeComponent)
				PerformAttack(g, weapon.CalculateDamage(), t, defenderPos, attAttr, defAttr)
			} else {
				fmt.Println("Out of range")
			}

		}

	} else {
		log.Print("Failed to attack. No ranged weapon")
	}

}

// Todo need to handle player death differently
// TOdo can also just call GetPosition instead of passing defenderPos
func RemoveDeadEntity(g *Game, defender *ecs.Entity, defAttr *Attributes, defenderPos *Position) {

	if g.playerData.position.IsEqual(defenderPos) {
		fmt.Println("Player dead")
	} else if defAttr.CurrentHealth <= 0 {
		//Todo removing an entity is really closely coupled to teh map right now.
		//Do it differently in the future
		index := IndexFromXY(defenderPos.X, defenderPos.Y)

		g.gameMap.Tiles[index].Blocked = false
		g.World.DisposeEntity(defender)
	}

}

func GetCreatureAtPosition(g *Game, pos *Position) *ecs.Entity {

	var e *ecs.Entity = nil
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[PositionComponent].(*Position)

		if pos.IsEqual(curPos) {
			e = c.Entity
			break
		}

	}

	return e

}
