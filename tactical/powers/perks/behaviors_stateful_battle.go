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
	"fmt"

	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterPerkBehavior(&OpeningSalvoBehavior{})
	RegisterPerkBehavior(&ResoluteBehavior{})
	RegisterPerkBehavior(&GrudgeBearerBehavior{})
}

// ========================================
// Opening Salvo: +35% damage on first attack of combat
// ========================================

type OpeningSalvoBehavior struct{ BasePerkBehavior }

func (b *OpeningSalvoBehavior) PerkID() PerkID { return PerkOpeningSalvo }

// openingSalvoDamageMod gives +35% damage on the squad's first attack of the combat.
// State: reads/writes OpeningSalvoState via GetBattleState/SetBattleState (per-battle).
func (b *OpeningSalvoBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	state := GetBattleState[*OpeningSalvoState](ctx.RoundState, PerkOpeningSalvo)
	if state != nil && state.HasAttackedThisCombat {
		return
	}
	modifiers.DamageMultiplier *= PerkBalance.OpeningSalvo.DamageMult
	SetBattleState(ctx.RoundState, PerkOpeningSalvo, &OpeningSalvoState{HasAttackedThisCombat: true})
	logPerkActivation(PerkOpeningSalvo, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (opening attack)", int((PerkBalance.OpeningSalvo.DamageMult-1)*100)))
}

// ========================================
// Resolute: Prevents death once per combat if unit had >50% HP at round start
// ========================================

type ResoluteBehavior struct{ BasePerkBehavior }

func (b *ResoluteBehavior) PerkID() PerkID { return PerkResolute }

// resoluteTurnStart snapshots current HP for the resolute death-save check.
// State: writes ResoluteState.RoundStartHP via GetBattleState/SetBattleState (per-battle).
func (b *ResoluteBehavior) TurnStart(ctx *HookContext) {
	state := GetOrInitBattleState(ctx.RoundState, PerkResolute, func() *ResoluteState {
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
		logPerkActivation(PerkResolute, ctx.SquadID, "unit survives lethal damage at 1 HP")
		return true
	}
	return false
}

// ========================================
// Grudge Bearer: +damage per grudge stack from enemy squad damage
// ========================================

type GrudgeBearerBehavior struct{ BasePerkBehavior }

func (b *GrudgeBearerBehavior) PerkID() PerkID { return PerkGrudgeBearer }

// grudgeBearerPostDamage tracks damage received from enemy squads.
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

// grudgeBearerDamageMod applies +20% damage per grudge stack (max +40%).
// State: reads GrudgeBearerState.Stacks via GetBattleState (per-battle).
func (b *GrudgeBearerBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	state := GetBattleState[*GrudgeBearerState](ctx.RoundState, PerkGrudgeBearer)
	if state != nil {
		stacks := state.Stacks[ctx.DefenderSquadID]
		if stacks > 0 {
			bonus := 1.0 + float64(stacks)*PerkBalance.GrudgeBearer.PerStackBonus
			modifiers.DamageMultiplier *= bonus
			logPerkActivation(PerkGrudgeBearer, ctx.AttackerSquadID, fmt.Sprintf("+%d%% damage (%d grudge stacks)", int((bonus-1)*100), stacks))
		}
	}
}
