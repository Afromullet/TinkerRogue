package guispells

import (
	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/encounter"
	"game_main/world/coords"
	"game_main/world/worldmap"
)

// SpellCastingDeps holds dependencies the spell system needs from the combat mode.
// This decouples guispells from CombatModeDeps - it only gets what it needs.
type SpellCastingDeps struct {
	BattleState      *framework.TacticalState
	ECSManager       *common.EntityManager
	EncounterService *encounter.EncounterService
	GameMap          *worldmap.GameMap
	PlayerPos        *coords.LogicalPosition
	AddCombatLog     func(string)
	Queries          *framework.GUIQueries
}
