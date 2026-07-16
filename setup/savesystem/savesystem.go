package savesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"game_main/core/common"
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
	Version   int                      `json:"version"`
	Timestamp string                   `json:"timestamp"`
	Checksum  string                   `json:"checksum,omitempty"`
	Chunks    map[string]chunkEnvelope `json:"chunks"`
}

// chunkEnvelope wraps one chunk's serialized data with the schema version that
// produced it. Storing the version alongside the data lets LoadGame detect a
// schema change (e.g. progression's v1->v2 move) instead of silently mis-loading,
// and the version is covered by the envelope checksum for free.
type chunkEnvelope struct {
	Version int             `json:"version"`
	Data    json.RawMessage `json:"data"`
}

// Validatable is an optional interface that chunks can implement
// to run post-load validation after all IDs have been remapped.
type Validatable interface {
	Validate(em *common.EntityManager) error
}

const (
	// CurrentSaveVersion is the envelope format version. Bumped to 2 when per-chunk
	// versions were wired into the chunk map; pre-v2 saves store raw chunk data with
	// no version and are no longer loadable.
	CurrentSaveVersion = 2
	// MinLoadableSaveVersion is the oldest envelope format LoadGame can read.
	MinLoadableSaveVersion = 2
	SaveDirectory          = "saves"
	SaveFileName           = "roguelike_save.json"
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
		Chunks:    make(map[string]chunkEnvelope),
	}

	for _, chunk := range registeredChunks {
		data, err := chunk.Save(em)
		if err != nil {
			return fmt.Errorf("failed to save chunk %q: %w", chunk.ChunkID(), err)
		}
		if data != nil {
			envelope.Chunks[chunk.ChunkID()] = chunkEnvelope{
				Version: chunk.ChunkVersion(),
				Data:    data,
			}
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

// readAndValidateEnvelope reads the save file and validates it (unmarshal + version
// + checksum) WITHOUT mutating the world. It performs every check that must pass
// before LoadGame starts creating entities, so callers can verify a save is loadable
// before tearing down a running world.
func readAndValidateEnvelope() (SaveEnvelope, error) {
	savePath := filepath.Join(SaveDirectory, SaveFileName)

	var envelope SaveEnvelope

	bytes, err := os.ReadFile(savePath)
	if err != nil {
		return envelope, fmt.Errorf("failed to read save file: %w", err)
	}

	if err := json.Unmarshal(bytes, &envelope); err != nil {
		return envelope, fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	// Version check (future: migration logic)
	if envelope.Version > CurrentSaveVersion {
		return envelope, fmt.Errorf("save file version %d is newer than supported version %d", envelope.Version, CurrentSaveVersion)
	}
	if envelope.Version < MinLoadableSaveVersion {
		return envelope, fmt.Errorf("save file version %d predates the versioned-chunk format (v%d); start a new game", envelope.Version, MinLoadableSaveVersion)
	}

	// Verify checksum if present
	if envelope.Checksum != "" {
		chunksBytes, err := json.Marshal(envelope.Chunks)
		if err != nil {
			return envelope, fmt.Errorf("failed to marshal chunks for checksum verification: %w", err)
		}
		hash := sha256.Sum256(chunksBytes)
		expected := hex.EncodeToString(hash[:])
		if envelope.Checksum != expected {
			return envelope, fmt.Errorf("save file checksum mismatch: file may be corrupted")
		}
	}

	// Per-chunk version check: reject a save whose stored chunk version differs from
	// what the current code expects, so a schema change fails loudly instead of
	// silently mis-loading. Runs here (not just in LoadGame) so ValidateSaveFile —
	// the in-session-load pre-flight — rejects an incompatible save before the world
	// is reset.
	if err := checkChunkVersions(envelope, registeredChunks); err != nil {
		return envelope, err
	}

	return envelope, nil
}

// checkChunkVersions returns an error if any chunk present in the envelope was
// saved with a schema version different from what the corresponding registered
// chunk currently expects. Chunks absent from the save are skipped (a chunk added
// after the save was written is simply not restored).
func checkChunkVersions(envelope SaveEnvelope, chunks []SaveChunk) error {
	for _, chunk := range chunks {
		stored, exists := envelope.Chunks[chunk.ChunkID()]
		if !exists {
			continue
		}
		if stored.Version != chunk.ChunkVersion() {
			return fmt.Errorf("chunk %q version mismatch: save has v%d, code expects v%d", chunk.ChunkID(), stored.Version, chunk.ChunkVersion())
		}
	}
	return nil
}

// ValidateSaveFile checks that the save file exists, parses, and is a supported,
// uncorrupted version — WITHOUT mutating the world. Safe to call before a world
// reset so a bad save doesn't tear down a running game.
func ValidateSaveFile() error {
	_, err := readAndValidateEnvelope()
	return err
}

// LoadGame deserializes a save file and rebuilds ECS state.
// The EntityManager should already have its components/tags initialized
// (i.e., InitializeSubsystems has been called) but no game entities created yet.
func LoadGame(em *common.EntityManager) error {
	envelope, err := readAndValidateEnvelope()
	if err != nil {
		return err
	}

	idMap := NewEntityIDMap()

	// Phase 1: Create entities from each chunk
	for _, chunk := range registeredChunks {
		ce, exists := envelope.Chunks[chunk.ChunkID()]
		if !exists {
			// Chunk not in save file — skip (backward compatibility)
			continue
		}
		if err := chunk.Load(em, ce.Data, idMap); err != nil {
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
