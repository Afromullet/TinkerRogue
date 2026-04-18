package raid

import (
	"fmt"

	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// RaidCombatStarter prepares raid encounter combat.
// Calls SetupRaidFactions() to create factions and position squads,
// then returns a CombatSetup with Type=CombatTypeRaid and PostCombatReturnMode="raid".
type RaidCombatStarter struct {
	RaidEntityID     ecs.EntityID
	GarrisonSquadIDs []ecs.EntityID
	DeployedSquadIDs []ecs.EntityID
	CombatPos        coords.LogicalPosition
	CommanderID      ecs.EntityID
}

func (s *RaidCombatStarter) Prepare(manager *common.EntityManager) (*combatlifecycle.CombatSetup, error) {
	playerFactionID, enemyFactionID, err := SetupRaidFactions(
		manager, s.RaidEntityID,
		s.GarrisonSquadIDs, s.DeployedSquadIDs, s.CombatPos,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup raid factions: %w", err)
	}

	return &combatlifecycle.CombatSetup{
		PlayerFactionID:       playerFactionID,
		EnemyFactionID:        enemyFactionID,
		EnemySquadIDs:         s.GarrisonSquadIDs,
		CombatPosition:        s.CombatPos,
		EncounterID:           s.RaidEntityID,
		ThreatName:            "Garrison Raid",
		RosterOwnerID:         s.CommanderID,
		Type:                  combatlifecycle.CombatTypeRaid,
		PostCombatReturnMode:  combatlifecycle.PostCombatReturnRaid,
		SkipServiceResolution: true,
	}, nil
}
