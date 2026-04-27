package gamesetup

import (
	"log"
	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
	"runtime"

	"game_main/campaign/overworld/core"
	"game_main/core/config"
	"game_main/templates"
	"game_main/world/worldmapcore"
)

// InitWalkableGridFromMap initializes the walkable grid and marks valid positions.
// Used by overworld, roguelike, and save-load initialization paths.
func InitWalkableGridFromMap(gm *worldmapcore.GameMap) {
	core.InitWalkableGrid(templates.GameConfig.Display.MapWidth, templates.GameConfig.Display.MapHeight)
	for _, pos := range gm.ValidPositions {
		core.SetTileWalkable(pos, true)
	}
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
