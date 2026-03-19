package encounter

import (
	"fmt"
	"math"

	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/mind/evaluation"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/tactical/combat"
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

func (s *OverworldCombatStarter) Prepare(manager *common.EntityManager) (*combat.CombatSetup, error) {
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

	// Hide encounter sprite during combat (tracked for rollback)
	renderable := common.GetComponentType[*common.Renderable](
		encounterEntity,
		common.RenderableComponent,
	)
	if renderable != nil {
		renderable.Visible = false
		s.hiddenRenderable = renderable
	}

	// Spawn enemies using balanced encounter system
	enemySquadIDs, playerFactionID, enemyFactionID, err := SpawnCombatEntities(manager, s.RosterOwnerID, s.PlayerPos, encounterData, s.EncounterID)
	if err != nil {
		// Rollback sprite hiding on spawn failure
		if renderable != nil {
			renderable.Visible = true
		}
		return nil, fmt.Errorf("failed to spawn enemies: %w", err)
	}
	return &combat.CombatSetup{
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

func (s *GarrisonDefenseStarter) Prepare(manager *common.EntityManager) (*combat.CombatSetup, error) {
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

	// Create factions
	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	playerFactionID, enemyFactionID := fm.CreateStandardFactions("Garrison Defense", "Attacking Forces", s.EncounterID)

	// Add garrison squads to player faction (they defend)
	garrisonPositions := generatePositionsAroundPoint(*nodePos, len(garrisonData.SquadIDs), -math.Pi/2, math.Pi/2, PlayerMinDistance, PlayerMaxDistance)
	for i, squadID := range garrisonData.SquadIDs {
		pos := garrisonPositions[i]
		if err := combatlifecycle.EnrollSquadInFaction(fm, manager, playerFactionID, squadID, pos, true); err != nil {
			return nil, fmt.Errorf("failed to add garrison squad %d: %w", squadID, err)
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

	enemySquadSpecs := generateEnemySquadsByPower(
		manager, targetEnemyPower, difficultyMod, encounterData, *nodePos, powerConfig,
	)

	enemySquadIDs := make([]ecs.EntityID, 0, len(enemySquadSpecs))
	for i, enemySpec := range enemySquadSpecs {
		if err := combatlifecycle.EnrollSquadInFaction(fm, manager, enemyFactionID, enemySpec.SquadID, enemySpec.Position, false); err != nil {
			return nil, fmt.Errorf("failed to add enemy squad %d: %w", i, err)
		}
		enemySquadIDs = append(enemySquadIDs, enemySpec.SquadID)
	}

	return &combat.CombatSetup{
		PlayerFactionID:  playerFactionID,
		EnemyFactionID:   enemyFactionID,
		EnemySquadIDs:    enemySquadIDs,
		CombatPosition:   *nodePos,
		EncounterID:      s.EncounterID,
		ThreatID:         s.TargetNodeID,
		ThreatName:       encounterData.Name,
		RosterOwnerID:    0,
		Type:             combat.CombatTypeGarrisonDefense,
		DefendedNodeID:   s.TargetNodeID,
	}, nil
}
