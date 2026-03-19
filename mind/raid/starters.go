package raid

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/combat"
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

func (s *RaidCombatStarter) Prepare(manager *common.EntityManager) (*combat.CombatSetup, error) {
	playerFactionID, enemyFactionID, err := SetupRaidFactions(
		manager, s.RaidEntityID,
		s.GarrisonSquadIDs, s.DeployedSquadIDs, s.CombatPos,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup raid factions: %w", err)
	}

	return &combat.CombatSetup{
		PlayerFactionID:      playerFactionID,
		EnemyFactionID:       enemyFactionID,
		EnemySquadIDs:        s.GarrisonSquadIDs,
		CombatPosition:       s.CombatPos,
		EncounterID:          s.RaidEntityID,
		ThreatName:           "Garrison Raid",
		RosterOwnerID:        s.CommanderID,
		Type:                 combat.CombatTypeRaid,
		PostCombatReturnMode: "raid",
	}, nil
}
