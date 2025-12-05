package main

import (
	"game_main/combat"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/graphics"

	"game_main/gui/core"
	"game_main/gui/guicombat"
	"game_main/gui/guimodes"
	"game_main/gui/guisquads"
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
	//g.gameMap = worldmap.NewGameMapDefault()
	g.gameMap = worldmap.NewGameMap("overworld")
	//g.gameMap = worldmap.NewGameMap("hybrid_tactical")
	//g.gameMap = worldmap.NewGameMap("wavelet_procedural")
	//g.gameMap = worldmap.NewGameMap("cave_tactical")

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

	// 10. Setup gameplay factions and squads for testing
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

// SetupUI initializes the new modal UI system with separate context managers.
// Must be called after game initialization but before input coordinator.
func SetupUI(g *Game) {
	// Create UI context with shared game state
	uiContext := &core.UIContext{
		ECSManager:   &g.em,
		PlayerData:   &g.playerData,
		ScreenWidth:  graphics.ScreenInfo.GetCanvasWidth(),
		ScreenHeight: graphics.ScreenInfo.GetCanvasHeight(),
		TileSize:     graphics.ScreenInfo.TileSize,
	}

	// Create game mode coordinator (manages two separate contexts)
	g.gameModeCoordinator = core.NewGameModeCoordinator(uiContext)

	// Set coordinator reference in context so modes can trigger context switches
	uiContext.ModeCoordinator = g.gameModeCoordinator

	// Get references to both managers for registration
	battleMapManager := g.gameModeCoordinator.GetBattleMapManager()
	overworldManager := g.gameModeCoordinator.GetOverworldManager()

	// ===== BATTLE MAP MODES (tactical layer) =====

	// Exploration mode - dungeon exploration
	explorationMode := guimodes.NewExplorationMode(battleMapManager)
	if err := g.gameModeCoordinator.RegisterBattleMapMode(explorationMode); err != nil {
		log.Fatalf("Failed to register exploration mode: %v", err)
	}

	// Info mode - inspect entities on battlefield
	infoMode := guimodes.NewInfoMode(battleMapManager)
	if err := g.gameModeCoordinator.RegisterBattleMapMode(infoMode); err != nil {
		log.Fatalf("Failed to register info mode: %v", err)
	}

	// Combat mode - turn-based squad combat
	combatMode := guicombat.NewCombatMode(battleMapManager)
	if err := g.gameModeCoordinator.RegisterBattleMapMode(combatMode); err != nil {
		log.Fatalf("Failed to register combat mode: %v", err)
	}

	// Combat animation mode - full-screen battle scene during attacks
	combatAnimMode := guicombat.NewCombatAnimationMode(battleMapManager)
	if err := g.gameModeCoordinator.RegisterBattleMapMode(combatAnimMode); err != nil {
		log.Fatalf("Failed to register combat animation mode: %v", err)
	}

	// Squad deployment mode - deploy squads to battle map
	squadDeploymentMode := guisquads.NewSquadDeploymentMode(battleMapManager)
	if err := g.gameModeCoordinator.RegisterBattleMapMode(squadDeploymentMode); err != nil {
		log.Fatalf("Failed to register squad deployment mode: %v", err)
	}

	// Inventory mode (battle map instance)
	inventoryModeBattle := guimodes.NewInventoryMode(battleMapManager)
	inventoryModeBattle.SetReturnMode("exploration") // ESC returns to exploration
	if err := g.gameModeCoordinator.RegisterBattleMapMode(inventoryModeBattle); err != nil {
		log.Fatalf("Failed to register inventory mode (battle): %v", err)
	}

	// ===== OVERWORLD MODES (strategic layer) =====

	// Squad management mode - manage squads between missions
	squadManagementMode := guisquads.NewSquadManagementMode(overworldManager)
	if err := g.gameModeCoordinator.RegisterOverworldMode(squadManagementMode); err != nil {
		log.Fatalf("Failed to register squad management mode: %v", err)
	}

	// Formation editor mode - edit squad formations
	formationEditorMode := guisquads.NewFormationEditorMode(overworldManager)
	if err := g.gameModeCoordinator.RegisterOverworldMode(formationEditorMode); err != nil {
		log.Fatalf("Failed to register formation editor mode: %v", err)
	}

	// Squad builder mode - create new squads
	squadBuilderMode := guisquads.NewSquadBuilderMode(overworldManager)
	if err := g.gameModeCoordinator.RegisterOverworldMode(squadBuilderMode); err != nil {
		log.Fatalf("Failed to register squad builder mode: %v", err)
	}

	// Unit purchase mode - buy units for roster
	unitPurchaseMode := guisquads.NewUnitPurchaseMode(overworldManager)
	if err := g.gameModeCoordinator.RegisterOverworldMode(unitPurchaseMode); err != nil {
		log.Fatalf("Failed to register unit purchase mode: %v", err)
	}

	// Squad editor mode - edit existing squads (add/remove units, change leader, etc.)
	squadEditorMode := guisquads.NewSquadEditorMode(overworldManager)
	if err := g.gameModeCoordinator.RegisterOverworldMode(squadEditorMode); err != nil {
		log.Fatalf("Failed to register squad editor mode: %v", err)
	}

	// Inventory mode (overworld instance)
	inventoryModeOverworld := guimodes.NewInventoryMode(overworldManager)
	inventoryModeOverworld.SetReturnMode("squad_management") // ESC returns to squad management
	if err := g.gameModeCoordinator.RegisterOverworldMode(inventoryModeOverworld); err != nil {
		log.Fatalf("Failed to register inventory mode (overworld): %v", err)
	}

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
