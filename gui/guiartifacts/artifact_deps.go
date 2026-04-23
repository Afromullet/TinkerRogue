package guiartifacts

import (
	"game_main/gui/framework"
	"game_main/mind/encounter"
	"game_main/tactical/combat/combatservices"
)

// ArtifactActivationDeps holds dependencies the artifact activation system needs from combat mode.
type ArtifactActivationDeps struct {
	BattleState   *framework.TacticalState
	CombatService *combatservices.CombatService
	Queries       *framework.GUIQueries

	// GUI's narrow command/query port onto EncounterService
	Encounter encounter.EncounterController
}
