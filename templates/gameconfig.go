package templates

import "game_main/core/config"

// JSONGameConfig is the root container for game balance configuration.
// Loaded from gamedata/gameconfig.json at startup.
type JSONGameConfig struct {
	Player    PlayerConfig    `json:"player"`
	Commander CommanderConfig `json:"commander"`
	FactionAI FactionAIConfig2 `json:"factionAI"`
	Combat    CombatConfig    `json:"combat"`
	Display   DisplayConfig   `json:"display"`
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
	MaxUnits               int `json:"maxUnits"`
	MaxSquads              int `json:"maxSquads"`
	MaxArtifacts           int `json:"maxArtifacts"`
	MaxArtifactsPerCommander int `json:"maxArtifactsPerCommander"`
}

type CommanderConfig struct {
	MovementSpeed int `json:"movementSpeed"`
	MaxCommanders int `json:"maxCommanders"`
	Cost          int `json:"cost"`
	MaxSquads     int `json:"maxSquads"`
	StartingMana  int `json:"startingMana"`
	MaxMana       int `json:"maxMana"`
}

// FactionAIConfig2 avoids name collision with the existing FactionAIConfig in jsonschema.go.
type FactionAIConfig2 struct {
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

// GameConfig holds the loaded game configuration. Initialized by ReadGameConfig().
var GameConfig JSONGameConfig

func ReadGameConfig() {
	readAndUnmarshal("gamedata/gameconfig.json", &GameConfig)
	validateGameConfig(&GameConfig)

	// Populate config package variables for packages that can't import templates
	config.SetConfigFromJSON(
		GameConfig.Combat.DefaultMovementSpeed,
		GameConfig.Combat.DefaultAttackRange,
		GameConfig.Combat.BaseHitChance,
		GameConfig.Combat.MaxHitRate,
		GameConfig.Combat.MaxCritChance,
		GameConfig.Combat.MaxDodgeChance,
		GameConfig.Combat.BaseCapacity,
		GameConfig.Combat.MaxCapacity,
		GameConfig.Combat.BaseMagicResist,
		GameConfig.Combat.CritDamageBonus,
		GameConfig.Display.TilePixels,
		GameConfig.Display.ScaleFactor,
		GameConfig.Display.RightPadding,
		GameConfig.Display.ZoomSquares,
		GameConfig.Display.StaticUIOffset,
		GameConfig.Display.MapWidth,
		GameConfig.Display.MapHeight,
	)

	println("Game config loaded")
}

func validateGameConfig(cfg *JSONGameConfig) {
	// Player attributes
	if cfg.Player.Attributes.Strength < 0 || cfg.Player.Attributes.Dexterity < 0 {
		panic("Player attributes must be non-negative")
	}

	// Player resources
	if cfg.Player.Resources.Gold < 0 {
		panic("Player starting gold must be non-negative")
	}

	// Player limits
	if cfg.Player.Limits.MaxUnits <= 0 || cfg.Player.Limits.MaxArtifacts <= 0 {
		panic("Player limits must be positive")
	}

	// Commander
	if cfg.Commander.MovementSpeed <= 0 || cfg.Commander.MaxCommanders <= 0 {
		panic("Commander config values must be positive")
	}

	// Combat
	if cfg.Combat.DefaultMovementSpeed <= 0 || cfg.Combat.DefaultAttackRange <= 0 {
		panic("Combat movement speed and attack range must be positive")
	}
	if cfg.Combat.BaseHitChance < 0 || cfg.Combat.MaxHitRate <= 0 {
		panic("Combat hit chance values invalid")
	}
	if cfg.Combat.BaseCapacity <= 0 || cfg.Combat.MaxCapacity <= 0 {
		panic("Combat capacity values must be positive")
	}
	if cfg.Combat.CritDamageBonus < 0 {
		panic("Crit damage bonus must be non-negative")
	}

	// Display
	if cfg.Display.MapWidth <= 0 || cfg.Display.MapHeight <= 0 {
		panic("Display map dimensions must be positive")
	}
	if cfg.Display.TilePixels <= 0 || cfg.Display.ScaleFactor <= 0 {
		panic("Display tile/scale values must be positive")
	}
}
