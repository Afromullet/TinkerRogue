package templates

import "log"

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

// artifactDataFile is the JSON wrapper for artifact definitions.
type artifactDataFile struct {
	Artifacts []ArtifactDefinition `json:"artifacts"`
}

// LoadArtifactDefinitions reads artifact definitions from JSON files and populates ArtifactRegistry.
// Missing files are non-fatal (registry stays empty).
func LoadArtifactDefinitions() error {
	total := 0
	n, err := loadArtifactFile(MinorArtifactDataPath)
	if err != nil {
		return err
	}
	total += n
	n, err = loadArtifactFile(MajorArtifactDataPath)
	if err != nil {
		return err
	}
	total += n
	log.Printf("[templates] artifacts loaded: %d definitions", total)
	return nil
}

// loadArtifactFile reads a single artifact JSON file and adds entries to ArtifactRegistry.
// Returns the number of artifacts loaded.
func loadArtifactFile(path string) (int, error) {
	loader := Loader[artifactDataFile]{
		Name:     "artifacts",
		Path:     path,
		Optional: true,
	}
	artifactFile, err := loader.Load()
	if err != nil {
		return 0, err
	}

	for i := range artifactFile.Artifacts {
		artifact := &artifactFile.Artifacts[i]
		// Major artifacts use their ID as the behavior key
		if artifact.Tier == "major" && artifact.Behavior == "" {
			artifact.Behavior = artifact.ID
		}
		ArtifactRegistry[artifact.ID] = artifact
	}

	return len(artifactFile.Artifacts), nil
}
