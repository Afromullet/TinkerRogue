package framework

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// TacticalState holds UI state for the tactical context
// This contains ONLY transient UI selection and mode state used during combat
//
// IMPORTANT: This is UI STATE ONLY - do not cache computed game data here
// - UI state: User selections, mode flags, display preferences
// - Game state: Combat data, positions, stats (stored in ECS)
// - Computed data: Calculate on-demand from ECS via services
type TacticalState struct {
	// UI Selection State
	SelectedSquadID  ecs.EntityID // Currently selected squad
	SelectedTargetID ecs.EntityID // Target squad for attacks

	// UI Overlay Flags (controls what visual overlay is showing)
	// These are PURELY visual - game logic should NOT check these flags
	InAttackMode   bool // Whether attack overlay is showing
	InMoveMode     bool // Whether movement overlay is showing
	ShowHealthBars bool // Whether health bars are displayed above squads

	// Encounter Tracking
	TriggeredEncounterID ecs.EntityID // Encounter that triggered this combat (0 if none)
}

// NewTacticalState creates a default tactical state
func NewTacticalState() *TacticalState {
	return &TacticalState{
		// UI Selection State
		SelectedSquadID:  ecs.EntityID(0),
		SelectedTargetID: ecs.EntityID(0),

		// UI Mode Flags
		InAttackMode:   false,
		InMoveMode:     false,
		ShowHealthBars: false,

		// Encounter Tracking
		TriggeredEncounterID: ecs.EntityID(0),
	}
}

// Reset clears the tactical state (for starting a new battle)
func (ts *TacticalState) Reset() {
	// Clear UI selection state
	ts.SelectedSquadID = ecs.EntityID(0)
	ts.SelectedTargetID = ecs.EntityID(0)

	// Clear UI mode flags
	ts.InAttackMode = false
	ts.InMoveMode = false
	ts.ShowHealthBars = false

	// Clear encounter tracking
	ts.TriggeredEncounterID = ecs.EntityID(0)
}

// OverworldState holds UI-specific state for the overworld context
// This contains ONLY transient UI selection and visualization state
//
// IMPORTANT: This is UI STATE ONLY - do not cache computed game data here
// - UI state: Camera position, selections, display toggles
// - Game state: Tick data, threats, travel (stored in ECS)
type OverworldState struct {
	CameraX         int
	CameraY         int
	SelectedNodeID  ecs.EntityID            // Currently selected node (threat or friendly)
	HoveredPosition *coords.LogicalPosition // Current mouse hover position
	ShowInfluence   bool                    // Toggle influence visualization
	IsAutoTraveling bool                    // Auto-advance ticks during travel
}

// NewOverworldState creates a default overworld state
func NewOverworldState() *OverworldState {
	return &OverworldState{
		CameraX:        0,
		CameraY:        0,
		SelectedNodeID: 0,
		HoveredPosition: nil,
		ShowInfluence:  false,
	}
}

// ClearSelection clears the currently selected node
func (os *OverworldState) ClearSelection() {
	os.SelectedNodeID = 0
}

// HasSelection returns true if a node is currently selected
func (os *OverworldState) HasSelection() bool {
	return os.SelectedNodeID != 0
}
