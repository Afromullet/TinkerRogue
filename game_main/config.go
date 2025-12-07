package main

import "game_main/config"

//TODO, remove this reimportingi n the future

// Re-export config constants for backward compatibility within game_main package
const (
	DEBUG_MODE          = config.DEBUG_MODE
	ENABLE_BENCHMARKING = config.ENABLE_BENCHMARKING
	ENABLE_COMBAT_LOG   = config.ENABLE_COMBAT_LOG
)

const (
	DefaultPlayerStrength     = config.DefaultPlayerStrength
	DefaultPlayerDexterity    = config.DefaultPlayerDexterity
	DefaultPlayerMagic        = config.DefaultPlayerMagic
	DefaultPlayerLeadership   = config.DefaultPlayerLeadership
	DefaultPlayerArmor        = config.DefaultPlayerArmor
	DefaultPlayerWeapon       = config.DefaultPlayerWeapon
	DefaultPlayerStartingGold = config.DefaultPlayerStartingGold
	DefaultPlayerMaxUnits     = config.DefaultPlayerMaxUnits
)

const (
	PlayerImagePath = config.PlayerImagePath
	AssetItemsDir   = config.AssetItemsDir
)

const (
	ProfileServerAddr = config.ProfileServerAddr
	CPUProfileRate    = config.CPUProfileRate
	MemoryProfileRate = config.MemoryProfileRate
)
