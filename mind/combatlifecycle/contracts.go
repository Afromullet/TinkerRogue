// Package combatlifecycle owns the combat lifecycle vocabulary: the contracts
// that describe how combat is started, transitioned, resolved, and cleaned up,
// plus the orchestration entry points (ExecuteCombatStart, ExecuteResolution).
//
// Implementations of these contracts live in their domain packages by design:
//   - mind/encounter: overworld and garrison-defense starters/resolvers, EncounterService transitioner
//   - mind/raid: raid starters/resolvers
//   - tactical/combat/combatservices: CombatService (cleaner)
//
// Domain knowledge stays with the domain; this package only defines the shared contracts.
package combatlifecycle

import (
	"game_main/core/common"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// CombatStarter handles context-specific combat initialization.
// Each combat type implements this: overworld, garrison defense, raid.
// Prepare() creates factions, positions squads, and returns a CombatSetup
// describing what was set up for the shared mode transition.
type CombatStarter interface {
	Prepare(manager *common.EntityManager) (*CombatSetup, error)
}

// CombatType distinguishes combat encounter types.
// Replaces the IsGarrisonDefense/IsRaidCombat bool flags to prevent invalid states.
type CombatType int

const (
	CombatTypeOverworld       CombatType = iota // Standard overworld threat encounter
	CombatTypeGarrisonDefense                   // Defending a garrisoned node
	CombatTypeRaid                              // Raid room encounter
	CombatTypeDebug                             // Debug/test encounters
)

// String returns a human-readable name for the combat type.
func (ct CombatType) String() string {
	switch ct {
	case CombatTypeOverworld:
		return "Overworld"
	case CombatTypeGarrisonDefense:
		return "GarrisonDefense"
	case CombatTypeRaid:
		return "Raid"
	case CombatTypeDebug:
		return "Debug"
	default:
		return "Unknown"
	}
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

	Type                 CombatType
	DefendedNodeID       ecs.EntityID
	PostCombatReturnMode string // "" = default, PostCombatReturnRaid = return to raid mode

	// SkipServiceResolution indicates the starter's domain handles resolution
	// via its own post-combat callback (e.g., RaidRunner). When true,
	// EncounterService.ExitCombat skips resolveEncounterOutcome and markEncounterDefeated.
	SkipServiceResolution bool
}

// PostCombatReturnMode constants for compile-time safety.
const (
	PostCombatReturnDefault = ""     // Return to default mode (exploration/overworld)
	PostCombatReturnRaid    = "raid" // Return to raid mode
)

// CombatTransitioner abstracts EncounterService for the pipeline.
// EncounterService satisfies this via Go structural typing (no explicit implements).
type CombatTransitioner interface {
	TransitionToCombat(setup *CombatSetup) error
}

// CombatStartRollback is an optional interface for starters that need cleanup
// if TransitionToCombat fails after Prepare succeeds (e.g., restoring sprite visibility).
// Starters that don't need rollback simply don't implement this.
type CombatStartRollback interface {
	Rollback()
}

// CombatExitReason describes why combat ended.
type CombatExitReason int

const (
	ExitVictory CombatExitReason = iota
	ExitDefeat
	ExitFlee
)

// DetermineExitReason maps combat-end state to the lifecycle exit enum.
// Flee takes precedence over victory state: a player who wins on the same tick
// they pressed flee is still treated as fleeing.
func DetermineExitReason(fleeRequested, playerVictory bool) CombatExitReason {
	switch {
	case fleeRequested:
		return ExitFlee
	case playerVictory:
		return ExitVictory
	default:
		return ExitDefeat
	}
}

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

// CombatCleaner handles tactical-side entity disposal when exiting combat.
// Implemented by CombatService (satisfies via Go structural typing, no import needed).
// Returns the player squad IDs that were in combat so the caller can finish
// cross-cutting cleanup (stripping FactionMembership, PerkRoundState, etc. via
// StripCombatComponents) without tactical/combat depending on mind/.
type CombatCleaner interface {
	CleanupCombat(enemySquadIDs []ecs.EntityID) []ecs.EntityID
}

// EncounterCallbacks is the GUI's narrow view of encounter services.
// EncounterService structurally satisfies this interface — pass it directly.
type EncounterCallbacks interface {
	ExitCombat(reason CombatExitReason, result *EncounterOutcome, cleaner CombatCleaner)
	GetRosterOwnerID() ecs.EntityID
	GetCurrentEncounterID() ecs.EntityID
}
