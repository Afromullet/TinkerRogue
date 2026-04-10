package perks

import (
	"encoding/json"
	"fmt"
	"game_main/setup/config"
	"os"
)

// PerkTier classifies perk implementation complexity.
type PerkTier int

const (
	PerkTierConditioning   PerkTier = iota // Tier 1: Simple conditionals
	PerkTierSpecialization                 // Tier 2: Event reactions, targeting, state tracking
)

func (t PerkTier) String() string {
	switch t {
	case PerkTierConditioning:
		return "Combat Conditioning"
	case PerkTierSpecialization:
		return "Combat Specialization"
	default:
		return "Unknown"
	}
}

// PerkCategory classifies the tactical purpose of a perk.
type PerkCategory int

const (
	CategoryOffense  PerkCategory = iota // Damage-oriented perks
	CategoryDefense                      // Damage reduction, cover perks
	CategoryTactical                     // Targeting, positioning perks
	CategoryReactive                     // Event-triggered perks
	CategoryDoctrine                     // Squad-wide behavioral changes
)

func (c PerkCategory) String() string {
	switch c {
	case CategoryOffense:
		return "Offense"
	case CategoryDefense:
		return "Defense"
	case CategoryTactical:
		return "Tactical"
	case CategoryReactive:
		return "Reactive"
	case CategoryDoctrine:
		return "Doctrine"
	default:
		return "Unknown"
	}
}

// PerkDefinition is a static blueprint loaded from JSON.
// The perk's ID is used as the key into the hook registry.
type PerkDefinition struct {
	ID            PerkID       `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Tier          PerkTier     `json:"tier"`
	Category      PerkCategory `json:"category"`
	Roles         []string     `json:"roles"`         // ["Tank"], ["DPS", "Support"], etc.
	ExclusiveWith []PerkID     `json:"exclusiveWith"` // Mutually exclusive perk IDs
	UnlockCost    int          `json:"unlockCost"`    // Perk points to unlock
}

// PerkDataPath is the relative path within assets to the perk data file.
const PerkDataPath = "gamedata/perkdata.json"

// PerkRegistry is the global registry of all perk definitions, keyed by perk ID.
var PerkRegistry = make(map[PerkID]*PerkDefinition)

// GetPerkDefinition looks up a perk by ID. Returns nil if not found.
func GetPerkDefinition(id PerkID) *PerkDefinition {
	return PerkRegistry[id]
}

// GetAllPerkIDs returns all perk IDs from the registry.
func GetAllPerkIDs() []PerkID {
	ids := make([]PerkID, 0, len(PerkRegistry))
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

	// Validate that every perk has registered hooks and vice versa
	validateHookCoverage()
}

// validateHookCoverage checks that JSON definitions and behavior registrations are in sync.
func validateHookCoverage() {
	for id := range PerkRegistry {
		if GetPerkBehavior(id) == nil {
			fmt.Printf("WARNING: Perk %q has a JSON definition but no registered behavior\n", id)
		}
	}
	for id := range behaviorRegistry {
		if PerkRegistry[id] == nil {
			fmt.Printf("WARNING: Perk %q has a registered behavior but no JSON definition\n", id)
		}
	}
}
