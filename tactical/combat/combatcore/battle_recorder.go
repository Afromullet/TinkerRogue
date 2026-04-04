package combatcore

// Re-exports from battlelog for backward compatibility.

import (
	"game_main/tactical/combat/battlelog"
)

type BattleRecord = battlelog.BattleRecord
type EngagementRecord = battlelog.EngagementRecord
type GridPosition = battlelog.GridPosition
type UnitEngagement = battlelog.UnitEngagement
type UnitActionSummary = battlelog.UnitActionSummary
type HealEngagement = battlelog.HealEngagement
type EngagementSummary = battlelog.EngagementSummary
type BattleRecorder = battlelog.BattleRecorder
type VictoryInfo = battlelog.VictoryInfo

func NewBattleRecorder() *BattleRecorder {
	return battlelog.NewBattleRecorder()
}

func GenerateEngagementSummary(log *CombatLog) *EngagementSummary {
	return battlelog.GenerateEngagementSummary(log)
}

func ExportBattleJSON(record *BattleRecord, outputDir string) error {
	return battlelog.ExportBattleJSON(record, outputDir)
}
