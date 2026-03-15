package combat

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

// CombatExitReason describes why combat ended.
type CombatExitReason int

const (
	ExitVictory CombatExitReason = iota
	ExitDefeat
	ExitFlee
)

// String returns a human-readable name for the exit reason.
func (r CombatExitReason) String() string {
	switch r {
	case ExitVictory:
		return "Victory"
	case ExitDefeat:
		return "Defeat"
	case ExitFlee:
		return "Fled"
	default:
		return "Unknown"
	}
}

// EncounterOutcome captures the combat outcome for the exit pipeline.
// Built by the GUI layer from CombatService.CheckVictoryCondition().
type EncounterOutcome struct {
	IsPlayerVictory  bool
	VictorFaction    ecs.EntityID
	VictorName       string
	RoundsCompleted  int
	DefeatedFactions []ecs.EntityID
}

// CombatCleaner handles entity disposal when exiting combat.
// Implemented by CombatService (satisfies via Go structural typing, no import needed).
type CombatCleaner interface {
	CleanupCombat(enemySquadIDs []ecs.EntityID)
}

// EncounterCallbacks is the GUI's narrow view of encounter services.
// EncounterService structurally satisfies this interface — pass it directly.
// GUI packages import tactical/combat (not mind/encounter), preserving the dependency boundary.
type EncounterCallbacks interface {
	ExitCombat(reason CombatExitReason, result *EncounterOutcome, cleaner CombatCleaner)
	GetRosterOwnerID() ecs.EntityID
	GetCurrentEncounterID() ecs.EntityID
}
