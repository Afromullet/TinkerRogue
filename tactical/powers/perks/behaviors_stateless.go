// behaviors_stateless.go — Perk implementations that are pure functions of HookContext.
//
// These perks read entity/squad data through the Manager but never read or write
// PerkRoundState shared tracking fields, PerkState, or PerkBattleState.
//
// Adding a new stateless perk? Put it here.
// If it needs state, use behaviors_stateful_round.go or behaviors_stateful_battle.go instead.
package perks

import (
	"math"

	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterPerkHooks("brace_for_impact", &PerkHooks{DefenderCoverMod: braceForImpactCoverMod})
	RegisterPerkHooks("executioners_instinct", &PerkHooks{AttackerDamageMod: executionerDamageMod})
	RegisterPerkHooks("shieldwall_discipline", &PerkHooks{DefenderDamageMod: shieldwallDamageMod})
	RegisterPerkHooks("isolated_predator", &PerkHooks{AttackerDamageMod: isolatedPredatorDamageMod})
	RegisterPerkHooks("vigilance", &PerkHooks{DefenderDamageMod: vigilanceDamageMod})
	RegisterPerkHooks("field_medic", &PerkHooks{TurnStart: fieldMedicTurnStart})
	RegisterPerkHooks("last_line", &PerkHooks{AttackerDamageMod: lastLineDamageMod})
	RegisterPerkHooks("cleave", &PerkHooks{
		TargetOverride:    cleaveTargetOverride,
		AttackerDamageMod: cleaveDamageMod,
	})
	RegisterPerkHooks("riposte", &PerkHooks{CounterMod: riposteCounterMod})
	RegisterPerkHooks("guardian_protocol", &PerkHooks{DamageRedirect: guardianDamageRedirect})
	RegisterPerkHooks("precision_strike", &PerkHooks{TargetOverride: precisionStrikeTargetOverride})
}

// Brace for Impact: +15% cover bonus when defending
func braceForImpactCoverMod(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown) {
	coverBreakdown.TotalReduction += PerkBalance.BraceForImpact.CoverBonus
	if coverBreakdown.TotalReduction > 1.0 {
		coverBreakdown.TotalReduction = 1.0
	}
}

// Executioner's Instinct: +25% crit chance vs squads with any unit below 30% HP
func executionerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
	for _, unitID := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](ctx.Manager, unitID, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}
		maxHP := attr.GetMaxHealth()
		if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < PerkBalance.ExecutionersInstinct.HPThreshold {
			modifiers.CritBonus += PerkBalance.ExecutionersInstinct.CritBonus
			return
		}
	}
}

// Shieldwall Discipline: Per Tank in row 0, -5% damage (max 15%)
func shieldwallDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	tankCount := 0
	unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
	for _, unitID := range unitIDs {
		entity := ctx.Manager.FindEntityByID(unitID)
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
	if tankCount > PerkBalance.ShieldwallDiscipline.MaxTanks {
		tankCount = PerkBalance.ShieldwallDiscipline.MaxTanks
	}
	if tankCount > 0 {
		reduction := float64(tankCount) * PerkBalance.ShieldwallDiscipline.PerTankReduction
		modifiers.DamageMultiplier *= (1.0 - reduction)
	}
}

// Isolated Predator: +25% damage when no friendly squads within 3 tiles
func isolatedPredatorDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		ctx.Manager, ctx.AttackerSquadID, common.PositionComponent,
	)
	if squadPos == nil {
		return
	}
	attackerFaction := combatcore.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if attackerFaction == 0 {
		return
	}
	friendlySquads := combatcore.GetActiveSquadsForFaction(attackerFaction, ctx.Manager)
	for _, friendlyID := range friendlySquads {
		if friendlyID == ctx.AttackerSquadID {
			continue
		}
		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			ctx.Manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			continue
		}
		if squadPos.ChebyshevDistance(friendlyPos) <= PerkBalance.IsolatedPredator.Range {
			return
		}
	}
	modifiers.DamageMultiplier *= PerkBalance.IsolatedPredator.DamageMult
}

// Vigilance: Crits become normal hits
func vigilanceDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	modifiers.SkipCrit = true
}

// Field Medic: At round start, lowest-HP unit heals 10% max HP
func fieldMedicTurnStart(ctx *HookContext) {
	unitIDs := squadcore.GetUnitIDsInSquad(ctx.SquadID, ctx.Manager)
	var lowestID ecs.EntityID
	lowestHP := math.MaxInt

	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](
			ctx.Manager, uid, common.AttributeComponent,
		)
		if attr != nil && attr.CurrentHealth > 0 && attr.CurrentHealth < lowestHP {
			lowestHP = attr.CurrentHealth
			lowestID = uid
		}
	}

	if lowestID != 0 {
		attr := common.GetComponentTypeByID[*common.Attributes](
			ctx.Manager, lowestID, common.AttributeComponent,
		)
		if attr != nil {
			maxHP := attr.GetMaxHealth()
			healAmount := maxHP / PerkBalance.FieldMedic.HealPercent
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

// Last Line: When last friendly squad alive, +20% hit and damage
func lastLineDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	faction := combatcore.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if faction == 0 {
		return
	}
	aliveSquads := combatcore.GetActiveSquadsForFaction(faction, ctx.Manager)
	if len(aliveSquads) == 1 && aliveSquads[0] == ctx.AttackerSquadID {
		modifiers.DamageMultiplier *= PerkBalance.LastLine.DamageMult
		modifiers.HitPenalty -= PerkBalance.LastLine.HitBonus
	}
}

// Cleave: Hit target row + row behind, but -30% damage to ALL targets
func cleaveTargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
	)
	if targetData == nil || targetData.AttackType != unitdefs.AttackTypeMeleeRow {
		return defaultTargets
	}
	if len(defaultTargets) == 0 {
		return defaultTargets
	}
	pos := common.GetComponentTypeByID[*squadcore.GridPositionData](
		ctx.Manager, defaultTargets[0], squadcore.GridPositionComponent,
	)
	if pos == nil {
		return defaultTargets
	}
	nextRow := pos.AnchorRow + 1
	if nextRow <= 2 {
		extraTargets := GetUnitsInRow(ctx.DefenderSquadID, nextRow, ctx.Manager)
		if len(extraTargets) > 0 {
			return append(defaultTargets, extraTargets...)
		}
	}
	return defaultTargets
}

func cleaveDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
	)
	if targetData != nil && targetData.AttackType == unitdefs.AttackTypeMeleeRow {
		modifiers.DamageMultiplier *= PerkBalance.Cleave.DamageMult
	}
}

// Riposte: Counterattacks have no hit penalty (normally -20)
func riposteCounterMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) bool {
	modifiers.HitPenalty = 0
	return false
}

// Guardian Protocol: Redirect 25% damage to adjacent tank
func guardianDamageRedirect(ctx *HookContext) (int, ecs.EntityID, int) {
	damageAmount := ctx.DamageAmount
	defenderSquadID := ctx.SquadID

	defenderPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		ctx.Manager, defenderSquadID, common.PositionComponent,
	)
	if defenderPos == nil {
		return damageAmount, 0, 0
	}
	faction := combatcore.GetSquadFaction(defenderSquadID, ctx.Manager)
	if faction == 0 {
		return damageAmount, 0, 0
	}
	friendlySquads := combatcore.GetActiveSquadsForFaction(faction, ctx.Manager)
	for _, friendlyID := range friendlySquads {
		if friendlyID == defenderSquadID {
			continue
		}
		if !HasPerk(friendlyID, "guardian_protocol", ctx.Manager) {
			continue
		}
		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			ctx.Manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			continue
		}
		if defenderPos.ChebyshevDistance(friendlyPos) > 1 {
			continue
		}
		unitIDs := squadcore.GetUnitIDsInSquad(friendlyID, ctx.Manager)
		for _, unitID := range unitIDs {
			attr := squadcore.GetAliveUnitAttributes(unitID, ctx.Manager)
			if attr == nil {
				continue
			}
			entity := ctx.Manager.FindEntityByID(unitID)
			if entity == nil {
				continue
			}
			roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
			if roleData != nil && roleData.Role == unitdefs.RoleTank {
				guardianDmg := damageAmount / PerkBalance.GuardianProtocol.RedirectFraction
				remainingDmg := damageAmount - guardianDmg
				return remainingDmg, unitID, guardianDmg
			}
		}
	}
	return damageAmount, 0, 0
}

// Precision Strike: Highest-dex DPS unit targets the lowest-HP enemy
func precisionStrikeTargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
	attackerSquadID := getSquadIDForUnit(ctx.AttackerID, ctx.Manager)
	if attackerSquadID == 0 {
		return defaultTargets
	}

	unitIDs := squadcore.GetUnitIDsInSquad(attackerSquadID, ctx.Manager)
	var highestDexID ecs.EntityID
	highestDex := 0
	for _, uid := range unitIDs {
		entity := ctx.Manager.FindEntityByID(uid)
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

	if highestDexID != ctx.AttackerID {
		return defaultTargets
	}

	enemyUnits := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
	var lowestHPID ecs.EntityID
	lowestHP := math.MaxInt
	for _, uid := range enemyUnits {
		attr := squadcore.GetAliveUnitAttributes(uid, ctx.Manager)
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

// GetUnitsInRow returns all alive units in a specific row of a squad.
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
