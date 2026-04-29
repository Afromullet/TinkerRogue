package visualizer

import (
	"time"

	sharedtypes "game_main/tools/combat_analysis/shared"
)

// Type aliases for the data fields the visualizer doesn't customize.
// BattleRecord and EngagementRecord are defined locally because the
// visualizer's EngagementRecord adds a Summary field, and BattleRecord
// must reference the local EngagementRecord slice.
type UnitSnapshot = sharedtypes.UnitSnapshot
type HealEvent = sharedtypes.HealEvent

// BattleRecord is the root structure for exported combat JSON files.
// It aggregates all combat engagements from a single battle.
type BattleRecord struct {
	BattleID    string             `json:"battle_id"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	FinalRound  int                `json:"final_round"`
	VictorName  string             `json:"victor_name,omitempty"`
	Engagements []EngagementRecord `json:"engagements"`
}

// EngagementRecord wraps a CombatLog with battle metadata.
// Each engagement represents a single squad-vs-squad attack.
// The visualizer adds a Summary field on top of the shared shape.
type EngagementRecord struct {
	Index     int                    `json:"index"`
	Round     int                    `json:"round"`
	CombatLog *sharedtypes.CombatLog `json:"combat_log"`
	Summary   *EngagementSummary     `json:"summary"`
}

// EngagementSummary contains per-unit summaries for both squads.
type EngagementSummary struct {
	AttackerSummaries []UnitActionSummary `json:"attacker_summaries"`
	DefenderSummaries []UnitActionSummary `json:"defender_summaries"`
}

// UnitActionSummary aggregates all actions performed by a single unit in an engagement.
type UnitActionSummary struct {
	UnitID          int64            `json:"unit_id"`
	UnitName        string           `json:"unit_name"`
	Role            string           `json:"role"`
	GridPos         GridPosition     `json:"grid_pos"`
	TargetedRows    []int            `json:"targeted_rows"`
	TargetedColumns []int            `json:"targeted_columns"`
	TargetedCells   []GridPosition   `json:"targeted_cells"`
	TargetMode      string           `json:"target_mode"`
	UnitsEngaged    []UnitEngagement `json:"units_engaged"`
	TotalAttacks    int              `json:"total_attacks"`
	Hits            int              `json:"hits"`
	Misses          int              `json:"misses"`
	Dodges          int              `json:"dodges"`
	Criticals       int              `json:"criticals"`
	TotalDamage     int              `json:"total_damage"`
	UnitsKilled     int              `json:"units_killed"`
	TotalHealing    int              `json:"total_healing"`
	UnitsHealed     int              `json:"units_healed"`
	HealsPerformed  []HealEngagement `json:"heals_performed,omitempty"`
	Summary         string           `json:"summary"`
}

// HealEngagement details a heal action on a specific target.
type HealEngagement struct {
	TargetID   int64  `json:"target_id"`
	TargetName string `json:"target_name"`
	HealAmount int    `json:"heal_amount"`
}

// GridPosition represents a grid cell location.
type GridPosition struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// UnitEngagement details interaction with a specific target.
type UnitEngagement struct {
	TargetID    int64  `json:"target_id"`
	TargetName  string `json:"target_name"`
	Outcome     string `json:"outcome"` // "HIT", "MISS", "DODGE", "CRITICAL"
	DamageDealt int    `json:"damage_dealt"`
	WasKilled   bool   `json:"was_killed"`
}
