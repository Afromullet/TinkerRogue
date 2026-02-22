package main

import (
	"fmt"
	"log"

	"game_main/common"
	"game_main/config"
	"game_main/gui/framework"
	"game_main/gui/guicombat"
	"game_main/gui/guiexploration"
	"game_main/gui/guiraid"
	"game_main/gui/guisquads"
	"game_main/gui/guiunitview"
	"game_main/mind/encounter"
	"game_main/mind/raid"
	"game_main/overworld/core"
	"game_main/savesystem"
	"game_main/savesystem/chunks"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/testing"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/worldmap"
)

// SetupRoguelikeMode performs a complete roguelike initialization from scratch:
// cavern map, squad modes in tactical context, no overworld systems.
func SetupRoguelikeMode(g *Game) {
	boot := NewGameBootstrap()
	boot.CreateWorld(&g.gameMap, "cavern")
	g.renderingCache = rendering.NewRenderingCache(&g.em)
	boot.CreatePlayer(&g.em, &g.playerData, &g.gameMap)
	boot.SetupDebugContent(&g.em, &g.gameMap, &g.playerData)

	// Walkable grid only — no overworld systems (ticks, factions, nodes)
	core.InitWalkableGrid(config.DefaultMapWidth, config.DefaultMapHeight)
	for _, pos := range g.gameMap.ValidPositions {
		core.SetTileWalkable(pos, true)
	}

	coordinator, encounterService := setupUICore(g)

	// Set the single commander as the selected commander so squad UI can find the roster
	roster := commander.GetPlayerCommanderRoster(g.playerData.PlayerEntityID, &g.em)
	if roster != nil && len(roster.CommanderIDs) > 0 {
		coordinator.GetOverworldState().SelectedCommanderID = roster.CommanderIDs[0]
	}

	tacticalManager := coordinator.GetTacticalManager()
	raidMode := registerRoguelikeTacticalModes(coordinator, tacticalManager, encounterService)

	// Create raid runner and inject into raid mode
	raidRunner := raid.NewRaidRunner(&g.em, encounterService)
	raidMode.SetRaidRunner(raidRunner)

	SetupInputCoordinator(g)
	testing.CreateTestItems(&g.gameMap)

	if err := coordinator.EnterTactical("exploration"); err != nil {
		log.Fatalf("Failed to enter exploration mode: %v", err)
	}
}

// SetupRoguelikeFromSave loads a saved roguelike game from disk.
// It restores the map, player, squads, commanders, gear, and raid state,
// then wires up the UI and systems just like SetupRoguelikeMode.
func SetupRoguelikeFromSave(g *Game) error {
	// Configure the MapChunk with a pointer to GameMap before loading
	configureMapChunk(&g.gameMap)

	// Load saved state into ECS
	if err := savesystem.LoadGame(&g.em); err != nil {
		return fmt.Errorf("failed to load save: %w", err)
	}

	g.renderingCache = rendering.NewRenderingCache(&g.em)

	// Restore PlayerData from the loaded player entity
	restorePlayerData(&g.em, &g.playerData)

	// Load unit templates — CreatePlayer does this for fresh games,
	// but the load path skips CreatePlayer so we must do it here.
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		return fmt.Errorf("failed to load unit templates: %w", err)
	}

	// Rebuild walkable grid from loaded map
	core.InitWalkableGrid(config.DefaultMapWidth, config.DefaultMapHeight)
	for _, pos := range g.gameMap.ValidPositions {
		core.SetTileWalkable(pos, true)
	}

	coordinator, encounterService := setupUICore(g)

	// Set selected commander from restored roster
	roster := commander.GetPlayerCommanderRoster(g.playerData.PlayerEntityID, &g.em)
	if roster != nil && len(roster.CommanderIDs) > 0 {
		coordinator.GetOverworldState().SelectedCommanderID = roster.CommanderIDs[0]
	}

	tacticalManager := coordinator.GetTacticalManager()
	raidMode := registerRoguelikeTacticalModes(coordinator, tacticalManager, encounterService)

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
	configureMapChunk(&g.gameMap)
	return savesystem.SaveGame(&g.em)
}

// configureMapChunk sets the GameMap pointer on the MapChunk so it can
// read/write map data during save/load.
func configureMapChunk(gm *worldmap.GameMap) {
	if chunk := savesystem.GetChunk("map"); chunk != nil {
		if mc, ok := chunk.(*chunks.MapChunk); ok {
			mc.GameMap = gm
		}
	}
}

// restorePlayerData reconstructs the PlayerData struct from the loaded player entity.
func restorePlayerData(em *common.EntityManager, pd *common.PlayerData) {
	playerTag, ok := em.WorldTags["players"]
	if !ok {
		return
	}
	results := em.World.Query(playerTag)
	if len(results) == 0 {
		return
	}
	entity := results[0].Entity
	pd.PlayerEntityID = entity.GetID()
	if pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent); pos != nil {
		pd.Pos = pos
	}
}

// registerRoguelikeTacticalModes registers squad + core tactical modes for roguelike.
// Squad modes are registered first so ExplorationMode.Initialize() detects squad_editor
// in the tactical manager and shows only the "Squad" button (no overworld button).
// Returns the RaidMode reference for RaidRunner injection.
func registerRoguelikeTacticalModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService) *guiraid.RaidMode {
	raidMode := guiraid.NewRaidMode(manager)

	modes := []framework.UIMode{
		guisquads.NewSquadEditorMode(manager),
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewArtifactMode(manager),
		guiunitview.NewUnitViewMode(manager),
		guiexploration.NewExplorationMode(manager),
		guicombat.NewCombatMode(manager, encounterService),
		guicombat.NewCombatAnimationMode(manager),
		guisquads.NewSquadDeploymentMode(manager),
		raidMode,
	}

	for _, mode := range modes {
		if err := coordinator.RegisterTacticalMode(mode); err != nil {
			log.Fatalf("Failed to register tactical mode '%s': %v", mode.GetModeName(), err)
		}
	}

	return raidMode
}
