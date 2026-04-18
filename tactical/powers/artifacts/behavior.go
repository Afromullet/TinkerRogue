package artifacts

import (
	"fmt"
	"game_main/tactical/combat/combattypes"

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

func (BaseBehavior) TargetType() BehaviorTargetType                             { return TargetNone }
func (BaseBehavior) OnPostReset(*BehaviorContext, ecs.EntityID, []ecs.EntityID) {}
func (BaseBehavior) OnAttackComplete(*BehaviorContext, ecs.EntityID, ecs.EntityID, *combattypes.CombatResult) {
}
func (BaseBehavior) OnTurnEnd(*BehaviorContext, int) {}
func (BaseBehavior) IsPlayerActivated() bool         { return false }
func (BaseBehavior) Activate(*BehaviorContext, ecs.EntityID) error {
	return fmt.Errorf("not player-activated")
}
