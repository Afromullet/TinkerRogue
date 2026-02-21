package templates

import (
	"encoding/json"
	"fmt"
	"game_main/visual/graphics"
	"os"
)

// SpellTargetType determines how a spell selects its targets.
type SpellTargetType string

const (
	TargetSingleSquad SpellTargetType = "single"
	TargetAoETile     SpellTargetType = "aoe"
)

// SpellEffectType determines what a spell does to its targets.
type SpellEffectType string

const (
	EffectDamage SpellEffectType = "damage"
	EffectBuff   SpellEffectType = "buff"
	EffectDebuff SpellEffectType = "debuff"
)

// SpellStatModifier defines one stat change a buff/debuff spell applies.
type SpellStatModifier struct {
	Stat     string `json:"stat"`     // "strength", "dexterity", "magic", etc.
	Modifier int    `json:"modifier"` // positive or negative
}

// SpellDefinition is a static blueprint for a spell loaded from JSON.
type SpellDefinition struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	ManaCost      int                 `json:"manaCost"`
	Damage        int                 `json:"damage"`
	TargetType    SpellTargetType     `json:"targetType"`
	EffectType    SpellEffectType     `json:"effectType"`
	Shape         *JSONTargetArea     `json:"shape,omitempty"`
	VXType        string              `json:"vxType"`
	VXDuration    int                 `json:"vxDuration"`
	Duration      int                 `json:"duration,omitempty"`      // turns for buff/debuff
	StatModifiers []SpellStatModifier `json:"statModifiers,omitempty"` // stat changes
}

// SpellRegistry is the global registry of all spell definitions, keyed by spell ID.
var SpellRegistry = make(map[string]*SpellDefinition)

// GetSpellDefinition looks up a spell by ID. Returns nil if not found.
func GetSpellDefinition(id string) *SpellDefinition {
	return SpellRegistry[id]
}

// GetAllSpellIDs returns all spell IDs from the registry.
func GetAllSpellIDs() []string {
	ids := make([]string, 0, len(SpellRegistry))
	for id := range SpellRegistry {
		ids = append(ids, id)
	}
	return ids
}

// CreateAoEShape creates a TileBasedShape from the spell's shape definition.
// Returns a default 1x1 square if the spell has no shape defined.
func (sd *SpellDefinition) CreateAoEShape() graphics.TileBasedShape {
	return CreateTargetArea(sd.Shape)
}

// CreateVisualEffect creates a VX effect at the given pixel coordinates.
func (sd *SpellDefinition) CreateVisualEffect(x, y int) graphics.VisualEffect {
	switch sd.VXType {
	case "fire":
		return graphics.NewFireEffect(x, y, sd.VXDuration)
	case "ice":
		return graphics.NewIceEffect(x, y, sd.VXDuration)
	case "electricity":
		return graphics.NewElectricityEffect(x, y, sd.VXDuration)
	case "cloud":
		return graphics.NewCloudEffect(x, y, sd.VXDuration)
	default:
		return graphics.NewFireEffect(x, y, sd.VXDuration)
	}
}

// IsSingleTarget returns true if this spell targets a single squad.
func (sd *SpellDefinition) IsSingleTarget() bool {
	return sd.TargetType == TargetSingleSquad
}

// IsAoE returns true if this spell targets an area of tiles.
func (sd *SpellDefinition) IsAoE() bool {
	return sd.TargetType == TargetAoETile
}

// IsDamage returns true if this spell deals damage.
func (sd *SpellDefinition) IsDamage() bool {
	return sd.EffectType == EffectDamage
}

// IsBuff returns true if this spell applies buffs.
func (sd *SpellDefinition) IsBuff() bool {
	return sd.EffectType == EffectBuff
}

// IsDebuff returns true if this spell applies debuffs.
func (sd *SpellDefinition) IsDebuff() bool {
	return sd.EffectType == EffectDebuff
}

// spellDataFile is the JSON wrapper for spell definitions.
type spellDataFile struct {
	Spells []SpellDefinition `json:"spells"`
}

// LoadSpellDefinitions reads spell definitions from a JSON file and populates SpellRegistry.
func LoadSpellDefinitions() {
	data, err := os.ReadFile(assetPath(SpellDataPath))
	if err != nil {
		fmt.Printf("WARNING: Failed to read spell data: %v\n", err)
		return
	}

	var spellFile spellDataFile
	if err := json.Unmarshal(data, &spellFile); err != nil {
		fmt.Printf("WARNING: Failed to parse spell data: %v\n", err)
		return
	}

	for i := range spellFile.Spells {
		spell := &spellFile.Spells[i]
		SpellRegistry[spell.ID] = spell
	}

	fmt.Printf("Loaded %d spell definitions\n", len(spellFile.Spells))
}
