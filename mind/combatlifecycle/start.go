package combatlifecycle

import (
	"game_main/core/common"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// CombatStarter handles context-specific combat initialization.
// Each combat type implements this: overworld, garrison defense, raid.
//
// Prepare creates factions, positions squads, and returns a CombatSetup
// describing what was set up for the shared mode transition.
//
// The optional rollback closure undoes any side effects Prepare applied
// (e.g., hidden encounter sprites) when TransitionToCombat fails after
// Prepare has already succeeded. Starters with no side effects to undo
// return nil. The closure is invoked by ExecuteCombatStart at most once,
// before the encounter entity is torn down, so any pointers it captures
// are guaranteed to be live at call time.
type CombatStarter interface {
	Prepare(manager *common.EntityManager) (*CombatSetup, func(), error)
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

	Type           CombatType
	DefendedNodeID ecs.EntityID

	// Resolver is the type-appropriate resolver, built eagerly by the starter's Prepare().
	// Runtime information (PlayerVictory, PlayerEntityID, PlayerSquadIDs) is passed to
	// Resolve via ResolutionContext at exit time. Nil means no resolution needed (e.g., debug).
	Resolver CombatResolver
}

// PostCombatReturnMode constants for compile-time safety.
const (
	PostCombatReturnDefault = ""     // Return to default mode (exploration/overworld)
	PostCombatReturnRaid    = "raid" // Return to raid mode
)

// PostCombatReturnMode returns the mode key the GUI should transition into after
// combat ends. Derived from Type: raid encounters return to raid mode; everything
// else falls back to the default (exploration/overworld). Keeping this as a
// derived value prevents Type and the return mode from drifting out of sync.
func (s *CombatSetup) PostCombatReturnMode() string {
	if s.Type == CombatTypeRaid {
		return PostCombatReturnRaid
	}
	return PostCombatReturnDefault
}

// CombatTransitioner abstracts EncounterService for the pipeline.
// EncounterService satisfies this via Go structural typing (no explicit implements).
type CombatTransitioner interface {
	TransitionToCombat(setup *CombatSetup) error
}

// ExecuteCombatStart is THE single entry point for all combat initiation.
// Sequences starter.Prepare → transitioner.TransitionToCombat. If the transition
// fails after Prepare succeeded, the rollback closure returned by Prepare
// (when non-nil) is invoked to undo any side effects (e.g., hidden sprites).
func ExecuteCombatStart(
	transitioner CombatTransitioner,
	manager *common.EntityManager,
	starter CombatStarter,
) error {
	setup, rollback, err := starter.Prepare(manager)
	if err != nil {
		return err
	}
	if err := transitioner.TransitionToCombat(setup); err != nil {
		if rollback != nil {
			rollback()
		}
		return err
	}
	return nil
}
