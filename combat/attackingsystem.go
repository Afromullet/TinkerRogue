package combat

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/graphics"
	"game_main/randgen"
	resmanager "game_main/resourcemanager"
	"game_main/worldmap"
	"log"
	"strconv"

	"github.com/bytearena/ecs"
)

// Rolls 1d20+AttackBonus and compares it to defenders armorclass. Has to be greater than or equal to the armor class to hit
// Then the defender does a dodge roll. If the dodge roll is greater than or equal to its dodge value, the attack hits
// If the attacker hits, subtract the Defenders protection value from the damage
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, attackerPos *coords.LogicalPosition, defenderPos *coords.LogicalPosition) {

	var attacker *ecs.Entity = nil
	var defender *ecs.Entity = nil
	var weapon *gear.MeleeWeapon = nil
	attackSuccess := false
	playerAttacking := false

	if pl.Pos.IsEqual(attackerPos) {
		playerAttacking = true
		attacker = pl.PlayerEntity
		defender = common.GetCreatureAtPosition(ecsmanager, defenderPos)
		weapon = pl.Equipment.MeleeWeapon()

	} else {
		attacker = common.GetCreatureAtPosition(ecsmanager, attackerPos)
		defender = pl.PlayerEntity
		weapon = common.GetComponentType[*gear.MeleeWeapon](attacker, gear.MeleeWeaponComponent)

	}

	if weapon != nil {

		damage := weapon.CalculateDamage()
		attackSuccess = PerformAttack(ecsmanager, pl, gm, damage, attacker, defender, playerAttacking)
		UpdateAttackMessage(attacker, attackSuccess, playerAttacking, damage)

	} else {
		log.Print("Failed to attack. No weapon")
		// Todo add a fist attack
	}

}

// A monster performing a ranged attack is simple right now.
// It ignores the weapons AOE and selects only the player as the target
// Todo add nil check for when there is no weapon for a player or monster attacker
func RangedAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, attackerPos *coords.LogicalPosition) {

	var attacker *ecs.Entity = nil
	var weapon *gear.RangedWeapon = nil
	var targets []*ecs.Entity
	attackSuccess := false
	playerAttacking := false

	if pl.Pos.IsEqual(attackerPos) {
		attacker = pl.PlayerEntity
		playerAttacking = true
		weapon = pl.Equipment.RangedWeapon()
		if weapon != nil {
			targets = weapon.GetTargets(ecsmanager)
		}
	} else {
		attacker = common.GetCreatureAtPosition(ecsmanager, attackerPos)
		weapon = common.GetComponentType[*gear.RangedWeapon](attacker, gear.RangedWeaponComponent)

		targets = append(targets, pl.PlayerEntity)
	}

	if weapon != nil {

		for _, t := range targets {

			defenderPos := common.GetPosition(t)
			if attackerPos.InRange(defenderPos, weapon.ShootingRange) {

				damage := weapon.CalculateDamage()

				attackSuccess = PerformAttack(ecsmanager, pl, gm, weapon.CalculateDamage(), attacker, t, playerAttacking)

				if graphics.MAP_SCROLLING_ENABLED {
					weapon.DisplayCenteredShootingVX(attackerPos, defenderPos)
				} else {
					weapon.DisplayShootingVX(attackerPos, defenderPos)
				}
				UpdateAttackMessage(attacker, attackSuccess, playerAttacking, damage)

			}
		}

	} else {
		// Todo add a panic here, since this should not happen
		log.Print("Failed to attack. No ranged weapon")
		//Not adding a default option here since it doesn't make sense to have a default ranged attack.
		//For melee, the entity can use its fists. For a ranged attack, that does not work

	}

}

// Passing the damage rather than the weapon so that Melee and Ranged Attacks can use the same function
// Currently Melee and Ranged Weapons are different types without a common interface
// Returns true if attack hits. False otherwise.
func PerformAttack(em *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, damage int, attacker *ecs.Entity, defender *ecs.Entity, isPlayerAttacking bool) bool {

	attAttr := common.GetAttributes(attacker)
	defAttr := common.GetAttributes(defender)

	attackRoll := randgen.GetDiceRoll(20) + attAttr.AttackBonus

	if attackRoll >= defAttr.TotalArmorClass {

		dodgeRoll := randgen.GetRandomBetween(0, 100)

		if dodgeRoll >= int(defAttr.TotalDodgeChance) {

			totalDamage := damage - defAttr.TotalProtection

			if totalDamage < 0 {
				totalDamage = 1
			}

			defAttr.CurrentHealth -= totalDamage
			return true

		} else {
			//TODO: Add feedback for dodge
		}

	} else {
		//TODO: Add feedback for miss
	}

	//Only check if it's following a player attack.
	// Not handling player removal yet
	if isPlayerAttacking {
		resmanager.RemoveEntity(em.World, gm, defender)
	}

	return false

}

// Used to update the messages that will be displayed in the GUI
func UpdateAttackMessage(attacker *ecs.Entity, attackSuccess, isPlayerAttacking bool, damage int) {

	attackerMessage := ""
	msg := common.GetComponentType[*common.UserMessage](attacker, common.UserMsgComponent)

	if isPlayerAttacking && attackSuccess {

		if attackSuccess {
			attackerMessage = "You hit for " + strconv.Itoa(damage) + " damage"
		} else {
			attackerMessage = "Your attack misses"
		}

	} else {

		//Todo, this kept on crashing for some components. Something must not have a name added
		if attacker.HasComponent(common.NameComponent) {
			attackerMessage = common.GetComponentType[*common.Name](attacker, common.NameComponent).NameStr + " attacks and "
		}

		if attackSuccess {

			attackerMessage += "hits for " + strconv.Itoa(damage) + " damage"

		} else {
			attackerMessage = " misses"

		}

	}

	msg.AttackMessage = attackerMessage

}
