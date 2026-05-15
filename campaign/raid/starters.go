package raid

import (
	"fmt"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// RaidCombatStarter prepares raid encounter combat.
// Calls SetupRaidFactions() to create factions and position squads,
// then returns a CombatSetup with a RaidEncounterResolver attached.
type RaidCombatStarter struct {
	RaidEntityID     ecs.EntityID
	GarrisonSquadIDs []ecs.EntityID
	DeployedSquadIDs []ecs.EntityID
	CombatPos        coords.LogicalPosition
	CommanderID      ecs.EntityID
	RoomNodeID       int
}

func (s *RaidCombatStarter) Prepare(manager *common.EntityManager) (*combatlifecycle.CombatSetup, func(), error) {
	playerFactionID, enemyFactionID, err := SetupRaidFactions(
		manager, s.RaidEntityID,
		s.GarrisonSquadIDs, s.DeployedSquadIDs, s.CombatPos,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup raid factions: %w", err)
	}

	resolver := &RaidEncounterResolver{
		RaidState:  GetRaidState(manager),
		RoomNodeID: s.RoomNodeID,
	}

	return combatlifecycle.NewRaidSetup(
		playerFactionID,
		enemyFactionID,
		s.GarrisonSquadIDs,
		s.CombatPos,
		s.RaidEntityID,
		"Garrison Raid",
		s.CommanderID,
		resolver,
	), nil, nil
}
