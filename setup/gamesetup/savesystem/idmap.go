package savesystem

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// EntityIDMap tracks old (saved) -> new (loaded) entity ID mappings.
// During load, each chunk creates new entities and registers the mapping
// from the saved ID to the new ID. During RemapIDs, chunks use this map
// to fix cross-entity references.
//
// The LoadContext map allows chunks to pass intermediate data between
// Load and RemapIDs phases without storing mutable state on the chunk struct.
type EntityIDMap struct {
	oldToNew    map[ecs.EntityID]ecs.EntityID
	LoadContext map[string]interface{}
}

// NewEntityIDMap creates an empty ID mapping.
func NewEntityIDMap() *EntityIDMap {
	return &EntityIDMap{
		oldToNew:    make(map[ecs.EntityID]ecs.EntityID),
		LoadContext: make(map[string]interface{}),
	}
}

// Register records an old->new entity ID mapping.
func (m *EntityIDMap) Register(oldID, newID ecs.EntityID) {
	m.oldToNew[oldID] = newID
}

// Remap returns the new ID for an old (saved) ID.
// Returns 0 if the old ID was not registered (or was 0 itself).
func (m *EntityIDMap) Remap(oldID ecs.EntityID) ecs.EntityID {
	if oldID == 0 {
		return 0
	}
	return m.oldToNew[oldID]
}

// RemapSlice remaps a slice of old IDs to new IDs.
// IDs that don't have a mapping are replaced with 0.
func (m *EntityIDMap) RemapSlice(ids []ecs.EntityID) []ecs.EntityID {
	result := make([]ecs.EntityID, len(ids))
	for i, oldID := range ids {
		result[i] = m.Remap(oldID)
	}
	return result
}

// RemapStrict returns the new ID for an old (saved) ID.
// Returns an error if a non-zero old ID has no mapping (indicating data loss).
// Use this for references that MUST resolve (e.g., squad->commander links).
func (m *EntityIDMap) RemapStrict(oldID ecs.EntityID) (ecs.EntityID, error) {
	if oldID == 0 {
		return 0, nil
	}
	newID, ok := m.oldToNew[oldID]
	if !ok {
		return 0, fmt.Errorf("unmapped entity ID %d: referenced entity was not loaded", oldID)
	}
	return newID, nil
}

// Count returns the number of registered mappings.
func (m *EntityIDMap) Count() int {
	return len(m.oldToNew)
}
