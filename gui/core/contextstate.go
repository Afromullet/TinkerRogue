package core

import (
	"game_main/coords"

	"github.com/bytearena/ecs"
)

// OverworldState holds persistent data for the overworld context
// This data is saved when entering battle map and restored when returning
type OverworldState struct {
	// Squad management state
	SelectedSquadID ecs.EntityID   // Currently selected squad in squad management
	SquadListScroll int            // Scroll position in squad list
	SquadIDs        []ecs.EntityID // All available squads (for quick access)

	// Formation editor state
	EditingSquadID     ecs.EntityID // Squad being edited in formation editor
	FormationDirty     bool         // Whether formation has unsaved changes
	SelectedFormation  string       // Name of selected formation preset

	// Squad builder state
	BuilderSelectedUnits []ecs.EntityID // Units selected in squad builder
	BuilderSquadName     string         // Name being entered for new squad

	// World map state (for future overworld map mode)
	WorldMapPosition   [2]int         // Current position on world map
	WorldMapZoom       float64        // Zoom level
	VisitedLocations   []ecs.EntityID // Locations player has visited
	ActiveQuestIDs     []ecs.EntityID // Active quests

	// UI state
	LastActiveModeOverworld string // Last mode used in overworld context
}

// BattleMapState holds persistent data for the battle map context
// This data is saved when returning to overworld
type BattleMapState struct {
	// Map state
	CurrentMapID     string       // ID of the current dungeon/battle map
	MapSeed          int64        // Random seed for map generation
	ExploredTiles    map[int]bool // Map of explored tile indices (using map for sparse storage)
	RevealedEnemies  []ecs.EntityID

	// Combat state
	InCombat           bool           // Whether currently in combat
	CombatTurnOrder    []ecs.EntityID // Entity IDs in turn order
	CurrentTurnIndex   int            // Current position in turn order
	CombatRound        int            // Current combat round number
	CombatLogMessages  []string       // Combat log history

	// Deployment state
	DeployedSquadID    ecs.EntityID   // Squad currently deployed to map
	DeploymentComplete bool           // Whether squad deployment is finished
	DeployedUnitIDs    []ecs.EntityID // Individual units deployed to map

	// Battle results (for carrying back to overworld)
	BattleComplete     bool           // Whether battle is finished
	BattleVictory      bool           // Whether player won
	LootCollected      []ecs.EntityID // Items/loot collected during battle
	ExperienceGained   int            // Total XP earned
	CasualtiesEntityIDs []ecs.EntityID // Units lost in battle

	// Camera/view state
	CameraOffsetX      int     // Camera scroll position X
	CameraOffsetY      int     // Camera scroll position Y
	ZoomLevel          float64 // Camera zoom level

	// UI state
	InfoTargetEntity   ecs.EntityID // Entity being inspected in info mode
	LastActiveModeBattleMap string  // Last mode used in battle map context

	// Combat UI state (previously in CombatStateManager)
	SelectedSquadID  ecs.EntityID                // Currently selected squad
	SelectedTargetID ecs.EntityID                // Target squad for attacks
	InAttackMode     bool                        // Whether attack mode is active
	InMoveMode       bool                        // Whether movement mode is active
	ValidMoveTiles   []coords.LogicalPosition    // Valid movement positions
}

// NewOverworldState creates a default overworld state
func NewOverworldState() *OverworldState {
	return &OverworldState{
		SelectedSquadID:         ecs.EntityID(0),
		SquadListScroll:         0,
		SquadIDs:                make([]ecs.EntityID, 0),
		EditingSquadID:          ecs.EntityID(0),
		FormationDirty:          false,
		SelectedFormation:       "",
		BuilderSelectedUnits:    make([]ecs.EntityID, 0),
		BuilderSquadName:        "",
		WorldMapPosition:        [2]int{0, 0},
		WorldMapZoom:            1.0,
		VisitedLocations:        make([]ecs.EntityID, 0),
		ActiveQuestIDs:          make([]ecs.EntityID, 0),
		LastActiveModeOverworld: "",
	}
}

// NewBattleMapState creates a default battle map state
func NewBattleMapState() *BattleMapState {
	return &BattleMapState{
		CurrentMapID:            "",
		MapSeed:                 0,
		ExploredTiles:           make(map[int]bool),
		RevealedEnemies:         make([]ecs.EntityID, 0),
		InCombat:                false,
		CombatTurnOrder:         make([]ecs.EntityID, 0),
		CurrentTurnIndex:        0,
		CombatRound:             0,
		CombatLogMessages:       make([]string, 0),
		DeployedSquadID:         ecs.EntityID(0),
		DeploymentComplete:      false,
		DeployedUnitIDs:         make([]ecs.EntityID, 0),
		BattleComplete:          false,
		BattleVictory:           false,
		LootCollected:           make([]ecs.EntityID, 0),
		ExperienceGained:        0,
		CasualtiesEntityIDs:     make([]ecs.EntityID, 0),
		CameraOffsetX:           0,
		CameraOffsetY:           0,
		ZoomLevel:               1.0,
		InfoTargetEntity:        ecs.EntityID(0),
		LastActiveModeBattleMap: "",
		SelectedSquadID:         ecs.EntityID(0),
		SelectedTargetID:        ecs.EntityID(0),
		InAttackMode:            false,
		InMoveMode:              false,
		ValidMoveTiles:          make([]coords.LogicalPosition, 0),
	}
}

// Reset clears the battle map state (for starting a new battle)
func (bms *BattleMapState) Reset() {
	bms.CurrentMapID = ""
	bms.MapSeed = 0
	bms.ExploredTiles = make(map[int]bool)
	bms.RevealedEnemies = make([]ecs.EntityID, 0)
	bms.InCombat = false
	bms.CombatTurnOrder = make([]ecs.EntityID, 0)
	bms.CurrentTurnIndex = 0
	bms.CombatRound = 0
	bms.CombatLogMessages = make([]string, 0)
	bms.DeploymentComplete = false
	bms.DeployedUnitIDs = make([]ecs.EntityID, 0)
	bms.BattleComplete = false
	bms.BattleVictory = false
	bms.LootCollected = make([]ecs.EntityID, 0)
	bms.ExperienceGained = 0
	bms.CasualtiesEntityIDs = make([]ecs.EntityID, 0)
	// Clear combat UI state
	bms.SelectedSquadID = ecs.EntityID(0)
	bms.SelectedTargetID = ecs.EntityID(0)
	bms.InAttackMode = false
	bms.InMoveMode = false
	bms.ValidMoveTiles = make([]coords.LogicalPosition, 0)
	// Keep camera and UI state
}
