package encounter

import (
	"fmt"
	"math"

	"game_main/common"
	"game_main/mind/combatpipeline"
	"game_main/mind/evaluation"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/visual/rendering"
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
	hiddenRenderable *rendering.Renderable
}

func (s *OverworldCombatStarter) Prepare(manager *common.EntityManager) (*combatpipeline.CombatSetup, error) {
	if s.EncounterID == 0 {
		return nil, fmt.Errorf("invalid encounter ID: 0")
	}

	// Validate encounter entity exists
	encounterEntity := manager.FindEntityByID(s.EncounterID)
	if encounterEntity == nil {
		return nil, fmt.Errorf("encounter entity %d not found", s.EncounterID)
	}
	encounterData := common.GetComponentType[*core.OverworldEncounterData](encounterEntity, core.OverworldEncounterComponent)
	if encounterData == nil {
		return nil, fmt.Errorf("encounter %d missing core.OverworldEncounterData", s.EncounterID)
	}

	fmt.Printf("OverworldCombatStarter: Preparing encounter %d (%s)\n", s.EncounterID, s.ThreatName)

	// Hide encounter sprite during combat (tracked for rollback)
	renderable := common.GetComponentType[*rendering.Renderable](
		encounterEntity,
		rendering.RenderableComponent,
	)
	if renderable != nil {
		renderable.Visible = false
		s.hiddenRenderable = renderable
		fmt.Println("Hiding overworld encounter sprite during combat")
	}

	// Spawn enemies using balanced encounter system
	fmt.Println("Starting combat encounter - spawning entities")
	enemySquadIDs, playerFactionID, enemyFactionID, err := SpawnCombatEntities(manager, s.RosterOwnerID, s.PlayerPos, encounterData, s.EncounterID)
	if err != nil {
		// Rollback sprite hiding on spawn failure
		if renderable != nil {
			renderable.Visible = true
		}
		return nil, fmt.Errorf("failed to spawn enemies: %w", err)
	}
	fmt.Printf("Spawned %d enemy squads: %v\n", len(enemySquadIDs), enemySquadIDs)

	return &combatpipeline.CombatSetup{
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
		EnemySquadIDs:   enemySquadIDs,
		CombatPosition:  s.PlayerPos,
		EncounterID:     s.EncounterID,
		ThreatID:        s.ThreatID,
		ThreatName:      s.ThreatName,
		RosterOwnerID:   s.RosterOwnerID,
	}, nil
}

// Rollback restores sprite visibility if TransitionToCombat fails after Prepare.
// Satisfies combatpipeline.CombatStartRollback.
func (s *OverworldCombatStarter) Rollback() {
	if s.hiddenRenderable != nil {
		s.hiddenRenderable.Visible = true
		s.hiddenRenderable = nil
		fmt.Println("Rollback: Restoring overworld encounter sprite after transition failure")
	}
}

// GarrisonDefenseStarter prepares garrison defense encounters.
// Gets garrison data, creates factions, adds garrison squads to player faction,
// generates enemy squads via power budget.
type GarrisonDefenseStarter struct {
	EncounterID  ecs.EntityID
	TargetNodeID ecs.EntityID
}

func (s *GarrisonDefenseStarter) Prepare(manager *common.EntityManager) (*combatpipeline.CombatSetup, error) {
	if s.EncounterID == 0 {
		return nil, fmt.Errorf("invalid encounter ID: 0")
	}

	// Validate encounter entity
	encounterEntity := manager.FindEntityByID(s.EncounterID)
	if encounterEntity == nil {
		return nil, fmt.Errorf("encounter entity %d not found", s.EncounterID)
	}
	encounterData := common.GetComponentType[*core.OverworldEncounterData](encounterEntity, core.OverworldEncounterComponent)
	if encounterData == nil {
		return nil, fmt.Errorf("encounter %d missing data", s.EncounterID)
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

	fmt.Printf("GarrisonDefenseStarter: Preparing garrison defense at node %d\n", s.TargetNodeID)

	// Create factions
	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	playerFactionID := fm.CreateFactionWithPlayer("Garrison Defense", 1, "Player 1", s.EncounterID)
	enemyFactionID := fm.CreateFactionWithPlayer("Attacking Forces", 0, "", s.EncounterID)

	// Add garrison squads to player faction (they defend)
	garrisonPositions := generatePositionsAroundPoint(*nodePos, len(garrisonData.SquadIDs), -math.Pi/2, math.Pi/2, PlayerMinDistance, PlayerMaxDistance)
	for i, squadID := range garrisonData.SquadIDs {
		pos := garrisonPositions[i]
		if err := fm.AddSquadToFaction(playerFactionID, squadID, pos); err != nil {
			return nil, fmt.Errorf("failed to add garrison squad %d: %w", squadID, err)
		}
		EnsureUnitPositions(manager, squadID, pos)
		combat.CreateActionStateForSquad(manager, squadID)

		// Mark squad as deployed for combat
		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
		if squadData != nil {
			squadData.IsDeployed = true
		}
	}

	// Calculate attacker power from garrison strength (not roster owner)
	powerConfig := evaluation.GetPowerConfigByProfile(DefaultPowerProfile)
	totalGarrisonPower := 0.0
	for _, squadID := range garrisonData.SquadIDs {
		totalGarrisonPower += evaluation.CalculateSquadPower(squadID, manager, powerConfig)
	}
	avgGarrisonPower := totalGarrisonPower / float64(len(garrisonData.SquadIDs))

	difficultyMod := getDifficultyModifier(encounterData.Level)
	targetEnemyPower := avgGarrisonPower * difficultyMod.PowerMultiplier
	if avgGarrisonPower <= 0.0 {
		targetEnemyPower = difficultyMod.MinTargetPower
	}
	if targetEnemyPower > difficultyMod.MaxTargetPower {
		targetEnemyPower = difficultyMod.MaxTargetPower
	}

	fmt.Printf("Garrison defense: avg garrison power %.2f, target enemy power %.2f\n",
		avgGarrisonPower, targetEnemyPower)

	enemySquadSpecs := generateEnemySquadsByPower(
		manager, targetEnemyPower, difficultyMod, encounterData, *nodePos, powerConfig,
	)

	enemySquadIDs := make([]ecs.EntityID, 0, len(enemySquadSpecs))
	for i, enemySpec := range enemySquadSpecs {
		if err := fm.AddSquadToFaction(enemyFactionID, enemySpec.SquadID, enemySpec.Position); err != nil {
			return nil, fmt.Errorf("failed to add enemy squad %d: %w", i, err)
		}
		combat.CreateActionStateForSquad(manager, enemySpec.SquadID)
		enemySquadIDs = append(enemySquadIDs, enemySpec.SquadID)
	}

	return &combatpipeline.CombatSetup{
		PlayerFactionID:   playerFactionID,
		EnemyFactionID:    enemyFactionID,
		EnemySquadIDs:     enemySquadIDs,
		CombatPosition:    *nodePos,
		EncounterID:       s.EncounterID,
		ThreatID:          s.TargetNodeID,
		ThreatName:        encounterData.Name,
		RosterOwnerID:     0,
		IsGarrisonDefense: true,
		DefendedNodeID:    s.TargetNodeID,
	}, nil
}
