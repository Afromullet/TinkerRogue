package guispells

import (
	"game_main/common"
	"game_main/gui/framework"
	"game_main/tactical/combat/combattypes"
	"game_main/world/coords"
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
	Encounter combattypes.EncounterCallbacks
}
