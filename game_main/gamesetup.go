package main

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/visual/graphics"
	"game_main/world/coords"

	"game_main/gui/framework"
	"game_main/gui/guicombat"
	"game_main/gui/guiexploration"
	"game_main/gui/guinodeplacement"
	"game_main/gui/guioverworld"

	"game_main/gui/guisquads"
	"game_main/gui/widgetresources"
	"game_main/input"
	"game_main/mind/encounter"
	"game_main/templates"

	"game_main/overworld/core"
	"game_main/overworld/node"
	"game_main/overworld/tick"
	"game_main/testing"
	"game_main/testing/bootstrap"
	"game_main/visual/rendering"
	"game_main/world/worldmap"
	"log"
	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
	"runtime"

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
func (gb *GameBootstrap) CreateWorld(gm *worldmap.GameMap) {
	// Multiple map generation algorithms available:
	// - "overworld" (default)
	// - "hybrid_tactical"
	*gm = worldmap.NewGameMap("overworld")
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
		config.DefaultCommanderStartingSpells,
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
	}

	// Spawn starting content (test enemies, items, etc.)
	testing.UpdateContentsForTest(em, gm)
}

// InitializeGameplay sets up squad system and exploration squads.
// Phase 5: Depends on CreatePlayer for faction positioning.
// Overworld factions spawn threats dynamically during gameplay.
func (gb *GameBootstrap) InitializeGameplay(em *common.EntityManager, pd *common.PlayerData, gm *worldmap.GameMap) {

	// Initialize overworld tick state
	tick.CreateTickStateEntity(em)

	// Create additional starting commanders near player position
	if err := bootstrap.CreateTestCommanders(em, pd, *pd.Pos); err != nil {
		fmt.Printf("WARNING: Failed to create test commanders: %v\n", err)
	}

	// Initialize commander turn state and action states
	commander.CreateOverworldTurnState(em)
	commander.StartNewTurn(em, pd.PlayerEntityID)

	// Initialize walkable grid from map tiles
	core.InitWalkableGrid(config.DefaultMapWidth, config.DefaultMapHeight)
	for _, pos := range gm.ValidPositions {
		core.SetTileWalkable(pos, true)
	}

	// Create initial overworld factions (they will spawn threats dynamically)
	bootstrap.InitializeOverworldFactions(em, pd, gm)

	// Convert POIs from world generation into neutral overworld nodes
	gb.ConvertPOIsToNodes(em, gm)
}

// ConvertPOIsToNodes converts POIs from world generation into neutral overworld nodes.
// This allows POIs to participate in the influence system (mildly suppress nearby threats).
func (gb *GameBootstrap) ConvertPOIsToNodes(em *common.EntityManager, gm *worldmap.GameMap) {
	currentTick := core.GetCurrentTick(em)
	for _, poi := range gm.POIs {
		nodeID, err := node.CreateNode(em, node.CreateNodeParams{
			Position:    poi.Position,
			NodeTypeID:  poi.NodeID,
			OwnerID:     core.OwnerNeutral,
			CurrentTick: currentTick,
		})
		if err != nil {
			log.Printf("Failed to convert POI '%s' to node: %v", poi.NodeID, err)
			continue
		}
		log.Printf("Converted POI '%s' at (%d, %d) to neutral node (ID: %d)",
			poi.NodeID, poi.Position.X, poi.Position.Y, nodeID)
	}
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
	bootstrap.InitializeGameplay(&g.em, &g.playerData, &g.gameMap)
}

// SetupGameplayFactions has been removed and replaced with SetupBalancedEncounter
// Combat encounters now use power-based balancing (see tactical/combat/encounter_spawner.go)
// This comment left for reference - function removed as part of encounter system overhaul

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

// gameSquadChecker implements core.SquadChecker interface
// This allows the overworld package to check squad status without circular dependency
type gameSquadChecker struct{}

// HasActiveSquads checks if player has any squads with living units
func (gsc *gameSquadChecker) HasActiveSquads(manager *common.EntityManager) bool {
	// Query all squad entities
	for _, result := range manager.World.Query(squads.SquadTag) {
		squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
		if squadData == nil {
			continue
		}

		if !squads.IsSquadDestroyed(squadData.SquadID, manager) {
			return true // Found at least one active squad
		}
	}

	return false // All squads are destroyed or no squads exist
}

// SetupUI initializes the new modal UI system with separate context managers.
// Must be called after game initialization but before input coordinator.
func SetupUI(g *Game) {
	// Pre-cache ScrollContainer backgrounds for performance (reduces NineSlice overhead by 80%)
	// This warms the cache before UI creation to avoid first-frame rendering stutter
	widgetresources.PreCacheScrollContainerBackgrounds()

	// Create UI context with shared game state
	uiContext := &framework.UIContext{
		ECSManager:   &g.em,
		PlayerData:   &g.playerData,
		GameMap:      &g.gameMap,
		ScreenWidth:  graphics.ScreenInfo.GetCanvasWidth(),
		ScreenHeight: graphics.ScreenInfo.GetCanvasHeight(),
		TileSize:     graphics.ScreenInfo.TileSize,
		Queries:      framework.NewGUIQueries(&g.em),
	}

	// Pre-cache based on actual screen dimensions for optimal cache hit rate
	widgetresources.PreCacheScrollContainerSizes(uiContext.ScreenWidth, uiContext.ScreenHeight)

	// Create game mode coordinator (manages two separate contexts)
	g.gameModeCoordinator = framework.NewGameModeCoordinator(uiContext)

	// Set coordinator reference in context so modes can trigger context switches
	uiContext.ModeCoordinator = g.gameModeCoordinator

	// Create encounter service (centralizes encounter lifecycle management)
	encounterService := encounter.NewEncounterService(&g.em, g.gameModeCoordinator)

	// Get references to both managers for registration
	tacticalManager := g.gameModeCoordinator.GetTacticalManager()
	overworldManager := g.gameModeCoordinator.GetOverworldManager()

	// Register all tactical modes (tactical layer)
	registerTacticalModes(g.gameModeCoordinator, tacticalManager, encounterService)

	// Register all overworld modes (strategic layer)
	registerOverworldModes(g.gameModeCoordinator, overworldManager, encounterService)

	// Set initial context and mode (start in tactical context, exploration mode)
	if err := g.gameModeCoordinator.EnterTactical("exploration"); err != nil {
		log.Fatalf("Failed to set initial tactical mode: %v", err)
	}
}

// SetupInputCoordinator initializes the input handling system.
// Must be called after UI is created.
func SetupInputCoordinator(g *Game) {
	// Pass ModeCoordinator for context switching
	g.inputCoordinator = input.NewInputCoordinator(&g.em, &g.playerData, &g.gameMap, g.gameModeCoordinator)
}

// registerTacticalModes registers all tactical UI modes with the coordinator.
func registerTacticalModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService) {
	modes := []framework.UIMode{
		guiexploration.NewExplorationMode(manager),
		guicombat.NewCombatMode(manager, encounterService),
		guicombat.NewCombatAnimationMode(manager),
		guisquads.NewSquadDeploymentMode(manager),
	}

	for _, mode := range modes {
		if err := coordinator.RegisterTacticalMode(mode); err != nil {
			log.Fatalf("Failed to register tactical mode '%s': %v", mode.GetModeName(), err)
		}
	}
}

// registerOverworldModes registers all overworld UI modes with the coordinator.
// This reduces boilerplate by iterating over a slice of mode constructors.
func registerOverworldModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService) {
	modes := []framework.UIMode{
		guioverworld.NewOverworldMode(manager, encounterService),
		guinodeplacement.NewNodePlacementMode(manager),
		guisquads.NewSquadManagementMode(manager),
		guisquads.NewSquadBuilderMode(manager),
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewSquadEditorMode(manager),
		newInventoryModeWithReturn(manager, "overworld"),
	}

	for _, mode := range modes {
		if err := coordinator.RegisterOverworldMode(mode); err != nil {
			log.Fatalf("Failed to register overworld mode '%s': %v", mode.GetModeName(), err)
		}
	}
}

// newInventoryModeWithReturn creates an inventory mode configured with a return mode.
func newInventoryModeWithReturn(manager *framework.UIModeManager, returnMode string) framework.UIMode {
	mode := guiexploration.NewInventoryMode(manager)
	mode.SetReturnMode(returnMode)
	return mode
}
