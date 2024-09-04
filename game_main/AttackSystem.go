package main

import (
	"fmt"
	"log"

	"github.com/bytearena/ecs"
)

// Rolls 1d20+AttackBonus and compares it to defenders armorclass. Has to be greater than or equal to the armor class to hit
// Then the defender does a dodge roll. If the dodge roll is greater than or equal to its dodge value, the attack hits
// If the attacker hits, subtract the Defenders protection value from the damage
func AttackSystem(g *Game, attackerPos *Position, defenderPos *Position) {

	//var attacker *ecs.QueryResult = nil
	//log.Print(attacker)

	//Determine if the player is the attacker or defender

	var attacker *ecs.Entity = nil
	var defender *ecs.Entity = nil
	var attackerMessage *UserMessage = nil

	//var weaponComponent any
	var weapon *Weapon = nil

	if g.playerData.position.IsEqual(attackerPos) {
		attacker = g.playerData.PlayerEntity
		defender = GetCreatureAtPosition(g, defenderPos)
		weapon = g.playerData.GetPlayerWeapon()

	} else {
		attacker = GetCreatureAtPosition(g, defenderPos)
		defender = g.playerData.PlayerEntity

		weapon = GetComponentType[*Weapon](attacker, WeaponComponent)

	}

	attAttr := GetComponentType[*Attributes](attacker, attributeComponent)
	defAttr := GetComponentType[*Attributes](defender, attributeComponent)
	attackerMessage = GetComponentType[*UserMessage](attacker, userMessage)

	fmt.Println("Defender AC, Prot, Dodge, and Health ", defAttr.TotalArmorClass, defAttr.TotalProtection,
		defAttr.TotalDodgeChance, defAttr.CurrentHealth)

	if weapon != nil && attAttr != nil && defAttr != nil {

		damage := weapon.CalculateDamage()

		attackRoll := GetDiceRoll(20) + attAttr.AttackBonus

		fmt.Println("Attack Roll vs AC", attackRoll, defAttr.TotalArmorClass)

		if attackRoll >= defAttr.TotalArmorClass {

			dodgeRoll := GetRandomBetween(0, 100)

			fmt.Println("Dodge Roll Result ", dodgeRoll)

			if dodgeRoll >= int(defAttr.TotalDodgeChance) {

				totalDamage := damage - defAttr.TotalProtection

				if totalDamage < 0 {
					totalDamage = 0
				}

				fmt.Println("Damage Roll", damage)
				fmt.Println("Actual  Damage", totalDamage)

				defAttr.CurrentHealth -= totalDamage

			}

			attackerMessage.AttackMessage = fmt.Sprintf("Damage Done: %d\n", damage)

		}

		if defAttr.CurrentHealth <= 0 {
			//Todo removing an entity is really closely coupled to teh map right now.
			//Do it differently in the future
			index := IndexFromXY(defenderPos.X, defenderPos.Y)

			g.gameMap.Tiles[index].Blocked = false
			g.World.DisposeEntity(defender)
		}

	} else {
		log.Print("Failed to attack. No weapon")
	}

}

func GetCreatureAtPosition(g *Game, pos *Position) *ecs.Entity {

	var e *ecs.Entity = nil
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[position].(*Position)

		if pos.IsEqual(curPos) {
			e = c.Entity
			break
		}

	}

	return e

}
