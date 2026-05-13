package combatlifecycle

import (
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// ResolutionContext carries runtime information that resolvers need at exit time.
// Built by EncounterService.ExitCombat (or any other exit orchestrator) and
// passed into CombatResolver.Resolve so resolvers can be constructed eagerly
// in CombatStarter.Prepare without capturing runtime state in closures.
type ResolutionContext struct {
	Reason         CombatExitReason
	PlayerVictory  bool
	PlayerEntityID ecs.EntityID
	PlayerSquadIDs []ecs.EntityID
}

// CombatResolver handles context-specific combat resolution.
// Each combat type implements this: overworld, raid, garrison defense.
// Resolve() applies domain-specific state changes (threat damage, room clearing, etc.)
// and returns a ResolutionPlan describing rewards to grant.
type CombatResolver interface {
	Resolve(manager *common.EntityManager, ctx ResolutionContext) *ResolutionPlan
}

// ResolutionPlan describes what to grant after resolution.
// Return nil from Resolve() for no-reward scenarios.
type ResolutionPlan struct {
	Rewards     Reward
	Target      GrantTarget
	Description string // Summary for logging/GUI ("Threat 42 destroyed")
}

// ResolutionResult is the unified output from ExecuteResolution.
type ResolutionResult struct {
	Rewards     Reward
	RewardText  string // Human-readable from Grant() ("150 gold, 75 XP")
	Description string // From resolver
}

// ExecuteResolution is THE single entry point for all combat resolution.
// All combat types call this. All rewards flow through here.
func ExecuteResolution(manager *common.EntityManager, resolver CombatResolver, ctx ResolutionContext) *ResolutionResult {
	plan := resolver.Resolve(manager, ctx)
	if plan == nil {
		return &ResolutionResult{}
	}

	result := &ResolutionResult{
		Description: plan.Description,
		Rewards:     plan.Rewards,
	}

	if plan.Rewards.Gold > 0 || plan.Rewards.Experience > 0 || plan.Rewards.Mana > 0 {
		result.RewardText = Grant(manager, plan.Rewards, plan.Target)
	}

	return result
}
