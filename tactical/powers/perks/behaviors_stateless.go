// behaviors_stateless.go — Perk implementations that are pure functions of HookContext.
//
// These perks read entity/squad data through the Manager but never read or write
// PerkRoundState shared tracking fields, PerkState, or PerkBattleState.
//
// Adding a new stateless perk? Put it here.
// If it needs state, use behaviors_stateful_round.go or behaviors_stateful_battle.go instead.
package perks

import (
	"fmt"
	"math"

	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterPerkBehavior(&BraceForImpactBehavior{})
	RegisterPerkBehavior(&ExecutionersInstinctBehavior{})
	RegisterPerkBehavior(&ShieldwallDisciplineBehavior{})
	RegisterPerkBehavior(&IsolatedPredatorBehavior{})
	RegisterPerkBehavior(&VigilanceBehavior{})
	RegisterPerkBehavior(&FieldMedicBehavior{})
	RegisterPerkBehavior(&LastLineBehavior{})
	RegisterPerkBehavior(&CleaveBehavior{})
	RegisterPerkBehavior(&RiposteBehavior{})
	RegisterPerkBehavior(&GuardianProtocolBehavior{})
	RegisterPerkBehavior(&PrecisionStrikeBehavior{})
}

// ========================================
// Brace for Impact: +15% cover bonus when defending
// ========================================

type BraceForImpactBehavior struct{ BasePerkBehavior }

func (b *BraceForImpactBehavior) PerkID() PerkID { return PerkBraceForImpact }

func (b *BraceForImpactBehavior) DefenderCoverMod(ctx *HookContext, coverBreakdown *combattypes.CoverBreakdown) {
	coverBreakdown.TotalReduction += PerkBalance.BraceForImpact.CoverBonus
	if coverBreakdown.TotalReduction > 1.0 {
		coverBreakdown.TotalReduction = 1.0
	}
	logPerkActivation(PerkBraceForImpact, ctx.DefenderSquadID, "cover bonus applied")
}

// ========================================
// Executioner's Instinct: +25% crit chance vs squads with any unit below 30% HP
// ========================================

type ExecutionersInstinctBehavior struct{ BasePerkBehavior }

func (b *ExecutionersInstinctBehavior) PerkID() PerkID { return PerkExecutionersInstinct }

func (b *ExecutionersInstinctBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
	for _, unitID := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](ctx.Manager, unitID, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}
		maxHP := attr.GetMaxHealth()
		if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < PerkBalance.ExecutionersInstinct.HPThreshold {
			modifiers.CritBonus += PerkBalance.ExecutionersInstinct.CritBonus
			logPerkActivation(PerkExecutionersInstinct, ctx.AttackerSquadID, "crit bonus vs wounded target")
			return
		}
	}
}

// ========================================
// Shieldwall Discipline: Per Tank in row 0, -5% damage (max 15%)
// ========================================

type ShieldwallDisciplineBehavior struct{ BasePerkBehavior }

func (b *ShieldwallDisciplineBehavior) PerkID() PerkID { return PerkShieldwallDiscipline }

func (b *ShieldwallDisciplineBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
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
		logPerkActivation(PerkShieldwallDiscipline, ctx.DefenderSquadID, fmt.Sprintf("-%d%% damage from %d front-row tanks", int(reduction*100), tankCount))
	}
}

// ========================================
// Isolated Predator: +25% damage when no friendly squads within 3 tiles
// ========================================

type IsolatedPredatorBehavior struct{ BasePerkBehavior }

func (b *IsolatedPredatorBehavior) PerkID() PerkID { return PerkIsolatedPredator }

func (b *IsolatedPredatorBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		ctx.Manager, ctx.AttackerSquadID, common.PositionComponent,
	)
	if squadPos == nil {
		return
	}
	attackerFaction := combatstate.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if attackerFaction == 0 {
		return
	}
	friendlySquads := combatstate.GetActiveSquadsForFaction(attackerFaction, ctx.Manager)
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
	logPerkActivation(PerkIsolatedPredator, ctx.AttackerSquadID, "damage bonus from isolation")
}

// ========================================
// Vigilance: Crits become normal hits
// ========================================

type VigilanceBehavior struct{ BasePerkBehavior }

func (b *VigilanceBehavior) PerkID() PerkID { return PerkVigilance }

func (b *VigilanceBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	modifiers.SkipCrit = true
	logPerkActivation(PerkVigilance, ctx.DefenderSquadID, "critical hit negated")
}

// ========================================
// Field Medic: At round start, lowest-HP unit heals 10% max HP
// ========================================

type FieldMedicBehavior struct{ BasePerkBehavior }

func (b *FieldMedicBehavior) PerkID() PerkID { return PerkFieldMedic }

func (b *FieldMedicBehavior) TurnStart(ctx *HookContext) {
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
			healAmount := maxHP / PerkBalance.FieldMedic.HealDivisor
			if healAmount < 1 {
				healAmount = 1
			}
			attr.CurrentHealth += healAmount
			if attr.CurrentHealth > maxHP {
				attr.CurrentHealth = maxHP
			}
			logPerkActivation(PerkFieldMedic, ctx.SquadID, fmt.Sprintf("healed unit for %d HP", healAmount))
		}
	}
}

// ========================================
// Last Line: When last friendly squad alive, +20% hit and damage
// ========================================

type LastLineBehavior struct{ BasePerkBehavior }

func (b *LastLineBehavior) PerkID() PerkID { return PerkLastLine }

func (b *LastLineBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	faction := combatstate.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if faction == 0 {
		return
	}
	aliveSquads := combatstate.GetActiveSquadsForFaction(faction, ctx.Manager)
	if len(aliveSquads) == 1 && aliveSquads[0] == ctx.AttackerSquadID {
		modifiers.DamageMultiplier *= PerkBalance.LastLine.DamageMult
		modifiers.HitPenalty -= PerkBalance.LastLine.HitBonus
		logPerkActivation(PerkLastLine, ctx.AttackerSquadID, "last squad standing bonus")
	}
}

// ========================================
// Cleave: Hit target row + row behind, but -30% damage to ALL targets
// ========================================

type CleaveBehavior struct{ BasePerkBehavior }

func (b *CleaveBehavior) PerkID() PerkID { return PerkCleave }

func (b *CleaveBehavior) TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
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
			logPerkActivation(PerkCleave, ctx.AttackerSquadID, fmt.Sprintf("cleaving %d extra targets in row %d", len(extraTargets), nextRow))
			return append(defaultTargets, extraTargets...)
		}
	}
	return defaultTargets
}

func (b *CleaveBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
	)
	if targetData != nil && targetData.AttackType == unitdefs.AttackTypeMeleeRow {
		modifiers.DamageMultiplier *= PerkBalance.Cleave.DamageMult
	}
}

// ========================================
// Riposte: Counterattacks have no hit penalty (normally -20)
// ========================================

type RiposteBehavior struct{ BasePerkBehavior }

func (b *RiposteBehavior) PerkID() PerkID { return PerkRiposte }

func (b *RiposteBehavior) CounterMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) bool {
	modifiers.HitPenalty = 0
	logPerkActivation(PerkRiposte, ctx.DefenderSquadID, "counter hit penalty removed")
	return false
}

// ========================================
// Guardian Protocol: Redirect 25% damage to adjacent tank
// ========================================

type GuardianProtocolBehavior struct{ BasePerkBehavior }

func (b *GuardianProtocolBehavior) PerkID() PerkID { return PerkGuardianProtocol }

func (b *GuardianProtocolBehavior) DamageRedirect(ctx *HookContext) (int, ecs.EntityID, int) {
	damageAmount := ctx.DamageAmount
	defenderSquadID := ctx.SquadID

	defenderPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		ctx.Manager, defenderSquadID, common.PositionComponent,
	)
	if defenderPos == nil {
		return damageAmount, 0, 0
	}
	faction := combatstate.GetSquadFaction(defenderSquadID, ctx.Manager)
	if faction == 0 {
		return damageAmount, 0, 0
	}
	friendlySquads := combatstate.GetActiveSquadsForFaction(faction, ctx.Manager)
	for _, friendlyID := range friendlySquads {
		if friendlyID == defenderSquadID {
			continue
		}
		if !HasPerk(friendlyID, PerkGuardianProtocol, ctx.Manager) {
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
				logPerkActivation(PerkGuardianProtocol, defenderSquadID, fmt.Sprintf("tank absorbs %d damage", guardianDmg))
				return remainingDmg, unitID, guardianDmg
			}
		}
	}
	return damageAmount, 0, 0
}

// ========================================
// Precision Strike: Highest-dex DPS unit targets the lowest-HP enemy
// ========================================

type PrecisionStrikeBehavior struct{ BasePerkBehavior }

func (b *PrecisionStrikeBehavior) PerkID() PerkID { return PerkPrecisionStrike }

func (b *PrecisionStrikeBehavior) TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
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
	logPerkActivation(PerkPrecisionStrike, ctx.AttackerSquadID, "targeting lowest-HP enemy")

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
