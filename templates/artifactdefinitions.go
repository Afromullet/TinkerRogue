package templates

import (
	"encoding/json"
	"fmt"
	"os"
)

// ArtifactStatModifier defines one stat change an artifact applies.
type ArtifactStatModifier struct {
	Stat     string `json:"stat"`
	Modifier int    `json:"modifier"`
}

// ArtifactDefinition is a static blueprint for an artifact loaded from JSON.
type ArtifactDefinition struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Tier          string                 `json:"tier"`     // "minor" or "major"
	Behavior      string                 `json:"behavior"` // major artifact behavior key (empty for minor)
	StatModifiers []ArtifactStatModifier `json:"statModifiers,omitempty"`
}

// ArtifactRegistry is the global registry of all artifact definitions, keyed by artifact ID.
var ArtifactRegistry = make(map[string]*ArtifactDefinition)

// GetArtifactDefinition looks up an artifact by ID. Returns nil if not found.
func GetArtifactDefinition(id string) *ArtifactDefinition {
	return ArtifactRegistry[id]
}

// ArtifactDataPath is the relative path within assets to the artifact data file.
const ArtifactDataPath = "gamedata/artifactdata.json"

// artifactDataFile is the JSON wrapper for artifact definitions.
type artifactDataFile struct {
	Artifacts []ArtifactDefinition `json:"artifacts"`
}

// LoadArtifactDefinitions reads artifact definitions from a JSON file and populates ArtifactRegistry.
func LoadArtifactDefinitions() {
	data, err := os.ReadFile(assetPath(ArtifactDataPath))
	if err != nil {
		fmt.Printf("WARNING: Failed to read artifact data: %v\n", err)
		return
	}

	var artifactFile artifactDataFile
	if err := json.Unmarshal(data, &artifactFile); err != nil {
		fmt.Printf("WARNING: Failed to parse artifact data: %v\n", err)
		return
	}

	for i := range artifactFile.Artifacts {
		artifact := &artifactFile.Artifacts[i]
		ArtifactRegistry[artifact.ID] = artifact
	}

	fmt.Printf("Loaded %d artifact definitions\n", len(artifactFile.Artifacts))
}
