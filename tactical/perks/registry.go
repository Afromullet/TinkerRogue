package perks

import (
	"encoding/json"
	"fmt"
	"game_main/setup/config"
	"os"
)

// PerkDataPath is the relative path within assets to the perk data file.
const PerkDataPath = "gamedata/perkdata.json"

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

// GetPerksByTier returns all perks of a given tier.
func GetPerksByTier(tier PerkTier) []*PerkDefinition {
	var perks []*PerkDefinition
	for _, perk := range PerkRegistry {
		if perk.Tier == tier {
			perks = append(perks, perk)
		}
	}
	return perks
}

// GetPerksByRole returns all perks available to a given role string.
func GetPerksByRole(role string) []*PerkDefinition {
	var perks []*PerkDefinition
	for _, perk := range PerkRegistry {
		for _, r := range perk.Roles {
			if r == role {
				perks = append(perks, perk)
				break
			}
		}
	}
	return perks
}

// perkDataFile is the JSON wrapper for perk definitions.
type perkDataFile struct {
	Perks []PerkDefinition `json:"perks"`
}

// LoadPerkDefinitions reads perk definitions from a JSON file and populates PerkRegistry.
func LoadPerkDefinitions() {
	data, err := os.ReadFile(config.AssetPath(PerkDataPath))
	if err != nil {
		fmt.Printf("WARNING: Failed to read perk data: %v\n", err)
		return
	}

	var perkFile perkDataFile
	if err := json.Unmarshal(data, &perkFile); err != nil {
		fmt.Printf("WARNING: Failed to parse perk data: %v\n", err)
		return
	}

	// Validate and populate registry
	for i := range perkFile.Perks {
		perk := &perkFile.Perks[i]

		// Check for duplicate IDs
		if _, exists := PerkRegistry[perk.ID]; exists {
			fmt.Printf("WARNING: Duplicate perk ID %q, skipping\n", perk.ID)
			continue
		}

		// Validate roles
		for _, role := range perk.Roles {
			if role != "Tank" && role != "DPS" && role != "Support" {
				fmt.Printf("WARNING: Perk %q has invalid role %q\n", perk.ID, role)
			}
		}

		PerkRegistry[perk.ID] = perk
	}

	// Validate exclusive pairs are symmetric
	for _, perk := range PerkRegistry {
		for _, exID := range perk.ExclusiveWith {
			other := PerkRegistry[exID]
			if other == nil {
				fmt.Printf("WARNING: Perk %q has exclusiveWith %q which doesn't exist\n", perk.ID, exID)
				continue
			}
			found := false
			for _, otherExID := range other.ExclusiveWith {
				if otherExID == perk.ID {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("WARNING: Perk %q is exclusive with %q but not vice versa\n", perk.ID, exID)
			}
		}
	}

	fmt.Printf("Loaded %d perk definitions\n", len(PerkRegistry))
}
