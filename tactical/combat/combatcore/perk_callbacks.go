package combatcore

// Re-exports from combattypes for backward compatibility.

import (
	"game_main/tactical/combat/combattypes"
)

type DamageHookRunner = combattypes.DamageHookRunner
type CoverHookRunner = combattypes.CoverHookRunner
type TargetHookRunner = combattypes.TargetHookRunner
type PostDamageRunner = combattypes.PostDamageRunner
type DeathOverrideRunner = combattypes.DeathOverrideRunner
type CounterModRunner = combattypes.CounterModRunner
type DamageRedirectRunner = combattypes.DamageRedirectRunner
type PerkCallbacks = combattypes.PerkCallbacks
