package combatcore

// Re-exports from combattypes for backward compatibility.

import (
	"game_main/tactical/combat/combattypes"
)

type CombatStarter = combattypes.CombatStarter
type CombatType = combattypes.CombatType
type CombatSetup = combattypes.CombatSetup
type CombatTransitioner = combattypes.CombatTransitioner
type CombatStartRollback = combattypes.CombatStartRollback
type CombatExitReason = combattypes.CombatExitReason
type EncounterOutcome = combattypes.EncounterOutcome
type CombatCleaner = combattypes.CombatCleaner
type EncounterCallbacks = combattypes.EncounterCallbacks

const (
	CombatTypeOverworld       = combattypes.CombatTypeOverworld
	CombatTypeGarrisonDefense = combattypes.CombatTypeGarrisonDefense
	CombatTypeRaid            = combattypes.CombatTypeRaid
	CombatTypeDebug           = combattypes.CombatTypeDebug

	PostCombatReturnDefault = combattypes.PostCombatReturnDefault
	PostCombatReturnRaid    = combattypes.PostCombatReturnRaid

	ExitVictory = combattypes.ExitVictory
	ExitDefeat  = combattypes.ExitDefeat
	ExitFlee    = combattypes.ExitFlee
)
