package encounter

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/overworld/threat"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// combatOutcome describes result of a tactical battle (victory or defeat only;
// retreat is handled separately by resolveFleeToOverworld)
type combatOutcome struct {
	ThreatNodeID   ecs.EntityID
	PlayerVictory  bool
	PlayerEntityID ecs.EntityID
	PlayerSquadIDs []ecs.EntityID
	Casualties     casualtyReport
	RewardsEarned  rewardTable
}

// casualtyReport tracks units lost in combat
type casualtyReport struct {
	PlayerUnitsLost  int
	EnemyUnitsKilled int
}

// applyCombatOutcome applies combat outcome to overworld state
// This is the feedback loop from tactical combat back to strategic layer
func applyCombatOutcome(
	manager *common.EntityManager,
	outcome *combatOutcome,
) error {
	// Find threat node
	threatEntity := manager.FindEntityByID(outcome.ThreatNodeID)
	if threatEntity == nil {
		return fmt.Errorf("threat node %d not found", outcome.ThreatNodeID)
	}

	nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		return fmt.Errorf("entity is not an overworld node")
	}

	// Get current tick for event logging
	currentTick := core.GetCurrentTick(manager)

	if outcome.PlayerVictory {
		// Player won - reduce or destroy threat
		damageDealt := calculateThreatDamage(outcome.Casualties.EnemyUnitsKilled)
		oldIntensity := nodeData.Intensity
		nodeData.Intensity -= damageDealt

		if nodeData.Intensity <= 0 {
			// Destroy threat node completely
			threat.DestroyThreatNode(manager, threatEntity)

			// Grant full rewards
			grantRewards(manager, outcome, outcome.RewardsEarned)

			// Log combat resolution event
			core.LogEvent(core.EventCombatResolved, currentTick, outcome.ThreatNodeID,
				fmt.Sprintf("Combat victory - Threat %d destroyed", outcome.ThreatNodeID),
				map[string]interface{}{
					"victory":            true,
					"intensity_reduced":  oldIntensity,
					"rewards_gold":       outcome.RewardsEarned.Gold,
					"rewards_xp":         outcome.RewardsEarned.Experience,
					"player_units_lost":  outcome.Casualties.PlayerUnitsLost,
					"enemy_units_killed": outcome.Casualties.EnemyUnitsKilled,
				})

			fmt.Printf("Threat %d destroyed! Rewards: %d gold, %d XP\n",
				outcome.ThreatNodeID, outcome.RewardsEarned.Gold, outcome.RewardsEarned.Experience)
		} else {
			// Weakened but not destroyed - partial rewards
			partialRewards := rewardTable{
				Gold:       outcome.RewardsEarned.Gold / 2,
				Experience: outcome.RewardsEarned.Experience / 2,
			}
			grantRewards(manager, outcome, partialRewards)

			// Reset growth progress (player setback the threat)
			nodeData.GrowthProgress = 0.0

			// Log combat resolution event
			core.LogEvent(core.EventCombatResolved, currentTick, outcome.ThreatNodeID,
				fmt.Sprintf("Combat victory - Threat %d weakened to intensity %d", outcome.ThreatNodeID, nodeData.Intensity),
				map[string]interface{}{
					"victory":            true,
					"intensity_reduced":  damageDealt,
					"new_intensity":      nodeData.Intensity,
					"rewards_gold":       partialRewards.Gold,
					"rewards_xp":         partialRewards.Experience,
					"player_units_lost":  outcome.Casualties.PlayerUnitsLost,
					"enemy_units_killed": outcome.Casualties.EnemyUnitsKilled,
				})

			fmt.Printf("Threat %d weakened to intensity %d. Partial rewards: %d gold, %d XP\n",
				outcome.ThreatNodeID, nodeData.Intensity, partialRewards.Gold, partialRewards.Experience)
		}
	} else {
		// Player defeat - threat grows stronger
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

		// Log combat resolution event
		core.LogEvent(core.EventCombatResolved, currentTick, outcome.ThreatNodeID,
			fmt.Sprintf("Combat defeat - Threat %d grew to intensity %d", outcome.ThreatNodeID, nodeData.Intensity),
			map[string]interface{}{
				"victory":            false,
				"old_intensity":      oldIntensity,
				"new_intensity":      nodeData.Intensity,
				"player_units_lost":  outcome.Casualties.PlayerUnitsLost,
				"enemy_units_killed": outcome.Casualties.EnemyUnitsKilled,
			})

		fmt.Printf("Defeated by threat %d! Threat grew to intensity %d\n",
			outcome.ThreatNodeID, nodeData.Intensity)
	}

	return nil
}

// calculateThreatDamage converts enemy casualties to threat intensity damage
// Every 5 enemies killed = 1 intensity reduction
// TODO: Threat damage calculation may need non-linear scaling based on threat level
func calculateThreatDamage(enemiesKilled int) int {
	return enemiesKilled / 5
}

// createCombatOutcome creates outcome from combat state
// Helper function to construct outcome from combat results
func createCombatOutcome(
	threatNodeID ecs.EntityID,
	playerWon bool,
	playerEntityID ecs.EntityID,
	playerSquadIDs []ecs.EntityID,
	playerUnitsLost int,
	enemyUnitsKilled int,
	rewards rewardTable,
) *combatOutcome {
	return &combatOutcome{
		ThreatNodeID:   threatNodeID,
		PlayerVictory:  playerWon,
		PlayerEntityID: playerEntityID,
		PlayerSquadIDs: playerSquadIDs,
		Casualties: casualtyReport{
			PlayerUnitsLost:  playerUnitsLost,
			EnemyUnitsKilled: enemyUnitsKilled,
		},
		RewardsEarned: rewards,
	}
}

// resolveCombatToOverworld applies combat outcome to overworld threat state.
// Caller (EndEncounter) already validates activeEncounter != nil.
func (es *EncounterService) resolveCombatToOverworld(
	threatNodeID ecs.EntityID,
	playerVictory bool,
	victorFaction ecs.EntityID,
	defeatedFactions []ecs.EntityID,
	roundsCompleted int,
) {
	// Calculate casualties
	playerUnitsLost, enemyUnitsKilled := es.calculateCasualties(victorFaction, defeatedFactions)

	// Get all player squad IDs for reward distribution
	playerSquadIDs := es.getAllPlayerSquadIDs()

	// Calculate rewards from threat
	threatEntity := es.manager.FindEntityByID(threatNodeID)
	if threatEntity == nil {
		fmt.Printf("WARNING: Threat node %d not found for resolution\n", threatNodeID)
		return
	}

	nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		fmt.Printf("WARNING: Entity %d is not an overworld node\n", threatNodeID)
		return
	}

	// Get the specific encounter that was assigned to this node
	selectedEncounter := core.GetNodeRegistry().GetEncounterByID(nodeData.EncounterID)
	rewards := calculateRewards(nodeData.Intensity, selectedEncounter)

	// Create combat outcome
	outcome := createCombatOutcome(
		threatNodeID,
		playerVictory,
		es.activeEncounter.PlayerEntityID,
		playerSquadIDs,
		playerUnitsLost,
		enemyUnitsKilled,
		rewards,
	)

	// Apply to overworld
	if err := applyCombatOutcome(es.manager, outcome); err != nil {
		fmt.Printf("ERROR resolving combat to overworld: %v\n", err)
	} else {
		fmt.Printf("Combat resolved to overworld: %d enemy killed, %d player lost\n",
			enemyUnitsKilled, playerUnitsLost)
	}
}

// calculateCasualties counts units killed in combat.
// Caller (resolveCombatToOverworld via EndEncounter) already validates activeEncounter != nil.
func (es *EncounterService) calculateCasualties(
	victorFaction ecs.EntityID,
	defeatedFactions []ecs.EntityID,
) (playerUnitsLost int, enemyUnitsKilled int) {
	// Use cached faction IDs from encounter creation (avoids O(n) query)
	playerFactionID := es.activeEncounter.PlayerFactionID
	enemyFactionID := es.activeEncounter.EnemyFactionID

	// Count dead units in each faction
	for _, result := range es.manager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)
		if memberData == nil {
			continue
		}

		// Get squad to check faction membership
		squadEntity := es.manager.FindEntityByID(memberData.SquadID)
		if squadEntity == nil {
			continue
		}

		squadFaction := common.GetComponentType[*combat.CombatFactionData](squadEntity, combat.FactionMembershipComponent)
		if squadFaction == nil {
			continue
		}

		// Check if unit is dead
		unitAttr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if unitAttr != nil && unitAttr.CurrentHealth <= 0 {
			if squadFaction.FactionID == playerFactionID {
				playerUnitsLost++
			} else if squadFaction.FactionID == enemyFactionID {
				enemyUnitsKilled++
			}
		}
	}

	return playerUnitsLost, enemyUnitsKilled
}

// getAllPlayerSquadIDs returns all player squad IDs from the roster
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

// resolveGarrisonDefense handles the outcome of a garrison defense encounter.
// Player wins: garrison holds, node stays player-owned.
// Player loses: node ownership transfers to the attacking faction.
func (es *EncounterService) resolveGarrisonDefense(playerWon bool, encounterData *core.OverworldEncounterData) {
	if es.activeEncounter == nil {
		return
	}

	nodeID := es.activeEncounter.DefendedNodeID
	currentTick := core.GetCurrentTick(es.manager)

	if playerWon {
		core.LogEvent(core.EventGarrisonDefended, currentTick, nodeID,
			fmt.Sprintf("Garrison at node %d successfully defended against %s",
				nodeID, encounterData.AttackingFactionType.String()), nil)
		fmt.Printf("Garrison at node %d held! Defense successful.\n", nodeID)
	} else {
		// Transfer ownership to attacking faction
		newOwner := encounterData.AttackingFactionType.String()
		if err := garrison.TransferNodeOwnership(es.manager, nodeID, newOwner); err != nil {
			fmt.Printf("ERROR: Failed to transfer node ownership: %v\n", err)
		} else {
			fmt.Printf("Garrison at node %d fell. Node captured by %s.\n", nodeID, newOwner)
		}
	}
}

// resolveFleeToOverworld logs the retreat event to overworld.
// No rewards, no casualties, no threat changes.
func (es *EncounterService) resolveFleeToOverworld() {
	if es.activeEncounter == nil {
		return
	}

	_, encounterData := es.getEncounterData(es.activeEncounter.EncounterID)
	if encounterData == nil || encounterData.ThreatNodeID == 0 {
		return
	}

	threatNodeID := encounterData.ThreatNodeID

	// Log retreat event
	currentTick := core.GetCurrentTick(es.manager)

	core.LogEvent(core.EventCombatResolved, currentTick, threatNodeID,
		fmt.Sprintf("Retreated from threat %d", threatNodeID),
		map[string]interface{}{
			"victory":            false,
			"retreat":            true,
			"player_units_lost":  0,
			"enemy_units_killed": 0,
		})

	fmt.Printf("Retreated from threat %d (no changes)\n", threatNodeID)
}

// returnGarrisonSquadsToNode returns garrison squads to their garrison after a successful defense.
// Removes combat components but keeps the squad entities alive.
func (es *EncounterService) returnGarrisonSquadsToNode(nodeID ecs.EntityID) {
	garrisonData := garrison.GetGarrisonAtNode(es.manager, nodeID)
	if garrisonData == nil {
		return
	}

	for _, squadID := range garrisonData.SquadIDs {
		squadEntity := es.manager.FindEntityByID(squadID)
		if squadEntity == nil {
			continue
		}

		// Remove combat components
		if squadEntity.HasComponent(combat.FactionMembershipComponent) {
			squadEntity.RemoveComponent(combat.FactionMembershipComponent)
		}

		// Remove position (garrison squads don't need world positions)
		pos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		if pos != nil {
			common.GlobalPositionSystem.RemoveEntity(squadID, *pos)
			squadEntity.RemoveComponent(common.PositionComponent)
		}

		// Remove unit positions too
		unitIDs := squads.GetUnitIDsInSquad(squadID, es.manager)
		for _, unitID := range unitIDs {
			unitEntity := es.manager.FindEntityByID(unitID)
			if unitEntity == nil {
				continue
			}
			unitPos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			if unitPos != nil {
				common.GlobalPositionSystem.RemoveEntity(unitID, *unitPos)
				unitEntity.RemoveComponent(common.PositionComponent)
			}
		}

		// Reset deployment flag
		squadData := common.GetComponentTypeByID[*squads.SquadData](es.manager, squadID, squads.SquadComponent)
		if squadData != nil {
			squadData.IsDeployed = false
		}

		fmt.Printf("Returned garrison squad %d to node %d\n", squadID, nodeID)
	}
}
