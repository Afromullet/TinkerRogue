package gamesetup

import (
	"fmt"
	"log"

	"game_main/common"
	"game_main/config"
	"game_main/overworld/core"
	"game_main/overworld/node"
	"game_main/overworld/tick"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/templates"
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

	// Seed 5 random unassigned units into the player's roster
	if err := bootstrap.CreateInitialRosterUnits(pd.PlayerEntityID, em, 5); err != nil {
		log.Fatalf("Failed to create initial roster units: %v", err)
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

// InitializeGameplay sets up squad system and exploration squads.
// Phase 5: Depends on CreatePlayer for faction positioning.
// Overworld factions spawn threats dynamically during gameplay.
func (gb *GameBootstrap) InitializeGameplay(em *common.EntityManager, pd *common.PlayerData, gm *worldmap.GameMap) {

	// Initialize overworld tick state
	tick.CreateTickStateEntity(em)

	// Create additional starting commanders near player position (debug only)
	if config.DEBUG_MODE {
		if err := bootstrap.CreateTestCommanders(em, pd, *pd.Pos); err != nil {
			fmt.Printf("WARNING: Failed to create test commanders: %v\n", err)
		}
	}

	// Initialize commander turn state and action states
	commander.CreateOverworldTurnState(em)
	commander.StartNewTurn(em, pd.PlayerEntityID)

	// Initialize walkable grid from map tiles
	InitWalkableGridFromMap(gm)

	// Create initial overworld factions (they will spawn threats dynamically, debug only)
	if config.DEBUG_MODE {
		bootstrap.InitializeOverworldFactions(em, pd, gm)
	}

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
