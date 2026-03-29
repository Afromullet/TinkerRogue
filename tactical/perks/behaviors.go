package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func init() {
	registerAllPerkHooks()
}

// ========================================
// TIER 1: COMBAT CONDITIONING (10 perks)
// ========================================

// Brace for Impact: When defending (not attacking), all units gain +0.15 cover bonus
func braceForImpactCoverMod(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// This perk only activates on the defender side.
	// The defenderID's squad must have this perk, and the squad must be the one being attacked.
	defenderSquadID := getSquadIDForUnit(defenderID, manager)
	if !HasPerk(defenderSquadID, "brace_for_impact", manager) {
		return
	}
	coverBreakdown.TotalReduction += 0.15
	if coverBreakdown.TotalReduction > 1.0 {
		coverBreakdown.TotalReduction = 1.0
	}
}

// Reckless Assault: +30% damage dealt, but +20% damage received until next turn
func recklessAssaultDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// When this squad is the ATTACKER (not counterattacking)
	if !modifiers.IsCounterattack && attackerSquadID != 0 && HasPerk(attackerSquadID, "reckless_assault", manager) {
		modifiers.DamageMultiplier *= 1.3
		roundState.RecklessVulnerable = true
		return
	}

	// When this squad is the DEFENDER and has the vulnerability flag
	if defenderSquadID != 0 && HasPerk(defenderSquadID, "reckless_assault", manager) {
		defenderState := GetRoundState(defenderSquadID, manager)
		if defenderState != nil && defenderState.RecklessVulnerable {
			modifiers.DamageMultiplier *= 1.2
		}
	}
}

// Stalwart: Full-damage counters if squad did NOT move
func stalwartCounterMod(defenderID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) bool {
	if !roundState.MovedThisTurn {
		modifiers.DamageMultiplier = 1.0 // Override 0.5 default
	}
	return false // Don't skip counter
}

// Executioner's Instinct: +25% crit chance vs squads with any unit below 30% HP
func executionerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// Check if any unit in the defender's squad is below 30% HP
	unitIDs := squadcore.GetUnitIDsInSquad(defenderSquadID, manager)
	for _, unitID := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}
		maxHP := attr.GetMaxHealth()
		if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < 0.3 {
			modifiers.CritBonus += 25
			return
		}
	}
}

// Shieldwall Discipline: Per Tank unit in row 0, all squad units take 5% less damage (max 15%)
func shieldwallDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// Only activates as a defender perk
	if !HasPerk(defenderSquadID, "shieldwall_discipline", manager) {
		return
	}

	// Count alive tanks in row 0
	tankCount := 0
	unitIDs := squadcore.GetUnitIDsInSquad(defenderSquadID, manager)
	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}

		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData == nil || roleData.Role != unitdefs.RoleTank {
			continue
		}

		gridPos := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent)
		if gridPos != nil && gridPos.AnchorRow == 0 {
			tankCount++
		}
	}

	if tankCount > 3 {
		tankCount = 3
	}
	if tankCount > 0 {
		reduction := float64(tankCount) * 0.05
		modifiers.DamageMultiplier *= (1.0 - reduction)
	}
}

// Isolated Predator: +25% damage when no friendly squads within 3 tiles
func isolatedPredatorDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		manager, attackerSquadID, common.PositionComponent,
	)
	if squadPos == nil {
		return
	}

	// Get this squad's faction
	attackerFaction := combatcore.GetSquadFaction(attackerSquadID, manager)
	if attackerFaction == 0 {
		return
	}

	// Check all friendly squads for distance
	friendlySquads := combatcore.GetActiveSquadsForFaction(attackerFaction, manager)
	for _, friendlyID := range friendlySquads {
		if friendlyID == attackerSquadID {
			continue
		}
		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			continue
		}
		distance := squadPos.ChebyshevDistance(friendlyPos)
		if distance <= 3 {
			return // A friendly squad is nearby, no bonus
		}
	}

	// No friendly squads within 3 tiles
	modifiers.DamageMultiplier *= 1.25
}

// Vigilance: Crits become normal hits (defender perk)
func vigilanceDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// Only activates as defender
	if HasPerk(defenderSquadID, "vigilance", manager) {
		modifiers.SkipCrit = true
	}
}

// Field Medic: At round start, lowest-HP unit heals 10% max HP
func fieldMedicTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
	var lowestID ecs.EntityID
	lowestHP := 999999

	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](
			manager, uid, common.AttributeComponent,
		)
		if attr != nil && attr.CurrentHealth > 0 && attr.CurrentHealth < lowestHP {
			lowestHP = attr.CurrentHealth
			lowestID = uid
		}
	}

	if lowestID != 0 {
		attr := common.GetComponentTypeByID[*common.Attributes](
			manager, lowestID, common.AttributeComponent,
		)
		if attr != nil {
			maxHP := attr.GetMaxHealth()
			healAmount := maxHP / 10 // 10% max HP
			if healAmount < 1 {
				healAmount = 1
			}
			attr.CurrentHealth += healAmount
			if attr.CurrentHealth > maxHP {
				attr.CurrentHealth = maxHP
			}
		}
	}
}

// Opening Salvo: +35% damage on squad's first attack of the combat only
func openingSalvoDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	if modifiers.IsCounterattack {
		return
	}
	if !roundState.HasAttackedThisCombat {
		modifiers.DamageMultiplier *= 1.35
		roundState.HasAttackedThisCombat = true
	}
}

// Last Line: When last friendly squad alive, +20% hit, dodge, and damage
func lastLineDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// Get this squad's faction
	faction := combatcore.GetSquadFaction(attackerSquadID, manager)
	if faction == 0 {
		return
	}

	// Check if this is the last friendly squad
	aliveSquads := combatcore.GetActiveSquadsForFaction(faction, manager)
	if len(aliveSquads) == 1 && aliveSquads[0] == attackerSquadID {
		modifiers.DamageMultiplier *= 1.2
		modifiers.HitPenalty -= 20 // +20 hit (reduce penalty)
	}
}

// ========================================
// TIER 2: COMBAT SPECIALIZATION (14 perks)
// ========================================

// Cleave: Hit target row + row behind, but -30% damage to ALL targets
func cleaveTargetOverride(attackerID, defenderSquadID ecs.EntityID,
	defaultTargets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// Only applies to MeleeRow attack types
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		manager, attackerID, squadcore.TargetRowComponent,
	)
	if targetData == nil || targetData.AttackType != unitdefs.AttackTypeMeleeRow {
		return defaultTargets
	}

	if len(defaultTargets) == 0 {
		return defaultTargets
	}

	// Find the row of the first target
	pos := common.GetComponentTypeByID[*squadcore.GridPositionData](
		manager, defaultTargets[0], squadcore.GridPositionComponent,
	)
	if pos == nil {
		return defaultTargets
	}

	// Add units from the row behind
	nextRow := pos.AnchorRow + 1
	if nextRow <= 2 {
		extraTargets := GetUnitsInRow(defenderSquadID, nextRow, manager)
		if len(extraTargets) > 0 {
			return append(defaultTargets, extraTargets...)
		}
	}

	return defaultTargets
}

// cleaveDamageMod applies the -30% damage penalty when Cleave is active
func cleaveDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// Only applies to melee row attackers
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		manager, attackerID, squadcore.TargetRowComponent,
	)
	if targetData != nil && targetData.AttackType == unitdefs.AttackTypeMeleeRow {
		modifiers.DamageMultiplier *= 0.7
	}
}

// Riposte: Counterattacks have no hit penalty (normally -20)
func riposteCounterMod(defenderID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) bool {
	modifiers.HitPenalty = 0 // Override -20 default
	return false             // Don't skip counter
}

// Disruption: Dealing damage reduces target squad's next attack by -15% this round
func disruptionPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, roundState *PerkRoundState,
	manager *common.EntityManager) {
	if damageDealt <= 0 {
		return
	}
	if roundState.DisruptionTargets == nil {
		roundState.DisruptionTargets = make(map[ecs.EntityID]bool)
	}
	// Mark the defender squad as disrupted
	roundState.DisruptionTargets[defenderSquadID] = true

	// Also mark the defender's round state (if they have one)
	defenderState := GetRoundState(defenderSquadID, manager)
	if defenderState != nil {
		if defenderState.DisruptionTargets == nil {
			defenderState.DisruptionTargets = make(map[ecs.EntityID]bool)
		}
		defenderState.DisruptionTargets[attackerSquadID] = true
	}
}

// Guardian Protocol: Redirect 25% damage to adjacent tank
func guardianDamageRedirect(defenderID, defenderSquadID ecs.EntityID,
	damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
	// Find adjacent friendly squads with Guardian Protocol that have a Tank unit
	defenderPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		manager, defenderSquadID, common.PositionComponent,
	)
	if defenderPos == nil {
		return damageAmount, 0, 0
	}

	faction := combatcore.GetSquadFaction(defenderSquadID, manager)
	if faction == 0 {
		return damageAmount, 0, 0
	}

	friendlySquads := combatcore.GetActiveSquadsForFaction(faction, manager)
	for _, friendlyID := range friendlySquads {
		if friendlyID == defenderSquadID {
			continue
		}
		if !HasPerk(friendlyID, "guardian_protocol", manager) {
			continue
		}

		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			continue
		}

		// Check adjacency (Chebyshev distance of 1)
		distance := defenderPos.ChebyshevDistance(friendlyPos)
		if distance > 1 {
			continue
		}

		// Find a Tank unit in the guardian squad to absorb damage
		unitIDs := squadcore.GetUnitIDsInSquad(friendlyID, manager)
		for _, unitID := range unitIDs {
			attr := squadcore.GetAliveUnitAttributes(unitID, manager)
			if attr == nil {
				continue
			}
			entity := manager.FindEntityByID(unitID)
			if entity == nil {
				continue
			}
			roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
			if roleData != nil && roleData.Role == unitdefs.RoleTank {
				// Found a tank to absorb. Redirect 25% of damage.
				guardianDmg := damageAmount / 4
				remainingDmg := damageAmount - guardianDmg
				return remainingDmg, unitID, guardianDmg
			}
		}
	}

	return damageAmount, 0, 0
}

// Overwatch: Skip attack to auto-attack at 75% damage next enemy that moves in range
func overwatchTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	// Overwatch is set via a separate action (skip attack).
	// TurnStart just manages the flag lifecycle.
	// The actual trigger happens in the movement system (not implemented in v1).
	// For now, this is a placeholder that manages state.
	// If the squad has not acted and set overwatch, the flag persists.
}

// Adaptive Armor: -10% damage from same attacker per hit (stacks to 30%, resets per round)
func adaptiveArmorDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// Only activates as defender
	if !HasPerk(defenderSquadID, "adaptive_armor", manager) {
		return
	}

	defenderState := GetRoundState(defenderSquadID, manager)
	if defenderState == nil {
		return
	}

	if defenderState.AttackedBy == nil {
		defenderState.AttackedBy = make(map[ecs.EntityID]int)
	}

	// Get current hit count from this attacker
	hits := defenderState.AttackedBy[attackerSquadID]
	if hits > 3 {
		hits = 3 // Cap at 30%
	}

	if hits > 0 {
		reduction := float64(hits) * 0.10
		modifiers.DamageMultiplier *= (1.0 - reduction)
	}

	// Increment hit counter for next time
	defenderState.AttackedBy[attackerSquadID]++
}

// Bloodlust: Track kills and boost damage
func bloodlustPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, roundState *PerkRoundState,
	manager *common.EntityManager) {
	if wasKill {
		roundState.KillsThisRound++
	}
}

func bloodlustDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	if roundState.KillsThisRound > 0 {
		bonus := 1.0 + float64(roundState.KillsThisRound)*0.15
		modifiers.DamageMultiplier *= bonus
	}
}

// Fortify: +0.05 cover per consecutive turn not moving (max +0.15 after 3 turns, moving resets)
func fortifyTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.MovedThisTurn {
		roundState.TurnsStationary = 0
	} else {
		if roundState.TurnsStationary < 3 {
			roundState.TurnsStationary++
		}
	}
}

func fortifyCoverMod(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown, roundState *PerkRoundState,
	manager *common.EntityManager) {
	defenderSquadID := getSquadIDForUnit(defenderID, manager)
	if !HasPerk(defenderSquadID, "fortify", manager) {
		return
	}

	defenderState := GetRoundState(defenderSquadID, manager)
	if defenderState == nil {
		return
	}

	if defenderState.TurnsStationary > 0 {
		bonus := float64(defenderState.TurnsStationary) * 0.05
		coverBreakdown.TotalReduction += bonus
		if coverBreakdown.TotalReduction > 1.0 {
			coverBreakdown.TotalReduction = 1.0
		}
	}
}

// Precision Strike: Highest-dex unit targets the lowest-HP enemy
func precisionStrikeTargetOverride(attackerID, defenderSquadID ecs.EntityID,
	defaultTargets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// Check if this attacker is the highest-dex DPS unit in its squad
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	if attackerSquadID == 0 {
		return defaultTargets
	}

	// Find the highest-dex DPS unit in the attacker's squad
	unitIDs := squadcore.GetUnitIDsInSquad(attackerSquadID, manager)
	var highestDexID ecs.EntityID
	highestDex := 0
	for _, uid := range unitIDs {
		entity := manager.FindEntityByID(uid)
		if entity == nil {
			continue
		}
		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData == nil || roleData.Role != unitdefs.RoleDPS {
			continue
		}
		if attr.Dexterity > highestDex {
			highestDex = attr.Dexterity
			highestDexID = uid
		}
	}

	// Only override for the highest-dex DPS unit
	if highestDexID != attackerID {
		return defaultTargets
	}

	// Find the lowest-HP alive enemy unit
	enemyUnits := squadcore.GetUnitIDsInSquad(defenderSquadID, manager)
	var lowestHPID ecs.EntityID
	lowestHP := 999999
	for _, uid := range enemyUnits {
		attr := squadcore.GetAliveUnitAttributes(uid, manager)
		if attr == nil {
			continue
		}
		if attr.CurrentHealth < lowestHP {
			lowestHP = attr.CurrentHealth
			lowestHPID = uid
		}
	}

	if lowestHPID != 0 {
		return []ecs.EntityID{lowestHPID}
	}
	return defaultTargets
}

// Resolute: Survive lethal damage at 1 HP (once per unit per battle)
func resoluteTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.RoundStartHP == nil {
		roundState.RoundStartHP = make(map[ecs.EntityID]int)
	}
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](
			manager, uid, common.AttributeComponent,
		)
		if attr != nil && attr.CurrentHealth > 0 {
			roundState.RoundStartHP[uid] = attr.CurrentHealth
		}
	}
}

func resoluteDeathOverride(unitID, squadID ecs.EntityID,
	roundState *PerkRoundState, manager *common.EntityManager) bool {
	if roundState.ResoluteUsed == nil {
		roundState.ResoluteUsed = make(map[ecs.EntityID]bool)
	}
	if roundState.ResoluteUsed[unitID] {
		return false // Already used
	}

	attr := common.GetComponentTypeByID[*common.Attributes](
		manager, unitID, common.AttributeComponent,
	)
	if attr == nil {
		return false
	}

	roundStartHP, ok := roundState.RoundStartHP[unitID]
	if !ok {
		return false
	}

	maxHP := attr.GetMaxHealth()
	if maxHP > 0 && float64(roundStartHP)/float64(maxHP) > 0.5 {
		roundState.ResoluteUsed[unitID] = true
		return true // Prevent death, caller sets HP to 1
	}
	return false
}

// Grudge Bearer: +20% damage vs enemy squads that have damaged this squad (stacks to +40%)
func grudgeBearerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// This is called as a DEFENDER hook -- track who attacked us
	if damageDealt <= 0 {
		return
	}
	if roundState.GrudgeStacks == nil {
		roundState.GrudgeStacks = make(map[ecs.EntityID]int)
	}
	current := roundState.GrudgeStacks[attackerSquadID]
	if current < 2 { // Cap at +40% (2 stacks * 20%)
		roundState.GrudgeStacks[attackerSquadID] = current + 1
	}
}

func grudgeBearerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	if roundState.GrudgeStacks != nil {
		stacks := roundState.GrudgeStacks[defenderSquadID]
		if stacks > 0 {
			bonus := 1.0 + float64(stacks)*0.20
			modifiers.DamageMultiplier *= bonus
		}
	}
}

// Counterpunch: +40% damage if attacked last turn AND did not attack last turn
func counterpunchTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.WasAttackedLastTurn && roundState.DidNotAttackLastTurn {
		roundState.CounterpunchReady = true
	} else {
		roundState.CounterpunchReady = false
	}
}

func counterpunchDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	if roundState.CounterpunchReady {
		modifiers.DamageMultiplier *= 1.4
		roundState.CounterpunchReady = false // Consumed on first attack
	}
}

// Marked for Death: +25% damage to marked target
func markedForDeathDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	// Check if the defender squad is marked by ANY friendly squad
	faction := combatcore.GetSquadFaction(attackerSquadID, manager)
	if faction == 0 {
		return
	}

	friendlySquads := combatcore.GetActiveSquadsForFaction(faction, manager)
	for _, friendlyID := range friendlySquads {
		friendlyState := GetRoundState(friendlyID, manager)
		if friendlyState != nil && friendlyState.MarkedSquad == defenderSquadID {
			modifiers.DamageMultiplier *= 1.25
			// Consume the mark
			friendlyState.MarkedSquad = 0
			return
		}
	}
}

// Deadshot's Patience: +50% damage and +20 accuracy if completely idle last turn
func deadshotTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.WasIdleLastTurn {
		roundState.DeadshotReady = true
	} else {
		roundState.DeadshotReady = false
	}
}

func deadshotDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) {
	if !roundState.DeadshotReady {
		return
	}
	// Verify ranged attack type
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		manager, attackerID, squadcore.TargetRowComponent,
	)
	if targetData == nil {
		return
	}
	if targetData.AttackType == unitdefs.AttackTypeRanged || targetData.AttackType == unitdefs.AttackTypeMagic {
		modifiers.DamageMultiplier *= 1.5
		modifiers.HitPenalty -= 20 // +20 accuracy (reduce penalty)
		roundState.DeadshotReady = false // Consumed
	}
}

// ========================================
// HELPER: GetUnitsInRow (exported for Cleave)
// ========================================

// GetUnitsInRow returns all alive units in a specific row of a squad.
// Exported version of the combattargeting helper for use by perk behaviors.
func GetUnitsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
	var units []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	for col := 0; col <= 2; col++ {
		cellUnits := squadcore.GetUnitIDsAtGridPosition(squadID, row, col, manager)
		for _, unitID := range cellUnits {
			if !seen[unitID] {
				if squadcore.GetAliveUnitAttributes(unitID, manager) != nil {
					units = append(units, unitID)
					seen[unitID] = true
				}
			}
		}
	}

	return units
}
