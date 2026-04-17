// behaviors_stateful_round.go — Perk implementations that depend on per-round or per-turn state.
//
// These perks fall into two sub-categories:
//
//   - Shared tracking readers: read shared fields on PerkRoundState (MovedThisTurn,
//     RecklessVulnerable, TurnsStationary, WasAttackedLastTurn, etc.) that are set
//     by the dispatch layer or by ResetPerTurn snapshots.
//
//   - Per-perk round state: use GetPerkState/SetPerkState for isolated state that
//     resets each round via ResetPerRound.
//
// All state here is ephemeral — it resets between rounds or turns.
//
// Adding a new per-round stateful perk? Put it here.
// If it persists across the entire combat, use behaviors_stateful_battle.go instead.
// If it needs no state at all, use behaviors_stateless.go.
package perks

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"

	"github.com/bytearena/ecs"
)

func init() {
	// Shared tracking readers
	RegisterPerkBehavior(&RecklessAssaultBehavior{})
	RegisterPerkBehavior(&StalwartBehavior{})
	RegisterPerkBehavior(&FortifyBehavior{})

	// Per-perk round state (uses GetPerkState/SetPerkState)
	RegisterPerkBehavior(&CounterpunchBehavior{})
	RegisterPerkBehavior(&DeadshotsPatienceBehavior{})
	RegisterPerkBehavior(&AdaptiveArmorBehavior{})
	RegisterPerkBehavior(&BloodlustBehavior{})
}

// ========================================
// SHARED TRACKING READERS
// ========================================

// Reckless Assault: +damage on attack, becomes vulnerable to incoming damage.

type RecklessAssaultBehavior struct{ BasePerkBehavior }

func (b *RecklessAssaultBehavior) PerkID() PerkID { return PerkRecklessAssault }

// recklessAssaultTurnStart resets vulnerability at the start of each turn.
// State: writes RecklessAssaultState via SetPerkState (per-round).
func (b *RecklessAssaultBehavior) TurnStart(ctx *HookContext) {
	SetPerkState(ctx.RoundState, PerkRecklessAssault, &RecklessAssaultState{Vulnerable: false})
}

// recklessAssaultAttackerMod boosts outgoing damage and sets vulnerability.
// State: writes RecklessAssaultState via SetPerkState (per-round).
func (b *RecklessAssaultBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	modifiers.DamageMultiplier *= PerkBalance.RecklessAssault.AttackerMult
	SetPerkState(ctx.RoundState, PerkRecklessAssault, &RecklessAssaultState{Vulnerable: true})
	ctx.LogPerk(PerkRecklessAssault, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage, now vulnerable", int((PerkBalance.RecklessAssault.AttackerMult-1)*100)))
}

// recklessAssaultDefenderMod increases incoming damage when vulnerable.
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

// stalwartCounterMod gives full-damage counters if the squad did not move.
// State: reads PerkRoundState.MovedThisTurn (shared tracking, set by dispatch).
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

// fortifyTurnStart increments stationary counter if squad didn't move.
// State: reads MovedThisTurn, writes TurnsStationary (shared tracking).
func (b *FortifyBehavior) TurnStart(ctx *HookContext) {
	if ctx.RoundState.MovedThisTurn {
		ctx.RoundState.TurnsStationary = 0
	} else {
		ctx.IncrementTurnsStationary(PerkBalance.Fortify.MaxStationaryTurns)
	}
}

// fortifyCoverMod adds cover based on consecutive stationary turns.
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

// ========================================
// PER-PERK ROUND STATE
// ========================================

// Counterpunch: Armed if attacked last turn and didn't attack. +40% damage.

type CounterpunchBehavior struct{ BasePerkBehavior }

func (b *CounterpunchBehavior) PerkID() PerkID { return PerkCounterpunch }

// counterpunchTurnStart arms the counterpunch bonus based on last-turn snapshots.
// State: reads WasAttackedLastTurn, DidNotAttackLastTurn (shared snapshots);
//
//	writes CounterpunchState via SetPerkState (per-round).
func (b *CounterpunchBehavior) TurnStart(ctx *HookContext) {
	ready := ctx.RoundState.WasAttackedLastTurn && ctx.RoundState.DidNotAttackLastTurn
	SetPerkState(ctx.RoundState, PerkCounterpunch, &CounterpunchState{Ready: ready})
}

// counterpunchDamageMod applies +40% damage when armed.
// State: reads/writes CounterpunchState via GetPerkState (per-round).
func (b *CounterpunchBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetPerkState[*CounterpunchState](ctx.RoundState, PerkCounterpunch)
	if state != nil && state.Ready {
		modifiers.DamageMultiplier *= PerkBalance.Counterpunch.DamageMult
		state.Ready = false
		ctx.LogPerk(PerkCounterpunch, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (retaliating)", int((PerkBalance.Counterpunch.DamageMult-1)*100)))
	}
}

// Deadshot's Patience: Armed if idle last turn. +50% damage and +20 accuracy for ranged/magic.

type DeadshotsPatienceBehavior struct{ BasePerkBehavior }

func (b *DeadshotsPatienceBehavior) PerkID() PerkID { return PerkDeadshotsPatience }

// deadshotTurnStart arms the deadshot bonus if the squad was idle last turn.
// State: reads WasIdleLastTurn (shared snapshot);
//
//	writes DeadshotState via SetPerkState (per-round).
func (b *DeadshotsPatienceBehavior) TurnStart(ctx *HookContext) {
	ready := ctx.RoundState.WasIdleLastTurn
	SetPerkState(ctx.RoundState, PerkDeadshotsPatience, &DeadshotState{Ready: ready})
}

// deadshotDamageMod applies +50% damage and +20 accuracy for ranged/magic attacks when armed.
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

// adaptiveArmorDamageMod reduces damage from repeated attackers.
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

// bloodlustPostDamage tracks kills this round.
// State: writes BloodlustState via SetPerkState (per-round).
func (b *BloodlustBehavior) AttackerPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if wasKill {
		state := GetOrInitPerkState(ctx.RoundState, PerkBloodlust, func() *BloodlustState {
			return &BloodlustState{}
		})
		state.KillsThisRound++
	}
}

// bloodlustDamageMod applies bonus damage based on kills this round.
// State: reads BloodlustState via GetPerkState (per-round).
func (b *BloodlustBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
	state := GetPerkState[*BloodlustState](ctx.RoundState, PerkBloodlust)
	if state != nil && state.KillsThisRound > 0 {
		bonus := 1.0 + float64(state.KillsThisRound)*PerkBalance.Bloodlust.PerKillBonus
		modifiers.DamageMultiplier *= bonus
		ctx.LogPerk(PerkBloodlust, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (%d kills)", int((bonus-1)*100), state.KillsThisRound))
	}
}
