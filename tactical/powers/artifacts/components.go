package artifacts

import "github.com/bytearena/ecs"

// EquipmentData stores which artifacts are equipped on a squad.
// EquippedArtifacts holds up to MaxSlots artifact definition IDs. MaxSlots == 0
// means "fall back to the configured default" — see defaultMaxArtifactSlots in
// system.go. Persisted per-squad so future per-commander/per-squad slot
// variants (e.g. an artifact that grants +1 slot) can be supported without
// changing the global config read at every equip call.
type EquipmentData struct {
	EquippedArtifacts []string
	MaxSlots          int
}

// ArtifactInstance represents a single copy of an artifact.
// EquippedOn is 0 when the instance is available, or the squad EntityID when equipped.
type ArtifactInstance struct {
	EquippedOn ecs.EntityID
}

// ArtifactInstanceInfo is a flat view of one artifact instance for GUI display.
type ArtifactInstanceInfo struct {
	DefinitionID  string
	EquippedOn    ecs.EntityID
	InstanceIndex int
}

// ArtifactInventoryData tracks which artifacts a player owns and their equipped state.
// OwnedArtifacts maps artifactID -> slice of instances (supports multiple copies).
type ArtifactInventoryData struct {
	OwnedArtifacts map[string][]*ArtifactInstance
	MaxArtifacts   int
}

// Tier constants for artifact definitions (used for display/filtering, not slot assignment).
const (
	TierMinor = "minor"
	TierMajor = "major"
)

// ECS variables
var (
	EquipmentComponent         *ecs.Component
	ArtifactInventoryComponent *ecs.Component
)
