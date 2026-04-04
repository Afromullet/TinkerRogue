package combatcore

// Re-exports from combatstate for backward compatibility.
// Component/tag vars are NOT re-exported — import combatstate directly for those.
// The init() registration now lives in combatstate.

import (
	"game_main/tactical/combat/combatstate"
)

// Type aliases for component data structs
type FactionData = combatstate.FactionData
type TurnStateData = combatstate.TurnStateData
type ActionStateData = combatstate.ActionStateData
type CombatFactionData = combatstate.CombatFactionData

// Type alias for cache
type CombatQueryCache = combatstate.CombatQueryCache

// Type alias for faction manager
type CombatFactionManager = combatstate.CombatFactionManager
