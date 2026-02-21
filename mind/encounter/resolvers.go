package encounter

import (
	"fmt"

	"game_main/common"
	"game_main/mind/combatpipeline"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/overworld/threat"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// OverworldCombatResolver resolves standard overworld threat encounters.
// Replaces resolveCombatToOverworld, applyCombatOutcome, createCombatOutcome, combatOutcome.
type OverworldCombatResolver struct {
	ThreatNodeID   ecs.EntityID
	PlayerVictory  bool
	PlayerEntityID ecs.EntityID
	PlayerSquadIDs []ecs.EntityID
	EnemySquadIDs  []ecs.EntityID
}

func (r *OverworldCombatResolver) Resolve(manager *common.EntityManager) *combatpipeline.ResolutionPlan {
	// Count casualties
	enemyUnitsKilled := combatpipeline.CountDeadUnits(manager, r.EnemySquadIDs)

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

	currentTick := core.GetCurrentTick(manager)

	if r.PlayerVictory {
		damageDealt := calculateThreatDamage(enemyUnitsKilled)
		oldIntensity := nodeData.Intensity
		nodeData.Intensity -= damageDealt

		rewards := combatpipeline.CalculateIntensityReward(oldIntensity)

		if nodeData.Intensity <= 0 {
			// Destroy threat node completely
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

			return &combatpipeline.ResolutionPlan{
				Rewards: rewards,
				Target: combatpipeline.GrantTarget{
					PlayerEntityID: r.PlayerEntityID,
					SquadIDs:       r.PlayerSquadIDs,
				},
				Description: fmt.Sprintf("Threat %d destroyed", r.ThreatNodeID),
			}
		}

		// Weakened but not destroyed — partial rewards
		partialRewards := rewards.Scale(0.5)
		nodeData.GrowthProgress = 0.0

		core.LogEvent(core.EventCombatResolved, currentTick, r.ThreatNodeID,
			fmt.Sprintf("Combat victory - Threat %d weakened to intensity %d", r.ThreatNodeID, nodeData.Intensity),
			map[string]interface{}{
				"victory":          true,
				"intensity_reduced": damageDealt,
				"new_intensity":    nodeData.Intensity,
				"rewards_gold":     partialRewards.Gold,
				"rewards_xp":       partialRewards.Experience,
			})

		fmt.Printf("Threat %d weakened to intensity %d. Partial rewards: %d gold, %d XP\n",
			r.ThreatNodeID, nodeData.Intensity, partialRewards.Gold, partialRewards.Experience)

		return &combatpipeline.ResolutionPlan{
			Rewards: partialRewards,
			Target: combatpipeline.GrantTarget{
				PlayerEntityID: r.PlayerEntityID,
				SquadIDs:       r.PlayerSquadIDs,
			},
			Description: fmt.Sprintf("Threat %d weakened to intensity %d", r.ThreatNodeID, nodeData.Intensity),
		}
	}

	// Player defeat — threat grows stronger
	oldIntensity := nodeData.Intensity
	nodeData.Intensity += 1
	nodeData.GrowthProgress = 0.0

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

	return &combatpipeline.ResolutionPlan{
		Description: fmt.Sprintf("Defeated by threat %d", r.ThreatNodeID),
	}
}

// GarrisonDefenseResolver resolves garrison defense encounters.
// Replaces resolveGarrisonDefense.
type GarrisonDefenseResolver struct {
	PlayerVictory        bool
	DefendedNodeID       ecs.EntityID
	AttackingFactionType core.FactionType
}

func (r *GarrisonDefenseResolver) Resolve(manager *common.EntityManager) *combatpipeline.ResolutionPlan {
	currentTick := core.GetCurrentTick(manager)

	if r.PlayerVictory {
		core.LogEvent(core.EventGarrisonDefended, currentTick, r.DefendedNodeID,
			fmt.Sprintf("Garrison at node %d successfully defended against %s",
				r.DefendedNodeID, r.AttackingFactionType.String()), nil)
		fmt.Printf("Garrison at node %d held! Defense successful.\n", r.DefendedNodeID)

		return &combatpipeline.ResolutionPlan{
			Description: "Garrison defended",
		}
	}

	// Transfer ownership to attacking faction
	newOwner := r.AttackingFactionType.String()
	if err := garrison.TransferNodeOwnership(manager, r.DefendedNodeID, newOwner); err != nil {
		fmt.Printf("ERROR: Failed to transfer node ownership: %v\n", err)
	} else {
		fmt.Printf("Garrison at node %d fell. Node captured by %s.\n", r.DefendedNodeID, newOwner)
	}

	return &combatpipeline.ResolutionPlan{
		Description: fmt.Sprintf("Node %d captured", r.DefendedNodeID),
	}
}

// FleeResolver resolves flee/retreat from combat.
// Replaces resolveFleeToOverworld.
type FleeResolver struct {
	ThreatNodeID ecs.EntityID
}

func (r *FleeResolver) Resolve(manager *common.EntityManager) *combatpipeline.ResolutionPlan {
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

	return &combatpipeline.ResolutionPlan{
		Description: fmt.Sprintf("Retreated from threat %d", r.ThreatNodeID),
	}
}

// calculateThreatDamage converts enemy casualties to threat intensity damage.
// Every 5 enemies killed = 1 intensity reduction.
func calculateThreatDamage(enemiesKilled int) int {
	return enemiesKilled / 5
}

// getAllPlayerSquadIDs returns all player squad IDs from the roster.
func (es *EncounterService) getAllPlayerSquadIDs() []ecs.EntityID {
	if es.activeEncounter == nil {
		return nil
	}

	roster := squads.GetPlayerSquadRoster(es.activeEncounter.RosterOwnerID, es.manager)
	if roster != nil && len(roster.OwnedSquads) > 0 {
		return roster.OwnedSquads
	}
	return nil
}

// returnGarrisonSquadsToNode returns garrison squads to their garrison after a successful defense.
// Removes combat components but keeps the squad entities alive.
func (es *EncounterService) returnGarrisonSquadsToNode(nodeID ecs.EntityID) {
	garrisonData := garrison.GetGarrisonAtNode(es.manager, nodeID)
	if garrisonData == nil {
		return
	}
	combatpipeline.StripCombatComponents(es.manager, garrisonData.SquadIDs)
}
