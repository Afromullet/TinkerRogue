// behaviors_stateless.go — stateless perk behaviors.
//
// These behaviors are pure functions of HookContext + ECS queries. They read
// nothing from PerkRoundState maps and write nothing. The taxonomy split (this
// file vs behaviors_per_round.go vs behaviors_per_battle.go) is informational
// — nothing in the dispatch or registration path depends on which file a
// behavior lives in. Pick the section that best matches the behavior's state
// needs when adding a new perk.
//
// Shared unit-query helpers live in unithelpers.go; shared squad-iteration
// helpers (ForEachFriendlySquad, ForEachFriendlySquadWithinRange) live in
// queries.go. Prefer those over open-coded loops to keep behaviors short.
package perks

import (
	"fmt"

	"game_main/core/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"

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

// Brace for Impact: +15% cover bonus when defending.

type BraceForImpactBehavior struct{ BasePerkBehavior }

func (b *BraceForImpactBehavior) PerkID() PerkID { return PerkBraceForImpact }

func (b *BraceForImpactBehavior) DefenderCoverMod(ctx *HookContext, coverBreakdown *combattypes.CoverBreakdown) {
	coverBreakdown.TotalReduction += PerkBalance.BraceForImpact.CoverBonus
	if coverBreakdown.TotalReduction > 1.0 {
		coverBreakdown.TotalReduction = 1.0
	}
	ctx.LogPerk(PerkBraceForImpact, ctx.DefenderSquadID, "cover bonus applied")
}

// Executioner's Instinct: +25% crit chance vs squads with any unit below 30% HP.

type ExecutionersInstinctBehavior struct{ BasePerkBehavior }

func (b *ExecutionersInstinctBehavior) PerkID() PerkID { return PerkExecutionersInstinct }

func (b *ExecutionersInstinctBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	if HasWoundedUnit(ctx.DefenderSquadID, PerkBalance.ExecutionersInstinct.HPThreshold, ctx.Manager) {
		modifiers.CritBonus += PerkBalance.ExecutionersInstinct.CritBonus
		ctx.LogPerk(PerkExecutionersInstinct, ctx.AttackerSquadID, "crit bonus vs wounded target")
	}
}

// Shieldwall Discipline: Per Tank in row 0, -5% damage (max 15%).

type ShieldwallDisciplineBehavior struct{ BasePerkBehavior }

func (b *ShieldwallDisciplineBehavior) PerkID() PerkID { return PerkShieldwallDiscipline }

func (b *ShieldwallDisciplineBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	tankCount := CountTanksInRow(ctx.DefenderSquadID, 0, ctx.Manager)
	if tankCount > PerkBalance.ShieldwallDiscipline.MaxTanks {
		tankCount = PerkBalance.ShieldwallDiscipline.MaxTanks
	}
	if tankCount > 0 {
		reduction := float64(tankCount) * PerkBalance.ShieldwallDiscipline.PerTankReduction
		modifiers.DamageMultiplier *= (1.0 - reduction)
		ctx.LogPerk(PerkShieldwallDiscipline, ctx.DefenderSquadID, fmt.Sprintf("-%d%% damage from %d front-row tanks", int(reduction*100), tankCount))
	}
}

// Isolated Predator: +25% damage when no friendly squads within 3 tiles.

type IsolatedPredatorBehavior struct{ BasePerkBehavior }

func (b *IsolatedPredatorBehavior) PerkID() PerkID { return PerkIsolatedPredator }

func (b *IsolatedPredatorBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	nearby := false
	ForEachFriendlySquadWithinRange(ctx.AttackerSquadID, PerkBalance.IsolatedPredator.Range, ctx.Manager,
		func(_ ecs.EntityID) bool {
			nearby = true
			return false
		})
	if nearby {
		return
	}
	modifiers.DamageMultiplier *= PerkBalance.IsolatedPredator.DamageMult
	ctx.LogPerk(PerkIsolatedPredator, ctx.AttackerSquadID, "damage bonus from isolation")
}

// Vigilance: Crits become normal hits.

type VigilanceBehavior struct{ BasePerkBehavior }

func (b *VigilanceBehavior) PerkID() PerkID { return PerkVigilance }

func (b *VigilanceBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	modifiers.SkipCrit = true
	ctx.LogPerk(PerkVigilance, ctx.DefenderSquadID, "critical hit negated")
}

// Field Medic: At turn start, lowest-HP unit heals.

type FieldMedicBehavior struct{ BasePerkBehavior }

func (b *FieldMedicBehavior) PerkID() PerkID { return PerkFieldMedic }

func (b *FieldMedicBehavior) TurnStart(ctx *HookContext) {
	lowestID := FindLowestHPUnit(ctx.SquadID, ctx.Manager)
	if lowestID == 0 {
		return
	}
	attr := common.GetComponentTypeByID[*common.Attributes](
		ctx.Manager, lowestID, common.AttributeComponent,
	)
	if attr == nil {
		return
	}
	maxHP := attr.GetMaxHealth()
	healAmount := maxHP / PerkBalance.FieldMedic.HealDivisor
	if healAmount < 1 {
		healAmount = 1
	}
	attr.CurrentHealth += healAmount
	if attr.CurrentHealth > maxHP {
		attr.CurrentHealth = maxHP
	}
	ctx.LogPerk(PerkFieldMedic, ctx.SquadID, fmt.Sprintf("healed unit for %d HP", healAmount))
}

// Last Line: When last friendly squad alive, +20% hit and damage.

type LastLineBehavior struct{ BasePerkBehavior }

func (b *LastLineBehavior) PerkID() PerkID { return PerkLastLine }

func (b *LastLineBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	faction := combatstate.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if faction == 0 {
		return
	}
	hasAlly := false
	ForEachFriendlySquad(ctx.AttackerSquadID, ctx.Manager, func(_ ecs.EntityID) bool {
		hasAlly = true
		return false
	})
	if hasAlly {
		return
	}
	modifiers.DamageMultiplier *= PerkBalance.LastLine.DamageMult
	modifiers.HitModifier -= PerkBalance.LastLine.HitBonus
	ctx.LogPerk(PerkLastLine, ctx.AttackerSquadID, "last squad standing bonus")
}

// Cleave: Hit target row + row behind, but -30% damage to ALL targets.

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
			ctx.LogPerk(PerkCleave, ctx.AttackerSquadID, fmt.Sprintf("cleaving %d extra targets in row %d", len(extraTargets), nextRow))
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

// Riposte: Counterattacks have no hit penalty (normally -20).

type RiposteBehavior struct{ BasePerkBehavior }

func (b *RiposteBehavior) PerkID() PerkID { return PerkRiposte }

func (b *RiposteBehavior) CounterMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) bool {
	modifiers.HitModifier = 0
	ctx.LogPerk(PerkRiposte, ctx.DefenderSquadID, "counter hit penalty removed")
	return false
}

// Guardian Protocol: Redirect 25% damage to adjacent tank.

type GuardianProtocolBehavior struct{ BasePerkBehavior }

func (b *GuardianProtocolBehavior) PerkID() PerkID { return PerkGuardianProtocol }

func (b *GuardianProtocolBehavior) DamageRedirect(ctx *HookContext) (int, ecs.EntityID, int) {
	damageAmount := ctx.DamageAmount
	defenderSquadID := ctx.SquadID

	reducedDmg := damageAmount
	var redirectTarget ecs.EntityID
	redirectAmt := 0
	ForEachFriendlySquadWithinRange(defenderSquadID, 1, ctx.Manager, func(friendlyID ecs.EntityID) bool {
		if !HasPerk(friendlyID, PerkGuardianProtocol, ctx.Manager) {
			return true
		}
		tankID := FindFirstTankInSquad(friendlyID, ctx.Manager)
		if tankID == 0 {
			return true
		}
		guardianDmg := damageAmount / PerkBalance.GuardianProtocol.RedirectFraction
		reducedDmg = damageAmount - guardianDmg
		redirectTarget = tankID
		redirectAmt = guardianDmg
		ctx.LogPerk(PerkGuardianProtocol, defenderSquadID, fmt.Sprintf("tank absorbs %d damage", guardianDmg))
		return false
	})
	return reducedDmg, redirectTarget, redirectAmt
}

// Precision Strike: Highest-dex DPS unit targets the lowest-HP enemy.

type PrecisionStrikeBehavior struct{ BasePerkBehavior }

func (b *PrecisionStrikeBehavior) PerkID() PerkID { return PerkPrecisionStrike }

func (b *PrecisionStrikeBehavior) TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
	attackerSquadID := getSquadIDForUnit(ctx.AttackerID, ctx.Manager)
	if attackerSquadID == 0 {
		return defaultTargets
	}
	if FindHighestDexUnitByRole(attackerSquadID, unitdefs.RoleDPS, ctx.Manager) != ctx.AttackerID {
		return defaultTargets
	}
	ctx.LogPerk(PerkPrecisionStrike, ctx.AttackerSquadID, "targeting lowest-HP enemy")

	if lowestHPID := FindLowestHPUnit(ctx.DefenderSquadID, ctx.Manager); lowestHPID != 0 {
		return []ecs.EntityID{lowestHPID}
	}
	return defaultTargets
}
