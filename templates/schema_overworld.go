package templates

import "game_main/campaign/overworld/ids"

// schema_overworld.go holds JSON DTOs for the entire overworld strategic layer:
//   - Overworld config (threat growth, faction AI, scoring) from overworldconfig.json
//   - Influence interaction config from influenceconfig.json
//   - Node definitions (threats, settlements, fortresses) from nodeDefinitions.json
//   - Encounter definitions (combat mechanics) from encounterdata.json

// --- Overworld config (overworldconfig.json) ---

// PlayerNodeConfig defines player node placement rules.
type PlayerNodeConfig struct {
	MaxPlacementRange int `json:"maxPlacementRange"`
	MaxNodes          int `json:"maxNodes"`
}

// JSONOverworldConfig is the root container for overworld configuration.
type JSONOverworldConfig struct {
	ThreatGrowth          ThreatGrowthConfig             `json:"threatGrowth"`
	FactionAI             FactionAIConfig                `json:"factionAI"`
	SpawnProbabilities    SpawnProbabilitiesConfig       `json:"spawnProbabilities"`
	MapDimensions         MapDimensionsConfig            `json:"mapDimensions"`
	FactionScoring        FactionScoringConfig           `json:"factionScoring"`
	StrengthThresholds    StrengthThresholdsConfig       `json:"strengthThresholds"`
	VictoryConditions     VictoryConditionsConfig        `json:"victoryConditions"`
	FactionScoringControl FactionScoringControlConfig    `json:"factionScoringControl"`
	PlayerNodes           PlayerNodeConfig               `json:"playerNodes"`
	StrategyBonuses       map[string]StrategyBonusConfig `json:"strategyBonuses"`
}

// ThreatGrowthConfig defines threat growth parameters.
type ThreatGrowthConfig struct {
	ContainmentSlowdown     float64 `json:"containmentSlowdown"`
	MaxThreatIntensity      int     `json:"maxThreatIntensity"`
	ChildNodeSpawnThreshold int     `json:"childNodeSpawnThreshold"`
}

// FactionAIConfig defines faction AI behavior parameters. Distinct from
// FactionStartingResources in schema_game.go, which holds per-faction
// starting stockpile values from gameconfig.json.
type FactionAIConfig struct {
	DefaultIntentTickDuration int `json:"defaultIntentTickDuration"`
	MaxTerritorySize          int `json:"maxTerritorySize"`
}

// SpawnProbabilitiesConfig defines spawn probabilities.
type SpawnProbabilitiesConfig struct {
	ExpansionThreatSpawnChance int `json:"expansionThreatSpawnChance"`
	FortifyThreatSpawnChance   int `json:"fortifyThreatSpawnChance"`
}

// MapDimensionsConfig defines default map dimensions.
type MapDimensionsConfig struct {
	DefaultMapWidth  int `json:"defaultMapWidth"`
	DefaultMapHeight int `json:"defaultMapHeight"`
}

// FactionScoringConfig defines faction intent scoring parameters.
type FactionScoringConfig struct {
	Expansion     ExpansionScoringConfig     `json:"expansion"`
	Fortification FortificationScoringConfig `json:"fortification"`
	Raiding       RaidingScoringConfig       `json:"raiding"`
	Retreat       RetreatScoringConfig       `json:"retreat"`
}

// ExpansionScoringConfig defines expansion scoring parameters.
type ExpansionScoringConfig struct {
	StrongBonus         float64 `json:"strongBonus"`
	SmallTerritoryBonus float64 `json:"smallTerritoryBonus"`
	MaxTerritoryPenalty float64 `json:"maxTerritoryPenalty"`
}

// FortificationScoringConfig defines fortification scoring parameters.
type FortificationScoringConfig struct {
	WeakBonus float64 `json:"weakBonus"`
	BaseValue float64 `json:"baseValue"`
}

// RaidingScoringConfig defines raiding scoring parameters.
type RaidingScoringConfig struct {
	StrongBonus      float64 `json:"strongBonus"`
	VeryStrongOffset int     `json:"veryStrongOffset"` // Offset above strong threshold for bonus raid score
}

// RetreatScoringConfig defines retreat scoring parameters.
type RetreatScoringConfig struct {
	CriticalWeakBonus     float64 `json:"criticalWeakBonus"`
	SmallTerritoryPenalty float64 `json:"smallTerritoryPenalty"`
	MinTerritorySize      int     `json:"minTerritorySize"`
}

// StrengthThresholdsConfig defines unified strength thresholds.
type StrengthThresholdsConfig struct {
	Weak     int `json:"weak"`     // 0-weak = weak (fortify/retreat)
	Strong   int `json:"strong"`   // strong+ = strong (expand/raid)
	Critical int `json:"critical"` // 0-critical = critically weak (must retreat)
}

// FactionArchetypeConfig defines strategic archetype per faction.
type FactionArchetypeConfig struct {
	Strategy string `json:"strategy"`
}

// VictoryConditionsConfig defines victory/defeat thresholds.
type VictoryConditionsConfig struct {
	HighIntensityThreshold  int     `json:"highIntensityThreshold"`
	MaxHighIntensityThreats int     `json:"maxHighIntensityThreats"`
	MaxThreatInfluence      float64 `json:"maxThreatInfluence"`
}

// StrategyBonusConfig defines per-strategy scoring bonuses.
type StrategyBonusConfig struct {
	ExpansionBonus     float64 `json:"expansionBonus"`
	FortificationBonus float64 `json:"fortificationBonus"`
	RaidingBonus       float64 `json:"raidingBonus"`
	RetreatPenalty     float64 `json:"retreatPenalty"`
}

// FactionScoringControlConfig defines faction scoring control parameters.
type FactionScoringControlConfig struct {
	IdleScoreThreshold float64 `json:"idleScoreThreshold"`
	RaidBaseIntensity  int     `json:"raidBaseIntensity"`
	RaidIntensityScale float64 `json:"raidIntensityScale"`
}

// --- Influence interaction config (influenceconfig.json) ---

// JSONInfluenceConfig is the root container for influence interaction configuration.
type JSONInfluenceConfig struct {
	BaseMagnitudeMultiplier    float64                    `json:"baseMagnitudeMultiplier"`
	DefaultPlayerNodeMagnitude float64                    `json:"defaultPlayerNodeMagnitude"`
	DefaultPlayerNodeRadius    int                        `json:"defaultPlayerNodeRadius"`
	Synergy                    InfluenceSynergyConfig     `json:"synergy"`
	Competition                InfluenceCompetitionConfig `json:"competition"`
	Suppression                InfluenceSuppressionConfig `json:"suppression"`
}

// InfluenceSynergyConfig defines same-faction threat synergy parameters.
type InfluenceSynergyConfig struct {
	GrowthBonus float64 `json:"growthBonus"`
}

// InfluenceCompetitionConfig defines rival-faction competition parameters.
type InfluenceCompetitionConfig struct {
	GrowthPenalty float64 `json:"growthPenalty"`
}

// InfluenceSuppressionConfig defines player node suppression parameters.
type InfluenceSuppressionConfig struct {
	GrowthPenalty       float64                    `json:"growthPenalty"`
	NodeTypeMultipliers map[ids.NodeTypeID]float64 `json:"nodeTypeMultipliers"`
}

// --- Node definitions (nodeDefinitions.json) ---

// JSONNodeOverworld defines overworld behavior for a node.
type JSONNodeOverworld struct {
	BaseGrowthRate   float64 `json:"baseGrowthRate,omitempty"`   // Growth rate per tick
	BaseRadius       int     `json:"baseRadius"`                 // Influence radius
	CanSpawnChildren bool    `json:"canSpawnChildren,omitempty"` // Can spawn child nodes
}

// JSONNodeDefinition is the unified node configuration.
// A single entry defines an overworld node (threat, settlement, fortress, etc.).
type JSONNodeDefinition struct {
	ID          ids.NodeTypeID `json:"id"`          // Unique identifier, e.g., "necromancer" - this IS the type
	Category    string         `json:"category"`    // "threat", "settlement", "fortress"
	DisplayName string         `json:"displayName"` // Human-readable name

	// Required marks node IDs that must be present after loading. Validators
	// derive the "required list" from this flag, so adding a new mandatory
	// threat node is a one-line JSON edit (no Go change).
	Required bool `json:"required,omitempty"`

	Color     JSONColor         `json:"color"`               // Display color on overworld map
	Overworld JSONNodeOverworld `json:"overworld"`           // Overworld behavior
	FactionID ids.FactionID     `json:"factionId,omitempty"` // Faction this node belongs to (for threat nodes)
	Cost      *JSONResourceCost `json:"cost,omitempty"`      // Resource cost to place this node
}

// JSONDefaultNode defines fallback configuration for unknown nodes.
type JSONDefaultNode struct {
	DisplayName string            `json:"displayName"`
	Color       JSONColor         `json:"color"`
	Overworld   JSONNodeOverworld `json:"overworld"`
}

// NodeDefinitionsData is the root container for node definitions.
type NodeDefinitionsData struct {
	NodeCategories []string             `json:"nodeCategories"` // Valid categories
	Nodes          []JSONNodeDefinition `json:"nodes"`
	DefaultNode    *JSONDefaultNode     `json:"defaultNode"`
}

// --- Encounter definitions (encounterdata.json) ---

// JSONEncounterDefinition defines combat mechanics for an encounter.
type JSONEncounterDefinition struct {
	ID                ids.EncounterID     `json:"id"`                // Unique identifier
	EncounterTypeID   ids.EncounterTypeID `json:"encounterTypeId"`   // e.g., "undead_basic"
	EncounterTypeName string              `json:"encounterTypeName"` // e.g., "Undead Horde"
	SquadPreferences  []string            `json:"squadPreferences"`  // e.g., ["melee", "melee", "magic"]
	DefaultDifficulty int                 `json:"defaultDifficulty"` // Default difficulty level
	Tags              []string            `json:"tags"`              // e.g., ["common", "undead"]
	FactionID         ids.FactionID       `json:"factionId"`         // Faction mapping
}
