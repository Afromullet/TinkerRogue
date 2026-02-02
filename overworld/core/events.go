package core

import (
	"fmt"
	"game_main/config"
	"game_main/overworld/overworldlog"

	"github.com/bytearena/ecs"
)

// OverworldEvent represents a significant game event
type OverworldEvent struct {
	Type        EventType
	Tick        int64
	EntityID    ecs.EntityID // Related entity (threat, faction, etc.)
	Description string
	Data        map[string]interface{} // Additional event data
}

// EventLog stores recent events for display
type EventLog struct {
	Events  []OverworldEvent
	MaxSize int // Maximum events to keep
	Unread  int // Count of unread events
}

// NewEventLog creates an event log
func NewEventLog(maxSize int) *EventLog {
	return &EventLog{
		Events:  make([]OverworldEvent, 0, maxSize),
		MaxSize: maxSize,
		Unread:  0,
	}
}

// AddEvent adds a new event to the log and records it if recording is enabled.
// This method is used internally by the event system.
func (el *EventLog) AddEvent(event OverworldEvent) {
	el.Events = append(el.Events, event)
	el.Unread++

	// Auto-record to recorder if enabled (accessed via context)
	ctx := GetContext()
	if ctx.Recorder != nil && ctx.Recorder.IsEnabled() {
		ctx.Recorder.RecordEvent(event.Tick, event.Type.String(), event.EntityID, event.Description, event.Data)
	}

	// Trim old events if over max size
	if len(el.Events) > el.MaxSize {
		// Remove oldest events
		excess := len(el.Events) - el.MaxSize
		el.Events = el.Events[excess:]
	}
}

// OverworldContext holds runtime state for the overworld system.
// This centralizes previously global variables for better testability and control.
type OverworldContext struct {
	EventLog *EventLog
	Recorder *overworldlog.OverworldRecorder
}

// NewOverworldContext creates a new context with default settings.
func NewOverworldContext() *OverworldContext {
	return &OverworldContext{
		EventLog: NewEventLog(100),
		Recorder: overworldlog.NewOverworldRecorder(),
	}
}

// Package-level context instance (initialized on first use)
var defaultContext *OverworldContext

// GetContext returns the current overworld context, initializing if needed.
// For testing, use SetContext to inject a custom context.
func GetContext() *OverworldContext {
	if defaultContext == nil {
		defaultContext = NewOverworldContext()
	}
	return defaultContext
}

// LogEvent adds an event to the event log.
// Pass nil for data if no additional metadata is needed.
func LogEvent(eventType EventType, tick int64, entityID ecs.EntityID, description string, data map[string]interface{}) {
	ctx := GetContext()

	if data == nil {
		data = make(map[string]interface{})
	}

	event := OverworldEvent{
		Type:        eventType,
		Tick:        tick,
		EntityID:    entityID,
		Description: description,
		Data:        data,
	}

	ctx.EventLog.AddEvent(event)

	// Also print to console for debugging
	fmt.Printf("[Tick %d] %s: %s\n", tick, eventType, description)
}

// StartRecordingSession initializes the overworld recorder for a new game session.
func StartRecordingSession(currentTick int64) {
	ctx := GetContext()
	ctx.Recorder.SetEnabled(config.ENABLE_OVERWORLD_LOG_EXPORT)
	if ctx.Recorder.IsEnabled() {
		ctx.Recorder.Start(currentTick)
		fmt.Printf("Overworld recording started (tick %d)\n", currentTick)
	}
}

// FinalizeRecording completes recording and exports to JSON.
// Returns an error if export fails.
func FinalizeRecording(outcome, reason string) error {
	ctx := GetContext()

	if !ctx.Recorder.IsEnabled() {
		return nil
	}

	record := ctx.Recorder.Finalize(outcome, reason)
	if err := overworldlog.ExportOverworldJSON(record, config.OVERWORLD_LOG_EXPORT_DIR); err != nil {
		return fmt.Errorf("failed to export overworld log: %w", err)
	}
	return nil
}

// ClearRecording resets the recorder for the next session.
func ClearRecording() {
	ctx := GetContext()
	ctx.Recorder.Clear()
}
