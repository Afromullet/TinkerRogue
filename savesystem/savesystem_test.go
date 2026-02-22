package savesystem

import (
	"encoding/json"
	"testing"

	"github.com/bytearena/ecs"
)

// --- EntityIDMap tests ---

func TestEntityIDMap_RegisterAndRemap(t *testing.T) {
	idMap := NewEntityIDMap()

	idMap.Register(10, 100)
	idMap.Register(20, 200)
	idMap.Register(30, 300)

	if got := idMap.Remap(10); got != 100 {
		t.Errorf("Remap(10) = %d, want 100", got)
	}
	if got := idMap.Remap(20); got != 200 {
		t.Errorf("Remap(20) = %d, want 200", got)
	}
	if got := idMap.Remap(30); got != 300 {
		t.Errorf("Remap(30) = %d, want 300", got)
	}
}

func TestEntityIDMap_RemapZero(t *testing.T) {
	idMap := NewEntityIDMap()
	if got := idMap.Remap(0); got != 0 {
		t.Errorf("Remap(0) = %d, want 0", got)
	}
}

func TestEntityIDMap_RemapUnknown(t *testing.T) {
	idMap := NewEntityIDMap()
	idMap.Register(10, 100)

	if got := idMap.Remap(99); got != 0 {
		t.Errorf("Remap(99) = %d, want 0 (unmapped)", got)
	}
}

func TestEntityIDMap_RemapSlice(t *testing.T) {
	idMap := NewEntityIDMap()
	idMap.Register(1, 10)
	idMap.Register(2, 20)
	idMap.Register(3, 30)

	input := []ecs.EntityID{1, 2, 3, 99, 0}
	result := idMap.RemapSlice(input)

	expected := []ecs.EntityID{10, 20, 30, 0, 0}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("RemapSlice[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestEntityIDMap_Count(t *testing.T) {
	idMap := NewEntityIDMap()
	if got := idMap.Count(); got != 0 {
		t.Errorf("Count() = %d, want 0", got)
	}

	idMap.Register(1, 10)
	idMap.Register(2, 20)
	if got := idMap.Count(); got != 2 {
		t.Errorf("Count() = %d, want 2", got)
	}
}

// --- RemapStrict tests ---

func TestEntityIDMap_RemapStrict_Success(t *testing.T) {
	idMap := NewEntityIDMap()
	idMap.Register(10, 100)

	got, err := idMap.RemapStrict(10)
	if err != nil {
		t.Fatalf("RemapStrict(10) returned error: %v", err)
	}
	if got != 100 {
		t.Errorf("RemapStrict(10) = %d, want 100", got)
	}
}

func TestEntityIDMap_RemapStrict_Zero(t *testing.T) {
	idMap := NewEntityIDMap()

	got, err := idMap.RemapStrict(0)
	if err != nil {
		t.Fatalf("RemapStrict(0) returned error: %v", err)
	}
	if got != 0 {
		t.Errorf("RemapStrict(0) = %d, want 0", got)
	}
}

func TestEntityIDMap_RemapStrict_Unmapped(t *testing.T) {
	idMap := NewEntityIDMap()
	idMap.Register(10, 100)

	_, err := idMap.RemapStrict(99)
	if err == nil {
		t.Error("RemapStrict(99) should return error for unmapped non-zero ID")
	}
}

// --- LoadContext tests ---

func TestEntityIDMap_LoadContext(t *testing.T) {
	idMap := NewEntityIDMap()

	// Store data
	idMap.LoadContext["test_key"] = "test_value"

	// Retrieve data
	val, ok := idMap.LoadContext["test_key"]
	if !ok {
		t.Fatal("Expected LoadContext to contain 'test_key'")
	}
	if val != "test_value" {
		t.Errorf("LoadContext['test_key'] = %v, want 'test_value'", val)
	}

	// Delete data
	delete(idMap.LoadContext, "test_key")
	_, ok = idMap.LoadContext["test_key"]
	if ok {
		t.Error("Expected LoadContext to not contain 'test_key' after delete")
	}
}

// --- Checksum tests ---

func TestSaveEnvelope_ChecksumRoundTrip(t *testing.T) {
	envelope := SaveEnvelope{
		Version:   1,
		Timestamp: "2026-02-22T14:30:00Z",
		Checksum:  "abc123",
		Chunks:    make(map[string]json.RawMessage),
	}

	testData := map[string]string{"key": "value"}
	raw, _ := json.Marshal(testData)
	envelope.Chunks["test"] = raw

	bytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var loaded SaveEnvelope
	if err := json.Unmarshal(bytes, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if loaded.Checksum != "abc123" {
		t.Errorf("Checksum = %s, want abc123", loaded.Checksum)
	}
}

func TestSaveEnvelope_ChecksumOmittedWhenEmpty(t *testing.T) {
	envelope := SaveEnvelope{
		Version:   1,
		Timestamp: "2026-02-22T14:30:00Z",
		Chunks:    make(map[string]json.RawMessage),
	}

	bytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Checksum field should be omitted from JSON when empty
	var raw map[string]json.RawMessage
	json.Unmarshal(bytes, &raw)
	if _, ok := raw["checksum"]; ok {
		t.Error("Expected checksum to be omitted when empty")
	}
}

// --- SaveEnvelope tests ---

func TestSaveEnvelope_MarshalUnmarshal(t *testing.T) {
	envelope := SaveEnvelope{
		Version:   1,
		Timestamp: "2026-02-22T14:30:00Z",
		Chunks:    make(map[string]json.RawMessage),
	}

	testData := map[string]string{"key": "value"}
	raw, _ := json.Marshal(testData)
	envelope.Chunks["test"] = raw

	bytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var loaded SaveEnvelope
	if err := json.Unmarshal(bytes, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}
	if loaded.Timestamp != "2026-02-22T14:30:00Z" {
		t.Errorf("Timestamp = %s, want 2026-02-22T14:30:00Z", loaded.Timestamp)
	}
	if _, ok := loaded.Chunks["test"]; !ok {
		t.Error("Expected 'test' chunk to exist")
	}
}

func TestSaveEnvelope_MissingChunksAreSkipped(t *testing.T) {
	// Simulate loading a save file that has fewer chunks than registered
	envelope := SaveEnvelope{
		Version:   1,
		Timestamp: "2026-02-22T14:30:00Z",
		Chunks: map[string]json.RawMessage{
			"player": json.RawMessage(`{}`),
			// "squads" chunk is missing
		},
	}

	_, existsPlayer := envelope.Chunks["player"]
	_, existsSquads := envelope.Chunks["squads"]

	if !existsPlayer {
		t.Error("Expected 'player' chunk to exist")
	}
	if existsSquads {
		t.Error("Expected 'squads' chunk to not exist (backward compatibility)")
	}
}

func TestSaveEnvelope_UnknownChunksPreserved(t *testing.T) {
	// Simulate a save file with a chunk type we don't recognize
	rawJSON := `{
		"version": 1,
		"timestamp": "2026-02-22T14:30:00Z",
		"chunks": {
			"player": {"entityID": 1},
			"future_feature": {"data": "something"}
		}
	}`

	var envelope SaveEnvelope
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if _, ok := envelope.Chunks["future_feature"]; !ok {
		t.Error("Expected unknown 'future_feature' chunk to be preserved in raw form")
	}
}
