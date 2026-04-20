// behaviors.go — all perk behavior implementations.
//
// Perks are grouped below by state lifecycle. The taxonomy is informational —
// nothing in the dispatch or registration path depends on which section a
// behavior lives in. Pick the section that best matches the behavior's state
// needs when adding a new perk.
//
//   - Stateless:             pure functions of HookContext + ECS queries.
//     Read nothing from PerkRoundState maps; write nothing.
//
//   - Per-round stateful:    read/write PerkRoundState (shared tracking fields
//     or the per-round PerkState map). State clears at
//     round boundaries via ResetPerkRoundStateRound.
//
//   - Per-battle stateful:   read/write PerkBattleState. Persists across rounds
//     until CleanupRoundState is called at combat end.
//
// Shared unit-query helpers live in unithelpers.go; shared squad-iteration
// helpers (ForEachFriendlySquad) live in queries.go. Prefer those over
// open-coded loops to keep behaviors short.
package perks

import (
	"fmt"

	"game_main/core/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

func init() {
	// Stateless
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

	// Per-round stateful
	RegisterPerkBehavior(&RecklessAssaultBehavior{})
	RegisterPerkBehavior(&StalwartBehavior{})
	RegisterPerkBehavior(&FortifyBehavior{})
	RegisterPerkBehavior(&CounterpunchBehavior{})
	RegisterPerkBehavior(&DeadshotsPatienceBehavior{})
	RegisterPerkBehavior(&AdaptiveArmorBehavior{})
	RegisterPerkBehavior(&BloodlustBehavior{})

	// Per-battle stateful
	RegisterPerkBehavior(&OpeningSalvoBehavior{})
	RegisterPerkBehavior(&ResoluteBehavior{})
	RegisterPerkBehavior(&GrudgeBearerBehavior{})
}

// ========================================
// STATELESS PERKS
// ========================================

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
	squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		ctx.Manager, ctx.AttackerSquadID, common.PositionComponent,
	)
	if squadPos == nil {
		return
	}
	nearby := false
	ForEachFriendlySquad(ctx.AttackerSquadID, ctx.Manager, func(friendlyID ecs.EntityID) bool {
		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			ctx.Manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			return true
		}
		if squadPos.ChebyshevDistance(friendlyPos) <= PerkBalance.IsolatedPredator.Range {
			nearby = true
			return false
		}
		return true
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
	modifiers.HitPenalty -= PerkBalance.LastLine.HitBonus
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
	modifiers.HitPenalty = 0
	ctx.LogPerk(PerkRiposte, ctx.DefenderSquadID, "counter hit penalty removed")
	return false
}

// Guardian Protocol: Redirect 25% damage to adjacent tank.

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

	reducedDmg := damageAmount
	var redirectTarget ecs.EntityID
	redirectAmt := 0
	ForEachFriendlySquad(defenderSquadID, ctx.Manager, func(friendlyID ecs.EntityID) bool {
		if !HasPerk(friendlyID, PerkGuardianProtocol, ctx.Manager) {
			return true
		}
		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			ctx.Manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			return true
		}
		if defenderPos.ChebyshevDistance(friendlyPos) > 1 {
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

// ========================================
// PER-ROUND STATEFUL PERKS
// ========================================

// Reckless Assault: +damage on attack, becomes vulnerable to incoming damage.

type RecklessAssaultBehavior struct{ BasePerkBehavior }

func (b *RecklessAssaultBehavior) PerkID() PerkID { return PerkRecklessAssault }

// TurnStart: resets vulnerability at the start of each turn.
// State: writes RecklessAssaultState via SetPerkState (per-round).
func (b *RecklessAssaultBehavior) TurnStart(ctx *HookContext) {
	SetPerkState(ctx.RoundState, PerkRecklessAssault, &RecklessAssaultState{Vulnerable: false})
}

// AttackerDamageMod: boosts outgoing damage and sets vulnerability.
// State: writes RecklessAssaultState via SetPerkState (per-round).
func (b *RecklessAssaultBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	modifiers.DamageMultiplier *= PerkBalance.RecklessAssault.AttackerMult
	SetPerkState(ctx.RoundState, PerkRecklessAssault, &RecklessAssaultState{Vulnerable: true})
	ctx.LogPerk(PerkRecklessAssault, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage, now vulnerable", int((PerkBalance.RecklessAssault.AttackerMult-1)*100)))
}

// DefenderDamageMod: increases incoming damage when vulnerable.
// State: reads RecklessAssaultState via GetPerkState (per-round).
func (b *RecklessAssaultBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetPerkState[*RecklessAssaultState](ctx.RoundState, PerkRecklessAssault)
	if state != nil && state.Vulnerable {
		modifiers.DamageMultiplier *= PerkBalance.RecklessAssault.DefenderMult
		ctx.LogPerk(PerkRecklessAssault, ctx.DefenderSquadID, fmt.Sprintf("+%d%% incoming damage (vulnerable)", int((PerkBalance.RecklessAssault.DefenderMult-1)*100)))
	}
}

// Stalwart: Full-damage counters if the squad did not move.

type StalwartBehavior struct{ BasePerkBehavior }

func (b *StalwartBehavior) PerkID() PerkID { return PerkStalwart }

// CounterMod: full-damage counters if the squad did not move.
// State: reads PerkRoundState.MovedThisTurn (shared tracking).
func (b *StalwartBehavior) CounterMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) bool {
	if !ctx.RoundState.MovedThisTurn {
		modifiers.DamageMultiplier = 1.0 // Override 0.5 default
		ctx.LogPerk(PerkStalwart, ctx.DefenderSquadID, "full-damage counterattack")
	}
	return false
}

// Fortify: Accumulates TurnsStationary, provides cover bonus.

type FortifyBehavior struct{ BasePerkBehavior }

func (b *FortifyBehavior) PerkID() PerkID { return PerkFortify }

// TurnStart: increments stationary counter if squad didn't move.
// State: reads MovedThisTurn, writes TurnsStationary (shared tracking).
func (b *FortifyBehavior) TurnStart(ctx *HookContext) {
	if ctx.RoundState.MovedThisTurn {
		ctx.RoundState.TurnsStationary = 0
	} else {
		ctx.IncrementTurnsStationary(PerkBalance.Fortify.MaxStationaryTurns)
	}
}

// DefenderCoverMod: adds cover based on consecutive stationary turns.
// State: reads PerkRoundState.TurnsStationary (shared tracking).
func (b *FortifyBehavior) DefenderCoverMod(ctx *HookContext, coverBreakdown *combattypes.CoverBreakdown) {
	stationary := ctx.RoundState.TurnsStationary
	if stationary > 0 {
		bonus := float64(stationary) * PerkBalance.Fortify.PerTurnCoverBonus
		coverBreakdown.TotalReduction += bonus
		if coverBreakdown.TotalReduction > 1.0 {
			coverBreakdown.TotalReduction = 1.0
		}
		ctx.LogPerk(PerkFortify, ctx.DefenderSquadID, fmt.Sprintf("+%d%% cover (%d turns stationary)", int(bonus*100), stationary))
	}
}

// Counterpunch: Armed if attacked last turn and didn't attack. +40% damage.

type CounterpunchBehavior struct{ BasePerkBehavior }

func (b *CounterpunchBehavior) PerkID() PerkID { return PerkCounterpunch }

// TurnStart: arms the counterpunch bonus based on last-turn snapshots.
// State: reads WasAttackedLastTurn, DidNotAttackLastTurn (shared snapshots);
//
//	writes CounterpunchState via SetPerkState (per-round).
func (b *CounterpunchBehavior) TurnStart(ctx *HookContext) {
	ready := ctx.RoundState.WasAttackedLastTurn && ctx.RoundState.DidNotAttackLastTurn
	SetPerkState(ctx.RoundState, PerkCounterpunch, &CounterpunchState{Ready: ready})
}

// AttackerDamageMod: applies +40% damage when armed.
// State: reads/writes CounterpunchState via GetPerkState (per-round).
func (b *CounterpunchBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetPerkState[*CounterpunchState](ctx.RoundState, PerkCounterpunch)
	if state != nil && state.Ready {
		modifiers.DamageMultiplier *= PerkBalance.Counterpunch.DamageMult
		state.Ready = false
		ctx.LogPerk(PerkCounterpunch, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (retaliating)", int((PerkBalance.Counterpunch.DamageMult-1)*100)))
	}
}

// Deadshot's Patience: Armed if idle last turn. +50% damage, +20 accuracy for ranged/magic.

type DeadshotsPatienceBehavior struct{ BasePerkBehavior }

func (b *DeadshotsPatienceBehavior) PerkID() PerkID { return PerkDeadshotsPatience }

// TurnStart: arms the deadshot bonus if the squad was idle last turn.
// State: reads WasIdleLastTurn (shared snapshot);
//
//	writes DeadshotState via SetPerkState (per-round).
func (b *DeadshotsPatienceBehavior) TurnStart(ctx *HookContext) {
	ready := ctx.RoundState.WasIdleLastTurn
	SetPerkState(ctx.RoundState, PerkDeadshotsPatience, &DeadshotState{Ready: ready})
}

// AttackerDamageMod: applies +50% damage and +20 accuracy for ranged/magic attacks when armed.
// State: reads/writes DeadshotState via GetPerkState (per-round).
func (b *DeadshotsPatienceBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetPerkState[*DeadshotState](ctx.RoundState, PerkDeadshotsPatience)
	if state == nil || !state.Ready {
		return
	}
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
	)
	if targetData == nil {
		return
	}
	if targetData.AttackType == unitdefs.AttackTypeRanged || targetData.AttackType == unitdefs.AttackTypeMagic {
		modifiers.DamageMultiplier *= PerkBalance.DeadshotsPatience.DamageMult
		modifiers.HitPenalty -= PerkBalance.DeadshotsPatience.AccuracyBonus
		state.Ready = false
		ctx.LogPerk(PerkDeadshotsPatience, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage, +%d accuracy (patient shot)", int((PerkBalance.DeadshotsPatience.DamageMult-1)*100), PerkBalance.DeadshotsPatience.AccuracyBonus))
	}
}

// Adaptive Armor: Reduces damage from repeated attackers.

type AdaptiveArmorBehavior struct{ BasePerkBehavior }

func (b *AdaptiveArmorBehavior) PerkID() PerkID { return PerkAdaptiveArmor }

// DefenderDamageMod: reduces damage from repeated attackers.
// State: reads/writes AdaptiveArmorState via GetPerkState/SetPerkState (per-round).
func (b *AdaptiveArmorBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetOrInitPerkState(ctx.RoundState, PerkAdaptiveArmor, func() *AdaptiveArmorState {
		return &AdaptiveArmorState{AttackedBy: make(map[ecs.EntityID]int)}
	})
	hits := state.AttackedBy[ctx.AttackerSquadID]
	if hits > PerkBalance.AdaptiveArmor.MaxHits {
		hits = PerkBalance.AdaptiveArmor.MaxHits
	}
	if hits > 0 {
		reduction := float64(hits) * PerkBalance.AdaptiveArmor.PerHitReduction
		modifiers.DamageMultiplier *= (1.0 - reduction)
		ctx.LogPerk(PerkAdaptiveArmor, ctx.DefenderSquadID, fmt.Sprintf("-%d%% damage (hit %d from same attacker)", int(reduction*100), hits))
	}
	state.AttackedBy[ctx.AttackerSquadID]++
}

// Bloodlust: +damage based on kills this round.

type BloodlustBehavior struct{ BasePerkBehavior }

func (b *BloodlustBehavior) PerkID() PerkID { return PerkBloodlust }

// AttackerPostDamage: tracks kills this round.
// State: writes BloodlustState via SetPerkState (per-round).
func (b *BloodlustBehavior) AttackerPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if wasKill {
		state := GetOrInitPerkState(ctx.RoundState, PerkBloodlust, func() *BloodlustState {
			return &BloodlustState{}
		})
		state.KillsThisRound++
	}
}

// AttackerDamageMod: applies bonus damage based on kills this round.
// State: reads BloodlustState via GetPerkState (per-round).
func (b *BloodlustBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetPerkState[*BloodlustState](ctx.RoundState, PerkBloodlust)
	if state != nil && state.KillsThisRound > 0 {
		bonus := 1.0 + float64(state.KillsThisRound)*PerkBalance.Bloodlust.PerKillBonus
		modifiers.DamageMultiplier *= bonus
		ctx.LogPerk(PerkBloodlust, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (%d kills)", int((bonus-1)*100), state.KillsThisRound))
	}
}

// ========================================
// PER-BATTLE STATEFUL PERKS
// ========================================

// Opening Salvo: +35% damage on first attack of combat.

type OpeningSalvoBehavior struct{ BasePerkBehavior }

func (b *OpeningSalvoBehavior) PerkID() PerkID { return PerkOpeningSalvo }

// AttackerDamageMod: +35% damage on the squad's first attack of the combat.
// State: reads/writes OpeningSalvoState via GetBattleState/SetBattleState (per-battle).
func (b *OpeningSalvoBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	state := GetBattleState[*OpeningSalvoState](ctx.RoundState, PerkOpeningSalvo)
	if state != nil && state.HasAttackedThisCombat {
		return
	}
	modifiers.DamageMultiplier *= PerkBalance.OpeningSalvo.DamageMult
	SetBattleState(ctx.RoundState, PerkOpeningSalvo, &OpeningSalvoState{HasAttackedThisCombat: true})
	ctx.LogPerk(PerkOpeningSalvo, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (opening attack)", int((PerkBalance.OpeningSalvo.DamageMult-1)*100)))
}

// Resolute: Prevents death once per combat if unit had >50% HP at round start.

type ResoluteBehavior struct{ BasePerkBehavior }

func (b *ResoluteBehavior) PerkID() PerkID { return PerkResolute }

// TurnStart: snapshots current HP for the resolute death-save check.
// State: writes ResoluteState.RoundStartHP via GetBattleState/SetBattleState (per-battle).
func (b *ResoluteBehavior) TurnStart(ctx *HookContext) {
	state := GetOrInitBattleState(ctx.RoundState, PerkResolute, func() *ResoluteState {
		return &ResoluteState{
			Used:         make(map[ecs.EntityID]bool),
			RoundStartHP: make(map[ecs.EntityID]int),
		}
	})
	for _, uid := range squadcore.GetUnitIDsInSquad(ctx.SquadID, ctx.Manager) {
		attr := squadcore.GetAliveUnitAttributes(uid, ctx.Manager)
		if attr != nil {
			state.RoundStartHP[uid] = attr.CurrentHealth
		}
	}
}

// DeathOverride: prevents death if the unit had >50% HP at round start (once per battle).
// State: reads/writes ResoluteState via GetBattleState (per-battle).
func (b *ResoluteBehavior) DeathOverride(ctx *HookContext) bool {
	state := GetBattleState[*ResoluteState](ctx.RoundState, PerkResolute)
	if state == nil {
		return false
	}
	if state.Used[ctx.UnitID] {
		return false
	}
	attr := common.GetComponentTypeByID[*common.Attributes](
		ctx.Manager, ctx.UnitID, common.AttributeComponent,
	)
	if attr == nil {
		return false
	}
	roundStartHP, ok := state.RoundStartHP[ctx.UnitID]
	if !ok {
		return false
	}
	maxHP := attr.GetMaxHealth()
	if maxHP > 0 && float64(roundStartHP)/float64(maxHP) > PerkBalance.Resolute.HPThreshold {
		state.Used[ctx.UnitID] = true
		ctx.LogPerk(PerkResolute, ctx.SquadID, "unit survives lethal damage at 1 HP")
		return true
	}
	return false
}

// Grudge Bearer: +damage per grudge stack from enemy squad damage.

type GrudgeBearerBehavior struct{ BasePerkBehavior }

func (b *GrudgeBearerBehavior) PerkID() PerkID { return PerkGrudgeBearer }

// DefenderPostDamage: tracks damage received from enemy squads.
// State: writes GrudgeBearerState.Stacks via GetBattleState/SetBattleState (per-battle).
func (b *GrudgeBearerBehavior) DefenderPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if damageDealt <= 0 {
		return
	}
	state := GetOrInitBattleState(ctx.RoundState, PerkGrudgeBearer, func() *GrudgeBearerState {
		return &GrudgeBearerState{Stacks: make(map[ecs.EntityID]int)}
	})
	current := state.Stacks[ctx.AttackerSquadID]
	if current < PerkBalance.GrudgeBearer.MaxStacks {
		state.Stacks[ctx.AttackerSquadID] = current + 1
	}
}

// AttackerDamageMod: applies +20% damage per grudge stack (max +40%).
// State: reads GrudgeBearerState.Stacks via GetBattleState (per-battle).
func (b *GrudgeBearerBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetBattleState[*GrudgeBearerState](ctx.RoundState, PerkGrudgeBearer)
	if state != nil {
		stacks := state.Stacks[ctx.DefenderSquadID]
		if stacks > 0 {
			bonus := 1.0 + float64(stacks)*PerkBalance.GrudgeBearer.PerStackBonus
			modifiers.DamageMultiplier *= bonus
			ctx.LogPerk(PerkGrudgeBearer, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (%d grudge stacks)", int((bonus-1)*100), stacks))
		}
	}
}
