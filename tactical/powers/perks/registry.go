package perks

import (
	"encoding/json"
	"fmt"
	"game_main/core/config"
	"game_main/tactical/squads/unitdefs"
	"log"
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

// LoadPerkDefinitions reads perk definitions from a JSON file and populates
// PerkRegistry. Returns the slice of problems found (duplicate IDs, invalid
// roles, asymmetric exclusiveWith pairs). Perks with at least one invalid role
// are skipped entirely so they cannot silently fail role-filtered UI. Callers
// decide whether errors are fatal (e.g. fail startup in debug builds).
func LoadPerkDefinitions() []error {
	var errs []error

	data, err := os.ReadFile(config.AssetPath(PerkDataPath))
	if err != nil {
		log.Printf("WARNING: Failed to read perk data: %v", err)
		return []error{fmt.Errorf("read perk data: %w", err)}
	}

	var perkFile perkDataFile
	if err := json.Unmarshal(data, &perkFile); err != nil {
		log.Printf("WARNING: Failed to parse perk data: %v", err)
		return []error{fmt.Errorf("parse perk data: %w", err)}
	}

	// Validate and populate registry
	for i := range perkFile.Perks {
		perk := &perkFile.Perks[i]

		// Check for duplicate IDs
		if _, exists := PerkRegistry[perk.ID]; exists {
			log.Printf("WARNING: Duplicate perk ID %q, skipping", perk.ID)
			errs = append(errs, fmt.Errorf("duplicate perk ID %q", perk.ID))
			continue
		}

		// Validate roles. Skip the entire perk if any role is invalid — a
		// perk with all-invalid roles silently disappears from role-filtered
		// UI, so fail loud instead.
		invalidRole := false
		for _, role := range perk.Roles {
			if _, err := unitdefs.GetRole(role); err != nil {
				log.Printf("WARNING: Perk %q has invalid role %q: %v", perk.ID, role, err)
				errs = append(errs, fmt.Errorf("perk %q has invalid role %q: %w", perk.ID, role, err))
				invalidRole = true
			}
		}
		if invalidRole {
			continue
		}

		PerkRegistry[perk.ID] = perk
	}

	// Validate exclusive pairs are symmetric
	for _, perk := range PerkRegistry {
		for _, exID := range perk.ExclusiveWith {
			other := PerkRegistry[exID]
			if other == nil {
				log.Printf("WARNING: Perk %q has exclusiveWith %q which doesn't exist", perk.ID, exID)
				errs = append(errs, fmt.Errorf("perk %q has exclusiveWith %q which doesn't exist", perk.ID, exID))
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
				log.Printf("WARNING: Perk %q is exclusive with %q but not vice versa", perk.ID, exID)
				errs = append(errs, fmt.Errorf("perk %q is exclusive with %q but not vice versa", perk.ID, exID))
			}
		}
	}

	log.Printf("Loaded %d perk definitions", len(PerkRegistry))
	return errs
}

// ValidateHookCoverage checks that JSON definitions and behavior registrations
// are in sync. Returns the slice of problems found; empty means everything
// matches. Callers decide whether mismatches are fatal (e.g. fail startup in
// debug builds).
func ValidateHookCoverage() []error {
	var errs []error
	for id := range PerkRegistry {
		if GetPerkBehavior(id) == nil {
			errs = append(errs, fmt.Errorf("perk %q has a JSON definition but no registered behavior", id))
		}
	}
	for id := range perkBehaviorImpls {
		if PerkRegistry[id] == nil {
			errs = append(errs, fmt.Errorf("perk %q has a registered behavior but no JSON definition", id))
		}
	}
	return errs
}
