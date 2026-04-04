package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"

	"github.com/bytearena/ecs"
)

// HookContext bundles common parameters passed to all perk hooks.
// Some fields may be zero-valued depending on the hook type (e.g., TurnStart has no attacker).
type HookContext struct {
	AttackerID      ecs.EntityID
	DefenderID      ecs.EntityID
	AttackerSquadID ecs.EntityID
	DefenderSquadID ecs.EntityID
	SquadID         ecs.EntityID // The squad that owns the perk (used by TurnStart, DeathOverride)
	UnitID          ecs.EntityID // Specific unit (used by DeathOverride, DamageRedirect)
	RoundNumber     int          // Current round (used by TurnStart)
	DamageAmount    int          // Incoming damage (used by DamageRedirect)
	RoundState      *PerkRoundState
	Manager         *common.EntityManager
}

// DamageModHook modifies DamageModifiers before damage calculation.
type DamageModHook func(ctx *HookContext, modifiers *combatcore.DamageModifiers)

// TargetOverrideHook overrides target selection. Returns modified target list.
type TargetOverrideHook func(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID

// CounterModHook modifies or suppresses counterattack.
// Returns true if counter should be skipped entirely.
type CounterModHook func(ctx *HookContext, modifiers *combatcore.DamageModifiers) (skipCounter bool)

// PostDamageHook runs after damage is recorded.
type PostDamageHook func(ctx *HookContext, damageDealt int, wasKill bool)

// TurnStartHook runs at start of a squad's turn.
// Uses ctx.SquadID, ctx.RoundNumber, ctx.RoundState, ctx.Manager.
type TurnStartHook func(ctx *HookContext)

// CoverModHook modifies cover calculation.
type CoverModHook func(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown)

// DamageRedirectHook intercepts damage before it is recorded.
// Uses ctx.UnitID (defender unit), ctx.SquadID (defender squad), ctx.DamageAmount.
// Returns reduced damage for original target, redirect target ID, and redirect amount.
type DamageRedirectHook func(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)

// DeathOverrideHook prevents lethal damage. Returns true to prevent death.
// Uses ctx.UnitID, ctx.SquadID, ctx.RoundState, ctx.Manager.
type DeathOverrideHook func(ctx *HookContext) (preventDeath bool)

// StateCategory classifies how a perk uses combat state.
type StateCategory int

const (
	StateNone       StateCategory = iota // Pure function of HookContext — no state read or written
	StateSharedRead                      // Reads shared PerkRoundState tracking fields (e.g., MovedThisTurn)
	StatePerRound                        // Uses GetPerkState/SetPerkState — resets each round
	StateBattle                          // Uses GetBattleState/SetBattleState — persists entire combat
)

// StateRequirements declares what state a perk reads and writes.
// This is documentation-as-code: the dispatch layer does not enforce these,
// but they make state coupling explicit at registration time.
type StateRequirements struct {
	Category     StateCategory
	ReadsFields  []string // Shared PerkRoundState fields read, e.g. ["MovedThisTurn"]
	WritesFields []string // Shared PerkRoundState fields written, e.g. ["RecklessVulnerable"]
}

// PerkHooks collects all hooks for a single perk.
// Attacker/Defender variants ensure hooks only fire on the correct side,
// eliminating the need for HasPerk() self-checks inside behaviors.
type PerkHooks struct {
	State StateRequirements // Declares state dependencies (zero value = StateNone = stateless)
	AttackerDamageMod DamageModHook // runs only when this squad is the attacker
	DefenderDamageMod DamageModHook // runs only when this squad is the defender
	DefenderCoverMod  CoverModHook  // runs only when this squad is the defender
	TargetOverride    TargetOverrideHook
	CounterMod        CounterModHook
	AttackerPostDamage PostDamageHook // runs only when this squad is the attacker
	DefenderPostDamage PostDamageHook // runs only when this squad is the defender
	TurnStart         TurnStartHook
	DamageRedirect    DamageRedirectHook
	DeathOverride     DeathOverrideHook
}

var hookRegistry = map[string]*PerkHooks{}

// RegisterPerkHooks registers a perk's hook implementations by perk ID.
func RegisterPerkHooks(perkID string, hooks *PerkHooks) {
	hookRegistry[perkID] = hooks
}

// GetPerkHooks returns the hook implementations for a perk, or nil if not found.
func GetPerkHooks(perkID string) *PerkHooks {
	return hookRegistry[perkID]
}
