package guicombat

import (
	"game_main/gui/framework"
	"game_main/mind/encounter"
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

	// GUI's narrow command/query port onto EncounterService
	Encounter encounter.EncounterController
}

// NewCombatModeDeps creates a new dependencies container
func NewCombatModeDeps(
	battleState *framework.TacticalState,
	combatService *combatservices.CombatService,
	queries *framework.GUIQueries,
	modeManager *framework.UIModeManager,
	encounterController encounter.EncounterController,
) *CombatModeDeps {
	return &CombatModeDeps{
		BattleState:   battleState,
		CombatService: combatService,
		Queries:       queries,
		ModeManager:   modeManager,
		Encounter:     encounterController,
	}
}
