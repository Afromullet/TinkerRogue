package templates

import "log"

// UnitSpellRegistry maps unit type names to the spell IDs they can cast.
// Populated from unitspells.json at startup.
var UnitSpellRegistry map[string][]SpellID

type jsonUnitSpellMapping struct {
	UnitType string    `json:"unitType"`
	Spells   []SpellID `json:"spells"`
}

type jsonUnitSpellFile struct {
	UnitSpells []jsonUnitSpellMapping `json:"unitSpells"`
}

var unitSpellLoader = Loader[jsonUnitSpellFile]{
	Name: "unitspells",
	Path: UnitSpellDataPath,
}

// LoadUnitSpellDefinitions reads unit-to-spell mappings from JSON. Spell IDs
// that don't resolve in SpellRegistry are logged as warnings rather than fatal
// errors — missing spells degrade the unit's spell list but don't crash boot.
func LoadUnitSpellDefinitions() error {
	data, err := unitSpellLoader.Load()
	if err != nil {
		return err
	}

	UnitSpellRegistry = make(map[string][]SpellID, len(data.UnitSpells))
	for _, mapping := range data.UnitSpells {
		for _, spellID := range mapping.Spells {
			if GetSpellDefinition(spellID) == nil {
				log.Printf("[templates] unitspells: unit type %q references unknown spell %q",
					mapping.UnitType, spellID)
			}
		}
		UnitSpellRegistry[mapping.UnitType] = mapping.Spells
	}

	log.Printf("[templates] unitspells loaded: %d unit types", len(UnitSpellRegistry))
	return nil
}

// GetSpellsForUnitType returns the spell IDs available to a given unit type.
// Returns nil if the unit type has no spells.
func GetSpellsForUnitType(unitType string) []SpellID {
	return UnitSpellRegistry[unitType]
}
