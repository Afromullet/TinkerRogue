package config

// Game configuration constants and default values

// Debug and profiling flags
const (
	DISPLAY_THREAT_MAP_LOG_OUTPUT = true

	DISPLAY_DEATAILED_COMBAT_OUTPUT = false

	// DEBUG_MODE enables debug visualization and logging
	DEBUG_MODE = true

	// ENABLE_BENCHMARKING enables pprof profiling server on localhost:6060
	ENABLE_BENCHMARKING = false

	// ENABLE_COMBAT_LOG enables the combat log UI and logging during combat
	// When disabled, no combat log panel is created and no log messages are recorded
	ENABLE_COMBAT_LOG = false
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
	DefaultPlayerStartingGold = 100000 // Starting gold for purchasing units
	DefaultPlayerMaxUnits     = 50     // Maximum units player can own
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
)

// Default Graphic and Display Related Values

const (
	DefaultMapWidth           = 100
	DefaultMapHeight          = 80
	DefaultTilePixels         = 32
	DefaultScaleFactor        = 3
	DefaultRightPadding       = 500
	DefaultZoomNumberOfSquare = 30   //Number of squares we see when zoomed it
	DefaultStaticUIOffset     = 1000 //Leftover from the old GUI. Todo, determine if this is still needed.
)

// Asset paths
const (
	PlayerImagePath = "../assets/creatures/player1.png"
	AssetItemsDir   = "../assets/items/"
)

// Profiling configuration
const (
	ProfileServerAddr = "localhost:6060"
	CPUProfileRate    = 1000
	MemoryProfileRate = 1
)
