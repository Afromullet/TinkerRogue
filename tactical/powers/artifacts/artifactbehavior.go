package artifacts

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"
	"game_main/templates"
	"sort"

	"github.com/bytearena/ecs"
)

// Behavior string constants for major artifact behaviors.
// Used as keys in charge tracking, hook registration, and artifact definitions.
const (
	BehaviorEngagementChains    = "engagement_chains"   // Forced Engagement Chains
	BehaviorSaboteurWsHourglass = "saboteurs_hourglass" // Saboteur's Hourglass
	BehaviorEchoDrums           = "echo_drums"          // Echo Drums
	BehaviorChainOfCommand      = "chain_of_command"    // Chain of Command Scepter
	BehaviorTwinStrike          = "twin_strike"         // Twin Strike Banner
	BehaviorDeadlockShackles    = "deadlock_shackles"   // Deadlock Shackles
)

// BehaviorContext bundles runtime dependencies for behavior hooks.
type BehaviorContext struct {
	Manager       *common.EntityManager
	cache         *combatstate.CombatQueryCache
	ChargeTracker *ArtifactChargeTracker
}

// NewBehaviorContext creates a BehaviorContext with the given dependencies.
func NewBehaviorContext(manager *common.EntityManager, cache *combatstate.CombatQueryCache, chargeTracker *ArtifactChargeTracker) *BehaviorContext {
	return &BehaviorContext{
		Manager:       manager,
		cache:         cache,
		ChargeTracker: chargeTracker,
	}
}

// GetActionState returns the ActionStateData for the given squad, or nil if not found.
func (ctx *BehaviorContext) GetActionState(squadID ecs.EntityID) *combatstate.ActionStateData {
	return ctx.cache.FindActionStateBySquadID(squadID)
}

// SetSquadLocked fully locks a squad so it cannot move or act this turn.
func (ctx *BehaviorContext) SetSquadLocked(squadID ecs.EntityID) {
	actionState := ctx.cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		return
	}
	actionState.HasActed = true
	actionState.HasMoved = true
	actionState.MovementRemaining = 0
}

// ResetSquadActions fully resets a squad's action state with the given movement speed.
func (ctx *BehaviorContext) ResetSquadActions(squadID ecs.EntityID, speed int) {
	actionState := ctx.cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		return
	}
	actionState.HasActed = false
	actionState.HasMoved = false
	actionState.MovementRemaining = speed
}

// GetSquadFaction returns the faction EntityID for the given squad, or 0 if not in combat.
func (ctx *BehaviorContext) GetSquadFaction(squadID ecs.EntityID) ecs.EntityID {
	return combatstate.GetSquadFaction(squadID, ctx.Manager)
}

// GetFactionSquads returns active squad IDs for the given faction.
func (ctx *BehaviorContext) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID {
	return combatstate.GetActiveSquadsForFaction(factionID, ctx.Manager)
}

// GetSquadSpeed returns the movement speed for a squad, falling back to DefaultMovementSpeed.
func (ctx *BehaviorContext) GetSquadSpeed(squadID ecs.EntityID) int {
	return squadcore.GetSquadMovementSpeedOrDefault(squadID, ctx.Manager)
}

// BehaviorTargetType describes what kind of target a behavior requires.
type BehaviorTargetType int

const (
	TargetNone     BehaviorTargetType = 0
	TargetFriendly BehaviorTargetType = 1
	TargetEnemy    BehaviorTargetType = 2
)

// String returns a display label for the target type.
func (t BehaviorTargetType) String() string {
	switch t {
	case TargetFriendly:
		return "Friendly Squad"
	case TargetEnemy:
		return "Enemy Squad"
	default:
		return "No Target"
	}
}

// ArtifactBehavior defines the contract for major artifact behaviors.
type ArtifactBehavior interface {
	BehaviorKey() string
	TargetType() BehaviorTargetType
	OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID)
	OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)
	OnTurnEnd(ctx *BehaviorContext, round int)
	IsPlayerActivated() bool
	Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error
}

// BaseBehavior provides no-op defaults. Concrete behaviors embed this
// and override only the hooks they need.
type BaseBehavior struct{}

func (BaseBehavior) TargetType() BehaviorTargetType                              { return TargetNone }
func (BaseBehavior) OnPostReset(*BehaviorContext, ecs.EntityID, []ecs.EntityID) {}
func (BaseBehavior) OnAttackComplete(*BehaviorContext, ecs.EntityID, ecs.EntityID, *combattypes.CombatResult) {
}
func (BaseBehavior) OnTurnEnd(*BehaviorContext, int) {}
func (BaseBehavior) IsPlayerActivated() bool         { return false }
func (BaseBehavior) Activate(*BehaviorContext, ecs.EntityID) error {
	return fmt.Errorf("not player-activated")
}

// ArtifactLogger is called when an artifact activates, for combat log feedback.
type ArtifactLogger func(behaviorKey string, squadID ecs.EntityID, message string)

var artifactLogger ArtifactLogger

// SetArtifactLogger sets the callback for artifact activation messages.
func SetArtifactLogger(fn ArtifactLogger) {
	artifactLogger = fn
}

// logArtifactActivation logs an artifact activation event if a logger is set.
func logArtifactActivation(behaviorKey string, squadID ecs.EntityID, message string) {
	if artifactLogger != nil {
		artifactLogger(behaviorKey, squadID, message)
	}
}

// Registry (same pattern as worldmap/generator.go)
var behaviorRegistry = map[string]ArtifactBehavior{}

// RegisterBehavior adds a behavior to the global registry.
func RegisterBehavior(b ArtifactBehavior) {
	behaviorRegistry[b.BehaviorKey()] = b
}

// GetBehavior returns the behavior for the given key, or nil.
func GetBehavior(key string) ArtifactBehavior {
	return behaviorRegistry[key]
}

// AllBehaviors returns all registered behaviors in deterministic order.
func AllBehaviors() []ArtifactBehavior {
	keys := make([]string, 0, len(behaviorRegistry))
	for k := range behaviorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]ArtifactBehavior, 0, len(keys))
	for _, k := range keys {
		result = append(result, behaviorRegistry[k])
	}
	return result
}

// ActivateArtifact dispatches activation to the registered behavior for the given key.
// Only player-activated behaviors can be triggered this way.
func ActivateArtifact(behavior string, targetSquadID ecs.EntityID, ctx *BehaviorContext) error {
	b := GetBehavior(behavior)
	if b == nil {
		return fmt.Errorf("unknown behavior %q", behavior)
	}
	if !b.IsPlayerActivated() {
		return fmt.Errorf("behavior %q is not player-activated", behavior)
	}
	return b.Activate(ctx, targetSquadID)
}

// ValidateBehaviorCoverage checks that JSON definitions and behavior registrations are in sync.
func ValidateBehaviorCoverage() {
	for id, def := range templates.ArtifactRegistry {
		if def.Behavior == "" {
			continue // minor artifact, no behavior expected
		}
		if GetBehavior(def.Behavior) == nil {
			fmt.Printf("WARNING: Artifact %q has behavior %q but no registered behavior implementation\n", id, def.Behavior)
		}
	}
	for key := range behaviorRegistry {
		found := false
		for _, def := range templates.ArtifactRegistry {
			if def.Behavior == key {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("WARNING: Behavior %q is registered but no artifact definition references it\n", key)
		}
	}
}

// CanActivateArtifact returns true if the given artifact behavior's charge is available.
func CanActivateArtifact(behavior string, charges *ArtifactChargeTracker) bool {
	if charges == nil {
		return false
	}
	return charges.IsAvailable(behavior)
}
