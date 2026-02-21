package combatpipeline

import "game_main/common"

// CombatResolver handles context-specific combat resolution.
// Each combat type implements this: overworld, raid, garrison defense, flee.
// Resolve() applies domain-specific state changes (threat damage, room clearing, etc.)
// and returns a ResolutionPlan describing rewards to grant.
type CombatResolver interface {
	Resolve(manager *common.EntityManager) *ResolutionPlan
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
func ExecuteResolution(manager *common.EntityManager, resolver CombatResolver) *ResolutionResult {
	plan := resolver.Resolve(manager)
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
