package savesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"game_main/common"
	"os"
	"path/filepath"
	"time"
)

// SaveChunk represents a subsystem that can save/load its own ECS data.
// Each chunk is responsible for one domain (squads, player, gear, etc.).
type SaveChunk interface {
	// ChunkID returns a unique string key for this chunk (e.g., "squads", "player")
	ChunkID() string

	// ChunkVersion returns the current serialization version for this chunk.
	// Embedded in save data to detect format changes and enable migration.
	ChunkVersion() int

	// Save extracts state from the ECS world into a JSON-serializable snapshot
	Save(em *common.EntityManager) (json.RawMessage, error)

	// Load recreates entities from serialized data, registering old->new ID mappings.
	// Called in Phase 1 of the two-phase load.
	Load(em *common.EntityManager, data json.RawMessage, idMap *EntityIDMap) error

	// RemapIDs fixes all cross-entity references using the completed ID map.
	// Called in Phase 2 AFTER all chunks have loaded.
	RemapIDs(em *common.EntityManager, idMap *EntityIDMap) error
}

// SaveEnvelope is the top-level save file structure.
type SaveEnvelope struct {
	Version   int                        `json:"version"`
	Timestamp string                     `json:"timestamp"`
	Checksum  string                     `json:"checksum,omitempty"`
	Chunks    map[string]json.RawMessage `json:"chunks"`
}

// Validatable is an optional interface that chunks can implement
// to run post-load validation after all IDs have been remapped.
type Validatable interface {
	Validate(em *common.EntityManager) error
}

const (
	CurrentSaveVersion = 1
	SaveDirectory      = "saves"
	SaveFileName       = "roguelike_save.json"
)

// registeredChunks holds all chunks registered via init().
var registeredChunks []SaveChunk

// RegisterChunk adds a SaveChunk to the global registry.
// Call this from init() in each chunk file.
func RegisterChunk(chunk SaveChunk) {
	registeredChunks = append(registeredChunks, chunk)
}

// GetRegisteredChunks returns all registered chunks (for testing).
func GetRegisteredChunks() []SaveChunk {
	return registeredChunks
}

// GetChunk returns a registered chunk by its ID, or nil if not found.
// Useful for configuring chunks that need external state (e.g., MapChunk needs GameMap pointer).
func GetChunk(chunkID string) SaveChunk {
	for _, chunk := range registeredChunks {
		if chunk.ChunkID() == chunkID {
			return chunk
		}
	}
	return nil
}

// SaveGame serializes the current game state to disk.
// Uses atomic write (write to .tmp, then rename) to prevent corruption on crash.
// Creates a .bak backup of the previous save before overwriting.
func SaveGame(em *common.EntityManager) error {
	envelope := SaveEnvelope{
		Version:   CurrentSaveVersion,
		Timestamp: time.Now().Format(time.RFC3339),
		Chunks:    make(map[string]json.RawMessage),
	}

	for _, chunk := range registeredChunks {
		data, err := chunk.Save(em)
		if err != nil {
			return fmt.Errorf("failed to save chunk %q: %w", chunk.ChunkID(), err)
		}
		if data != nil {
			envelope.Chunks[chunk.ChunkID()] = data
		}
	}

	// Compute checksum over the chunks data
	chunksBytes, err := json.Marshal(envelope.Chunks)
	if err != nil {
		return fmt.Errorf("failed to marshal chunks for checksum: %w", err)
	}
	hash := sha256.Sum256(chunksBytes)
	envelope.Checksum = hex.EncodeToString(hash[:])

	// Ensure save directory exists
	if err := os.MkdirAll(SaveDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	bytes, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal save data: %w", err)
	}

	savePath := filepath.Join(SaveDirectory, SaveFileName)
	tmpPath := savePath + ".tmp"
	bakPath := savePath + ".bak"

	// Write to temp file first (atomic write step 1)
	if err := os.WriteFile(tmpPath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write temp save file: %w", err)
	}

	// Backup existing save (if present) before overwriting
	if _, err := os.Stat(savePath); err == nil {
		// Remove old backup, ignore errors if it doesn't exist
		os.Remove(bakPath)
		if err := os.Rename(savePath, bakPath); err != nil {
			// Non-fatal: backup failed but we can still save
			os.Remove(bakPath)
		}
	}

	// Atomic rename temp -> final (atomic write step 2)
	if err := os.Rename(tmpPath, savePath); err != nil {
		return fmt.Errorf("failed to finalize save file: %w", err)
	}

	return nil
}

// LoadGame deserializes a save file and rebuilds ECS state.
// The EntityManager should already have its components/tags initialized
// (i.e., InitializeSubsystems has been called) but no game entities created yet.
func LoadGame(em *common.EntityManager) error {
	savePath := filepath.Join(SaveDirectory, SaveFileName)

	bytes, err := os.ReadFile(savePath)
	if err != nil {
		return fmt.Errorf("failed to read save file: %w", err)
	}

	var envelope SaveEnvelope
	if err := json.Unmarshal(bytes, &envelope); err != nil {
		return fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	// Version check (future: migration logic)
	if envelope.Version > CurrentSaveVersion {
		return fmt.Errorf("save file version %d is newer than supported version %d", envelope.Version, CurrentSaveVersion)
	}

	// Verify checksum if present
	if envelope.Checksum != "" {
		chunksBytes, err := json.Marshal(envelope.Chunks)
		if err != nil {
			return fmt.Errorf("failed to marshal chunks for checksum verification: %w", err)
		}
		hash := sha256.Sum256(chunksBytes)
		expected := hex.EncodeToString(hash[:])
		if envelope.Checksum != expected {
			return fmt.Errorf("save file checksum mismatch: file may be corrupted")
		}
	}

	idMap := NewEntityIDMap()

	// Phase 1: Create entities from each chunk
	for _, chunk := range registeredChunks {
		data, exists := envelope.Chunks[chunk.ChunkID()]
		if !exists {
			// Chunk not in save file — skip (backward compatibility)
			continue
		}
		if err := chunk.Load(em, data, idMap); err != nil {
			return fmt.Errorf("failed to load chunk %q: %w", chunk.ChunkID(), err)
		}
	}

	// Phase 2: Remap all cross-entity references
	for _, chunk := range registeredChunks {
		if _, exists := envelope.Chunks[chunk.ChunkID()]; !exists {
			continue
		}
		if err := chunk.RemapIDs(em, idMap); err != nil {
			return fmt.Errorf("failed to remap IDs for chunk %q: %w", chunk.ChunkID(), err)
		}
	}

	// Phase 3: Post-load validation (optional per chunk)
	for _, chunk := range registeredChunks {
		if _, exists := envelope.Chunks[chunk.ChunkID()]; !exists {
			continue
		}
		if v, ok := chunk.(Validatable); ok {
			if err := v.Validate(em); err != nil {
				return fmt.Errorf("validation failed for chunk %q: %w", chunk.ChunkID(), err)
			}
		}
	}

	return nil
}

// HasSaveFile returns true if a save file exists.
func HasSaveFile() bool {
	savePath := filepath.Join(SaveDirectory, SaveFileName)
	_, err := os.Stat(savePath)
	return err == nil
}

// DeleteSaveFile removes the save file (e.g., on permadeath).
func DeleteSaveFile() error {
	savePath := filepath.Join(SaveDirectory, SaveFileName)
	if err := os.Remove(savePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete save file: %w", err)
	}
	return nil
}
