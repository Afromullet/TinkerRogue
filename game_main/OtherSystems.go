package main

import (
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

	var weaponComponent any

	if g.playerData.position.IsEqual(attackerPos) {
		attacker = g.playerData.playerEntity
		defender = GetCreatureAtPosition(g, defenderPos)

		weaponComponent, _ = g.playerData.playerWeapon.GetComponentData(WeaponComponent)
	} else {
		attacker = GetCreatureAtPosition(g, defenderPos)
		defender = g.playerData.playerEntity

		weaponComponent, _ = attacker.GetComponentData(WeaponComponent)
	}

	//Todo add safety checks
	weapon := weaponComponent.(*Weapon)

	if weapon != nil {

		var defenderHealth *Health = nil

		if h, healthOK := defender.GetComponentData(healthComponent); healthOK {
			defenderHealth = h.(*Health)

			defenderHealth.CurrentHealth -= 1

			if defenderHealth.CurrentHealth <= 0 {

				//Todo removing an entity is really closely coupled to teh map right now.
				//Do it differently in the future
				index := GetIndexFromXY(defenderPos.X, defenderPos.Y)

				g.gameMap.Tiles[index].Blocked = false
				g.World.DisposeEntity(defender)

			}

		}

		if defenderHealth != nil {

		} else {
			log.Print("Error. Defender does not have a health component")
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
