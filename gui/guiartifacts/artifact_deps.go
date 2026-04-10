package guiartifacts

import (
	"game_main/gui/framework"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/combat/combatservices"
)

// ArtifactActivationDeps holds dependencies the artifact activation system needs from combat mode.
type ArtifactActivationDeps struct {
	BattleState   *framework.TacticalState
	CombatService *combatservices.CombatService
	Queries       *framework.GUIQueries

	// Encounter callbacks (replacing direct EncounterService dependency)
	Encounter combattypes.EncounterCallbacks
}
