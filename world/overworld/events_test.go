package overworld

import (
	"strings"
	"testing"
)

func TestEventLog_AddEvent(t *testing.T) {
	log := NewEventLog(10)

	event := OverworldEvent{
		Type:        EventThreatSpawned,
		Tick:        5,
		EntityID:    123,
		Description: "Test threat spawned",
		Data:        make(map[string]interface{}),
	}

	log.AddEvent(event)

	if len(log.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(log.Events))
	}

	if log.Unread != 1 {
		t.Errorf("Expected Unread=1, got %d", log.Unread)
	}

	if log.Events[0].Type != EventThreatSpawned {
		t.Errorf("Event type mismatch: expected %v, got %v", EventThreatSpawned, log.Events[0].Type)
	}
}

func TestEventLog_CapacityLimit(t *testing.T) {
	log := NewEventLog(5)

	// Add 10 events (exceeds capacity)
	for i := 0; i < 10; i++ {
		event := OverworldEvent{
			Type:        EventThreatEvolved,
			Tick:        int64(i),
			EntityID:    0,
			Description: "Event " + string(rune('0'+i)),
		}
		log.AddEvent(event)
	}

	// Should only keep 5 most recent
	if len(log.Events) != 5 {
		t.Errorf("Expected 5 events (max capacity), got %d", len(log.Events))
	}

	// Should keep events 5-9 (newest)
	if log.Events[0].Tick != 5 {
		t.Errorf("Expected oldest retained event tick=5, got %d", log.Events[0].Tick)
	}

	if log.Events[4].Tick != 9 {
		t.Errorf("Expected newest event tick=9, got %d", log.Events[4].Tick)
	}
}

func TestEventLog_GetRecentEvents(t *testing.T) {
	log := NewEventLog(10)

	// Add 5 events
	for i := 0; i < 5; i++ {
		event := OverworldEvent{
			Type:        EventFactionExpanded,
			Tick:        int64(i),
			EntityID:    0,
			Description: "Event " + string(rune('0'+i)),
		}
		log.AddEvent(event)
	}

	// Get 3 most recent
	recent := log.GetRecentEvents(3)

	if len(recent) != 3 {
		t.Errorf("Expected 3 recent events, got %d", len(recent))
	}

	// Should be events 2, 3, 4 (newest 3)
	if recent[0].Tick != 2 {
		t.Errorf("Expected first recent event tick=2, got %d", recent[0].Tick)
	}

	if recent[2].Tick != 4 {
		t.Errorf("Expected last recent event tick=4, got %d", recent[2].Tick)
	}
}

func TestEventLog_GetRecentEvents_ExceedsAvailable(t *testing.T) {
	log := NewEventLog(10)

	// Add 3 events
	for i := 0; i < 3; i++ {
		event := OverworldEvent{
			Type: EventThreatDestroyed,
			Tick: int64(i),
		}
		log.AddEvent(event)
	}

	// Request 10 events (more than available)
	recent := log.GetRecentEvents(10)

	// Should return all 3 available
	if len(recent) != 3 {
		t.Errorf("Expected 3 events (all available), got %d", len(recent))
	}
}

func TestEventLog_MarkRead(t *testing.T) {
	log := NewEventLog(10)

	// Add events
	for i := 0; i < 3; i++ {
		log.AddEvent(OverworldEvent{Type: EventVictory, Tick: int64(i)})
	}

	if log.Unread != 3 {
		t.Errorf("Expected Unread=3, got %d", log.Unread)
	}

	log.MarkRead()

	if log.Unread != 0 {
		t.Errorf("Expected Unread=0 after MarkRead, got %d", log.Unread)
	}

	// Events should still exist
	if len(log.Events) != 3 {
		t.Errorf("Expected events to persist after MarkRead, got %d events", len(log.Events))
	}
}

func TestEventLog_Clear(t *testing.T) {
	log := NewEventLog(10)

	// Add events
	for i := 0; i < 5; i++ {
		log.AddEvent(OverworldEvent{Type: EventDefeat, Tick: int64(i)})
	}

	log.Clear()

	if len(log.Events) != 0 {
		t.Errorf("Expected 0 events after Clear, got %d", len(log.Events))
	}

	if log.Unread != 0 {
		t.Errorf("Expected Unread=0 after Clear, got %d", log.Unread)
	}
}

func TestEventType_String(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventThreatSpawned, "Threat Spawned"},
		{EventThreatEvolved, "Threat Evolved"},
		{EventThreatDestroyed, "Threat Destroyed"},
		{EventFactionExpanded, "Faction Expanded"},
		{EventFactionRaid, "Faction Raid"},
		{EventFactionDefeated, "Faction Defeated"},
		{EventVictory, "Victory"},
		{EventDefeat, "Defeat"},
		{EventCombatResolved, "Combat Resolved"},
		{EventType(999), "Unknown Event"}, // Unknown type
	}

	for _, test := range tests {
		result := test.eventType.String()
		if result != test.expected {
			t.Errorf("EventType(%d).String() = %q, expected %q", test.eventType, result, test.expected)
		}
	}
}

func TestGetEventLogForUI(t *testing.T) {
	// Create new log for this test
	testLog := NewEventLog(10)

	// Add test events
	testLog.AddEvent(OverworldEvent{
		Type:        EventThreatSpawned,
		Tick:        1,
		Description: "First event",
	})
	testLog.AddEvent(OverworldEvent{
		Type:        EventThreatEvolved,
		Tick:        2,
		Description: "Second event",
	})

	// Temporarily swap global log
	originalLog := GlobalEventLog
	GlobalEventLog = testLog
	defer func() { GlobalEventLog = originalLog }()

	result := GetEventLogForUI(2)

	// Check formatting (newest first)
	if !strings.Contains(result, "[Tick 2] Second event") {
		t.Errorf("Expected formatted output to contain tick 2 event, got: %s", result)
	}

	if !strings.Contains(result, "[Tick 1] First event") {
		t.Errorf("Expected formatted output to contain tick 1 event, got: %s", result)
	}

	// Newest should appear before oldest
	idx1 := strings.Index(result, "Tick 2")
	idx2 := strings.Index(result, "Tick 1")
	if idx1 > idx2 {
		t.Error("Expected events in reverse chronological order (newest first)")
	}
}

func TestGetEventLogForUI_Empty(t *testing.T) {
	testLog := NewEventLog(10)

	originalLog := GlobalEventLog
	GlobalEventLog = testLog
	defer func() { GlobalEventLog = originalLog }()

	result := GetEventLogForUI(10)

	expected := "No recent events"
	if result != expected {
		t.Errorf("Expected %q for empty log, got %q", expected, result)
	}
}

func TestHasUnreadEvents(t *testing.T) {
	testLog := NewEventLog(10)

	originalLog := GlobalEventLog
	GlobalEventLog = testLog
	defer func() { GlobalEventLog = originalLog }()

	// Initially no unread
	if HasUnreadEvents() {
		t.Error("Expected no unread events initially")
	}

	// Add event
	testLog.AddEvent(OverworldEvent{Type: EventVictory, Tick: 1})

	if !HasUnreadEvents() {
		t.Error("Expected unread events after adding event")
	}

	// Mark read
	testLog.MarkRead()

	if HasUnreadEvents() {
		t.Error("Expected no unread events after MarkRead")
	}
}

func TestGetUnreadCount(t *testing.T) {
	testLog := NewEventLog(10)

	originalLog := GlobalEventLog
	GlobalEventLog = testLog
	defer func() { GlobalEventLog = originalLog }()

	if GetUnreadCount() != 0 {
		t.Errorf("Expected unread count=0 initially, got %d", GetUnreadCount())
	}

	// Add 3 events
	for i := 0; i < 3; i++ {
		testLog.AddEvent(OverworldEvent{Type: EventDefeat, Tick: int64(i)})
	}

	if GetUnreadCount() != 3 {
		t.Errorf("Expected unread count=3, got %d", GetUnreadCount())
	}

	testLog.MarkRead()

	if GetUnreadCount() != 0 {
		t.Errorf("Expected unread count=0 after MarkRead, got %d", GetUnreadCount())
	}
}

func TestLogEvent(t *testing.T) {
	testLog := NewEventLog(10)

	originalLog := GlobalEventLog
	GlobalEventLog = testLog
	defer func() { GlobalEventLog = originalLog }()

	// Suppress console output for test
	// (LogEvent prints to stdout, which is expected behavior)

	LogEvent(EventThreatSpawned, 5, 123, "Test event")

	if len(testLog.Events) != 1 {
		t.Errorf("Expected 1 event logged, got %d", len(testLog.Events))
	}

	event := testLog.Events[0]
	if event.Type != EventThreatSpawned {
		t.Errorf("Expected EventThreatSpawned, got %v", event.Type)
	}

	if event.Tick != 5 {
		t.Errorf("Expected tick=5, got %d", event.Tick)
	}

	if event.EntityID != 123 {
		t.Errorf("Expected entityID=123, got %d", event.EntityID)
	}

	if event.Description != "Test event" {
		t.Errorf("Expected description='Test event', got %q", event.Description)
	}
}

func TestLogEventWithData(t *testing.T) {
	testLog := NewEventLog(10)

	originalLog := GlobalEventLog
	GlobalEventLog = testLog
	defer func() { GlobalEventLog = originalLog }()

	data := map[string]interface{}{
		"intensity": 5,
		"position":  "50,40",
	}

	LogEventWithData(EventThreatEvolved, 10, 456, "Threat evolved", data)

	if len(testLog.Events) != 1 {
		t.Errorf("Expected 1 event logged, got %d", len(testLog.Events))
	}

	event := testLog.Events[0]
	if event.Data["intensity"] != 5 {
		t.Errorf("Expected data['intensity']=5, got %v", event.Data["intensity"])
	}

	if event.Data["position"] != "50,40" {
		t.Errorf("Expected data['position']='50,40', got %v", event.Data["position"])
	}
}
