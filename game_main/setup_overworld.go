package main

import (
	"fmt"
	"log"

	"game_main/common"
	"game_main/config"
	"game_main/gui/framework"
	"game_main/gui/guicombat"
	"game_main/gui/guiexploration"
	"game_main/gui/guinodeplacement"
	"game_main/gui/guioverworld"
	"game_main/gui/guisquads"
	"game_main/gui/guiunitview"
	"game_main/mind/encounter"
	"game_main/overworld/core"
	"game_main/overworld/node"
	"game_main/overworld/tick"
	"game_main/tactical/commander"
	"game_main/testing"
	"game_main/testing/bootstrap"
	"game_main/visual/rendering"
	"game_main/world/worldmap"
)

// SetupOverworldMode performs a complete overworld initialization:
// overworld map, full tactical + overworld UI, factions, nodes.
func SetupOverworldMode(g *Game) {
	boot := NewGameBootstrap()
	boot.CreateWorld(&g.gameMap, "overworld")
	g.renderingCache = rendering.NewRenderingCache(&g.em)
	boot.CreatePlayer(&g.em, &g.playerData, &g.gameMap)
	boot.SetupDebugContent(&g.em, &g.gameMap, &g.playerData)
	boot.InitializeGameplay(&g.em, &g.playerData, &g.gameMap)

	coordinator, encounterService := setupUICore(g)
	tacticalManager := coordinator.GetTacticalManager()
	overworldManager := coordinator.GetOverworldManager()
	registerTacticalModes(coordinator, tacticalManager, encounterService)
	registerOverworldModes(coordinator, overworldManager, encounterService)

	SetupInputCoordinator(g)
	testing.CreateTestItems(&g.gameMap)

	if err := coordinator.EnterTactical("exploration"); err != nil {
		log.Fatalf("Failed to enter exploration mode: %v", err)
	}
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
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewSquadEditorMode(manager),
		guisquads.NewArtifactMode(manager),
		guisquads.NewPerkMode(manager),
		guiunitview.NewUnitViewMode(manager),
	}

	for _, mode := range modes {
		if err := coordinator.RegisterOverworldMode(mode); err != nil {
			log.Fatalf("Failed to register overworld mode '%s': %v", mode.GetModeName(), err)
		}
	}
}
