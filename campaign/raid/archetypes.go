package raid

import (
	"encoding/json"
	"fmt"
	"os"
)

// ArchetypeUnit defines a single unit within a squad archetype.
type ArchetypeUnit struct {
	MonsterType string `json:"monsterType"` // Key into monsterdata.json (looked up via squads.GetTemplateByUnitType)
	GridRow     int    `json:"gridRow"`
	GridCol     int    `json:"gridCol"`
	GridWidth   int    `json:"gridWidth"`  // Default 1
	GridHeight  int    `json:"gridHeight"` // Default 1
	IsLeader    bool   `json:"isLeader"`
}

// SquadArchetype defines a pre-composed garrison squad template.
type SquadArchetype struct {
	Name           string          `json:"name"`
	DisplayName    string          `json:"displayName"`
	Units          []ArchetypeUnit `json:"units"`
	PreferredRooms []string        `json:"preferredRooms"`
}

// GarrisonArchetypes holds all garrison squad compositions. Populated by LoadArchetypeData.
var GarrisonArchetypes []SquadArchetype

// LoadArchetypeData reads and parses raidarchetypes.json into GarrisonArchetypes.
func LoadArchetypeData(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read archetype data: %w", err)
	}

	var file struct {
		Archetypes []SquadArchetype `json:"archetypes"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse archetype data: %w", err)
	}

	GarrisonArchetypes = file.Archetypes
	println("Archetype data loaded:", len(GarrisonArchetypes), "archetypes")
	return nil
}

// GetArchetype finds a squad archetype by name. Returns nil if not found.
func GetArchetype(name string) *SquadArchetype {
	for i := range GarrisonArchetypes {
		if GarrisonArchetypes[i].Name == name {
			return &GarrisonArchetypes[i]
		}
	}
	return nil
}
