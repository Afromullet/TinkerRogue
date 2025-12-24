package main

import (
	"game_main/common"
	"game_main/config"
	"game_main/tactical/squads"
	"game_main/visual/graphics"
	"game_main/world/coords"

	"game_main/gui/core"
	"game_main/gui/guicombat"
	"game_main/gui/guicomponents"

	"game_main/gui/guimodes"
	"game_main/gui/guiresources"
	"game_main/gui/guisquads"
	"game_main/input"
	"game_main/templates"

	"game_main/tactical/combat"
	"game_main/testing"
	"game_main/visual/rendering"
	"game_main/world/worldmap"
	"log"
	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
	"runtime"
)

// SetupNewGame creates and initializes all game systems in the correct order.
// This is the main orchestration function for game initialization.
func SetupNewGame(g *Game) {
	// 1. Load game data from JSON files
	templates.ReadGameData()

	// 2. Initialize core game systems
	//g.gameMap = worldmap.NewGameMapDefault()
	g.gameMap = worldmap.NewGameMap("overworld")
	//g.gameMap = worldmap.NewGameMap("hybrid_tactical")
	//g.gameMap = worldmap.NewGameMap("wavelet_procedural")
	//g.gameMap = worldmap.NewGameMap("cave_tactical")

	InitializeECS(&g.em)

	// 2a. Initialize Position System for O(1) position lookups (Phase 0 - MASTER_ROADMAP)
	common.GlobalPositionSystem = common.NewPositionSystem(g.em.World)

	g.renderingCache = rendering.NewRenderingCache(&g.em)

	// 3. Configure graphics system
	graphics.ScreenInfo.ScaleFactor = 1
	if coords.MAP_SCROLLING_ENABLED {
		graphics.ScreenInfo.ScaleFactor = 3
	}

	// 4. Initialize player
	InitializePlayerData(&g.em, &g.playerData, &g.gameMap)

	// 5. Setup test data if in debug mode
	if config.DEBUG_MODE {
		SetupTestData(&g.em, &g.gameMap, &g.playerData)
	}

	// 6. Spawn starting content
	testing.UpdateContentsForTest(&g.em, &g.gameMap)

	// 7. Register creatures with tracker
	AddCreaturesToTracker(&g.em)

	// 8. Initialize squad system (using game's EntityManager)
	if err := SetupSquadSystem(&g.em); err != nil {
		log.Fatalf("Failed to initialize squad system: %v", err)
	}

	// 9. Setup gameplay factions and squads for testing
	if err := SetupGameplayFactions(&g.em, &g.playerData); err != nil {
		log.Fatalf("Failed to setup gameplay factions: %v", err)
	}

}

// SetupSquadSystem initializes the squad combat system.
// Pass the game's EntityManager so squads exist in the same ECS world.
// Note: Squad components and tags are already registered in InitializeECS
func SetupSquadSystem(manager *common.EntityManager) error {
	// Initialize unit templates from JSON
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		return err
	}

	// Create test squads if in debug mode
	//if err := squads.CreateDummySquadsForTesting(manager); err != nil {
	//	return err
	//}

	return nil
}

// SetupGameplayFactions creates two factions with squads for gameplay testing.
// This sets up player and AI factions with 3 squads each positioned on the map.
func SetupGameplayFactions(manager *common.EntityManager, playerData *common.PlayerData) error {
	return combat.SetupGameplayFactions(manager, *playerData.Pos)
}

// SetupTestData creates test items and content for debugging.
// Only called when DEBUG_MODE is enabled.
func SetupTestData(em *common.EntityManager, gm *worldmap.GameMap, pd *common.PlayerData) {
	testing.CreateTestItems(em.World, em.WorldTags, gm)
	testing.InitTestActionManager(em, pd)
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

// SetupUI initializes the new modal UI system with separate context managers.
// Must be called after game initialization but before input coordinator.
func SetupUI(g *Game) {
	// Pre-cache ScrollContainer backgrounds for performance (reduces NineSlice overhead by 80%)
	// This warms the cache before UI creation to avoid first-frame rendering stutter
	guiresources.PreCacheScrollContainerBackgrounds()

	// Create UI context with shared game state
	uiContext := &core.UIContext{
		ECSManager:   &g.em,
		PlayerData:   &g.playerData,
		GameMap:      &g.gameMap, //Todo, remove in future. Used by dangervisualizer, which is for debugging
		ScreenWidth:  graphics.ScreenInfo.GetCanvasWidth(),
		ScreenHeight: graphics.ScreenInfo.GetCanvasHeight(),
		TileSize:     graphics.ScreenInfo.TileSize,
		Queries:      guicomponents.NewGUIQueries(&g.em),
	}

	// Pre-cache based on actual screen dimensions for optimal cache hit rate
	guiresources.PreCacheScrollContainerSizes(uiContext.ScreenWidth, uiContext.ScreenHeight)

	// Create game mode coordinator (manages two separate contexts)
	g.gameModeCoordinator = core.NewGameModeCoordinator(uiContext)

	// Set coordinator reference in context so modes can trigger context switches
	uiContext.ModeCoordinator = g.gameModeCoordinator

	// Get references to both managers for registration
	battleMapManager := g.gameModeCoordinator.GetBattleMapManager()
	overworldManager := g.gameModeCoordinator.GetOverworldManager()

	// Register all battle map modes (tactical layer)
	registerBattleMapModes(g.gameModeCoordinator, battleMapManager)

	// Register all overworld modes (strategic layer)
	registerOverworldModes(g.gameModeCoordinator, overworldManager)

	// Set initial context and mode (start in battle map, exploration mode)
	if err := g.gameModeCoordinator.EnterBattleMap("exploration"); err != nil {
		log.Fatalf("Failed to set initial battle map mode: %v", err)
	}
}

// SetupInputCoordinator initializes the input handling system.
// Must be called after UI is created.
func SetupInputCoordinator(g *Game) {
	// InputCoordinator now works without PlayerUI reference
	g.inputCoordinator = input.NewInputCoordinator(&g.em, &g.playerData, &g.gameMap, nil)
}

// registerBattleMapModes registers all battle map UI modes with the coordinator.
func registerBattleMapModes(coordinator *core.GameModeCoordinator, manager *core.UIModeManager) {
	modes := []core.UIMode{
		guimodes.NewExplorationMode(manager),
		guimodes.NewInfoMode(manager),
		guicombat.NewCombatMode(manager),
		guicombat.NewCombatAnimationMode(manager),
		guisquads.NewSquadDeploymentMode(manager),
		newInventoryModeWithReturn(manager, "exploration"),
	}

	for _, mode := range modes {
		if err := coordinator.RegisterBattleMapMode(mode); err != nil {
			log.Fatalf("Failed to register battle map mode '%s': %v", mode.GetModeName(), err)
		}
	}
}

// registerOverworldModes registers all overworld UI modes with the coordinator.
// This reduces boilerplate by iterating over a slice of mode constructors.
func registerOverworldModes(coordinator *core.GameModeCoordinator, manager *core.UIModeManager) {
	modes := []core.UIMode{
		guisquads.NewSquadManagementMode(manager),
		guisquads.NewFormationEditorMode(manager),
		guisquads.NewSquadBuilderMode(manager),
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewSquadEditorMode(manager),
		newInventoryModeWithReturn(manager, "squad_management"),
	}

	for _, mode := range modes {
		if err := coordinator.RegisterOverworldMode(mode); err != nil {
			log.Fatalf("Failed to register overworld mode '%s': %v", mode.GetModeName(), err)
		}
	}
}

// newInventoryModeWithReturn creates an inventory mode configured with a return mode.
// This helper eliminates duplicate inventory mode setup code.
func newInventoryModeWithReturn(manager *core.UIModeManager, returnMode string) core.UIMode {
	mode := guimodes.NewInventoryMode(manager)
	mode.SetReturnMode(returnMode)
	return mode
}
