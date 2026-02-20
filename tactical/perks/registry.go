package perks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PerkRegistry is the global registry of all perk definitions, keyed by perk ID.
var PerkRegistry = make(map[string]*PerkDefinition)

// GetPerkDefinition looks up a perk by ID. Returns nil if not found.
func GetPerkDefinition(id string) *PerkDefinition {
	return PerkRegistry[id]
}

// GetAllPerkIDs returns all perk IDs from the registry.
func GetAllPerkIDs() []string {
	ids := make([]string, 0, len(PerkRegistry))
	for id := range PerkRegistry {
		ids = append(ids, id)
	}
	return ids
}

// perkDataFile is the JSON wrapper for perk definitions.
type perkDataFile struct {
	Perks []PerkDefinition `json:"perks"`
}

// assetRoot caching for perk data loading.
var assetRoot string

func getAssetRoot() string {
	if assetRoot != "" {
		return assetRoot
	}
	if info, err := os.Stat("assets"); err == nil && info.IsDir() {
		assetRoot = "assets"
		return assetRoot
	}
	assetRoot = filepath.Join("..", "assets")
	return assetRoot
}

// LoadPerkDefinitions reads perk definitions from JSON and populates PerkRegistry.
func LoadPerkDefinitions() {
	path := filepath.Join(getAssetRoot(), "gamedata", "perkdata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("WARNING: Failed to read perk data: %v\n", err)
		return
	}

	var perkFile perkDataFile
	if err := json.Unmarshal(data, &perkFile); err != nil {
		fmt.Printf("WARNING: Failed to parse perk data: %v\n", err)
		return
	}

	// Validate and register
	seenIDs := make(map[string]bool)
	for i := range perkFile.Perks {
		perk := &perkFile.Perks[i]

		if perk.ID == "" {
			fmt.Printf("WARNING: Perk at index %d has no ID, skipping\n", i)
			continue
		}

		if seenIDs[perk.ID] {
			fmt.Printf("WARNING: Duplicate perk ID '%s', skipping\n", perk.ID)
			continue
		}

		seenIDs[perk.ID] = true
		PerkRegistry[perk.ID] = perk
	}

	// Validate exclusive pairs are symmetric
	for _, perk := range PerkRegistry {
		for _, exID := range perk.ExclusiveWith {
			other := PerkRegistry[exID]
			if other == nil {
				fmt.Printf("WARNING: Perk '%s' exclusive with unknown perk '%s'\n", perk.ID, exID)
				continue
			}
			found := false
			for _, otherEx := range other.ExclusiveWith {
				if otherEx == perk.ID {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("WARNING: Exclusive pair asymmetric: '%s' lists '%s', but not vice versa\n", perk.ID, exID)
			}
		}
	}

	fmt.Printf("Loaded %d perk definitions\n", len(PerkRegistry))
}
