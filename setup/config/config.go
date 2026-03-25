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

// Default values in case JSON loading fails
var (
	DefaultMovementSpeed  = 3
	DefaultAttackRange    = 1
	DefaultBaseHitChance  = 80
	DefaultMaxHitRate     = 100
	DefaultMaxCritChance  = 50
	DefaultMaxDodgeChance = 30
	DefaultBaseCapacity   = 60
	DefaultMaxCapacity    = 150
	BaseMagicResist       = 5
	CritDamageBonus       = 0.5

	// Display defaults (used by coords package which can't import templates)
	DefaultTilePixels         = 32
	DefaultScaleFactor        = 3
	DefaultRightPadding       = 500
	DefaultZoomNumberOfSquare = 30
	DefaultStaticUIOffset     = 1000
	DefaultMapWidth           = 100
	DefaultMapHeight          = 80
)

// SetConfigFromJSON updates all config variables from loaded JSON.
// Called by templates.ReadGameConfig() after parsing gameconfig.json.
func SetConfigFromJSON(
	movementSpeed, attackRange, baseHitChance, maxHitRate, maxCritChance, maxDodgeChance, baseCapacity, maxCapacity, baseMagicResist int, critDamageBonus float64,
	tilePixels, scaleFactor, rightPadding, zoomSquares, staticUIOffset, mapWidth, mapHeight int,
) {
	DefaultMovementSpeed = movementSpeed
	DefaultAttackRange = attackRange
	DefaultBaseHitChance = baseHitChance
	DefaultMaxHitRate = maxHitRate
	DefaultMaxCritChance = maxCritChance
	DefaultMaxDodgeChance = maxDodgeChance
	DefaultBaseCapacity = baseCapacity
	DefaultMaxCapacity = maxCapacity
	BaseMagicResist = baseMagicResist
	CritDamageBonus = critDamageBonus

	DefaultTilePixels = tilePixels
	DefaultScaleFactor = scaleFactor
	DefaultRightPadding = rightPadding
	DefaultZoomNumberOfSquare = zoomSquares
	DefaultStaticUIOffset = staticUIOffset
	DefaultMapWidth = mapWidth
	DefaultMapHeight = mapHeight
}

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
