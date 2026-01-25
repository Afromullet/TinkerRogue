package overworldlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytearena/ecs"
)

func TestRecorderLifecycle(t *testing.T) {
	recorder := NewOverworldRecorder()

	// Test initial state
	if recorder.IsEnabled() {
		t.Error("Expected recorder to be disabled initially")
	}
	if recorder.EventCount() != 0 {
		t.Errorf("Expected 0 events, got %d", recorder.EventCount())
	}

	// Enable and start
	recorder.SetEnabled(true)
	recorder.Start(0)

	if !recorder.IsEnabled() {
		t.Error("Expected recorder to be enabled")
	}
	if recorder.sessionID == "" {
		t.Error("Expected session ID to be set")
	}

	// Record some events
	recorder.RecordEvent(10, "Threat Spawned", 1, "Test threat spawned", map[string]interface{}{"intensity": 5})
	recorder.RecordEvent(20, "Threat Destroyed", 1, "Test threat destroyed", map[string]interface{}{"intensity_reduced": 5})

	if recorder.EventCount() != 2 {
		t.Errorf("Expected 2 events, got %d", recorder.EventCount())
	}

	// Finalize
	record := recorder.Finalize("Victory", "All threats eliminated")

	if record == nil {
		t.Fatal("Expected non-nil record")
	}
	if record.Outcome != "Victory" {
		t.Errorf("Expected outcome 'Victory', got '%s'", record.Outcome)
	}
	if len(record.Events) != 2 {
		t.Errorf("Expected 2 events in record, got %d", len(record.Events))
	}

	// Test Clear
	recorder.Clear()
	if recorder.EventCount() != 0 {
		t.Errorf("Expected 0 events after clear, got %d", recorder.EventCount())
	}
	if recorder.sessionID != "" {
		t.Error("Expected empty session ID after clear")
	}
}

func TestRecorderDisabled(t *testing.T) {
	recorder := NewOverworldRecorder()
	// Don't enable recorder

	recorder.Start(0)

	recorder.RecordEvent(10, "Threat Spawned", 1, "Test event", make(map[string]interface{}))

	// Should not record when disabled
	if recorder.EventCount() != 0 {
		t.Errorf("Expected 0 events when disabled, got %d", recorder.EventCount())
	}
}

func TestEventRecordConversion(t *testing.T) {
	recorder := NewOverworldRecorder()
	recorder.SetEnabled(true)
	recorder.Start(0)

	testData := map[string]interface{}{
		"intensity":   10,
		"threat_type": "goblin_camp",
	}

	recorder.RecordEvent(15, "Threat Spawned", 5, "Goblin camp spawned", testData)

	record := recorder.Finalize("In Progress", "")

	if len(record.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(record.Events))
	}

	eventRecord := record.Events[0]
	if eventRecord.Index != 1 {
		t.Errorf("Expected index 1, got %d", eventRecord.Index)
	}
	if eventRecord.Tick != 15 {
		t.Errorf("Expected tick 15, got %d", eventRecord.Tick)
	}
	if eventRecord.Type != "Threat Spawned" {
		t.Errorf("Expected type 'Threat Spawned', got '%s'", eventRecord.Type)
	}
	if eventRecord.EntityID != 5 {
		t.Errorf("Expected entity ID 5, got %d", eventRecord.EntityID)
	}
	if eventRecord.Description != "Goblin camp spawned" {
		t.Errorf("Expected description 'Goblin camp spawned', got '%s'", eventRecord.Description)
	}
}

func TestTotalTicksCalculation(t *testing.T) {
	recorder := NewOverworldRecorder()
	recorder.SetEnabled(true)
	recorder.Start(100)

	recorder.RecordEvent(150, "Threat Spawned", 1, "Test", make(map[string]interface{}))

	record := recorder.Finalize("Victory", "")

	if record.StartTick != 100 {
		t.Errorf("Expected start tick 100, got %d", record.StartTick)
	}
	if record.FinalTick != 150 {
		t.Errorf("Expected final tick 150, got %d", record.FinalTick)
	}
	if record.TotalTicks != 50 {
		t.Errorf("Expected total ticks 50, got %d", record.TotalTicks)
	}
}

func TestSessionIDFormat(t *testing.T) {
	recorder := NewOverworldRecorder()
	recorder.SetEnabled(true)
	recorder.Start(0)

	if recorder.sessionID == "" {
		t.Error("Expected non-empty session ID")
	}

	// Verify it starts with "journey_"
	if len(recorder.sessionID) < 8 || recorder.sessionID[:8] != "journey_" {
		t.Errorf("Expected session ID to start with 'journey_', got '%s'", recorder.sessionID)
	}
}

func TestSummaryGeneration(t *testing.T) {
	recorder := NewOverworldRecorder()
	recorder.SetEnabled(true)
	recorder.Start(0)

	// Add various event types
	recorder.RecordEvent(10, "Threat Spawned", 1, "Threat 1 spawned",
		map[string]interface{}{"intensity": 5, "threat_type": "goblin_camp"})
	recorder.RecordEvent(20, "Threat Spawned", 2, "Threat 2 spawned",
		map[string]interface{}{"intensity": 10, "threat_type": "orc_fortress"})
	recorder.RecordEvent(30, "Threat Destroyed", 1, "Threat 1 destroyed",
		map[string]interface{}{"threat_type": "goblin_camp"})
	recorder.RecordEvent(40, "Faction Expanded", ecs.EntityID(100), "Faction expanded",
		map[string]interface{}{"tiles_gained": 3})
	recorder.RecordEvent(50, "Combat Resolved", 1, "Combat victory",
		map[string]interface{}{"victory": true, "intensity_reduced": 5})

	record := recorder.Finalize("Victory", "All threats eliminated")

	// Test threat summary
	if record.ThreatSummary == nil {
		t.Fatal("Expected non-nil threat summary")
	}
	if record.ThreatSummary.TotalSpawned != 2 {
		t.Errorf("Expected 2 threats spawned, got %d", record.ThreatSummary.TotalSpawned)
	}
	if record.ThreatSummary.TotalDestroyed != 1 {
		t.Errorf("Expected 1 threat destroyed, got %d", record.ThreatSummary.TotalDestroyed)
	}
	if record.ThreatSummary.MaxIntensity != 10 {
		t.Errorf("Expected max intensity 10, got %d", record.ThreatSummary.MaxIntensity)
	}

	// Test faction summary
	if record.FactionSummary == nil {
		t.Fatal("Expected non-nil faction summary")
	}
	if record.FactionSummary.TotalExpansions != 1 {
		t.Errorf("Expected 1 faction expansion, got %d", record.FactionSummary.TotalExpansions)
	}

	// Test combat summary
	if record.CombatSummary == nil {
		t.Fatal("Expected non-nil combat summary")
	}
	if record.CombatSummary.TotalCombats != 1 {
		t.Errorf("Expected 1 combat, got %d", record.CombatSummary.TotalCombats)
	}
	if record.CombatSummary.Victories != 1 {
		t.Errorf("Expected 1 victory, got %d", record.CombatSummary.Victories)
	}
}

func TestExportOverworldJSON(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	recorder := NewOverworldRecorder()
	recorder.SetEnabled(true)
	recorder.Start(0)

	recorder.RecordEvent(10, "Threat Spawned", 1, "Test threat", map[string]interface{}{"intensity": 5})

	record := recorder.Finalize("Victory", "Test complete")

	// Test export
	err := ExportOverworldJSON(record, tempDir)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify file exists
	filename := generateOverworldFilename(record)
	filePath := filepath.Join(tempDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s", filePath)
	}

	// Verify file is not empty
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Error("Expected non-empty file")
	}
}

func TestExportNilRecord(t *testing.T) {
	err := ExportOverworldJSON(nil, "./test")
	if err == nil {
		t.Error("Expected error when exporting nil record")
	}
}

func TestExportEmptyDirectory(t *testing.T) {
	recorder := NewOverworldRecorder()
	recorder.SetEnabled(true)
	recorder.Start(0)
	record := recorder.Finalize("Test", "")

	err := ExportOverworldJSON(record, "")
	if err == nil {
		t.Error("Expected error when output directory is empty")
	}
}

func TestRecorderMultipleSessions(t *testing.T) {
	recorder := NewOverworldRecorder()
	recorder.SetEnabled(true)

	// First session
	recorder.Start(0)
	recorder.RecordEvent(10, "Threat Spawned", 1, "Session 1 event", make(map[string]interface{}))
	record1 := recorder.Finalize("Victory", "Session 1 complete")

	if len(record1.Events) != 1 {
		t.Errorf("Session 1: Expected 1 event, got %d", len(record1.Events))
	}

	// Clear and start second session
	recorder.Clear()
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	recorder.Start(0)

	recorder.RecordEvent(20, "Threat Destroyed", 2, "Session 2 event", make(map[string]interface{}))
	record2 := recorder.Finalize("Defeat", "Session 2 failed")

	if len(record2.Events) != 1 {
		t.Errorf("Session 2: Expected 1 event, got %d", len(record2.Events))
	}

	// Verify sessions are independent
	if record1.SessionID == record2.SessionID {
		t.Error("Expected different session IDs for different sessions")
	}
	if record1.Outcome == record2.Outcome {
		t.Error("Expected different outcomes for sessions")
	}
}
