package guicombat

import (
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/mind/encounter"
	"game_main/tactical/combatservices"
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
	BattleState *framework.BattleMapState

	// Services (game logic)
	CombatService    *combatservices.CombatService
	EncounterService *encounter.EncounterService

	// Queries (read access to ECS)
	Queries *framework.GUIQueries

	// UI Components
	CombatLogArea *widgets.CachedTextAreaWrapper
	LogManager    *CombatLogManager

	// Mode management
	ModeManager *framework.UIModeManager
}

// NewCombatModeDeps creates a new dependencies container
func NewCombatModeDeps(
	battleState *framework.BattleMapState,
	combatService *combatservices.CombatService,
	encounterService *encounter.EncounterService,
	queries *framework.GUIQueries,
	combatLogArea *widgets.CachedTextAreaWrapper,
	logManager *CombatLogManager,
	modeManager *framework.UIModeManager,
) *CombatModeDeps {
	return &CombatModeDeps{
		BattleState:      battleState,
		CombatService:    combatService,
		EncounterService: encounterService,
		Queries:          queries,
		CombatLogArea:    combatLogArea,
		LogManager:       logManager,
		ModeManager:      modeManager,
	}
}

// AddCombatLog adds a message to the combat log UI
func (d *CombatModeDeps) AddCombatLog(message string) {
	d.LogManager.UpdateTextArea(d.CombatLogArea, message)
}
