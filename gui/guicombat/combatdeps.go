package guicombat

import (
	"game_main/gui/framework"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/combat/combatservices"
)

// CombatModeDeps consolidates all dependencies for combat mode components.
// Instead of passing 6+ parameters to each handler, pass this single struct.
//
// Benefits:
//   - Add new dependency? One place to update.
//   - Constructor signatures stay stable.
//   - Easier to mock for testing.
type CombatModeDeps struct {
	// State (shared across all handlers)
	BattleState *framework.TacticalState

	// Services (game logic)
	CombatService *combatservices.CombatService

	// Queries (read access to ECS)
	Queries *framework.GUIQueries

	// Mode management
	ModeManager *framework.UIModeManager

	// Encounter callbacks (replacing direct EncounterService dependency)
	Encounter combattypes.EncounterCallbacks
}

// NewCombatModeDeps creates a new dependencies container
func NewCombatModeDeps(
	battleState *framework.TacticalState,
	combatService *combatservices.CombatService,
	queries *framework.GUIQueries,
	modeManager *framework.UIModeManager,
	encounter combattypes.EncounterCallbacks,
) *CombatModeDeps {
	return &CombatModeDeps{
		BattleState:   battleState,
		CombatService: combatService,
		Queries:       queries,
		ModeManager:   modeManager,
		Encounter:     encounter,
	}
}
