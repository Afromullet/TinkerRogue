package main

import (
	"fmt"
	"log"

	"github.com/bytearena/ecs"
)

type TurnState int

const (
	BeforePlayerAction = iota
	PlayerTurn
	MonsterTurn
)

func GetNextState(state TurnState) TurnState {
	switch state {
	case BeforePlayerAction:
		return PlayerTurn
	case PlayerTurn:
		return MonsterTurn
	case MonsterTurn:
		return BeforePlayerAction
	default:
		return PlayerTurn
	}
}

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

		weapon = ComponentType[*Weapon](attacker, WeaponComponent)

	}

	attackerMessage = ComponentType[*UserMessage](attacker, userMessage)
	log.Print(attackerMessage)

	if weapon != nil {

		defenderHealth := ComponentType[*Health](defender, healthComponent)
		if defenderHealth != nil {
			defenderHealth.CurrentHealth -= weapon.Damage
			attackerMessage.AttackMessage = fmt.Sprintf("Damage Done: %d\n", weapon.Damage)

			if defenderHealth.CurrentHealth <= 0 {
				//Todo removing an entity is really closely coupled to teh map right now.
				//Do it differently in the future
				index := IndexFromXY(defenderPos.X, defenderPos.Y)

				g.gameMap.Tiles[index].Blocked = false
				g.World.DisposeEntity(defender)
			}
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
