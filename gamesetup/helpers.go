package gamesetup

import (
	"log"
	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
	"runtime"

	"game_main/common"
	"game_main/config"
	"game_main/overworld/core"
	"game_main/testing"
	"game_main/world/worldmap"
)

// InitWalkableGridFromMap initializes the walkable grid and marks valid positions.
// Used by overworld, roguelike, and save-load initialization paths.
func InitWalkableGridFromMap(gm *worldmap.GameMap) {
	core.InitWalkableGrid(config.DefaultMapWidth, config.DefaultMapHeight)
	for _, pos := range gm.ValidPositions {
		core.SetTileWalkable(pos, true)
	}
}

// SetupTestData creates test items and content for debugging.
// Only called when DEBUG_MODE is enabled.
func SetupTestData(em *common.EntityManager, gm *worldmap.GameMap, pd *common.PlayerData) {
	testing.CreateTestItems(gm)
}

// SetupBenchmarking initializes performance profiling tools when enabled.
// It starts an HTTP server for pprof and configures CPU/memory profiling rates.
func SetupBenchmarking() {
	if !config.ENABLE_BENCHMARKING {
		return
	}

	// Start pprof HTTP server in background
	go func() {
		log.Println("Starting pprof server on", config.ProfileServerAddr)
		if err := http.ListenAndServe(config.ProfileServerAddr, nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()

	// Configure profiling rates
	runtime.SetCPUProfileRate(config.CPUProfileRate)
	runtime.MemProfileRate = config.MemoryProfileRate
}
