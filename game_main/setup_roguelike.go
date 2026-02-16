package main

import (
	"log"

	"game_main/config"
	"game_main/gui/framework"
	"game_main/gui/guicombat"
	"game_main/gui/guiexploration"
	"game_main/gui/guisquads"
	"game_main/mind/encounter"
	"game_main/overworld/core"
	"game_main/tactical/commander"
	"game_main/testing"
	"game_main/visual/rendering"
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
	registerRoguelikeTacticalModes(coordinator, tacticalManager, encounterService)

	SetupInputCoordinator(g)
	testing.CreateTestItems(&g.gameMap)

	if err := coordinator.EnterTactical("exploration"); err != nil {
		log.Fatalf("Failed to enter exploration mode: %v", err)
	}
}

// registerRoguelikeTacticalModes registers squad + core tactical modes for roguelike.
// Squad modes are registered first so ExplorationMode.Initialize() detects squad_editor
// in the tactical manager and shows only the "Squad" button (no overworld button).
func registerRoguelikeTacticalModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService) {
	modes := []framework.UIMode{
		guisquads.NewSquadEditorMode(manager),
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewArtifactMode(manager),
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
