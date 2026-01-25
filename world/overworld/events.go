package overworld

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// EventType categorizes overworld events
type EventType int

const (
	EventThreatSpawned    EventType = iota // New threat appeared
	EventThreatEvolved                     // Threat gained intensity
	EventThreatDestroyed                   // Threat eliminated
	EventFactionExpanded                   // Faction claimed territory
	EventFactionRaid                       // Faction launched raid
	EventFactionDefeated                   // Faction eliminated
	EventVictory                           // Player won
	EventDefeat                            // Player lost
	EventCombatResolved                    // Combat outcome applied
)

func (e EventType) String() string {
	switch e {
	case EventThreatSpawned:
		return "Threat Spawned"
	case EventThreatEvolved:
		return "Threat Evolved"
	case EventThreatDestroyed:
		return "Threat Destroyed"
	case EventFactionExpanded:
		return "Faction Expanded"
	case EventFactionRaid:
		return "Faction Raid"
	case EventFactionDefeated:
		return "Faction Defeated"
	case EventVictory:
		return "Victory"
	case EventDefeat:
		return "Defeat"
	case EventCombatResolved:
		return "Combat Resolved"
	default:
		return "Unknown Event"
	}
}

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
	Events   []OverworldEvent
	MaxSize  int // Maximum events to keep
	Unread   int // Count of unread events
}

// NewEventLog creates an event log
func NewEventLog(maxSize int) *EventLog {
	return &EventLog{
		Events:  make([]OverworldEvent, 0, maxSize),
		MaxSize: maxSize,
		Unread:  0,
	}
}

// AddEvent adds a new event to the log
func (el *EventLog) AddEvent(event OverworldEvent) {
	el.Events = append(el.Events, event)
	el.Unread++

	// Trim old events if over max size
	if len(el.Events) > el.MaxSize {
		// Remove oldest events
		excess := len(el.Events) - el.MaxSize
		el.Events = el.Events[excess:]
	}
}

// GetRecentEvents returns the N most recent events
func (el *EventLog) GetRecentEvents(count int) []OverworldEvent {
	if count > len(el.Events) {
		count = len(el.Events)
	}

	// Return last N events
	start := len(el.Events) - count
	return el.Events[start:]
}

// MarkRead resets unread counter
func (el *EventLog) MarkRead() {
	el.Unread = 0
}

// Clear empties the event log
func (el *EventLog) Clear() {
	el.Events = make([]OverworldEvent, 0, el.MaxSize)
	el.Unread = 0
}

// Global event log (singleton for simplicity)
var GlobalEventLog = NewEventLog(100)

// LogEvent adds an event to the global log
func LogEvent(eventType EventType, tick int64, entityID ecs.EntityID, description string) {
	event := OverworldEvent{
		Type:        eventType,
		Tick:        tick,
		EntityID:    entityID,
		Description: description,
		Data:        make(map[string]interface{}),
	}

	GlobalEventLog.AddEvent(event)

	// Also print to console for debugging
	fmt.Printf("[Tick %d] %s: %s\n", tick, eventType, description)
}

// LogEventWithData adds an event with custom data
func LogEventWithData(eventType EventType, tick int64, entityID ecs.EntityID, description string, data map[string]interface{}) {
	event := OverworldEvent{
		Type:        eventType,
		Tick:        tick,
		EntityID:    entityID,
		Description: description,
		Data:        data,
	}

	GlobalEventLog.AddEvent(event)

	fmt.Printf("[Tick %d] %s: %s\n", tick, eventType, description)
}

// GetEventLogForUI returns recent events formatted for display
func GetEventLogForUI(count int) string {
	events := GlobalEventLog.GetRecentEvents(count)
	if len(events) == 0 {
		return "No recent events"
	}

	result := ""
	for i := len(events) - 1; i >= 0; i-- { // Reverse order (newest first)
		event := events[i]
		result += fmt.Sprintf("[Tick %d] %s\n", event.Tick, event.Description)
	}

	return result
}

// HasUnreadEvents checks if there are new events
func HasUnreadEvents() bool {
	return GlobalEventLog.Unread > 0
}

// GetUnreadCount returns number of unread events
func GetUnreadCount() int {
	return GlobalEventLog.Unread
}
