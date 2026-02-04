package templates

import (
	"game_main/common"
	"game_main/visual/graphics"
)

// All structs for unmarshalling JSON data

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

type JSONMeleeWeapon struct {
	Name        string `json:"name,omitempty"`
	ImgName     string `json:"imgname,omitempty"`
	MinDamage   int    `json:"minDamage"`
	MaxDamage   int    `json:"maxDamage"`
	AttackSpeed int    `json:"attackSpeed"`
}

func NewJSONMeleeWeapon(w JSONWeapon) JSONMeleeWeapon {
	return JSONMeleeWeapon{
		ImgName:     w.ImgName,
		Name:        w.Name,
		MinDamage:   w.MinDamage,
		MaxDamage:   w.MaxDamage,
		AttackSpeed: w.AttackSpeed,
	}
}

// Different TileShapes require different parameters
// The JSONTargetArea struct contains optional fields for all of the options
type JSONTargetArea struct {
	Type   string `json:"type"`
	Size   int    `json:"size,omitempty"`
	Length int    `json:"length,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Radius int    `json:"radius,omitempty"`
}

// For creating the TileBasedShape from JSON data
func CreateTargetArea(area *JSONTargetArea) graphics.TileBasedShape {

	var s graphics.TileBasedShape

	//Default to a 1x1 square if the area is nil
	if area == nil {
		s = graphics.NewSquare(0, 0, graphics.MediumShape)
	} else if area.Type == "Rectangle" {

		s = graphics.NewRectangle(0, 0, graphics.MediumShape)

	} else if area.Type == "Cone" {

		s = graphics.NewCone(0, 0, graphics.LineDown, graphics.MediumShape)

	} else if area.Type == "Square" {

		s = graphics.NewSquare(0, 0, graphics.MediumShape)

	} else if area.Type == "Line" {

		s = graphics.NewLine(0, 0, graphics.LineDown, graphics.MediumShape)

	} else if area.Type == "Circle" {

		s = graphics.NewCircle(0, 0, graphics.MediumShape)

	}

	return s

}

type JSONRangedWeapon struct {
	Name            string          `json:"name,omitempty"`
	ShootingVXXName string          `json:"shootingVX"`
	ImgName         string          `json:"imgname,omitempty"`
	MinDamage       int             `json:"minDamage"`
	MaxDamage       int             `json:"maxDamage"`
	ShootingRange   int             `json:"shootingRange"`
	AttackSpeed     int             `json:"attackSpeed"`
	TargetArea      *JSONTargetArea `json:"targetArea"`
}

func NewJSONRangedWeapon(r JSONWeapon) JSONRangedWeapon {

	return JSONRangedWeapon{

		Name:            r.Name,
		ShootingVXXName: r.ShootingVX,
		ImgName:         r.ImgName,
		MinDamage:       r.MinDamage,
		MaxDamage:       r.MaxDamage,
		ShootingRange:   r.ShootingRange,
		AttackSpeed:     r.AttackSpeed,
		TargetArea:      r.TargetArea,
	}

}

type JSONMonster struct {
	Name       string         `json:"name"`
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
}

func NewJSONMonster(m JSONMonster) JSONMonster {
	return JSONMonster{
		Name:       m.Name,
		ImageName:  m.ImageName,
		Attributes: m.Attributes,
		Width:      m.Width,
		Height:     m.Height,
		Role:       m.Role,

		AttackType:     m.AttackType,
		TargetCells:    m.TargetCells,
		CoverValue:     m.CoverValue,
		CoverRange:     m.CoverRange,
		RequiresActive: m.RequiresActive,
		AttackRange:    m.AttackRange,
		MovementSpeed:  m.MovementSpeed,
	}
}

// Intermediate struct for reading data from weapondata.json
// The json file contains both melee and ranged weapons, which
// Is why we have optional fields.
type JSONWeapon struct {
	Type          string          `json:"type"` // Can be "MeleeWeapon" or "RangedWeapon"
	Name          string          `json:"name"` // Weapon name
	ImgName       string          `json:"imgname"`
	MinDamage     int             `json:"minDamage"`               // Minimum damage
	MaxDamage     int             `json:"maxDamage"`               // Maximum damage
	AttackSpeed   int             `json:"attackSpeed"`             // Attack speed
	ShootingRange int             `json:"shootingRange,omitempty"` // For ranged weapons only
	AmmoType      string          `json:"ammoType,omitempty"`      // For ranged weapons only
	ShootingVX    string          `json:"shootingvx,omitempty"`
	TargetArea    *JSONTargetArea `json:"targetArea"`
}

type JSONAttributeModifier struct {
	Name       string `json:"name"`
	ImgName    string `json:"imgname"`
	Strength   int    `json:"strength"`
	Dexterity  int    `json:"dexterity"`
	Magic      int    `json:"magic"`
	Leadership int    `json:"leadership"`
	Armor      int    `json:"armor"`
	Weapon     int    `json:"weapon"`
	Duration   int    `json:"duration"`
}

func NewJSONAttributeModifier(a JSONAttributeModifier) JSONAttributeModifier {
	return JSONAttributeModifier{
		Name:       a.Name,
		ImgName:    a.ImgName,
		Strength:   a.Strength,
		Dexterity:  a.Dexterity,
		Magic:      a.Magic,
		Leadership: a.Leadership,
		Armor:      a.Armor,
		Weapon:     a.Weapon,
		Duration:   a.Duration,
	}
}

func CreateAttributesFromJSON(a JSONAttributeModifier) common.Attributes {
	// For consumables, create an attributes struct with modifiers only
	// Don't use NewAttributes since we don't want to initialize health
	return common.Attributes{
		Strength:   a.Strength,
		Dexterity:  a.Dexterity,
		Magic:      a.Magic,
		Leadership: a.Leadership,
		Armor:      a.Armor,
		Weapon:     a.Weapon,
		// Health fields left at zero - consumables will modify them separately
	}
}

type JSONCreatureModifier struct {
	Name              string   `json:"name"`
	AttackBonus       int      `json:"attackBonus"`
	MaxHealth         int      `json:"maxHealth"`
	CurrentHealth     int      `json:"currentHealth"`
	BaseArmorClass    int      `json:"baseArmorClass"`
	BaseProtection    int      `json:"baseProtection"`
	BaseMovementSpeed int      `json:"baseMovementSpeed"`
	BaseDodgeChance   float32  `json:"baseDodgeChance"`
	DamageBonus       int      `json:"damagebonus"`
	Width             int      `json:"width"`
	Height            int      `json:"height"`
	Role              string   `json:"role"`
	TargetCells       [][2]int `json:"targetCells"` // Cell-based targeting pattern
}

func CreatureModifierFromJSON(a JSONCreatureModifier) JSONCreatureModifier {
	return JSONCreatureModifier{
		Name:              a.Name,
		MaxHealth:         a.MaxHealth,
		CurrentHealth:     a.CurrentHealth,
		AttackBonus:       a.AttackBonus,
		BaseArmorClass:    a.BaseArmorClass,
		BaseProtection:    a.BaseProtection,
		BaseDodgeChance:   a.BaseDodgeChance,
		BaseMovementSpeed: a.BaseMovementSpeed,
		DamageBonus:       a.DamageBonus,
		Width:             a.Width,
		Height:            a.Height,
		Role:              a.Role,
		TargetCells:       a.TargetCells,
	}
}

// JSONEncounterDifficulty defines difficulty scaling for encounters
type JSONEncounterDifficulty struct {
	Level           int     `json:"level"`
	Name            string  `json:"name"`
	PowerMultiplier float64 `json:"powerMultiplier"`
	SquadCount      int     `json:"squadCount"`
}

// JSONSquadType defines squad type metadata (for future filtering/validation)
type JSONSquadType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// EncounterData is the root container for encounter configuration
type EncounterData struct {
	DifficultyLevels  []JSONEncounterDifficulty `json:"difficultyLevels"`
	SquadTypes        []JSONSquadType           `json:"squadTypes"`
	ThreatDefinitions []JSONThreatDefinition    `json:"threatDefinitions"` // Unified threat config
	DefaultThreat     *JSONDefaultThreat        `json:"defaultThreat"`     // Fallback for unknown threats
}

// JSONAIConfig is the root container for AI behavior configuration
type JSONAIConfig struct {
	ThreatCalculation ThreatCalculationConfig `json:"threatCalculation"`
	RoleBehaviors     []RoleBehaviorConfig    `json:"roleBehaviors"`
	SupportLayer      SupportLayerConfig      `json:"supportLayer"`
}

// ThreatCalculationConfig defines threat calculation parameters
type ThreatCalculationConfig struct {
	FlankingThreatRangeBonus   int `json:"flankingThreatRangeBonus"`
	IsolationThreshold         int `json:"isolationThreshold"`
	RetreatSafeThreatThreshold int `json:"retreatSafeThreatThreshold"`
}

// RoleBehaviorConfig defines how a role weighs different threat layers
// RangedWeight and PositionalWeight are hardcoded as shared constants.
type RoleBehaviorConfig struct {
	Role          string  `json:"role"`
	MeleeWeight   float64 `json:"meleeWeight"`
	SupportWeight float64 `json:"supportWeight"`
}

// SupportLayerConfig defines support layer parameters
type SupportLayerConfig struct {
	HealRadius                  int `json:"healRadius"`
	BuffPriorityEngagementRange int `json:"buffPriorityEngagementRange"`
}

// JSONPowerConfig is the root container for power evaluation configuration
type JSONPowerConfig struct {
	Profiles           []PowerProfileConfig     `json:"profiles"`
	RoleMultipliers    []RoleMultiplierConfig   `json:"roleMultipliers"`
	AbilityValues      []AbilityValueConfig     `json:"abilityValues"`
	CompositionBonuses []CompositionBonusConfig `json:"compositionBonuses"`
	LeaderBonus        float64                  `json:"leaderBonus"`
}

// PowerProfileConfig defines a power calculation profile.
// Only top-level category weights are configurable; sub-calculations use fixed formulas.
type PowerProfileConfig struct {
	Name            string  `json:"name"`
	OffensiveWeight float64 `json:"offensiveWeight"` // Weight for offensive stats (damage output)
	DefensiveWeight float64 `json:"defensiveWeight"` // Weight for defensive stats (survivability)
	UtilityWeight   float64 `json:"utilityWeight"`   // Weight for utility (role, abilities, cover)
	HealthPenalty   float64 `json:"healthPenalty"`   // Exponent for health-based power scaling
}

// RoleMultiplierConfig defines role multiplier value
type RoleMultiplierConfig struct {
	Role       string  `json:"role"`
	Multiplier float64 `json:"multiplier"`
}

// AbilityValueConfig defines ability power value
type AbilityValueConfig struct {
	Ability string  `json:"ability"`
	Power   float64 `json:"power"`
}

// CompositionBonusConfig defines composition bonus
type CompositionBonusConfig struct {
	UniqueTypes int     `json:"uniqueTypes"`
	Bonus       float64 `json:"bonus"`
}

// JSONOverworldConfig is the root container for overworld configuration
type JSONOverworldConfig struct {
	ThreatGrowth          ThreatGrowthConfig                `json:"threatGrowth"`
	FactionAI             FactionAIConfig                   `json:"factionAI"`
	SpawnProbabilities    SpawnProbabilitiesConfig          `json:"spawnProbabilities"`
	MapDimensions         MapDimensionsConfig               `json:"mapDimensions"`
	FactionScoring        FactionScoringConfig              `json:"factionScoring"`
	StrengthThresholds    StrengthThresholdsConfig          `json:"strengthThresholds"`
	FactionArchetypes     map[string]FactionArchetypeConfig `json:"factionArchetypes"`
	VictoryConditions     VictoryConditionsConfig           `json:"victoryConditions"`
	FactionScoringControl FactionScoringControlConfig       `json:"factionScoringControl"`
}

// ThreatGrowthConfig defines threat growth parameters
type ThreatGrowthConfig struct {
	ContainmentSlowdown       float64 `json:"containmentSlowdown"`
	MaxThreatIntensity        int     `json:"maxThreatIntensity"`
	ChildNodeSpawnThreshold   int     `json:"childNodeSpawnThreshold"`
	MaxChildNodeSpawnAttempts int     `json:"maxChildNodeSpawnAttempts"`
}

// FactionAIConfig defines faction AI behavior parameters
type FactionAIConfig struct {
	DefaultIntentTickDuration int `json:"defaultIntentTickDuration"`
	ExpansionTerritoryLimit   int `json:"expansionTerritoryLimit"`
	FortificationStrengthGain int `json:"fortificationStrengthGain"`
	MaxTerritorySize          int `json:"maxTerritorySize"`
}

// SpawnProbabilitiesConfig defines spawn and drop probabilities
type SpawnProbabilitiesConfig struct {
	ExpansionThreatSpawnChance int `json:"expansionThreatSpawnChance"`
	FortifyThreatSpawnChance   int `json:"fortifyThreatSpawnChance"`
	BonusItemDropChance        int `json:"bonusItemDropChance"`
}

// MapDimensionsConfig defines default map dimensions
type MapDimensionsConfig struct {
	DefaultMapWidth  int `json:"defaultMapWidth"`
	DefaultMapHeight int `json:"defaultMapHeight"`
}

// FactionScoringConfig defines faction intent scoring parameters
type FactionScoringConfig struct {
	Expansion     ExpansionScoringConfig     `json:"expansion"`
	Fortification FortificationScoringConfig `json:"fortification"`
	Raiding       RaidingScoringConfig       `json:"raiding"`
	Retreat       RetreatScoringConfig       `json:"retreat"`
}

// ExpansionScoringConfig defines expansion scoring parameters
type ExpansionScoringConfig struct {
	StrongBonus         float64 `json:"strongBonus"`
	SmallTerritoryBonus float64 `json:"smallTerritoryBonus"`
	MaxTerritoryPenalty float64 `json:"maxTerritoryPenalty"`
}

// FortificationScoringConfig defines fortification scoring parameters
type FortificationScoringConfig struct {
	WeakBonus float64 `json:"weakBonus"`
	BaseValue float64 `json:"baseValue"`
}

// RaidingScoringConfig defines raiding scoring parameters
type RaidingScoringConfig struct {
	StrongBonus float64 `json:"strongBonus"`
}

// RetreatScoringConfig defines retreat scoring parameters
type RetreatScoringConfig struct {
	CriticalWeakBonus     float64 `json:"criticalWeakBonus"`
	SmallTerritoryPenalty float64 `json:"smallTerritoryPenalty"`
	MinTerritorySize      int     `json:"minTerritorySize"`
}

// StrengthThresholdsConfig defines unified strength thresholds
type StrengthThresholdsConfig struct {
	Weak     int `json:"weak"`     // 0-weak = weak (fortify/retreat)
	Strong   int `json:"strong"`   // strong+ = strong (expand/raid)
	Critical int `json:"critical"` // 0-critical = critically weak (must retreat)
}

// FactionArchetypeConfig defines strategic archetype per faction
type FactionArchetypeConfig struct {
	Strategy   string  `json:"strategy"`
	Aggression float64 `json:"aggression"`
}

// VictoryConditionsConfig defines victory/defeat thresholds
type VictoryConditionsConfig struct {
	HighIntensityThreshold  int     `json:"highIntensityThreshold"`
	MaxHighIntensityThreats int     `json:"maxHighIntensityThreats"`
	MaxThreatInfluence      float64 `json:"maxThreatInfluence"`
}

// FactionScoringControlConfig defines faction scoring control parameters
type FactionScoringControlConfig struct {
	IdleScoreThreshold float64 `json:"idleScoreThreshold"`
	RaidBaseIntensity  int     `json:"raidBaseIntensity"`
	RaidIntensityScale float64 `json:"raidIntensityScale"`
}

// --- Unified Threat Definition Structs ---
// These structs enable loading ALL threat configuration from a single JSON array,
// replacing the need to edit 6-8 files when adding a new threat type.

// JSONColor represents an RGBA color in JSON
type JSONColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

// JSONThreatEncounterConfig defines encounter parameters for a threat
type JSONThreatEncounterConfig struct {
	TypeID            string   `json:"typeId"`            // e.g., "undead_basic"
	TypeName          string   `json:"typeName"`          // e.g., "Undead Horde"
	SquadPreferences  []string `json:"squadPreferences"`  // e.g., ["melee", "melee", "magic"]
	DefaultDifficulty int      `json:"defaultDifficulty"` // Default difficulty level
	Tags              []string `json:"tags"`              // e.g., ["common", "undead"]
}


// JSONThreatOverworldConfig defines overworld behavior parameters
type JSONThreatOverworldConfig struct {
	BaseGrowthRate   float64 `json:"baseGrowthRate"`   // Growth rate per tick
	BaseRadius       int     `json:"baseRadius"`       // Influence radius
	PrimaryEffect    string  `json:"primaryEffect"`    // "SpawnBoost", "ResourceDrain", etc.
	CanSpawnChildren bool    `json:"canSpawnChildren"` // Can spawn child nodes
}


// JSONThreatDefinition is the unified threat configuration
// A single entry in the threatDefinitions array replaces configuration
// previously spread across 6-8 files.
type JSONThreatDefinition struct {
	ID          string `json:"id"`          // Unique identifier, e.g., "necromancer"
	DisplayName string `json:"displayName"` // Human-readable name, e.g., "Necromancer"
	EnumValue   int    `json:"enumValue"`   // Maps to ThreatType enum (-1 for JSON-only threats)

	Encounter     JSONThreatEncounterConfig `json:"encounter"`     // Encounter configuration
	Color         JSONColor                 `json:"color"`         // Display color on overworld map
	Overworld     JSONThreatOverworldConfig `json:"overworld"`     // Overworld behavior
	BasicDrops    []string                  `json:"basicDrops"`    // Normal item drops
	HighTierDrops []string                  `json:"highTierDrops"` // Drops at high intensity
	FactionID     string                    `json:"factionId"`     // Faction mapping, e.g., "Necromancers"
}

// JSONDefaultThreat defines fallback configuration for unknown threats
type JSONDefaultThreat struct {
	DisplayName   string                    `json:"displayName"`
	Color         JSONColor                 `json:"color"`         // Display color on overworld map
	Overworld     JSONThreatOverworldConfig `json:"overworld"`     // Overworld behavior
	BasicDrops    []string                  `json:"basicDrops"`    // Normal item drops
	HighTierDrops []string                  `json:"highTierDrops"` // Drops at high intensity
}
