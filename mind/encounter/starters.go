package encounter

import (
	"fmt"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/mind/spawning"
	"game_main/campaign/overworld/garrison"
	"game_main/core/coords"

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
}

func (s *OverworldCombatStarter) Prepare(manager *common.EntityManager) (*combatlifecycle.CombatSetup, func(), error) {
	encounterEntity, encounterData, err := ValidateEncounterEntity(manager, s.EncounterID)
	if err != nil {
		return nil, nil, err
	}

	// Hide encounter sprite during combat. The closure captures the renderable
	// so a later TransitionToCombat failure (or a Prepare-side spawn failure)
	// can restore it without per-struct state.
	renderable := common.GetComponentType[*common.Renderable](
		encounterEntity,
		common.RenderableComponent,
	)
	if renderable != nil {
		renderable.Visible = false
	}
	rollback := func() {
		if renderable != nil {
			renderable.Visible = true
		}
	}

	spawnResult, err := SpawnCombatEntities(manager, s.RosterOwnerID, s.PlayerPos, encounterData, s.EncounterID)
	if err != nil {
		rollback()
		return nil, nil, fmt.Errorf("failed to spawn enemies: %w", err)
	}
	threatNodeID := encounterData.ThreatNodeID
	enemySquadIDs := spawnResult.EnemySquadIDs

	// Debug encounters (no threat node) get a nil resolver — no resolution needed.
	var resolver combatlifecycle.CombatResolver
	if threatNodeID != 0 {
		resolver = &OverworldCombatResolver{
			ThreatNodeID:  threatNodeID,
			EnemySquadIDs: enemySquadIDs,
		}
	}

	return combatlifecycle.NewOverworldSetup(
		spawnResult.PlayerFactionID,
		spawnResult.EnemyFactionID,
		enemySquadIDs,
		s.PlayerPos,
		s.EncounterID,
		s.ThreatID,
		s.ThreatName,
		s.RosterOwnerID,
		resolver,
	), rollback, nil
}

// GarrisonDefenseStarter prepares garrison defense encounters.
// The garrison squads defend; attackers are generated via power budget against them.
type GarrisonDefenseStarter struct {
	EncounterID  ecs.EntityID
	TargetNodeID ecs.EntityID
}

func (s *GarrisonDefenseStarter) Prepare(manager *common.EntityManager) (*combatlifecycle.CombatSetup, func(), error) {
	_, encounterData, err := ValidateEncounterEntity(manager, s.EncounterID)
	if err != nil {
		return nil, nil, err
	}

	garrisonData := garrison.GetGarrisonAtNode(manager, s.TargetNodeID)
	if garrisonData == nil || len(garrisonData.SquadIDs) == 0 {
		return nil, nil, fmt.Errorf("no garrison at node %d", s.TargetNodeID)
	}

	nodeEntity := manager.FindEntityByID(s.TargetNodeID)
	if nodeEntity == nil {
		return nil, nil, fmt.Errorf("node entity %d not found", s.TargetNodeID)
	}
	nodePos := common.GetComponentType[*coords.LogicalPosition](nodeEntity, common.PositionComponent)
	if nodePos == nil {
		return nil, nil, fmt.Errorf("node %d has no position", s.TargetNodeID)
	}

	garrisonPositions := spawning.GeneratePlayerSquadPositions(*nodePos, len(garrisonData.SquadIDs))

	enemyIDs, enemyPositions, err := spawning.GenerateAttackerSquads(manager, *nodePos, garrisonData.SquadIDs, encounterData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate attackers: %w", err)
	}

	spawnResult, err := assembleCombatFactions(
		manager, s.EncounterID,
		"Garrison Defense", "Attacking Forces",
		garrisonData.SquadIDs, enemyIDs,
		garrisonPositions, enemyPositions,
		true,
	)
	if err != nil {
		return nil, nil, err
	}

	resolver := &GarrisonDefenseResolver{
		DefendedNodeID:       s.TargetNodeID,
		AttackingFactionType: encounterData.AttackingFactionType,
	}

	return combatlifecycle.NewGarrisonSetup(
		spawnResult.PlayerFactionID,
		spawnResult.EnemyFactionID,
		spawnResult.EnemySquadIDs,
		*nodePos,
		s.EncounterID,
		s.TargetNodeID,
		encounterData.Name,
		s.TargetNodeID,
		resolver,
	), nil, nil
}
