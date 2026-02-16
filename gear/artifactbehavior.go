package gear

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"sort"

	"github.com/bytearena/ecs"
)

// Behavior string constants for major artifact behaviors.
// Used as keys in charge tracking, hook registration, and artifact definitions.
const (
	// BehaviorInitiativeFirst is checked directly in combat initialization (not a per-turn hook).
	BehaviorInitiativeFirst     = "initiative_first"    // Commander's Initiative Badge
	BehaviorVanguardMovement    = "vanguard_movement"   // Vanguard's Oath
	BehaviorDoubleTime          = "double_time"         // Double Time Drums
	BehaviorEngagementChains    = "engagement_chains"   // Forced Engagement Chains
	BehaviorMomentumStandard    = "momentum_standard"   // Momentum Standard
	BehaviorSaboteurWsHourglass = "saboteurs_hourglass" // Saboteur's Hourglass
	BehaviorEchoDrums           = "echo_drums"          // Echo Drums
	BehaviorStandDown           = "stand_down"          // Stand Down Orders
	BehaviorChainOfCommand      = "chain_of_command"    // Chain of Command Scepter
	BehaviorRallyingHorn        = "rallying_horn"       // Rallying War Horn
	BehaviorAnthemPerseverance  = "anthem_perseverance" // Anthem of Perseverance
	BehaviorDeadlockShackles    = "deadlock_shackles"   // Deadlock Shackles
)

// BehaviorContext bundles runtime dependencies for behavior hooks.
type BehaviorContext struct {
	Manager       *common.EntityManager
	Cache         *combat.CombatQueryCache
	ChargeTracker *ArtifactChargeTracker
}

// GetSquadFaction returns the faction EntityID for the given squad, or 0 if not in combat.
func (ctx *BehaviorContext) GetSquadFaction(squadID ecs.EntityID) ecs.EntityID {
	return combat.GetSquadFaction(squadID, ctx.Manager)
}

// GetFactionSquads returns active squad IDs for the given faction.
func (ctx *BehaviorContext) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID {
	return combat.GetActiveSquadsForFaction(factionID, ctx.Manager)
}

// GetSquadSpeed returns the movement speed for a squad, falling back to DefaultMovementSpeed.
func (ctx *BehaviorContext) GetSquadSpeed(squadID ecs.EntityID) int {
	speed := squads.GetSquadMovementSpeed(squadID, ctx.Manager)
	if speed == 0 {
		speed = combat.DefaultMovementSpeed
	}
	return speed
}

// BehaviorTargetType describes what kind of target a behavior requires.
const (
	TargetNone     = 0
	TargetFriendly = 1
	TargetEnemy    = 2
)

// ArtifactBehavior defines the contract for major artifact behaviors.
type ArtifactBehavior interface {
	BehaviorKey() string
	TargetType() int
	OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID)
	OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *squads.CombatResult)
	OnTurnEnd(ctx *BehaviorContext, round int)
	IsPlayerActivated() bool
	Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error
}

// BaseBehavior provides no-op defaults. Concrete behaviors embed this
// and override only the hooks they need.
type BaseBehavior struct{}

func (BaseBehavior) TargetType() int                                              { return TargetNone }
func (BaseBehavior) OnPostReset(*BehaviorContext, ecs.EntityID, []ecs.EntityID) {}
func (BaseBehavior) OnAttackComplete(*BehaviorContext, ecs.EntityID, ecs.EntityID, *squads.CombatResult) {
}
func (BaseBehavior) OnTurnEnd(*BehaviorContext, int) {}
func (BaseBehavior) IsPlayerActivated() bool         { return false }
func (BaseBehavior) Activate(*BehaviorContext, ecs.EntityID) error {
	return fmt.Errorf("not player-activated")
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

// CanActivateArtifact returns true if the given artifact behavior's charge is available.
func CanActivateArtifact(behavior string, charges *ArtifactChargeTracker) bool {
	if charges == nil {
		return false
	}
	return charges.IsAvailable(behavior)
}
