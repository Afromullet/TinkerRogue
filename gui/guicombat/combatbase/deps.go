package combatbase

import (
	"game_main/gui/framework"
	"game_main/mind/encounter"
	"game_main/tactical/combat/combatservices"
)

// CombatModeDeps consolidates all dependencies for combat mode components.
type CombatModeDeps struct {
	BattleState   *framework.TacticalState
	CombatService *combatservices.CombatService
	Queries       *framework.GUIQueries
	ModeManager   *framework.UIModeManager
	Encounter     encounter.EncounterController
}

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
