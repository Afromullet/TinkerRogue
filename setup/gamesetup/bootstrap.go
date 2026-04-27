package gamesetup

import (
	"fmt"
	"log"

	"game_main/campaign/overworld/core"
	"game_main/campaign/overworld/ids"
	"game_main/campaign/overworld/node"
	"game_main/campaign/overworld/tick"
	"game_main/core/common"
	"game_main/core/config"
	"game_main/core/coords"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/commander"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/perks"
	"game_main/tactical/squads/unitdefs"
	"game_main/templates"
	"game_main/testing/bootstrap"
	"game_main/visual/graphics"
	"game_main/world/worldgen"
	"game_main/world/worldmapcore"
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
	perks.LoadPerkDefinitions()
	perks.LoadPerkBalanceConfig()
	artifacts.LoadArtifactBalanceConfig()
	reportCoverage("artifact", artifacts.ValidateBehaviorCoverage())
	reportCoverage("perk", perks.ValidateHookCoverage())
	combatcore.LoadCombatBalanceConfig()
	core.ValidateNodeRegistry()
}

// reportCoverage prints validation errors from the powers registries. In debug
// builds mismatches are fatal — they almost always indicate a developer bug
// (behavior registered with no JSON definition, or vice versa). In release
// builds they drop to warnings so a data issue doesn't prevent shipping.
func reportCoverage(label string, errs []error) {
	if len(errs) == 0 {
		return
	}
	if config.DEBUG_MODE {
		for _, err := range errs {
			log.Printf("FATAL %s coverage: %v", label, err)
		}
		log.Fatalf("%s registry has %d coverage errors; fix before continuing", label, len(errs))
		return
	}
	for _, err := range errs {
		log.Printf("WARNING %s coverage: %v", label, err)
	}
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
func (gb *GameBootstrap) CreateWorld(gm *worldmapcore.GameMap, mapType string) {
	*gm = worldmapcore.NewGameMap(worldgen.GetGenerator(mapType))
}

// CreatePlayer initializes the player entity, creates all commanders defined in
// initialsetup.json, and adds creatures to the position system.
// Phase 4: Depends on CreateWorld for starting position.
func (gb *GameBootstrap) CreatePlayer(em *common.EntityManager, pd *common.PlayerData, gm *worldmapcore.GameMap) {
	InitializePlayerData(em, pd, gm)

	// Initialize unit templates before creating initial squads
	if err := unitdefs.InitUnitTemplatesFromJSON(); err != nil {
		log.Fatalf("Failed to load unit templates: %v", err)
	}

	primaryID, err := CreateInitialCommanders(em, pd, *pd.Pos)
	if err != nil {
		log.Fatalf("Failed to create initial commanders: %v", err)
	}

	if err := CreateInitialRosterUnits(pd.PlayerEntityID, em); err != nil {
		log.Fatalf("Failed to create initial roster units: %v", err)
	}

	fmt.Printf("Primary commander ID: %d at (%d,%d)\n", primaryID, pd.Pos.X, pd.Pos.Y)
}

// SetupDebugContent creates test items and spawns debug content.
// Debug Phase: Only runs when DEBUG_MODE is enabled.
func (gb *GameBootstrap) SetupDebugContent(em *common.EntityManager, gm *worldmapcore.GameMap, pd *common.PlayerData) {
	if config.DEBUG_MODE {

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
func (gb *GameBootstrap) InitializeGameplay(em *common.EntityManager, pd *common.PlayerData, gm *worldmapcore.GameMap) {

	// Initialize overworld tick state
	tick.CreateTickStateEntity(em)

	// Initialize commander turn state and action states
	commander.CreateOverworldTurnState(em)
	commander.StartNewTurn(em, pd.PlayerEntityID)

	// Initialize walkable grid from map tiles
	InitWalkableGridFromMap(gm)

	// Create initial overworld factions (they spawn threats dynamically each tick).
	if err := CreateInitialFactions(em, pd, gm); err != nil {
		log.Fatalf("Failed to create initial factions: %v", err)
	}

	// Convert POIs from world generation into neutral overworld nodes
	gb.ConvertPOIsToNodes(em, gm)
}

// ConvertPOIsToNodes converts POIs from world generation into neutral overworld nodes.
// This allows POIs to participate in the influence system (mildly suppress nearby threats).
func (gb *GameBootstrap) ConvertPOIsToNodes(em *common.EntityManager, gm *worldmapcore.GameMap) {
	currentTick := core.GetCurrentTick(em)
	for _, poi := range gm.POIs {
		nodeID, err := node.CreateNode(em, node.CreateNodeParams{
			Position:    poi.Position,
			NodeTypeID:  ids.NodeTypeID(poi.NodeID),
			OwnerID:     ids.OwnerNeutral,
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
