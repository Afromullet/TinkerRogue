package templates

import "fmt"

// UnitSpellRegistry maps unit type names to the spell IDs they can cast.
// Populated from unitspells.json at startup.
var UnitSpellRegistry map[string][]string

type jsonUnitSpellMapping struct {
	UnitType string   `json:"unitType"`
	Spells   []string `json:"spells"`
}

type jsonUnitSpellFile struct {
	UnitSpells []jsonUnitSpellMapping `json:"unitSpells"`
}

// LoadUnitSpellDefinitions reads unit-to-spell mappings from JSON.
func LoadUnitSpellDefinitions() {
	var data jsonUnitSpellFile
	readAndUnmarshal("gamedata/unitspells.json", &data)

	UnitSpellRegistry = make(map[string][]string, len(data.UnitSpells))
	for _, mapping := range data.UnitSpells {
		// Validate spell IDs exist in the spell registry
		for _, spellID := range mapping.Spells {
			if GetSpellDefinition(spellID) == nil {
				fmt.Printf("WARNING: unit type %q references unknown spell %q\n", mapping.UnitType, spellID)
			}
		}
		UnitSpellRegistry[mapping.UnitType] = mapping.Spells
	}

	fmt.Printf("Unit spell mappings loaded: %d unit types\n", len(UnitSpellRegistry))
}

// GetSpellsForUnitType returns the spell IDs available to a given unit type.
// Returns nil if the unit type has no spells.
func GetSpellsForUnitType(unitType string) []string {
	return UnitSpellRegistry[unitType]
}
