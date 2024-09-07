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

func RangedAttackSystem(g *Game, attackerPos *Position) {

	var attacker *ecs.Entity = nil
	var weapon *RangedWeapon = nil

	if g.playerData.position.IsEqual(attackerPos) {
		attacker = g.playerData.PlayerEntity
		weapon = g.playerData.GetPlayerRangedWeapon()
	}

	if weapon != nil {

		targets := weapon.GetTargets(g)

		for _, t := range targets {

			defenderPos := GetPosition(t)
			if attackerPos.InRange(defenderPos, weapon.ShootingRange) {
				fmt.Println("Shooting")

				PerformAttack(g, weapon.CalculateDamage(), attacker, t)

				//RangedAttackDrawnigPlaceHolder(attackerPos, defenderPos, weapon)

			} else {
				fmt.Println("Out of range")
			}

		}

		RangedAttackAreaDrawnigPlaceHolder(attackerPos, weapon)

	} else {
		log.Print("Failed to attack. No ranged weapon")
	}

}

func RangedAttackDrawnigPlaceHolder(attackerPos *Position, defenderPos *Position, weapon *RangedWeapon) {

	attX, attY := PixelsFromPosition(attackerPos)
	defX, defY := PixelsFromPosition(defenderPos)

	//arr := NewFireEffect(attX, attY, 1, 5, 1, 0.5)
	arr := NewElectricArc(attX, attY, defX, defY, 5)

	vxHandler.AddVisualEffect(arr)

}

func RangedAttackAreaDrawnigPlaceHolder(attackerPos *Position, weapon *RangedWeapon) {

	attX, attY := PixelsFromPosition(attackerPos)
	//defX, defY := PixelsFromPosition(defenderPos)

	//arr := NewFireEffect(attX, attY, 1, 5, 1, 0.5)
	arr := NewElectricityEffect(attX, attY, 5)
	area := NewVisualEffectArea(weapon.TargetArea, arr)
	vxHandler.AddVisualEffecArea(area)

	//vxHandler.AddVisualEffect(arr)

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
