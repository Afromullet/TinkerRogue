package templates

import (
	"fmt"
	"game_main/core/config"
	"log"
)

// GameConfig holds the loaded game configuration. Initialized by ReadGameConfig().
var GameConfig JSONGameConfig

var gameConfigLoader = Loader[JSONGameConfig]{
	Name:     "gameconfig",
	Path:     GameConfigPath,
	Validate: validateGameConfig,
}

func ReadGameConfig() error {
	cfg, err := gameConfigLoader.Load()
	if err != nil {
		return err
	}
	GameConfig = cfg

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

	log.Printf("[templates] gameconfig loaded")
	return nil
}

func validateGameConfig(cfg *JSONGameConfig) error {
	// Player attributes
	if cfg.Player.Attributes.Strength < 0 || cfg.Player.Attributes.Dexterity < 0 {
		return fmt.Errorf("player attributes must be non-negative")
	}

	// Player resources
	if cfg.Player.Resources.Gold < 0 {
		return fmt.Errorf("player starting gold must be non-negative")
	}

	// Player limits
	if cfg.Player.Limits.MaxUnits <= 0 || cfg.Player.Limits.MaxArtifacts <= 0 {
		return fmt.Errorf("player limits must be positive")
	}

	// Commander
	if cfg.Commander.MovementSpeed <= 0 || cfg.Commander.MaxCommanders <= 0 {
		return fmt.Errorf("commander config values must be positive")
	}
	if len(cfg.Commander.StartingPerks) == 0 {
		return fmt.Errorf("commander.startingPerks must not be empty")
	}
	if len(cfg.Commander.StartingSpells) == 0 {
		return fmt.Errorf("commander.startingSpells must not be empty")
	}

	// Combat
	if cfg.Combat.DefaultMovementSpeed <= 0 || cfg.Combat.DefaultAttackRange <= 0 {
		return fmt.Errorf("combat movement speed and attack range must be positive")
	}
	if cfg.Combat.BaseHitChance < 0 || cfg.Combat.MaxHitRate <= 0 {
		return fmt.Errorf("combat hit chance values invalid")
	}
	if cfg.Combat.BaseCapacity <= 0 || cfg.Combat.MaxCapacity <= 0 {
		return fmt.Errorf("combat capacity values must be positive")
	}
	if cfg.Combat.CritDamageBonus < 0 {
		return fmt.Errorf("crit damage bonus must be non-negative")
	}

	// Display
	if cfg.Display.MapWidth <= 0 || cfg.Display.MapHeight <= 0 {
		return fmt.Errorf("display map dimensions must be positive")
	}
	if cfg.Display.TilePixels <= 0 || cfg.Display.ScaleFactor <= 0 {
		return fmt.Errorf("display tile/scale values must be positive")
	}

	// Encounter rewards
	if cfg.Encounter.BaseGold < 0 || cfg.Encounter.GoldPerIntensity < 0 {
		return fmt.Errorf("encounter gold values must be non-negative")
	}
	if cfg.Encounter.BaseXP < 0 || cfg.Encounter.XPPerIntensity < 0 {
		return fmt.Errorf("encounter XP values must be non-negative")
	}
	if cfg.Encounter.BasePoints < 0 || cfg.Encounter.PointsPerIntensity < 0 {
		return fmt.Errorf("encounter point values must be non-negative")
	}
	if cfg.Encounter.IntensityMultiplierStep < 0 {
		return fmt.Errorf("encounter intensityMultiplierStep must be non-negative")
	}
	return nil
}
