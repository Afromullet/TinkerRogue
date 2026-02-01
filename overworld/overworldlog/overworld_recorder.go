package overworldlog

import (
	"fmt"
	"time"

	"github.com/bytearena/ecs"
)

// OverworldRecord is the root structure exported to JSON for post-game analysis.
// It aggregates all overworld events from a single game session.
type OverworldRecord struct {
	SessionID      string                    `json:"session_id"`
	StartTime      time.Time                 `json:"start_time"`
	EndTime        time.Time                 `json:"end_time"`
	StartTick      int64                     `json:"start_tick"`
	FinalTick      int64                     `json:"final_tick"`
	TotalTicks     int64                     `json:"total_ticks"`
	Outcome        string                    `json:"outcome"`          // "Victory", "Defeat", etc.
	OutcomeReason  string                    `json:"outcome_reason"`
	Events         []EventRecord             `json:"events"`
	ThreatSummary  *ThreatActivitySummary    `json:"threat_summary"`
	FactionSummary *FactionActivitySummary   `json:"faction_summary"`
	CombatSummary  *CombatActivitySummary    `json:"combat_summary"`
}

// EventRecord wraps an OverworldEvent with metadata for JSON export.
// Each event represents a significant overworld occurrence.
type EventRecord struct {
	Index       int                    `json:"index"`
	Tick        int64                  `json:"tick"`
	Type        string                 `json:"type"`        // Stringified EventType
	EntityID    ecs.EntityID           `json:"entity_id"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
}

// OverworldRecorder accumulates overworld events during a game session for later export.
type OverworldRecorder struct {
	enabled   bool
	sessionID string
	startTime time.Time
	startTick int64
	events    []EventRecord
	nextIndex int
}

// NewOverworldRecorder creates a new disabled OverworldRecorder.
func NewOverworldRecorder() *OverworldRecorder {
	return &OverworldRecorder{
		enabled:   false,
		events:    make([]EventRecord, 0),
		nextIndex: 1,
	}
}

// SetEnabled enables or disables recording.
func (or *OverworldRecorder) SetEnabled(enabled bool) {
	or.enabled = enabled
}

// IsEnabled returns whether recording is enabled.
func (or *OverworldRecorder) IsEnabled() bool {
	return or.enabled
}

// Start initializes a new overworld recording session.
// Should be called when the game session begins.
func (or *OverworldRecorder) Start(currentTick int64) {
	or.startTime = time.Now()
	or.startTick = currentTick
	// Include milliseconds to prevent ID collisions when sessions run in quick succession
	or.sessionID = fmt.Sprintf("journey_%s", or.startTime.Format("20060102_150405.000"))
	or.events = make([]EventRecord, 0)
	or.nextIndex = 1
}

// RecordEvent adds an overworld event to the session record.
// Should be called whenever a significant overworld event occurs.
// Parameters are passed individually to avoid circular imports.
func (or *OverworldRecorder) RecordEvent(tick int64, eventType string, entityID ecs.EntityID, description string, data map[string]interface{}) {
	if !or.enabled {
		return
	}

	record := EventRecord{
		Index:       or.nextIndex,
		Tick:        tick,
		Type:        eventType,
		EntityID:    entityID,
		Description: description,
		Data:        data,
	}

	or.events = append(or.events, record)
	or.nextIndex++
}

// Finalize completes the overworld record with outcome information.
// Returns the complete OverworldRecord ready for export.
func (or *OverworldRecorder) Finalize(outcome, reason string) *OverworldRecord {
	endTime := time.Now()
	var finalTick int64
	if len(or.events) > 0 {
		finalTick = or.events[len(or.events)-1].Tick
	} else {
		finalTick = or.startTick
	}

	record := &OverworldRecord{
		SessionID:     or.sessionID,
		StartTime:     or.startTime,
		EndTime:       endTime,
		StartTick:     or.startTick,
		FinalTick:     finalTick,
		TotalTicks:    finalTick - or.startTick,
		Outcome:       outcome,
		OutcomeReason: reason,
		Events:        or.events,
	}

	// Generate summaries
	record.ThreatSummary = GenerateThreatSummary(or.events)
	record.FactionSummary = GenerateFactionSummary(or.events)
	record.CombatSummary = GenerateCombatSummary(or.events)

	return record
}

// Clear resets the recorder for the next session.
func (or *OverworldRecorder) Clear() {
	or.events = make([]EventRecord, 0)
	or.nextIndex = 1
	or.sessionID = ""
	or.startTime = time.Time{}
	or.startTick = 0
}

// EventCount returns the number of recorded events.
func (or *OverworldRecorder) EventCount() int {
	return len(or.events)
}
