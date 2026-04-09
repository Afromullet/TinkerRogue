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

// PerkBehavior defines the contract for all perk implementations.
// Perks embed BasePerkBehavior and override only the methods they need.
type PerkBehavior interface {
	PerkID() string

	// Damage pipeline hooks
	AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers)
	DefenderDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers)
	DefenderCoverMod(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown)
	TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID
	CounterMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) (skipCounter bool)
	AttackerPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
	DefenderPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
	TurnStart(ctx *HookContext)
	DamageRedirect(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
	DeathOverride(ctx *HookContext) (preventDeath bool)
}

// BasePerkBehavior provides no-op defaults. Concrete perks embed this
// and override only the hooks they need.
type BasePerkBehavior struct{}

func (BasePerkBehavior) AttackerDamageMod(*HookContext, *combatcore.DamageModifiers)  {}
func (BasePerkBehavior) DefenderDamageMod(*HookContext, *combatcore.DamageModifiers)  {}
func (BasePerkBehavior) DefenderCoverMod(*HookContext, *combatcore.CoverBreakdown)    {}
func (BasePerkBehavior) AttackerPostDamage(*HookContext, int, bool)                   {}
func (BasePerkBehavior) DefenderPostDamage(*HookContext, int, bool)                   {}
func (BasePerkBehavior) TurnStart(*HookContext)                                       {}
func (BasePerkBehavior) CounterMod(*HookContext, *combatcore.DamageModifiers) bool    { return false }
func (BasePerkBehavior) DeathOverride(*HookContext) bool                              { return false }
func (BasePerkBehavior) DamageRedirect(*HookContext) (int, ecs.EntityID, int)         { return 0, 0, 0 }
func (BasePerkBehavior) TargetOverride(_ *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
	return defaultTargets
}

// PerkLogger is called when a perk activates, for combat log feedback.
type PerkLogger func(perkID string, squadID ecs.EntityID, message string)

var perkLogger PerkLogger

// SetPerkLogger sets the callback for perk activation messages.
func SetPerkLogger(fn PerkLogger) {
	perkLogger = fn
}

// logPerkActivation logs a perk activation event if a logger is set.
func logPerkActivation(perkID string, squadID ecs.EntityID, message string) {
	if perkLogger != nil {
		perkLogger(perkID, squadID, message)
	}
}

var behaviorRegistry = map[string]PerkBehavior{}

// RegisterPerkBehavior registers a perk behavior by its PerkID.
func RegisterPerkBehavior(b PerkBehavior) {
	behaviorRegistry[b.PerkID()] = b
}

// GetPerkBehavior returns the behavior for a perk, or nil if not found.
func GetPerkBehavior(perkID string) PerkBehavior {
	return behaviorRegistry[perkID]
}
