package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
	"runtime"

	"game_main/common"
	"game_main/config"
	"game_main/gui/framework"
	"game_main/gui/widgetresources"
	"game_main/input"
	"game_main/mind/encounter"
	"game_main/mind/raid"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/templates"
	"game_main/testing"
	"game_main/testing/bootstrap"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
func (gb *GameBootstrap) CreateWorld(gm *worldmap.GameMap, mapType string) {
	*gm = worldmap.NewGameMap(mapType)
}

// CreatePlayer initializes the player entity, creates the initial commander, and adds creatures to position system.
// Phase 4: Depends on CreateWorld for starting position.
func (gb *GameBootstrap) CreatePlayer(em *common.EntityManager, pd *common.PlayerData, gm *worldmap.GameMap) {
	InitializePlayerData(em, pd, gm)

	// Initialize unit templates before creating initial squads
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		log.Fatalf("Failed to load unit templates: %v", err)
	}

	// Create initial commander at the player's starting position
	commanderImage, _, err := ebitenutil.NewImageFromFile(config.PlayerImagePath)
	if err != nil {
		log.Fatalf("Failed to load commander image: %v", err)
	}

	commanderID := commander.CreateCommander(
		em,
		"Commander",
		*pd.Pos,
		config.DefaultCommanderMovementSpeed,
		config.DefaultCommanderMaxSquads,
		commanderImage,
		config.DefaultCommanderStartingMana,
		config.DefaultCommanderMaxMana,
		templates.GetAllSpellIDs(),
	)

	// Add commander to player's roster
	roster := commander.GetPlayerCommanderRoster(pd.PlayerEntityID, em)
	if roster == nil {
		log.Fatal("Player has no commander roster component")
	}
	if err := roster.AddCommander(commanderID); err != nil {
		log.Fatalf("Failed to add initial commander: %v", err)
	}

	// Create initial squads on the commander (squad roster) using player's unit roster
	if err := bootstrap.CreateInitialPlayerSquads(commanderID, pd.PlayerEntityID, em, "Commander"); err != nil {
		log.Fatalf("Failed to create initial player squads: %v", err)
	}

	fmt.Printf("Created initial commander (ID: %d) at (%d,%d)\n", commanderID, pd.Pos.X, pd.Pos.Y)
}

// SetupDebugContent creates test items and spawns debug content.
// Debug Phase: Only runs when DEBUG_MODE is enabled.
func (gb *GameBootstrap) SetupDebugContent(em *common.EntityManager, gm *worldmap.GameMap, pd *common.PlayerData) {
	if config.DEBUG_MODE {
		SetupTestData(em, gm, pd)
		rosterData := commander.GetPlayerCommanderRoster(pd.PlayerEntityID, em)
		seedCount := 1
		if rosterData != nil {
			seedCount = len(rosterData.CommanderIDs)
		}
		bootstrap.SeedAllArtifacts(pd.PlayerEntityID, seedCount, em)
		bootstrap.EquipPlayerActivatedArtifacts(pd.PlayerEntityID, em)
	}

}

// SetupSharedSystems initializes only the systems shared by all game modes
// (data loading + ECS core). World creation is deferred until mode selection.
func SetupSharedSystems(g *Game) {
	bootstrap := NewGameBootstrap()
	bootstrap.LoadGameData()

	if err := raid.LoadRaidConfig(config.AssetPath("gamedata/raidconfig.json")); err != nil {
		fmt.Printf("WARNING: Failed to load raid config: %v (using defaults)\n", err)
	}

	initMapGenConfigOverride()
	bootstrap.InitializeCoreECS(&g.em)
}

// setupUICore initializes the shared UI infrastructure (coordinator, encounter service)
// without registering any modes. Called by both SetupOverworldMode and SetupRoguelikeMode.
func setupUICore(g *Game) (*framework.GameModeCoordinator, *encounter.EncounterService) {
	widgetresources.PreCacheScrollContainerBackgrounds()

	uiContext := &framework.UIContext{
		ECSManager:   &g.em,
		PlayerData:   &g.playerData,
		GameMap:      &g.gameMap,
		ScreenWidth:  graphics.ScreenInfo.GetCanvasWidth(),
		ScreenHeight: graphics.ScreenInfo.GetCanvasHeight(),
		TileSize:     graphics.ScreenInfo.TileSize,
		Queries:      framework.NewGUIQueries(&g.em),
	}

	widgetresources.PreCacheScrollContainerSizes(uiContext.ScreenWidth, uiContext.ScreenHeight)

	uiContext.SaveGameCallback = func() error {
		return SaveRoguelikeGame(g)
	}

	uiContext.LoadGameCallback = func() {
		g.pendingLoad = true
	}

	g.gameModeCoordinator = framework.NewGameModeCoordinator(uiContext)
	uiContext.ModeCoordinator = g.gameModeCoordinator

	encounterService := encounter.NewEncounterService(&g.em, g.gameModeCoordinator)
	return g.gameModeCoordinator, encounterService
}

// SetupInputCoordinator initializes the input handling system.
// Must be called after UI is created.
func SetupInputCoordinator(g *Game) {
	// Pass ModeCoordinator for context switching

	g.cameraController = input.NewCameraController(&g.em, &g.playerData, &g.gameMap, g.gameModeCoordinator)

}

// SetupTestData creates test items and content for debugging.
// Only called when DEBUG_MODE is enabled.
func SetupTestData(em *common.EntityManager, gm *worldmap.GameMap, pd *common.PlayerData) {
	testing.CreateTestItems(gm)
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

// gameSquadChecker implements core.SquadChecker interface
// This allows the overworld package to check squad status without circular dependency
type gameSquadChecker struct{}
