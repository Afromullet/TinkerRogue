package main

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/graphics"
	"game_main/gui"
	"game_main/input"
	"game_main/spawning"
	"game_main/squads"
	"game_main/systems"
	"game_main/testing"
	"game_main/worldmap"
	"log"
	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
	"runtime"
)

// SetupNewGame creates and initializes all game systems in the correct order.
// This is the main orchestration function for game initialization.
func SetupNewGame(g *Game) {
	// 1. Load game data from JSON files
	entitytemplates.ReadGameData()

	// 2. Initialize core game systems
	g.gameMap = worldmap.NewGameMap()
	InitializeECS(&g.em)

	// 2a. Initialize Position System for O(1) position lookups (Phase 0 - MASTER_ROADMAP)
	common.GlobalPositionSystem = systems.NewPositionSystem(g.em.World)

	// 3. Configure graphics system
	graphics.ScreenInfo.ScaleFactor = 1
	if graphics.MAP_SCROLLING_ENABLED {
		graphics.ScreenInfo.ScaleFactor = 3
	}

	// 4. Initialize player
	InitializePlayerData(&g.em, &g.playerData, &g.gameMap)

	// 5. Initialize spawning system
	spawning.InitLootSpawnTables()

	// 6. Setup test data if in debug mode
	if DEBUG_MODE {
		SetupTestData(&g.em, &g.gameMap, &g.playerData)
	}

	// 7. Spawn starting content
	testing.UpdateContentsForTest(&g.em, &g.gameMap)
	spawning.SpawnStartingCreatures(0, &g.em, &g.gameMap, &g.playerData)
	spawning.SpawnStartingEquipment(&g.em, &g.gameMap, &g.playerData)

	// 8. Register creatures with tracker
	AddCreaturesToTracker(&g.em)

	// 9. Initialize squad system (using game's EntityManager)
	if err := SetupSquadSystem(&g.em); err != nil {
		log.Fatalf("Failed to initialize squad system: %v", err)
	}

}

// SetupSquadSystem initializes the squad combat system.
// Pass the game's EntityManager so squads exist in the same ECS world.
func SetupSquadSystem(manager *common.EntityManager) error {
	if err := squads.InitializeSquadData(manager); err != nil {
		return err
	}

	// Create test squads if in debug mode

	if err := squads.CreateDummySquadsForTesting(manager); err != nil {
		return err
	}

	return nil
}

// SetupTestData creates test items and content for debugging.
// Only called when DEBUG_MODE is enabled.
func SetupTestData(em *common.EntityManager, gm *worldmap.GameMap, pd *avatar.PlayerData) {
	testing.CreateTestItems(em.World, em.Tags, gm)
	testing.InitTestActionManager(em, pd)
}

// SetupBenchmarking initializes performance profiling tools when enabled.
// It starts an HTTP server for pprof and configures CPU/memory profiling rates.
func SetupBenchmarking() {
	if !ENABLE_BENCHMARKING {
		return
	}

	// Start pprof HTTP server in background
	go func() {
		log.Println("Starting pprof server on", ProfileServerAddr)
		if err := http.ListenAndServe(ProfileServerAddr, nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()

	// Configure profiling rates
	runtime.SetCPUProfileRate(CPUProfileRate)
	runtime.MemProfileRate = MemoryProfileRate
}

// SetupUI initializes the new modal UI system.
// Must be called after game initialization but before input coordinator.
func SetupUI(g *Game) {
	// Create UI context with shared game state
	uiContext := &gui.UIContext{
		ECSManager:   &g.em,
		PlayerData:   &g.playerData,
		ScreenWidth:  graphics.ScreenInfo.GetCanvasWidth(),
		ScreenHeight: graphics.ScreenInfo.GetCanvasHeight(),
		TileSize:     graphics.ScreenInfo.TileSize,
	}

	// Create mode manager
	g.uiModeManager = gui.NewUIModeManager(uiContext)

	// Register all UI modes
	explorationMode := gui.NewExplorationMode(g.uiModeManager)
	if err := g.uiModeManager.RegisterMode(explorationMode); err != nil {
		log.Fatalf("Failed to register exploration mode: %v", err)
	}

	squadManagementMode := gui.NewSquadManagementMode(g.uiModeManager)
	if err := g.uiModeManager.RegisterMode(squadManagementMode); err != nil {
		log.Fatalf("Failed to register squad management mode: %v", err)
	}

	combatMode := gui.NewCombatMode(g.uiModeManager)
	if err := g.uiModeManager.RegisterMode(combatMode); err != nil {
		log.Fatalf("Failed to register combat mode: %v", err)
	}

	inventoryMode := gui.NewInventoryMode(g.uiModeManager)
	if err := g.uiModeManager.RegisterMode(inventoryMode); err != nil {
		log.Fatalf("Failed to register inventory mode: %v", err)
	}

	formationEditorMode := gui.NewFormationEditorMode(g.uiModeManager)
	if err := g.uiModeManager.RegisterMode(formationEditorMode); err != nil {
		log.Fatalf("Failed to register formation editor mode: %v", err)
	}

	// Set initial mode to exploration
	if err := g.uiModeManager.SetMode("exploration"); err != nil {
		log.Fatalf("Failed to set exploration mode: %v", err)
	}
}

// SetupInputCoordinator initializes the input handling system.
// Must be called after UI is created.
func SetupInputCoordinator(g *Game) {
	// InputCoordinator now works without PlayerUI reference
	g.inputCoordinator = input.NewInputCoordinator(&g.em, &g.playerData, &g.gameMap, nil)
}
