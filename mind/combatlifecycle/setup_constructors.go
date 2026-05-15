package combatlifecycle

import (
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// NewOverworldSetup builds a CombatSetup for a standard overworld threat encounter.
// Type is fixed to CombatTypeOverworld; per-type-only DefendedNodeID stays at the
// zero value so it cannot be set for the wrong type. The post-combat return mode
// is derived from Type via CombatSetup.PostCombatReturnMode().
func NewOverworldSetup(
	playerFactionID ecs.EntityID,
	enemyFactionID ecs.EntityID,
	enemySquadIDs []ecs.EntityID,
	combatPosition coords.LogicalPosition,
	encounterID ecs.EntityID,
	threatID ecs.EntityID,
	threatName string,
	rosterOwnerID ecs.EntityID,
	resolver CombatResolver,
) *CombatSetup {
	return &CombatSetup{
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
		EnemySquadIDs:   enemySquadIDs,
		CombatPosition:  combatPosition,
		EncounterID:     encounterID,
		ThreatID:        threatID,
		ThreatName:      threatName,
		RosterOwnerID:   rosterOwnerID,
		Type:            CombatTypeOverworld,
		Resolver:        resolver,
	}
}

// NewGarrisonSetup builds a CombatSetup for a garrison defense encounter.
// RosterOwnerID stays 0 (garrison defenders are not commander-owned).
func NewGarrisonSetup(
	playerFactionID ecs.EntityID,
	enemyFactionID ecs.EntityID,
	enemySquadIDs []ecs.EntityID,
	combatPosition coords.LogicalPosition,
	encounterID ecs.EntityID,
	threatID ecs.EntityID,
	threatName string,
	defendedNodeID ecs.EntityID,
	resolver CombatResolver,
) *CombatSetup {
	return &CombatSetup{
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
		EnemySquadIDs:   enemySquadIDs,
		CombatPosition:  combatPosition,
		EncounterID:     encounterID,
		ThreatID:        threatID,
		ThreatName:      threatName,
		RosterOwnerID:   0,
		Type:            CombatTypeGarrisonDefense,
		DefendedNodeID:  defendedNodeID,
		Resolver:        resolver,
	}
}

// NewRaidSetup builds a CombatSetup for a raid room encounter.
// CombatSetup.PostCombatReturnMode() derives PostCombatReturnRaid from Type,
// so the post-combat flow returns to raid mode rather than the default
// exploration mode.
func NewRaidSetup(
	playerFactionID ecs.EntityID,
	enemyFactionID ecs.EntityID,
	enemySquadIDs []ecs.EntityID,
	combatPosition coords.LogicalPosition,
	encounterID ecs.EntityID,
	threatName string,
	rosterOwnerID ecs.EntityID,
	resolver CombatResolver,
) *CombatSetup {
	return &CombatSetup{
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
		EnemySquadIDs:   enemySquadIDs,
		CombatPosition:  combatPosition,
		EncounterID:     encounterID,
		ThreatName:      threatName,
		RosterOwnerID:   rosterOwnerID,
		Type:            CombatTypeRaid,
		Resolver:        resolver,
	}
}
