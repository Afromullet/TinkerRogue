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
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"

	"github.com/bytearena/ecs"
)

func init() {
	// Shared tracking readers
	RegisterPerkHooks("reckless_assault", &PerkHooks{
		State: StateRequirements{
			Category:     StateSharedRead,
			ReadsFields:  []string{"RecklessVulnerable"},
			WritesFields: []string{"RecklessVulnerable"},
		},
		AttackerDamageMod: recklessAssaultAttackerMod,
		DefenderDamageMod: recklessAssaultDefenderMod,
	})
	RegisterPerkHooks("stalwart", &PerkHooks{
		State:      StateRequirements{Category: StateSharedRead, ReadsFields: []string{"MovedThisTurn"}},
		CounterMod: stalwartCounterMod,
	})
	RegisterPerkHooks("fortify", &PerkHooks{
		State: StateRequirements{
			Category:     StateSharedRead,
			ReadsFields:  []string{"MovedThisTurn", "TurnsStationary"},
			WritesFields: []string{"TurnsStationary"},
		},
		TurnStart:        fortifyTurnStart,
		DefenderCoverMod: fortifyCoverMod,
	})

	// Per-perk round state (uses GetPerkState/SetPerkState)
	RegisterPerkHooks("counterpunch", &PerkHooks{
		State: StateRequirements{
			Category:    StatePerRound,
			ReadsFields: []string{"WasAttackedLastTurn", "DidNotAttackLastTurn"},
		},
		TurnStart:         counterpunchTurnStart,
		AttackerDamageMod: counterpunchDamageMod,
	})
	RegisterPerkHooks("deadshots_patience", &PerkHooks{
		State: StateRequirements{
			Category:    StatePerRound,
			ReadsFields: []string{"WasIdleLastTurn"},
		},
		TurnStart:         deadshotTurnStart,
		AttackerDamageMod: deadshotDamageMod,
	})
	RegisterPerkHooks("disruption", &PerkHooks{
		State:              StateRequirements{Category: StatePerRound},
		AttackerPostDamage: disruptionPostDamage,
	})
	RegisterPerkHooks("adaptive_armor", &PerkHooks{
		State:             StateRequirements{Category: StatePerRound},
		DefenderDamageMod: adaptiveArmorDamageMod,
	})
	RegisterPerkHooks("bloodlust", &PerkHooks{
		State:              StateRequirements{Category: StatePerRound},
		AttackerPostDamage: bloodlustPostDamage,
		AttackerDamageMod:  bloodlustDamageMod,
	})
RegisterPerkHooks("overwatch", &PerkHooks{
		State:     StateRequirements{Category: StatePerRound},
		TurnStart: overwatchTurnStart,
	})
}

// ========================================
// SHARED TRACKING READERS
// ========================================

// recklessAssaultAttackerMod boosts outgoing damage and sets vulnerability.
// State: writes PerkRoundState.RecklessVulnerable (shared tracking).
func recklessAssaultAttackerMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	modifiers.DamageMultiplier *= PerkBalance.RecklessAssault.AttackerMult
	ctx.RoundState.RecklessVulnerable = true
}

// recklessAssaultDefenderMod increases incoming damage when vulnerable.
// State: reads PerkRoundState.RecklessVulnerable (shared tracking).
func recklessAssaultDefenderMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if ctx.RoundState.RecklessVulnerable {
		modifiers.DamageMultiplier *= PerkBalance.RecklessAssault.DefenderMult
	}
}

// stalwartCounterMod gives full-damage counters if the squad did not move.
// State: reads PerkRoundState.MovedThisTurn (shared tracking, set by dispatch).
func stalwartCounterMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) bool {
	if !ctx.RoundState.MovedThisTurn {
		modifiers.DamageMultiplier = 1.0 // Override 0.5 default
	}
	return false
}

// fortifyTurnStart increments stationary counter if squad didn't move.
// State: reads MovedThisTurn, writes TurnsStationary (shared tracking).
func fortifyTurnStart(ctx *HookContext) {
	if ctx.RoundState.MovedThisTurn {
		ctx.RoundState.TurnsStationary = 0
	} else {
		if ctx.RoundState.TurnsStationary < PerkBalance.Fortify.MaxStationaryTurns {
			ctx.RoundState.TurnsStationary++
		}
	}
}

// fortifyCoverMod adds cover based on consecutive stationary turns.
// State: reads PerkRoundState.TurnsStationary (shared tracking).
func fortifyCoverMod(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown) {
	if ctx.RoundState.TurnsStationary > 0 {
		bonus := float64(ctx.RoundState.TurnsStationary) * PerkBalance.Fortify.PerTurnCoverBonus
		coverBreakdown.TotalReduction += bonus
		if coverBreakdown.TotalReduction > 1.0 {
			coverBreakdown.TotalReduction = 1.0
		}
	}
}

// ========================================
// PER-PERK ROUND STATE
// ========================================

// counterpunchTurnStart arms the counterpunch bonus based on last-turn snapshots.
// State: reads WasAttackedLastTurn, DidNotAttackLastTurn (shared snapshots);
//
//	writes CounterpunchState via SetPerkState (per-round).
func counterpunchTurnStart(ctx *HookContext) {
	ready := ctx.RoundState.WasAttackedLastTurn && ctx.RoundState.DidNotAttackLastTurn
	SetPerkState(ctx.RoundState, "counterpunch", &CounterpunchState{Ready: ready})
}

// counterpunchDamageMod applies +40% damage when armed.
// State: reads/writes CounterpunchState via GetPerkState (per-round).
func counterpunchDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	state := GetPerkState[*CounterpunchState](ctx.RoundState, "counterpunch")
	if state != nil && state.Ready {
		modifiers.DamageMultiplier *= PerkBalance.Counterpunch.DamageMult
		state.Ready = false
	}
}

// deadshotTurnStart arms the deadshot bonus if the squad was idle last turn.
// State: reads WasIdleLastTurn (shared snapshot);
//
//	writes DeadshotState via SetPerkState (per-round).
func deadshotTurnStart(ctx *HookContext) {
	ready := ctx.RoundState.WasIdleLastTurn
	SetPerkState(ctx.RoundState, "deadshots_patience", &DeadshotState{Ready: ready})
}

// deadshotDamageMod applies +50% damage and +20 accuracy for ranged/magic attacks when armed.
// State: reads/writes DeadshotState via GetPerkState (per-round).
func deadshotDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	state := GetPerkState[*DeadshotState](ctx.RoundState, "deadshots_patience")
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
	}
}

// disruptionPostDamage marks the target squad as disrupted.
// State: writes DisruptionState via SetPerkState on both attacker and defender (per-round).
func disruptionPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if damageDealt <= 0 {
		return
	}
	state := GetOrInitPerkState(ctx.RoundState, "disruption", func() *DisruptionState {
		return &DisruptionState{Targets: make(map[ecs.EntityID]bool)}
	})
	state.Targets[ctx.DefenderSquadID] = true

	defenderRoundState := GetRoundState(ctx.DefenderSquadID, ctx.Manager)
	if defenderRoundState != nil {
		defState := GetOrInitPerkState(defenderRoundState, "disruption", func() *DisruptionState {
			return &DisruptionState{Targets: make(map[ecs.EntityID]bool)}
		})
		defState.Targets[ctx.AttackerSquadID] = true
	}
}

// adaptiveArmorDamageMod reduces damage from repeated attackers.
// State: reads/writes AdaptiveArmorState via GetPerkState/SetPerkState (per-round).
func adaptiveArmorDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	state := GetOrInitPerkState(ctx.RoundState, "adaptive_armor", func() *AdaptiveArmorState {
		return &AdaptiveArmorState{AttackedBy: make(map[ecs.EntityID]int)}
	})
	hits := state.AttackedBy[ctx.AttackerSquadID]
	if hits > PerkBalance.AdaptiveArmor.MaxHits {
		hits = PerkBalance.AdaptiveArmor.MaxHits
	}
	if hits > 0 {
		reduction := float64(hits) * PerkBalance.AdaptiveArmor.PerHitReduction
		modifiers.DamageMultiplier *= (1.0 - reduction)
	}
	state.AttackedBy[ctx.AttackerSquadID]++
}

// bloodlustPostDamage tracks kills this round.
// State: writes BloodlustState via SetPerkState (per-round).
func bloodlustPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if wasKill {
		state := GetOrInitPerkState(ctx.RoundState, "bloodlust", func() *BloodlustState {
			return &BloodlustState{}
		})
		state.KillsThisRound++
	}
}

// bloodlustDamageMod applies bonus damage based on kills this round.
// State: reads BloodlustState via GetPerkState (per-round).
func bloodlustDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	state := GetPerkState[*BloodlustState](ctx.RoundState, "bloodlust")
	if state != nil && state.KillsThisRound > 0 {
		bonus := 1.0 + float64(state.KillsThisRound)*PerkBalance.Bloodlust.PerKillBonus
		modifiers.DamageMultiplier *= bonus
	}
}


// overwatchTurnStart is a placeholder for the overwatch perk.
// State: placeholder — not implemented in v1.
func overwatchTurnStart(ctx *HookContext) {
	// Placeholder — the actual trigger happens in the movement system (not implemented in v1).
}
