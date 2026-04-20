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

	// Set by Prepare for rollback if TransitionToCombat fails
	hiddenRenderable *common.Renderable
}

func (s *OverworldCombatStarter) Prepare(manager *common.EntityManager) (*combatlifecycle.CombatSetup, error) {
	encounterEntity, encounterData, err := ValidateEncounterEntity(manager, s.EncounterID)
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

	spawnResult, err := SpawnCombatEntities(manager, s.RosterOwnerID, s.PlayerPos, encounterData, s.EncounterID)
	if err != nil {
		// Rollback sprite hiding on spawn failure
		if renderable != nil {
			renderable.Visible = true
		}
		return nil, fmt.Errorf("failed to spawn enemies: %w", err)
	}
	return &combatlifecycle.CombatSetup{
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
// The garrison squads defend; attackers are generated via power budget against them.
type GarrisonDefenseStarter struct {
	EncounterID  ecs.EntityID
	TargetNodeID ecs.EntityID
}

func (s *GarrisonDefenseStarter) Prepare(manager *common.EntityManager) (*combatlifecycle.CombatSetup, error) {
	_, encounterData, err := ValidateEncounterEntity(manager, s.EncounterID)
	if err != nil {
		return nil, err
	}

	garrisonData := garrison.GetGarrisonAtNode(manager, s.TargetNodeID)
	if garrisonData == nil || len(garrisonData.SquadIDs) == 0 {
		return nil, fmt.Errorf("no garrison at node %d", s.TargetNodeID)
	}

	nodeEntity := manager.FindEntityByID(s.TargetNodeID)
	if nodeEntity == nil {
		return nil, fmt.Errorf("node entity %d not found", s.TargetNodeID)
	}
	nodePos := common.GetComponentType[*coords.LogicalPosition](nodeEntity, common.PositionComponent)
	if nodePos == nil {
		return nil, fmt.Errorf("node %d has no position", s.TargetNodeID)
	}

	garrisonPositions := spawning.GeneratePlayerSquadPositions(*nodePos, len(garrisonData.SquadIDs))

	enemyIDs, enemyPositions, err := spawning.GenerateAttackerSquads(manager, *nodePos, garrisonData.SquadIDs, encounterData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate attackers: %w", err)
	}

	spawnResult, err := assembleCombatFactions(
		manager, s.EncounterID,
		"Garrison Defense", "Attacking Forces",
		garrisonData.SquadIDs, enemyIDs,
		garrisonPositions, enemyPositions,
		true,
	)
	if err != nil {
		return nil, err
	}

	return &combatlifecycle.CombatSetup{
		PlayerFactionID: spawnResult.PlayerFactionID,
		EnemyFactionID:  spawnResult.EnemyFactionID,
		EnemySquadIDs:   spawnResult.EnemySquadIDs,
		CombatPosition:  *nodePos,
		EncounterID:     s.EncounterID,
		ThreatID:        s.TargetNodeID,
		ThreatName:      encounterData.Name,
		RosterOwnerID:   0,
		Type:            combatlifecycle.CombatTypeGarrisonDefense,
		DefendedNodeID:  s.TargetNodeID,
	}, nil
}
