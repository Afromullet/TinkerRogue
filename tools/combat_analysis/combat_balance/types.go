package balance

import (
	sharedtypes "game_main/tools/combat_analysis/shared"
)

// HitType constants matching tactical/squads/combatevents.go
const (
	HitTypeMiss          = 0
	HitTypeDodge         = 1
	HitTypeNormal        = 2
	HitTypeCritical      = 3
	HitTypeCounterattack = 4
	HitTypeHeal          = 5
)

// Type aliases — battle record types live in the shared package so the
// combat_balance, combat_visualizer, and any future analyzers stay in sync.
type BattleRecord = sharedtypes.BattleRecord
type EngagementRecord = sharedtypes.EngagementRecord
type CombatLog = sharedtypes.CombatLog
type UnitSnapshot = sharedtypes.UnitSnapshot
type HealEvent = sharedtypes.HealEvent
type AttackEvent = sharedtypes.AttackEvent
type TargetInfo = sharedtypes.TargetInfo
type HitResult = sharedtypes.HitResult
