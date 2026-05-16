package templates

import "game_main/core/common"

// schema_monster.go holds the small data DTOs that don't belong to a specific
// subsystem: monsters, encounter difficulty, squad metadata, shared utility
// types (color, resource cost, target area), and name-generation pools.
// Grouped because each individual subset is too small to warrant its own file.

// --- Monsters and encounters ---

// JSONAttributes mirrors common.Attributes for JSON unmarshalling. The
// NewAttributesFromJson method converts to the runtime ECS type.
type JSONAttributes struct {
	Strength   int `json:"strength"`
	Dexterity  int `json:"dexterity"`
	Magic      int `json:"magic"`
	Leadership int `json:"leadership"`
	Armor      int `json:"armor"`
	Weapon     int `json:"weapon"`
}

func (attr JSONAttributes) NewAttributesFromJson() common.Attributes {
	return common.NewAttributes(
		attr.Strength,
		attr.Dexterity,
		attr.Magic,
		attr.Leadership,
		attr.Armor,
		attr.Weapon,
	)
}

// JSONStatGrowths defines per-stat growth rate grades for leveling.
// Each field is a grade string: S, A, B, C, D, E, or F.
type JSONStatGrowths struct {
	Strength   string `json:"strength"`
	Dexterity  string `json:"dexterity"`
	Magic      string `json:"magic"`
	Leadership string `json:"leadership"`
	Armor      string `json:"armor"`
	Weapon     string `json:"weapon"`
}

type JSONMonster struct {
	UnitType   string         `json:"unitType"`
	ImageName  string         `json:"imgname"`
	Attributes JSONAttributes `json:"attributes"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	Role       string         `json:"role"`

	// Targeting fields
	AttackType  string   `json:"attackType"`  // "MeleeRow", "MeleeColumn", "Ranged", or "Magic"
	TargetCells [][2]int `json:"targetCells"` // For magic: pattern cells

	CoverValue     float64 `json:"coverValue"`     // Damage reduction provided (0.0-1.0)
	CoverRange     int     `json:"coverRange"`     // Rows behind that receive cover (1-3)
	RequiresActive bool    `json:"requiresActive"` // If true, dead/stunned units don't provide cover
	AttackRange    int     `json:"attackRange"`    // World-based attack range (Melee=1, Ranged=3, Magic=4)
	MovementSpeed  int     `json:"movementSpeed"`  // Movement speed on world map (1 tile per speed point)
	Cost           int     `json:"cost"`           // Gold cost to purchase this unit

	StatGrowths JSONStatGrowths `json:"statGrowths"` // Per-stat growth rate grades for leveling
}

// JSONEncounterDifficulty defines difficulty scaling for encounters.
type JSONEncounterDifficulty struct {
	Level            int     `json:"level"`
	Name             string  `json:"name"`
	PowerMultiplier  float64 `json:"powerMultiplier"`
	SquadCount       int     `json:"squadCount"`
	MinUnitsPerSquad int     `json:"minUnitsPerSquad"`
	MaxUnitsPerSquad int     `json:"maxUnitsPerSquad"`
	MinTargetPower   float64 `json:"minTargetPower"`
	MaxTargetPower   float64 `json:"maxTargetPower"`
}

// JSONSquadType defines squad type metadata (for future filtering/validation).
type JSONSquadType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// --- Shared utility DTOs ---

// JSONTargetArea describes a targeted region in JSON form. Different TileShapes
// use different parameters; fields are optional and zero-default.
type JSONTargetArea struct {
	Type   string `json:"type"`
	Size   int    `json:"size,omitempty"`
	Length int    `json:"length,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Radius int    `json:"radius,omitempty"`
}

// JSONResourceCost represents a resource cost in JSON (iron, wood, stone).
type JSONResourceCost struct {
	Iron  int `json:"iron"`
	Wood  int `json:"wood"`
	Stone int `json:"stone"`
}

// JSONColor represents an RGBA color in JSON.
type JSONColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

// --- Name generation pools ---

// JSONNamePool defines a pool of syllable parts for name generation.
type JSONNamePool struct {
	Prefixes []string `json:"prefixes"`
	Middles  []string `json:"middles"`
	Suffixes []string `json:"suffixes"`
}

// JSONNameConfig is the root container for name generation configuration.
type JSONNameConfig struct {
	NameFormat   string                  `json:"nameFormat"`
	MinSyllables int                     `json:"minSyllables"`
	MaxSyllables int                     `json:"maxSyllables"`
	Pools        map[string]JSONNamePool `json:"pools"`
}
