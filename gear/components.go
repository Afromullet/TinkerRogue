package gear

import "github.com/bytearena/ecs"

// MaxArtifactSlots is the maximum number of artifacts a squad can equip.
const MaxArtifactSlots = 3

// EquipmentData stores which artifacts are equipped on a squad.
// EquippedArtifacts holds up to MaxArtifactSlots artifact definition IDs.
type EquipmentData struct {
	EquippedArtifacts []string
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
