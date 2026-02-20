package perks

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/effects"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// ========================================
// COUNTER MOD BEHAVIORS
// ========================================

// riposteCounterMod: Counterattacks deal 100% damage (override 50% default).
func riposteCounterMod(defenderID, attackerID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) bool {
	modifiers.DamageMultiplier = 1.0 // Override 0.5 default
	modifiers.HitPenalty = 0         // Override -20 default
	return false                     // Don't skip counter
}

// stoneWallCounterMod: Skip counterattack entirely.
func stoneWallCounterMod(defenderID, attackerID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) bool {
	return true // Skip counterattack entirely
}

// ========================================
// DAMAGE MOD BEHAVIORS
// ========================================

// stoneWallDamageMod: -30% damage taken (applied on defender side).
func stoneWallDamageMod(attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) {
	// Only applies when this perk holder is the defender
	if !HasPerk(defenderID, "stone_wall", manager) {
		return
	}
	modifiers.DamageMultiplier *= 0.7
}

// berserkerDamageMod: +30% damage when below 50% HP.
func berserkerDamageMod(attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) {
	attr := common.GetComponentTypeByID[*common.Attributes](manager, attackerID, common.AttributeComponent)
	if attr != nil && attr.MaxHealth > 0 && float64(attr.CurrentHealth)/float64(attr.MaxHealth) < 0.5 {
		modifiers.DamageMultiplier *= 1.3
	}
}

// armorPiercingDamageMod: Halve effective armor.
func armorPiercingDamageMod(attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) {
	modifiers.ArmorReduction = 0.5
}

// glassCannonDamageMod: +35% damage dealt (squad perk, applies to all units).
// When unit is attacking: boost damage.
// When unit is defending: handled by RunDefenderDamageModHooks increasing damage taken.
func glassCannonDamageMod(attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) {
	// Check if the attacker has glass_cannon -- boost outgoing damage
	if HasPerk(attackerID, "glass_cannon", manager) {
		modifiers.DamageMultiplier *= 1.35
		return
	}
	// Check if the defender has glass_cannon -- increase damage taken
	if HasPerk(defenderID, "glass_cannon", manager) {
		modifiers.DamageMultiplier *= 1.2
	}
}

// focusFireDamageMod: 2x damage when using focus fire.
func focusFireDamageMod(attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) {
	modifiers.DamageMultiplier *= 2.0
}

// ========================================
// TARGET OVERRIDE BEHAVIORS
// ========================================

// focusFireTargetOverride: Single target only.
func focusFireTargetOverride(attackerID, defenderSquadID ecs.EntityID,
	defaultTargets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	if len(defaultTargets) > 0 {
		return defaultTargets[:1]
	}
	return defaultTargets
}

// cleaveTargetOverride: Hit target row + row behind (melee row only).
func cleaveTargetOverride(attackerID, defenderSquadID ecs.EntityID,
	defaultTargets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// Only applies to melee row attackers
	targetData := common.GetComponentTypeByID[*squads.TargetRowData](
		manager, attackerID, squads.TargetRowComponent,
	)
	if targetData == nil || targetData.AttackType != squads.AttackTypeMeleeRow {
		return defaultTargets
	}

	if len(defaultTargets) == 0 {
		return defaultTargets
	}

	// Find what row the default targets are in
	pos := common.GetComponentTypeByID[*squads.GridPositionData](
		manager, defaultTargets[0], squads.GridPositionComponent,
	)
	if pos == nil {
		return defaultTargets
	}

	// Add units from the next row behind
	nextRow := pos.AnchorRow + 1
	if nextRow <= 2 {
		extraTargets := squads.GetUnitsInRow(defenderSquadID, nextRow, manager)
		if len(extraTargets) > 0 {
			return append(defaultTargets, extraTargets...)
		}
	}
	return defaultTargets
}

// ========================================
// POST-DAMAGE BEHAVIORS
// ========================================

// lifestealPostDamage: Heal 25% of damage dealt.
func lifestealPostDamage(attackerID, defenderID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	if damageDealt <= 0 {
		return
	}

	healAmount := damageDealt / 4
	if healAmount < 1 {
		healAmount = 1
	}

	attr := common.GetComponentTypeByID[*common.Attributes](manager, attackerID, common.AttributeComponent)
	if attr != nil && attr.CurrentHealth > 0 {
		attr.CurrentHealth += healAmount
		if attr.CurrentHealth > attr.MaxHealth {
			attr.CurrentHealth = attr.MaxHealth
		}
		fmt.Printf("[PERK] Lifesteal: entity %d healed %d HP\n", attackerID, healAmount)
	}
}

// inspirationPostDamage: On kill, +2 strength to all squad allies for 2 turns.
func inspirationPostDamage(attackerID, defenderID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	if !wasKill {
		return
	}

	memberData := common.GetComponentTypeByID[*squads.SquadMemberData](
		manager, attackerID, squads.SquadMemberComponent,
	)
	if memberData == nil {
		return
	}

	unitIDs := squads.GetUnitIDsInSquad(memberData.SquadID, manager)
	effect := effects.ActiveEffect{
		Name:           "Inspiration",
		Source:         effects.SourcePerk,
		Stat:           effects.StatStrength,
		Modifier:       2,
		RemainingTurns: 2,
	}
	effects.ApplyEffectToUnits(unitIDs, effect, manager)
	fmt.Printf("[PERK] Inspiration triggered by entity %d: +2 STR to squad\n", attackerID)
}

// ========================================
// COVER MOD BEHAVIORS
// ========================================

// impaleCoverMod: Melee column attacks ignore cover.
func impaleCoverMod(attackerID, defenderID ecs.EntityID,
	coverBreakdown *squads.CoverBreakdown, manager *common.EntityManager) {
	// Only applies to melee column attackers
	targetData := common.GetComponentTypeByID[*squads.TargetRowData](
		manager, attackerID, squads.TargetRowComponent,
	)
	if targetData == nil || targetData.AttackType != squads.AttackTypeMeleeColumn {
		return
	}

	coverBreakdown.TotalReduction = 0
	coverBreakdown.Providers = nil
}

// ========================================
// TURN START BEHAVIORS
// ========================================

// warMedicTurnStart: Heal 3 HP to the lowest-HP ally in the squad.
func warMedicTurnStart(squadID ecs.EntityID, manager *common.EntityManager) {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)

	var lowestID ecs.EntityID
	lowestHP := 999999

	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		if attr != nil && attr.CurrentHealth > 0 && attr.CurrentHealth < attr.MaxHealth && attr.CurrentHealth < lowestHP {
			lowestHP = attr.CurrentHealth
			lowestID = uid
		}
	}

	if lowestID != 0 {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, lowestID, common.AttributeComponent)
		if attr != nil {
			healAmount := 3
			attr.CurrentHealth += healAmount
			if attr.CurrentHealth > attr.MaxHealth {
				attr.CurrentHealth = attr.MaxHealth
			}
			fmt.Printf("[PERK] War Medic: healed entity %d for %d HP\n", lowestID, healAmount)
		}
	}
}
