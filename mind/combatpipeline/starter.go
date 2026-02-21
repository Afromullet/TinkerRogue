package combatpipeline

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CombatStarter handles context-specific combat initialization.
// Each combat type implements this: overworld, garrison defense, raid.
// Prepare() creates factions, positions squads, and returns a CombatSetup
// describing what was set up for the shared mode transition.
type CombatStarter interface {
	Prepare(manager *common.EntityManager) (*CombatSetup, error)
}

// CombatSetup is the unified output from Prepare().
// Contains everything ExecuteCombatStart() needs to do the mode transition.
type CombatSetup struct {
	PlayerFactionID ecs.EntityID
	EnemyFactionID  ecs.EntityID
	EnemySquadIDs   []ecs.EntityID
	CombatPosition  coords.LogicalPosition

	EncounterID   ecs.EntityID
	ThreatID      ecs.EntityID
	ThreatName    string
	RosterOwnerID ecs.EntityID // 0 for garrison defense

	IsGarrisonDefense    bool
	DefendedNodeID       ecs.EntityID
	IsRaidCombat         bool
	PostCombatReturnMode string // "" = default, "raid" = return to raid mode
}

// CombatTransitioner abstracts EncounterService for the pipeline.
// EncounterService satisfies this via Go structural typing (no explicit implements).
// This avoids combatpipeline importing encounter (which would create a cycle).
type CombatTransitioner interface {
	TransitionToCombat(setup *CombatSetup) error
}

// CombatStartRollback is an optional interface for starters that need cleanup
// if TransitionToCombat fails after Prepare succeeds (e.g., restoring sprite visibility).
// Starters that don't need rollback simply don't implement this.
type CombatStartRollback interface {
	Rollback()
}

// CombatStartResult is the output from ExecuteCombatStart.
type CombatStartResult struct {
	PlayerFactionID ecs.EntityID
	EnemyFactionID  ecs.EntityID
	EnemySquadIDs   []ecs.EntityID
}

// ExecuteCombatStart is THE single entry point for all combat initiation.
func ExecuteCombatStart(
	transitioner CombatTransitioner,
	manager *common.EntityManager,
	starter CombatStarter,
) (*CombatStartResult, error) {
	setup, err := starter.Prepare(manager)
	if err != nil {
		return nil, err
	}
	if err := transitioner.TransitionToCombat(setup); err != nil {
		// Let the starter undo any side effects from Prepare (e.g., hidden sprites)
		if rb, ok := starter.(CombatStartRollback); ok {
			rb.Rollback()
		}
		return nil, err
	}
	return &CombatStartResult{
		PlayerFactionID: setup.PlayerFactionID,
		EnemyFactionID:  setup.EnemyFactionID,
		EnemySquadIDs:   setup.EnemySquadIDs,
	}, nil
}
