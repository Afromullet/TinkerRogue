// behaviors_stateful_battle.go — Perk implementations that persist state across the entire combat.
//
// These perks use GetBattleState/SetBattleState. Their state survives round resets
// and is only cleaned up by CleanupRoundState at combat end.
//
// Adding a new battle-persistent perk? Put it here.
// If the state resets each round, use behaviors_stateful_round.go instead.
// If it needs no state at all, use behaviors_stateless.go.
package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterPerkHooks("opening_salvo", &PerkHooks{
		State:             StateRequirements{Category: StateBattle},
		AttackerDamageMod: openingSalvoDamageMod,
	})
	RegisterPerkHooks("resolute", &PerkHooks{
		State:         StateRequirements{Category: StateBattle},
		TurnStart:     resoluteTurnStart,
		DeathOverride: resoluteDeathOverride,
	})
	RegisterPerkHooks("grudge_bearer", &PerkHooks{
		State:             StateRequirements{Category: StateBattle},
		PostDamage:        grudgeBearerPostDamage,
		AttackerDamageMod: grudgeBearerDamageMod,
	})
}

// openingSalvoDamageMod gives +35% damage on the squad's first attack of the combat.
// State: reads/writes OpeningSalvoState via GetBattleState/SetBattleState (per-battle).
func openingSalvoDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	state := GetBattleState[*OpeningSalvoState](ctx.RoundState, "opening_salvo")
	if state != nil && state.HasAttackedThisCombat {
		return
	}
	modifiers.DamageMultiplier *= 1.35
	SetBattleState(ctx.RoundState, "opening_salvo", &OpeningSalvoState{HasAttackedThisCombat: true})
}

// resoluteTurnStart snapshots current HP for the resolute death-save check.
// State: writes ResoluteState.RoundStartHP via GetBattleState/SetBattleState (per-battle).
func resoluteTurnStart(ctx *HookContext) {
	state := GetOrInitBattleState(ctx.RoundState, "resolute", func() *ResoluteState {
		return &ResoluteState{
			Used:         make(map[ecs.EntityID]bool),
			RoundStartHP: make(map[ecs.EntityID]int),
		}
	})
	unitIDs := squadcore.GetUnitIDsInSquad(ctx.SquadID, ctx.Manager)
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](
			ctx.Manager, uid, common.AttributeComponent,
		)
		if attr != nil && attr.CurrentHealth > 0 {
			state.RoundStartHP[uid] = attr.CurrentHealth
		}
	}
}

// resoluteDeathOverride prevents death if the unit had >50% HP at round start (once per battle).
// State: reads/writes ResoluteState via GetBattleState (per-battle).
func resoluteDeathOverride(ctx *HookContext) bool {
	state := GetBattleState[*ResoluteState](ctx.RoundState, "resolute")
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
	if maxHP > 0 && float64(roundStartHP)/float64(maxHP) > 0.5 {
		state.Used[ctx.UnitID] = true
		return true
	}
	return false
}

// grudgeBearerPostDamage tracks damage received from enemy squads.
// State: writes GrudgeBearerState.Stacks via GetBattleState/SetBattleState (per-battle).
func grudgeBearerPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if damageDealt <= 0 {
		return
	}
	state := GetOrInitBattleState(ctx.RoundState, "grudge_bearer", func() *GrudgeBearerState {
		return &GrudgeBearerState{Stacks: make(map[ecs.EntityID]int)}
	})
	current := state.Stacks[ctx.AttackerSquadID]
	if current < 2 {
		state.Stacks[ctx.AttackerSquadID] = current + 1
	}
}

// grudgeBearerDamageMod applies +20% damage per grudge stack (max +40%).
// State: reads GrudgeBearerState.Stacks via GetBattleState (per-battle).
func grudgeBearerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	state := GetBattleState[*GrudgeBearerState](ctx.RoundState, "grudge_bearer")
	if state != nil {
		stacks := state.Stacks[ctx.DefenderSquadID]
		if stacks > 0 {
			bonus := 1.0 + float64(stacks)*0.20
			modifiers.DamageMultiplier *= bonus
		}
	}
}
