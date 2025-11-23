package core

import (
	"game_main/coords"

	"github.com/bytearena/ecs"
)

// OverworldState holds persistent UI state for the overworld context
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
}

// NewOverworldState creates a default overworld state
func NewOverworldState() *OverworldState {
	return &OverworldState{
		SelectedSquadID:      ecs.EntityID(0),
		SquadListScroll:      0,
		SquadIDs:             make([]ecs.EntityID, 0),
		EditingSquadID:       ecs.EntityID(0),
		FormationDirty:       false,
		SelectedFormation:    "",
		BuilderSelectedUnits: make([]ecs.EntityID, 0),
		BuilderSquadName:     "",
	}
}

// BattleMapState holds UI state for the battle map context
// This contains only transient UI selection and mode state used during combat
type BattleMapState struct {
	// UI Selection State
	SelectedSquadID  ecs.EntityID // Currently selected squad
	SelectedTargetID ecs.EntityID // Target squad for attacks

	// UI Mode Flags
	InAttackMode bool // Whether attack mode is active
	InMoveMode   bool // Whether movement mode is active

	// Computed UI State (cached from systems)
	ValidMoveTiles []coords.LogicalPosition // Valid movement positions (from MovementSystem)
}

// NewBattleMapState creates a default battle map state
func NewBattleMapState() *BattleMapState {
	return &BattleMapState{
		// UI Selection State
		SelectedSquadID:  ecs.EntityID(0),
		SelectedTargetID: ecs.EntityID(0),

		// UI Mode Flags
		InAttackMode: false,
		InMoveMode:   false,

		// Computed UI State
		ValidMoveTiles: make([]coords.LogicalPosition, 0),
	}
}

// Reset clears the battle map state (for starting a new battle)
func (bms *BattleMapState) Reset() {
	// Clear UI selection state
	bms.SelectedSquadID = ecs.EntityID(0)
	bms.SelectedTargetID = ecs.EntityID(0)

	// Clear UI mode flags
	bms.InAttackMode = false
	bms.InMoveMode = false

	// Clear computed UI state
	bms.ValidMoveTiles = make([]coords.LogicalPosition, 0)
}
