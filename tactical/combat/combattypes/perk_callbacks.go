package combattypes

import (
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// PerkDispatcher defines the contract for dispatching perk hooks into the damage pipeline.
// Implemented by the perks package; consumed by combat systems.
// All methods are safe to call unconditionally — the implementation handles
// "no perks equipped" by iterating zero behaviors (no-op).
type PerkDispatcher interface {
	AttackerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID, modifiers *DamageModifiers, manager *common.EntityManager)
	DefenderDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID, modifiers *DamageModifiers, manager *common.EntityManager)
	CoverMod(attackerID, defenderID ecs.EntityID, cover *CoverBreakdown, manager *common.EntityManager)
	TargetOverride(attackerID, defenderSquadID ecs.EntityID, targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID
	CounterMod(defenderSquadID, attackerID ecs.EntityID, modifiers *DamageModifiers, manager *common.EntityManager) bool
	AttackerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID, damage int, wasKill bool, manager *common.EntityManager)
	DefenderPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID, damage int, wasKill bool, manager *common.EntityManager)
	DeathOverride(unitID, squadID ecs.EntityID, manager *common.EntityManager) bool
	DamageRedirect(defenderID, defenderSquadID ecs.EntityID, damageAmount int, manager *common.EntityManager) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
}
