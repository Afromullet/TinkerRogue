package encounter

import (
	"fmt"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/campaign/overworld/core"
	"game_main/campaign/overworld/garrison"
	"game_main/campaign/overworld/threat"

	"github.com/bytearena/ecs"
)

// OverworldCombatResolver resolves standard overworld threat encounters.
// Runtime information (victory, player entity, player squads) is supplied via
// ResolutionContext at exit time, so this resolver can be built eagerly in
// OverworldCombatStarter.Prepare.
type OverworldCombatResolver struct {
	ThreatNodeID  ecs.EntityID
	EnemySquadIDs []ecs.EntityID
}

func (r *OverworldCombatResolver) Resolve(manager *common.EntityManager, ctx combatlifecycle.ResolutionContext) *combatlifecycle.ResolutionResult {
	// Flee: no rewards, no state changes, just a log entry.
	if ctx.Reason == combatlifecycle.ExitFlee {
		currentTick := core.GetCurrentTick(manager)
		core.LogEvent(core.EventCombatResolved, currentTick, r.ThreatNodeID,
			fmt.Sprintf("Retreated from threat %d", r.ThreatNodeID),
			map[string]interface{}{
				"victory":            false,
				"retreat":            true,
				"player_units_lost":  0,
				"enemy_units_killed": 0,
			})
		fmt.Printf("Retreated from threat %d (no changes)\n", r.ThreatNodeID)
		return &combatlifecycle.ResolutionResult{
			Description: fmt.Sprintf("Retreated from threat %d", r.ThreatNodeID),
		}
	}

	// Count casualties
	enemyUnitsKilled := combatlifecycle.CountDeadUnits(manager, r.EnemySquadIDs)

	// Find threat node
	threatEntity := manager.FindEntityByID(r.ThreatNodeID)
	if threatEntity == nil {
		fmt.Printf("WARNING: Threat node %d not found for resolution\n", r.ThreatNodeID)
		return nil
	}

	nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		fmt.Printf("WARNING: Entity %d is not an overworld node\n", r.ThreatNodeID)
		return nil
	}

	if ctx.Outcome.IsPlayerVictory {
		return r.resolveVictory(manager, threatEntity, nodeData, ctx, enemyUnitsKilled)
	}
	return r.resolveDefeat(manager, threatEntity, nodeData)
}

func (r *OverworldCombatResolver) resolveVictory(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
	nodeData *core.OverworldNodeData,
	ctx combatlifecycle.ResolutionContext,
	enemyUnitsKilled int,
) *combatlifecycle.ResolutionResult {
	damageDealt := enemyUnitsKilled / EnemiesPerIntensityPoint
	oldIntensity := nodeData.Intensity
	nodeData.Intensity -= damageDealt
	currentTick := core.GetCurrentTick(manager)

	rewards := CalculateIntensityReward(oldIntensity)

	target := combatlifecycle.GrantTarget{
		PlayerEntityID: ctx.PlayerEntityID,
		SquadIDs:       ctx.PlayerSquadIDs,
	}

	if nodeData.Intensity <= 0 {
		threat.DestroyThreatNode(manager, threatEntity)

		core.LogEvent(core.EventCombatResolved, currentTick, r.ThreatNodeID,
			fmt.Sprintf("Combat victory - Threat %d destroyed", r.ThreatNodeID),
			map[string]interface{}{
				"victory":           true,
				"intensity_reduced": oldIntensity,
				"rewards_gold":      rewards.Gold,
				"rewards_xp":        rewards.Experience,
			})

		fmt.Printf("Threat %d destroyed! Rewards: %d gold, %d XP\n",
			r.ThreatNodeID, rewards.Gold, rewards.Experience)

		return &combatlifecycle.ResolutionResult{
			Rewards:     rewards,
			Target:      target,
			Description: fmt.Sprintf("Threat %d destroyed", r.ThreatNodeID),
		}
	}

	// Weakened but not destroyed — partial rewards
	partialRewards := rewards.Scale(0.5)
	nodeData.GrowthProgress = 0.0

	core.LogEvent(core.EventCombatResolved, currentTick, r.ThreatNodeID,
		fmt.Sprintf("Combat victory - Threat %d weakened to intensity %d", r.ThreatNodeID, nodeData.Intensity),
		map[string]interface{}{
			"victory":           true,
			"intensity_reduced": damageDealt,
			"new_intensity":     nodeData.Intensity,
			"rewards_gold":      partialRewards.Gold,
			"rewards_xp":        partialRewards.Experience,
		})

	fmt.Printf("Threat %d weakened to intensity %d. Partial rewards: %d gold, %d XP\n",
		r.ThreatNodeID, nodeData.Intensity, partialRewards.Gold, partialRewards.Experience)

	return &combatlifecycle.ResolutionResult{
		Rewards:     partialRewards,
		Target:      target,
		Description: fmt.Sprintf("Threat %d weakened to intensity %d", r.ThreatNodeID, nodeData.Intensity),
	}
}

func (r *OverworldCombatResolver) resolveDefeat(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
	nodeData *core.OverworldNodeData,
) *combatlifecycle.ResolutionResult {
	oldIntensity := nodeData.Intensity
	nodeData.Intensity += DefeatIntensityGrowth
	nodeData.GrowthProgress = 0.0
	currentTick := core.GetCurrentTick(manager)

	// Update influence radius
	influenceData := common.GetComponentType[*core.InfluenceData](threatEntity, core.InfluenceComponent)
	if influenceData != nil {
		params := core.GetThreatTypeParamsFromConfig(core.ThreatType(nodeData.NodeTypeID))
		influenceData.Radius = params.BaseRadius + nodeData.Intensity
		influenceData.BaseMagnitude = core.CalculateBaseMagnitude(nodeData.Intensity)
	}

	core.LogEvent(core.EventCombatResolved, currentTick, r.ThreatNodeID,
		fmt.Sprintf("Combat defeat - Threat %d grew to intensity %d", r.ThreatNodeID, nodeData.Intensity),
		map[string]interface{}{
			"victory":       false,
			"old_intensity": oldIntensity,
			"new_intensity": nodeData.Intensity,
		})

	fmt.Printf("Defeated by threat %d! Threat grew to intensity %d\n",
		r.ThreatNodeID, nodeData.Intensity)

	return &combatlifecycle.ResolutionResult{
		Description: fmt.Sprintf("Defeated by threat %d", r.ThreatNodeID),
	}
}

// GarrisonDefenseResolver resolves garrison defense encounters.
// PlayerVictory is read from ResolutionContext at exit time.
type GarrisonDefenseResolver struct {
	DefendedNodeID       ecs.EntityID
	AttackingFactionType core.FactionType
}

func (r *GarrisonDefenseResolver) Resolve(manager *common.EntityManager, ctx combatlifecycle.ResolutionContext) *combatlifecycle.ResolutionResult {
	currentTick := core.GetCurrentTick(manager)

	if ctx.Outcome.IsPlayerVictory {
		core.LogEvent(core.EventGarrisonDefended, currentTick, r.DefendedNodeID,
			fmt.Sprintf("Garrison at node %d successfully defended against %s",
				r.DefendedNodeID, r.AttackingFactionType.String()), nil)
		fmt.Printf("Garrison at node %d held! Defense successful.\n", r.DefendedNodeID)

		return &combatlifecycle.ResolutionResult{
			Description: "Garrison defended",
		}
	}

	// Transfer ownership to attacking faction
	newOwner := core.OwnerIDFromFaction(r.AttackingFactionType)
	if err := garrison.TransferNodeOwnership(manager, r.DefendedNodeID, newOwner); err != nil {
		fmt.Printf("ERROR: Failed to transfer node ownership: %v\n", err)
	} else {
		fmt.Printf("Garrison at node %d fell. Node captured by %s.\n", r.DefendedNodeID, newOwner)
	}

	return &combatlifecycle.ResolutionResult{
		Description: fmt.Sprintf("Node %d captured", r.DefendedNodeID),
	}
}

