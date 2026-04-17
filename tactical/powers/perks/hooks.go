package perks

import (
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// HookContext bundles common parameters passed to all perk hooks.
// It embeds powercore.PowerContext by value so Manager, Cache, RoundNumber,
// and Logger come from one shared definition. Perk-specific fields
// (attacker/defender identities, damage amount, RoundState) stay on this
// struct. Value embedding keeps zero-value contexts usable in tests.
//
// Some fields may be zero-valued depending on the hook type (e.g. TurnStart
// has no attacker/defender).
type HookContext struct {
	powercore.PowerContext

	AttackerID      ecs.EntityID
	DefenderID      ecs.EntityID
	AttackerSquadID ecs.EntityID
	DefenderSquadID ecs.EntityID
	SquadID         ecs.EntityID // The squad that owns the perk (used by TurnStart, DeathOverride)
	UnitID          ecs.EntityID // Specific unit (used by DeathOverride, DamageRedirect)
	DamageAmount    int          // Incoming damage (used by DamageRedirect)
	RoundState      *PerkRoundState
}

// PerkBehavior defines the contract for all perk implementations.
// Perks embed BasePerkBehavior and override only the methods they need.
type PerkBehavior interface {
	PerkID() PerkID

	// Damage pipeline hooks
	AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers)
	DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers)
	DefenderCoverMod(ctx *HookContext, coverBreakdown *combattypes.CoverBreakdown)
	TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID
	CounterMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) (skipCounter bool)
	AttackerPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
	DefenderPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
	TurnStart(ctx *HookContext)
	DamageRedirect(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
	DeathOverride(ctx *HookContext) (preventDeath bool)
}

// BasePerkBehavior provides no-op defaults. Concrete perks embed this
// and override only the hooks they need.
type BasePerkBehavior struct{}

func (BasePerkBehavior) AttackerDamageMod(*HookContext, *combattypes.DamageModifiers)  {}
func (BasePerkBehavior) DefenderDamageMod(*HookContext, *combattypes.DamageModifiers)  {}
func (BasePerkBehavior) DefenderCoverMod(*HookContext, *combattypes.CoverBreakdown)    {}
func (BasePerkBehavior) AttackerPostDamage(*HookContext, int, bool)                   {}
func (BasePerkBehavior) DefenderPostDamage(*HookContext, int, bool)                   {}
func (BasePerkBehavior) TurnStart(*HookContext)                                       {}
func (BasePerkBehavior) CounterMod(*HookContext, *combattypes.DamageModifiers) bool    { return false }
func (BasePerkBehavior) DeathOverride(*HookContext) bool                              { return false }
func (BasePerkBehavior) DamageRedirect(*HookContext) (int, ecs.EntityID, int)         { return 0, 0, 0 }
func (BasePerkBehavior) TargetOverride(_ *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
	return defaultTargets
}

// LogPerk routes a perk activation message through the embedded PowerLogger,
// converting the typed PerkID to the string form expected by PowerLogger.
// Nil-safe via the underlying ctx.Log.
func (ctx *HookContext) LogPerk(perkID PerkID, squadID ecs.EntityID, message string) {
	ctx.Log(string(perkID), squadID, message)
}

var behaviorRegistry = map[PerkID]PerkBehavior{}

// RegisterPerkBehavior registers a perk behavior by its PerkID.
func RegisterPerkBehavior(b PerkBehavior) {
	behaviorRegistry[b.PerkID()] = b
}

// GetPerkBehavior returns the behavior for a perk, or nil if not found.
func GetPerkBehavior(perkID PerkID) PerkBehavior {
	return behaviorRegistry[perkID]
}
