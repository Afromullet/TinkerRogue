package combatcore

// Re-exports from combattypes for backward compatibility.
// External packages can continue importing combatcore for these types
// until they are migrated to import combattypes directly.

import (
	"game_main/tactical/combat/combattypes"
)

// Type aliases - these are identical to the combattypes originals
type AttackEvent = combattypes.AttackEvent
type TargetInfo = combattypes.TargetInfo
type HitResult = combattypes.HitResult
type HitType = combattypes.HitType
type CoverBreakdown = combattypes.CoverBreakdown
type CoverProvider = combattypes.CoverProvider
type HealEvent = combattypes.HealEvent
type CombatLog = combattypes.CombatLog
type UnitSnapshot = combattypes.UnitSnapshot
type SquadStatus = combattypes.SquadStatus
type UnitIdentity = combattypes.UnitIdentity
type DamageModifiers = combattypes.DamageModifiers
type CombatResult = combattypes.CombatResult

// Constant re-exports
const (
	HitTypeMiss         = combattypes.HitTypeMiss
	HitTypeDodge        = combattypes.HitTypeDodge
	HitTypeNormal       = combattypes.HitTypeNormal
	HitTypeCritical     = combattypes.HitTypeCritical
	HitTypeCounterattack = combattypes.HitTypeCounterattack
	HitTypeHeal         = combattypes.HitTypeHeal
)
