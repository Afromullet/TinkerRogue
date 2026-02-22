package config

import (
	"os"
	"path/filepath"
)

// Game configuration constants and default values

// Debug and profiling flags
const (
	DISPLAY_THREAT_MAP_LOG_OUTPUT = false

	DISPLAY_DEATAILED_COMBAT_OUTPUT = false

	// DEBUG_MODE enables debug visualization and logging
	DEBUG_MODE = true

	// ENABLE_BENCHMARKING enables pprof profiling server on localhost:6060
	ENABLE_BENCHMARKING = true

	// ENABLE_COMBAT_LOG_EXPORT enables JSON export of battle logs for post-combat analysis
	// When enabled, a JSON file is written to COMBAT_LOG_EXPORT_DIR after each battle
	ENABLE_COMBAT_LOG_EXPORT = false
	COMBAT_LOG_EXPORT_DIR    = "./combat_logs"

	// ENABLE_OVERWORLD_LOG_EXPORT enables JSON export of overworld session logs for post-game analysis
	// When enabled, a JSON file is written to OVERWORLD_LOG_EXPORT_DIR after game end (victory/defeat)
	ENABLE_OVERWORLD_LOG_EXPORT = false
	OVERWORLD_LOG_EXPORT_DIR    = "./overworld_logs"
)

// Default player starting attributes
// These are used when initializing a new player character
const (
	DefaultPlayerStrength   = 15 // → 50 HP (20 + 15*2)
	DefaultPlayerDexterity  = 20 // → 100% hit, 10% crit, 6% dodge
	DefaultPlayerMagic      = 0  // → Player starts without magic abilities1
	DefaultPlayerLeadership = 0  // → Player doesn't start with squad leadership
	DefaultPlayerArmor      = 2  // → 4 physical resistance (2*2)
	DefaultPlayerWeapon     = 3  // → 6 bonus damage (3*2)
)

// Default player resources and roster limits
const (
	DefaultPlayerStartingGold       = 100000 // Starting gold for purchasing units
	DefaultPlayerMaxUnits           = 500    // Maximum units player can own
	DefaultPlayerMaxSquads          = 50     // Maximum squads player can own
	DefaultPlayerStartingIron       = 50     // Starting iron for node placement
	DefaultPlayerStartingWood       = 50     // Starting wood for node placement
	DefaultPlayerStartingStone      = 50     // Starting stone for node placement
	DefaultPlayerMaxArtifacts       = 50     // Maximum artifacts player can own
	DefaultMaxArtifactsPerCommander = 30     //TODO, change later. The value is high for testing
)

// Commander system defaults
const (
	DefaultCommanderMovementSpeed = 25   // Tiles per overworld turn
	DefaultMaxCommanders          = 3    // Maximum commanders player can control
	DefaultCommanderCost          = 5000 // Gold cost to recruit a new commander
	DefaultCommanderMaxSquads     = 50   // Max squads per commander
	DefaultCommanderStartingMana  = 50   // Starting mana for new commanders
	DefaultCommanderMaxMana       = 50   // Maximum mana pool for new commanders
)

// Default faction AI starting resources
const (
	DefaultFactionStartingGold  = 100000
	DefaultFactionStartingIron  = 30
	DefaultFactionStartingWood  = 30
	DefaultFactionStartingStone = 30
)

// Default Unit Attributes
const (
	DefaultMovementSpeed  = 3
	DefaultAttackRange    = 1
	DefaultBaseHitChance  = 80
	DefaultMaxHitRate     = 100
	DefaultMaxCritChance  = 50
	DefaultMaxDodgeChance = 30
	DefaultBaseCapacity   = 6
	DefaultMaxCapacity    = 9
	BaseMagicResist       = 5
)

// Critical hit constants
const (
	// CritDamageBonus is the extra damage multiplier on critical hits.
	// Crits deal (1 + CritDamageBonus) = 1.5x damage.
	// Used in expected damage calculations: expectedMult = 1 + (critChance * CritDamageBonus)
	CritDamageBonus = 0.5
)

// Default Graphic and Display Related Values

const (
	DefaultMapWidth           = 100
	DefaultMapHeight          = 80
	DefaultTilePixels         = 32
	DefaultScaleFactor        = 3
	DefaultRightPadding       = 500
	DefaultZoomNumberOfSquare = 30 //Number of squares we see when zoomed it
	DefaultStaticUIOffset     = 1000
)

// assetRoot is the resolved path to the assets directory.
// Initialized once on first use via getAssetRoot().
var assetRoot string

// getAssetRoot returns the path to the assets directory, detecting whether
// we're running from game_main/ (legacy) or the project root.
func getAssetRoot() string {
	if assetRoot != "" {
		return assetRoot
	}
	if info, err := os.Stat("assets"); err == nil && info.IsDir() {
		assetRoot = "assets"
		return assetRoot
	}
	assetRoot = filepath.Join("..", "assets")
	return assetRoot
}

// AssetPath builds a path relative to the assets directory.
// Works regardless of whether the binary runs from the project root or game_main/.
func AssetPath(relative string) string {
	return filepath.Join(getAssetRoot(), relative)
}

// Asset paths
var (
	PlayerImagePath = AssetPath("creatures/player1.png")
	AssetItemsDir   = AssetPath("items")
)

// Profiling configuration
const (
	ProfileServerAddr = "localhost:6060"
	CPUProfileRate    = 1000
	MemoryProfileRate = 1
)
