package main

import (
	"fmt"
	"log"

	"game_main/config"
	"game_main/gamesetup"
	"game_main/gui/framework"
	"game_main/gui/widgetresources"
	"game_main/input"
	"game_main/mind/encounter"
	"game_main/mind/raid"
	"game_main/savesystem"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/visual/graphics"
	"game_main/visual/rendering"
)

// ---------------------------------------------------------------------------
// Shared systems
// ---------------------------------------------------------------------------

// SetupSharedSystems initializes only the systems shared by all game modes
// (data loading + ECS core). World creation is deferred until mode selection.
func SetupSharedSystems(g *Game) {
	bootstrap := gamesetup.NewGameBootstrap()
	bootstrap.LoadGameData()

	if err := raid.LoadRaidConfig(config.AssetPath("gamedata/raidconfig.json")); err != nil {
		fmt.Printf("WARNING: Failed to load raid config: %v (using defaults)\n", err)
	}

	gamesetup.InitMapGenConfigOverride()
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
	g.cameraController = input.NewCameraController(&g.em, &g.playerData, &g.gameMap, g.gameModeCoordinator)
}

// ---------------------------------------------------------------------------
// Overworld mode
// ---------------------------------------------------------------------------

// SetupOverworldMode performs a complete overworld initialization:
// overworld map, full tactical + overworld UI, factions, nodes.
func SetupOverworldMode(g *Game) {
	boot := gamesetup.NewGameBootstrap()
	boot.CreateWorld(&g.gameMap, "overworld")
	g.renderingCache = rendering.NewRenderingCache(&g.em)
	boot.CreatePlayer(&g.em, &g.playerData, &g.gameMap)
	boot.SetupDebugContent(&g.em, &g.gameMap, &g.playerData)
	boot.InitializeGameplay(&g.em, &g.playerData, &g.gameMap)

	coordinator, encounterService := setupUICore(g)
	tacticalManager := coordinator.GetTacticalManager()
	overworldManager := coordinator.GetOverworldManager()
	gamesetup.RegisterTacticalModes(coordinator, tacticalManager, encounterService)
	gamesetup.RegisterOverworldModes(coordinator, overworldManager, encounterService)

	SetupInputCoordinator(g)

	if err := coordinator.EnterTactical("exploration"); err != nil {
		log.Fatalf("Failed to enter exploration mode: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Roguelike mode
// ---------------------------------------------------------------------------

// SetupRoguelikeMode performs a complete roguelike initialization from scratch:
// cavern map, squad modes in tactical context, no overworld systems.
func SetupRoguelikeMode(g *Game) {
	boot := gamesetup.NewGameBootstrap()
	boot.CreateWorld(&g.gameMap, "cavern")
	g.renderingCache = rendering.NewRenderingCache(&g.em)
	boot.CreatePlayer(&g.em, &g.playerData, &g.gameMap)
	boot.SetupDebugContent(&g.em, &g.gameMap, &g.playerData)

	// Walkable grid only — no overworld systems (ticks, factions, nodes)
	gamesetup.InitWalkableGridFromMap(&g.gameMap)

	coordinator, encounterService := setupUICore(g)

	// Set the single commander as the selected commander so squad UI can find the roster
	roster := commander.GetPlayerCommanderRoster(g.playerData.PlayerEntityID, &g.em)
	if roster != nil && len(roster.CommanderIDs) > 0 {
		coordinator.GetOverworldState().SelectedCommanderID = roster.CommanderIDs[0]
	}

	tacticalManager := coordinator.GetTacticalManager()
	raidMode := gamesetup.RegisterRoguelikeTacticalModes(coordinator, tacticalManager, encounterService)

	// Create raid runner and inject into raid mode
	raidRunner := raid.NewRaidRunner(&g.em, encounterService)
	raidMode.SetRaidRunner(raidRunner)

	SetupInputCoordinator(g)

	if err := coordinator.EnterTactical("exploration"); err != nil {
		log.Fatalf("Failed to enter exploration mode: %v", err)
	}
}

// SetupRoguelikeFromSave loads a saved roguelike game from disk.
// It restores the map, player, squads, commanders, gear, and raid state,
// then wires up the UI and systems just like SetupRoguelikeMode.
func SetupRoguelikeFromSave(g *Game) error {
	// Configure the MapChunk with a pointer to GameMap before loading
	gamesetup.ConfigureMapChunk(&g.gameMap)

	// Load saved state into ECS
	if err := savesystem.LoadGame(&g.em); err != nil {
		return fmt.Errorf("failed to load save: %w", err)
	}

	g.renderingCache = rendering.NewRenderingCache(&g.em)

	// Restore PlayerData from the loaded player entity
	gamesetup.RestorePlayerData(&g.em, &g.playerData)

	// Load unit templates — CreatePlayer does this for fresh games,
	// but the load path skips CreatePlayer so we must do it here.
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		return fmt.Errorf("failed to load unit templates: %w", err)
	}

	// Reconstruct renderables from saved metadata (images can't be serialized)
	if err := gamesetup.RestoreRenderables(&g.em); err != nil {
		return fmt.Errorf("failed to restore renderables: %w", err)
	}

	// Rebuild walkable grid from loaded map
	gamesetup.InitWalkableGridFromMap(&g.gameMap)

	coordinator, encounterService := setupUICore(g)

	// Set selected commander from restored roster
	roster := commander.GetPlayerCommanderRoster(g.playerData.PlayerEntityID, &g.em)
	if roster != nil && len(roster.CommanderIDs) > 0 {
		coordinator.GetOverworldState().SelectedCommanderID = roster.CommanderIDs[0]
	}

	tacticalManager := coordinator.GetTacticalManager()
	raidMode := gamesetup.RegisterRoguelikeTacticalModes(coordinator, tacticalManager, encounterService)

	raidRunner := raid.NewRaidRunner(&g.em, encounterService)
	raidMode.SetRaidRunner(raidRunner)

	// If a raid was in progress when saved, restore the runner's entity ID
	// so IsActive() returns true and autoStartRaid() doesn't duplicate entities.
	if raidState := raid.GetRaidState(&g.em); raidState != nil {
		for _, result := range g.em.World.Query(g.em.WorldTags["raidstate"]) {
			raidRunner.RestoreFromSave(result.Entity.GetID())
			break
		}
	}

	SetupInputCoordinator(g)

	if err := coordinator.EnterTactical("exploration"); err != nil {
		log.Fatalf("Failed to enter exploration mode: %v", err)
	}

	fmt.Println("Roguelike game loaded from save")
	return nil
}

// SaveRoguelikeGame saves the current roguelike game state to disk.
// Call this from exploration mode (e.g., via a keybind or button).
func SaveRoguelikeGame(g *Game) error {
	gamesetup.ConfigureMapChunk(&g.gameMap)
	return savesystem.SaveGame(&g.em)
}
