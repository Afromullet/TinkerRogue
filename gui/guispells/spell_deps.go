package guispells

import (
	"game_main/core/common"
	"game_main/gui/framework"
	"game_main/mind/combatlifecycle"
	"game_main/core/coords"
	"game_main/world/worldmapcore"
)

// SpellCastingDeps holds dependencies the spell system needs from the combat mode.
// This decouples guispells from CombatModeDeps - it only gets what it needs.
type SpellCastingDeps struct {
	BattleState *framework.TacticalState
	ECSManager  *common.EntityManager
	GameMap     *worldmapcore.GameMap
	PlayerPos   *coords.LogicalPosition
	Queries     *framework.GUIQueries

	// Encounter callbacks (replacing direct EncounterService dependency)
	Encounter combatlifecycle.EncounterCallbacks
}
