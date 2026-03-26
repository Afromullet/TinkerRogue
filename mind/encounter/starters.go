package encounter

import (
	"fmt"
	"math"

	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/mind/evaluation"
	"game_main/overworld/garrison"
	"game_main/tactical/combat/combatcore"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// OverworldCombatStarter prepares standard overworld threat encounters.
// Validates encounter entity, hides sprite, spawns combat entities via SpawnCombatEntities.
type OverworldCombatStarter struct {
	EncounterID   ecs.EntityID
	ThreatID      ecs.EntityID
	ThreatName    string
	PlayerPos     coords.LogicalPosition
	RosterOwnerID ecs.EntityID

	// Set by Prepare for rollback if TransitionToCombat fails
	hiddenRenderable *common.Renderable
}

func (s *OverworldCombatStarter) Prepare(manager *common.EntityManager) (*combatcore.CombatSetup, error) {
	encounterEntity, encounterData, err := combatlifecycle.ValidateEncounterEntity(manager, s.EncounterID)
	if err != nil {
		return nil, err
	}

	// Hide encounter sprite during combat (tracked for rollback)
	renderable := common.GetComponentType[*common.Renderable](
		encounterEntity,
		common.RenderableComponent,
	)
	if renderable != nil {
		renderable.Visible = false
		s.hiddenRenderable = renderable
	}

	// Check if the threat node has an NPC garrison; if so, use garrison squads directly
	var spawnResult *SpawnResult
	garrisonData := getGarrisonForEncounter(manager, encounterData)
	if garrisonData != nil {
		spawnResult, err = spawnGarrisonEncounter(manager, s.RosterOwnerID, s.PlayerPos, garrisonData, s.EncounterID)
	} else {
		spawnResult, err = SpawnCombatEntities(manager, s.RosterOwnerID, s.PlayerPos, encounterData, s.EncounterID)
	}
	if err != nil {
		// Rollback sprite hiding on spawn failure
		if renderable != nil {
			renderable.Visible = true
		}
		return nil, fmt.Errorf("failed to spawn enemies: %w", err)
	}
	return &combatcore.CombatSetup{
		PlayerFactionID: spawnResult.PlayerFactionID,
		EnemyFactionID:  spawnResult.EnemyFactionID,
		EnemySquadIDs:   spawnResult.EnemySquadIDs,
		CombatPosition:  s.PlayerPos,
		EncounterID:     s.EncounterID,
		ThreatID:        s.ThreatID,
		ThreatName:      s.ThreatName,
		RosterOwnerID:   s.RosterOwnerID,
	}, nil
}

// Rollback restores sprite visibility if TransitionToCombat fails after Prepare.
// Satisfies combat.CombatStartRollback.
func (s *OverworldCombatStarter) Rollback() {
	if s.hiddenRenderable != nil {
		s.hiddenRenderable.Visible = true
		s.hiddenRenderable = nil
	}
}

// GarrisonDefenseStarter prepares garrison defense encounters.
// Gets garrison data, creates factions, adds garrison squads to player faction,
// generates enemy squads via power budget.
type GarrisonDefenseStarter struct {
	EncounterID  ecs.EntityID
	TargetNodeID ecs.EntityID
}

func (s *GarrisonDefenseStarter) Prepare(manager *common.EntityManager) (*combatcore.CombatSetup, error) {
	_, encounterData, err := combatlifecycle.ValidateEncounterEntity(manager, s.EncounterID)
	if err != nil {
		return nil, err
	}

	// Get garrison data
	garrisonData := garrison.GetGarrisonAtNode(manager, s.TargetNodeID)
	if garrisonData == nil || len(garrisonData.SquadIDs) == 0 {
		return nil, fmt.Errorf("no garrison at node %d", s.TargetNodeID)
	}

	// Get node position for combat
	nodeEntity := manager.FindEntityByID(s.TargetNodeID)
	if nodeEntity == nil {
		return nil, fmt.Errorf("node entity %d not found", s.TargetNodeID)
	}
	nodePos := common.GetComponentType[*coords.LogicalPosition](nodeEntity, common.PositionComponent)
	if nodePos == nil {
		return nil, fmt.Errorf("node %d has no position", s.TargetNodeID)
	}

	// Create factions
	fm, playerFactionID, enemyFactionID := combatlifecycle.CreateFactionPair(manager, "Garrison Defense", "Attacking Forces", s.EncounterID)

	// Add garrison squads to player faction (they defend)
	garrisonPositions := generatePositionsAroundPoint(*nodePos, len(garrisonData.SquadIDs), -math.Pi/2, math.Pi/2, PlayerMinDistance, PlayerMaxDistance)
	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, playerFactionID, garrisonData.SquadIDs, garrisonPositions, true); err != nil {
		return nil, fmt.Errorf("failed to add garrison squads: %w", err)
	}

	// Calculate attacker power from garrison strength (not roster owner)
	powerConfig := evaluation.GetPowerConfigByProfile(DefaultPowerProfile)
	difficultyMod := getDifficultyModifier(encounterData.Level)
	targetEnemyPower := calculateTargetPower(manager, garrisonData.SquadIDs, powerConfig, difficultyMod)

	enemySquadSpecs := generateEnemySquadsByPower(
		manager, targetEnemyPower, difficultyMod, encounterData, *nodePos, powerConfig,
	)

	enemySquadIDs := make([]ecs.EntityID, len(enemySquadSpecs))
	enemyPositions := make([]coords.LogicalPosition, len(enemySquadSpecs))
	for i, spec := range enemySquadSpecs {
		enemySquadIDs[i] = spec.SquadID
		enemyPositions[i] = spec.Position
	}
	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, enemyFactionID, enemySquadIDs, enemyPositions, false); err != nil {
		return nil, fmt.Errorf("failed to add enemy squads: %w", err)
	}

	return &combatcore.CombatSetup{
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
		EnemySquadIDs:   enemySquadIDs,
		CombatPosition:  *nodePos,
		EncounterID:     s.EncounterID,
		ThreatID:        s.TargetNodeID,
		ThreatName:      encounterData.Name,
		RosterOwnerID:   0,
		Type:            combatcore.CombatTypeGarrisonDefense,
		DefendedNodeID:  s.TargetNodeID,
	}, nil
}
