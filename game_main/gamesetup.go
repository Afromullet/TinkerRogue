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
	"game_main/world/encounter"
	"game_main/world/worldmap"
	"log"
	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
	"runtime"
)

// GameBootstrap encapsulates game initialization logic with explicit phases.
// Each phase method represents a discrete initialization step with clear dependencies.
type GameBootstrap struct{}

// NewGameBootstrap creates a new bootstrap orchestrator.
func NewGameBootstrap() *GameBootstrap {
	return &GameBootstrap{}
}

// LoadGameData loads static game data from JSON files.
// Phase 1: No dependencies, must run first.
func (gb *GameBootstrap) LoadGameData() {
	templates.ReadGameData()
}

// InitializeCoreECS initializes the ECS world and global systems.
// Phase 2: Depends on LoadGameData for templates.
func (gb *GameBootstrap) InitializeCoreECS(em *common.EntityManager) {
	InitializeECS(em)

	// Initialize Position System for O(1) position lookups (Phase 0 - MASTER_ROADMAP)
	common.GlobalPositionSystem = common.NewPositionSystem(em.World)

	// Configure graphics system
	graphics.ScreenInfo.ScaleFactor = 1
	if coords.MAP_SCROLLING_ENABLED {
		graphics.ScreenInfo.ScaleFactor = 3
	}
}

// CreateWorld generates the game map.
// Phase 3: Depends on InitializeCoreECS for coordinate system.
func (gb *GameBootstrap) CreateWorld(gm *worldmap.GameMap) {
	// Multiple map generation algorithms available:
	// - "overworld" (default)
	// - "hybrid_tactical"
	*gm = worldmap.NewGameMap("overworld")
}

// CreatePlayer initializes the player entity and adds creatures to position system.
// Phase 4: Depends on CreateWorld for starting position.
func (gb *GameBootstrap) CreatePlayer(em *common.EntityManager, pd *common.PlayerData, gm *worldmap.GameMap) {
	InitializePlayerData(em, pd, gm)
	AddCreaturesToTracker(em)
}

// SetupDebugContent creates test items and spawns debug content.
// Debug Phase: Only runs when DEBUG_MODE is enabled.
func (gb *GameBootstrap) SetupDebugContent(em *common.EntityManager, gm *worldmap.GameMap, pd *common.PlayerData) {
	if config.DEBUG_MODE {
		SetupTestData(em, gm, pd)
	}

	// Spawn starting content (test enemies, items, etc.)
	testing.UpdateContentsForTest(em, gm)
}

// InitializeGameplay sets up squad system and exploration squads.
// Phase 5: Depends on CreatePlayer for faction positioning.
func (gb *GameBootstrap) InitializeGameplay(em *common.EntityManager, pd *common.PlayerData) {
	// Initialize squad system
	if err := SetupSquadSystem(em); err != nil {
		log.Fatalf("Failed to initialize squad system: %v", err)
	}

	// Spawn test encounters on overworld
	encounter.SpawnTestEncounters(em, *pd.Pos)
}

// SetupNewGame orchestrates game initialization through explicit phases.
// Each phase is named and testable, making dependencies clear.
func SetupNewGame(g *Game) {
	bootstrap := NewGameBootstrap()

	// Phase 1: Load static data
	bootstrap.LoadGameData()

	// Phase 2: Initialize ECS and global systems
	bootstrap.InitializeCoreECS(&g.em)

	// Phase 3: Create game world
	bootstrap.CreateWorld(&g.gameMap)

	// Initialize rendering cache (depends on ECS)
	g.renderingCache = rendering.NewRenderingCache(&g.em)

	// Phase 4: Create player entity
	bootstrap.CreatePlayer(&g.em, &g.playerData, &g.gameMap)

	// Debug Phase: Setup test content
	bootstrap.SetupDebugContent(&g.em, &g.gameMap, &g.playerData)

	// Phase 5: Initialize gameplay systems
	bootstrap.InitializeGameplay(&g.em, &g.playerData)
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
// NOTE: This is now called ONLY when entering combat mode, not during game initialization
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
	// Pass ModeCoordinator so MovementController can trigger encounters
	g.inputCoordinator = input.NewInputCoordinator(&g.em, &g.playerData, &g.gameMap, g.gameModeCoordinator)
}

// registerBattleMapModes registers all battle map UI modes with the coordinator.
func registerBattleMapModes(coordinator *core.GameModeCoordinator, manager *core.UIModeManager) {
	modes := []core.UIMode{
		guimodes.NewExplorationMode(manager),
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
