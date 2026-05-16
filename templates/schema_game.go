package templates

// JSONGameConfig is the root container for game balance configuration.
// Loaded from gamedata/gameconfig.json at startup.
type JSONGameConfig struct {
	Player    PlayerConfig             `json:"player"`
	Commander CommanderConfig          `json:"commander"`
	FactionAI FactionStartingResources `json:"factionAI"`
	Combat    CombatConfig             `json:"combat"`
	Display   DisplayConfig            `json:"display"`
	Encounter EncounterRewardsConfig   `json:"encounter"`
}

type PlayerConfig struct {
	Attributes PlayerAttributesConfig `json:"attributes"`
	Resources  PlayerResourcesConfig  `json:"resources"`
	Limits     PlayerLimitsConfig     `json:"limits"`
}

type PlayerAttributesConfig struct {
	Strength   int `json:"strength"`
	Dexterity  int `json:"dexterity"`
	Magic      int `json:"magic"`
	Leadership int `json:"leadership"`
	Armor      int `json:"armor"`
	Weapon     int `json:"weapon"`
}

type PlayerResourcesConfig struct {
	Gold  int `json:"gold"`
	Iron  int `json:"iron"`
	Wood  int `json:"wood"`
	Stone int `json:"stone"`
}

type PlayerLimitsConfig struct {
	MaxUnits                 int `json:"maxUnits"`
	MaxSquads                int `json:"maxSquads"`
	MaxArtifacts             int `json:"maxArtifacts"`
	MaxArtifactsPerCommander int `json:"maxArtifactsPerCommander"`
}

type CommanderConfig struct {
	MovementSpeed  int      `json:"movementSpeed"`
	MaxCommanders  int      `json:"maxCommanders"`
	Cost           int      `json:"cost"`
	MaxSquads      int      `json:"maxSquads"`
	StartingMana   int      `json:"startingMana"`
	MaxMana        int      `json:"maxMana"`
	StartingPerks  []string `json:"startingPerks"`
	StartingSpells []string `json:"startingSpells"`
}

// FactionStartingResources defines the per-faction starting stockpile populated
// from gameconfig.json's "factionAI" key. Distinct from FactionAIConfig in
// schema_overworld.go, which holds faction tactical AI tunables.
type FactionStartingResources struct {
	StartingGold  int `json:"startingGold"`
	StartingIron  int `json:"startingIron"`
	StartingWood  int `json:"startingWood"`
	StartingStone int `json:"startingStone"`
}

type CombatConfig struct {
	DefaultMovementSpeed int     `json:"defaultMovementSpeed"`
	DefaultAttackRange   int     `json:"defaultAttackRange"`
	BaseHitChance        int     `json:"baseHitChance"`
	MaxHitRate           int     `json:"maxHitRate"`
	MaxCritChance        int     `json:"maxCritChance"`
	MaxDodgeChance       int     `json:"maxDodgeChance"`
	BaseCapacity         int     `json:"baseCapacity"`
	MaxCapacity          int     `json:"maxCapacity"`
	BaseMagicResist      int     `json:"baseMagicResist"`
	CritDamageBonus      float64 `json:"critDamageBonus"`
}

type DisplayConfig struct {
	MapWidth       int `json:"mapWidth"`
	MapHeight      int `json:"mapHeight"`
	TilePixels     int `json:"tilePixels"`
	ScaleFactor    int `json:"scaleFactor"`
	RightPadding   int `json:"rightPadding"`
	ZoomSquares    int `json:"zoomSquares"`
	StaticUIOffset int `json:"staticUIOffset"`
}

// EncounterRewardsConfig holds the tunables for overworld threat-encounter reward
// scaling. Consumed by mind/encounter.CalculateIntensityReward.
//
// Formula: reward = (Base + PerIntensity * intensity) * (1.0 + MultiplierStep * intensity)
type EncounterRewardsConfig struct {
	BaseGold                int     `json:"baseGold"`
	GoldPerIntensity        int     `json:"goldPerIntensity"`
	BaseXP                  int     `json:"baseXP"`
	XPPerIntensity          int     `json:"xpPerIntensity"`
	BasePoints              int     `json:"basePoints"`
	PointsPerIntensity      int     `json:"pointsPerIntensity"`
	IntensityMultiplierStep float64 `json:"intensityMultiplierStep"`
}
