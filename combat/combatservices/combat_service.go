package combatservices

import (
	"fmt"
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// CombatService encapsulates all combat game logic and system ownership
type CombatService struct {
	entityManager   *common.EntityManager
	turnManager     *combat.TurnManager
	factionManager  *combat.FactionManager
	movementSystem  *combat.CombatMovementSystem
	combatCache     *combat.CombatQueryCache
	combatActSystem *combat.CombatActionSystem
}

// NewCombatService creates a new combat service
func NewCombatService(manager *common.EntityManager) *CombatService {
	return &CombatService{
		entityManager:   manager,
		turnManager:     combat.NewTurnManager(manager),
		factionManager:  combat.NewFactionManager(manager),
		movementSystem:  combat.NewMovementSystem(manager, common.GlobalPositionSystem),
		combatCache:     combat.NewCombatQueryCache(manager),
		combatActSystem: combat.NewCombatActionSystem(manager), // Create once, reuse for all attacks
	}
}

// GetMovementSystem returns the movement system for command pattern integration
func (cs *CombatService) GetMovementSystem() *combat.CombatMovementSystem {
	return cs.movementSystem
}

// GetCombatActionSystem returns the combat action system for executing attacks
func (cs *CombatService) GetCombatActionSystem() *combat.CombatActionSystem {
	return cs.combatActSystem
}

// GetTurnManager returns the turn manager for turn operations
func (cs *CombatService) GetTurnManager() *combat.TurnManager {
	return cs.turnManager
}

// InitializeCombat initializes combat with the given factions
// Also assigns any unassigned squads (from squad deployment) to the player faction.
// TODO: Assinging unassigned squads to the player faction is a temporary fix. remove.
func (cs *CombatService) InitializeCombat(factionIDs []ecs.EntityID) error {
	// Find player faction (has IsPlayerControlled = true)
	var playerFactionID ecs.EntityID
	for _, factionID := range factionIDs {
		// Use cached query for performance
		factionData := cs.combatCache.FindFactionDataByID(factionID, cs.entityManager)
		if factionData != nil && factionData.IsPlayerControlled {
			playerFactionID = factionID
			break
		}
	}

	// Assign any unassigned squads to player faction
	// These are squads deployed via SquadDeploymentMode that have positions but no CombatFactionComponent
	if playerFactionID != 0 {
		cs.assignDeployedSquadsToPlayerFaction(playerFactionID)
	}

	return cs.turnManager.InitializeCombat(factionIDs)
}

// assignDeployedSquadsToPlayerFaction finds all squads with positions but no CombatFactionComponent
// and assigns them to the player faction. These are squads that were deployed via SquadDeploymentMode.
// TODO: Assinging unassigned squads to the player faction is a temporary fix. Squads will have to be assigned to the
// Correct Faction. There can be multiple players
func (cs *CombatService) assignDeployedSquadsToPlayerFaction(playerFactionID ecs.EntityID) {
	for _, result := range cs.entityManager.World.Query(squads.SquadTag) {
		squadEntity := result.Entity
		squadID := squadEntity.GetID()

		// Check if squad already has a faction (skip if it does)
		combatFaction := common.GetComponentType[*combat.CombatFactionData](squadEntity, combat.CombatFactionComponent)
		if combatFaction != nil {
			continue // Already assigned to a faction
		}

		// Check if squad has a position (deployed squads have positions)
		position := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		if position == nil {
			continue // No position, not a deployed squad
		}

		// Squad is unassigned and deployed - add it to player faction
		fm := combat.NewFactionManager(cs.entityManager)
		if err := fm.AddSquadToFaction(playerFactionID, squadID, *position); err != nil {
			// Log error but continue with other squads
			continue
		}
	}
}

// GetAliveSquadsInFaction returns all alive squads for a faction
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
	squadIDs := cs.factionManager.GetFactionSquads(factionID)
	result := []ecs.EntityID{}
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, cs.entityManager) {
			result = append(result, squadID)
		}
	}
	return result
}

// VictoryCheckResult contains battle outcome information.
type VictoryCheckResult struct {
	BattleOver       bool
	VictorFaction    ecs.EntityID
	VictorName       string
	DefeatedFactions []ecs.EntityID
	RoundsCompleted  int
}

// CheckVictoryCondition checks if battle has ended
// TODO: Add actual victory conditions
func (cs *CombatService) CheckVictoryCondition() *VictoryCheckResult {
	result := &VictoryCheckResult{
		RoundsCompleted: cs.turnManager.GetCurrentRound(),
	}

	// Count alive squads per faction
	aliveByFaction := make(map[ecs.EntityID]int)

	for _, queryResult := range cs.entityManager.World.Query(combat.FactionTag) {
		entity := queryResult.Entity
		factionID := entity.GetID()
		aliveByFaction[factionID] = 0
	}

	// Count squads
	for _, queryResult := range cs.entityManager.World.Query(squads.SquadTag) {
		entity := queryResult.Entity
		squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
		if squadData != nil && !squads.IsSquadDestroyed(entity.GetID(), cs.entityManager) {
			factionData := common.GetComponentTypeByID[*combat.FactionData](
				cs.entityManager, entity.GetID(), combat.FactionComponent)
			if factionData != nil {
				aliveByFaction[factionData.FactionID]++
			}
		}
	}

	// Check victory: only one faction with alive squads
	factionsWithSquads := 0
	var victorFaction ecs.EntityID
	for factionID, count := range aliveByFaction {
		if count > 0 {
			factionsWithSquads++
			victorFaction = factionID
		} else {
			result.DefeatedFactions = append(result.DefeatedFactions, factionID)
		}
	}

	if factionsWithSquads <= 1 {
		result.BattleOver = true
		result.VictorFaction = victorFaction

		// Get faction data to include player name
		factionData := cs.combatCache.FindFactionDataByID(victorFaction, cs.entityManager)
		if factionData != nil {
			if factionData.PlayerID > 0 {
				// Player victory - include player name
				result.VictorName = fmt.Sprintf("%s (%s)", factionData.Name, factionData.PlayerName)
			} else {
				// AI victory
				result.VictorName = factionData.Name
			}
		} else {
			result.VictorName = "Unknown"
		}
	}

	return result
}

// getSquadNameByID is a helper to get squad name from ID
func getSquadNameByID(squadID ecs.EntityID, manager *common.EntityManager) string {
	squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
	if squadData != nil {
		return squadData.Name
	}
	return "Unknown"
}
