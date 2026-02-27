package battlelog

import (
	"fmt"
	"game_main/tactical/squads"
	"time"

	"github.com/bytearena/ecs"
)

// BattleRecord is the root structure exported to JSON for post-combat analysis.
// It aggregates all combat engagements from a single battle.
type BattleRecord struct {
	BattleID        string             `json:"battle_id"`
	StartTime       time.Time          `json:"start_time"`
	EndTime         time.Time          `json:"end_time"`
	FinalRound      int                `json:"final_round"`
	VictorFactionID ecs.EntityID       `json:"victor_faction_id,omitempty"`
	VictorName      string             `json:"victor_name,omitempty"`
	Engagements     []EngagementRecord `json:"engagements"`
}

// EngagementRecord wraps a CombatLog with battle metadata.
// Each engagement represents a single squad-vs-squad attack.
type EngagementRecord struct {
	Index     int                `json:"index"`
	Round     int                `json:"round"`
	CombatLog *squads.CombatLog  `json:"combat_log"`
	Summary   *EngagementSummary `json:"summary"` // Per-unit action summaries
}

// GridPosition represents a grid cell location.
type GridPosition struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// UnitEngagement details interaction with a specific target.
type UnitEngagement struct {
	TargetID    ecs.EntityID `json:"target_id"`
	TargetName  string       `json:"target_name"`
	Outcome     string       `json:"outcome"` // "HIT", "MISS", "DODGE", "CRITICAL"
	DamageDealt int          `json:"damage_dealt"`
	WasKilled   bool         `json:"was_killed"`
}

// UnitActionSummary aggregates all actions performed by a single unit in an engagement.
type UnitActionSummary struct {
	UnitID   ecs.EntityID `json:"unit_id"`
	UnitName string       `json:"unit_name"`
	Role     string       `json:"role"`     // "Tank", "DPS", "Support"
	GridPos  GridPosition `json:"grid_pos"` // Unit's position

	// Targeting breakdown
	TargetedRows    []int          `json:"targeted_rows"`    // Unique rows attacked
	TargetedColumns []int          `json:"targeted_columns"` // Unique columns attacked
	TargetedCells   []GridPosition `json:"targeted_cells"`   // Specific cells (for Magic mode)
	TargetMode      string         `json:"target_mode"`      // Primary attack type

	// Affected units
	UnitsEngaged []UnitEngagement `json:"units_engaged"` // Per-target details

	// Outcome aggregation
	TotalAttacks int `json:"total_attacks"`
	Hits         int `json:"hits"`
	Misses       int `json:"misses"`
	Dodges       int `json:"dodges"`
	Criticals    int `json:"criticals"`
	TotalDamage  int `json:"total_damage"`
	UnitsKilled  int `json:"units_killed"`

	// Healing aggregation
	TotalHealing   int              `json:"total_healing"`
	UnitsHealed    int              `json:"units_healed"`
	HealsPerformed []HealEngagement `json:"heals_performed,omitempty"`

	// Human-readable summary
	Summary string `json:"summary"`
}

// HealEngagement details a heal action on a specific target.
type HealEngagement struct {
	TargetID   ecs.EntityID `json:"target_id"`
	TargetName string       `json:"target_name"`
	HealAmount int          `json:"heal_amount"`
}

// EngagementSummary contains per-unit summaries for both squads.
type EngagementSummary struct {
	AttackerSummaries []UnitActionSummary `json:"attacker_summaries"`
	DefenderSummaries []UnitActionSummary `json:"defender_summaries"` // Now includes counterattacks
}

// BattleRecorder accumulates combat events during a battle for later export.
type BattleRecorder struct {
	enabled      bool
	battleID     string
	startTime    time.Time
	engagements  []EngagementRecord
	nextIndex    int
	currentRound int // Updated by CombatService when turns change
}

// NewBattleRecorder creates a new disabled BattleRecorder.
func NewBattleRecorder() *BattleRecorder {
	return &BattleRecorder{
		enabled:     false,
		engagements: make([]EngagementRecord, 0),
		nextIndex:   1,
	}
}

// SetEnabled enables or disables recording.
func (br *BattleRecorder) SetEnabled(enabled bool) {
	br.enabled = enabled
}

// IsEnabled returns whether recording is enabled.
func (br *BattleRecorder) IsEnabled() bool {
	return br.enabled
}

// Start initializes a new battle recording session.
// Should be called when combat begins.
func (br *BattleRecorder) Start() {
	br.startTime = time.Now()
	// Include milliseconds to prevent ID collisions when battles run in quick succession
	br.battleID = fmt.Sprintf("battle_%s", br.startTime.Format("20060102_150405.000"))
	br.engagements = make([]EngagementRecord, 0)
	br.nextIndex = 1
}

// SetCurrentRound updates the current round for recording.
// Should be called by CombatService when the round changes.
func (br *BattleRecorder) SetCurrentRound(round int) {
	br.currentRound = round
}

// RecordEngagement adds a combat log to the battle record.
// Uses the current round set via SetCurrentRound.
// Should be called after each squad attack completes.
func (br *BattleRecorder) RecordEngagement(log *squads.CombatLog) {
	if !br.enabled || log == nil {
		return
	}

	// Generate summary from combat log
	summary := GenerateEngagementSummary(log)

	record := EngagementRecord{
		Index:     br.nextIndex,
		Round:     br.currentRound,
		CombatLog: log,
		Summary:   summary,
	}

	br.engagements = append(br.engagements, record)
	br.nextIndex++
}

// VictoryInfo contains battle outcome information for the recorder.
// This is a simplified version to avoid circular imports with combatservices.
type VictoryInfo struct {
	RoundsCompleted int
	VictorFaction   ecs.EntityID
	VictorName      string
}

// Finalize completes the battle record with victory information.
// Returns the complete BattleRecord ready for export.
func (br *BattleRecorder) Finalize(victor *VictoryInfo) *BattleRecord {
	record := &BattleRecord{
		BattleID:    br.battleID,
		StartTime:   br.startTime,
		EndTime:     time.Now(),
		Engagements: br.engagements,
	}

	if victor != nil {
		record.FinalRound = victor.RoundsCompleted
		record.VictorFactionID = victor.VictorFaction
		record.VictorName = victor.VictorName
	}

	return record
}

// Clear resets the recorder for the next battle.
func (br *BattleRecorder) Clear() {
	br.engagements = make([]EngagementRecord, 0)
	br.nextIndex = 1
	br.battleID = ""
	br.startTime = time.Time{}
}

// EngagementCount returns the number of recorded engagements.
func (br *BattleRecorder) EngagementCount() int {
	return len(br.engagements)
}
