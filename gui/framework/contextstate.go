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

	// Spell Casting State
	InSpellMode     bool   // Whether spell mode is active
	SelectedSpellID string // Which spell is being targeted
	HasCastSpell    bool   // One spell per turn limit

	// Artifact Activation State
	InArtifactMode           bool   // Whether artifact mode is active
	SelectedArtifactBehavior string // Which artifact behavior is being targeted

	// Encounter Tracking
	TriggeredEncounterID ecs.EntityID // Encounter that triggered this combat (0 if none)

	// Post-Combat Routing
	PostCombatReturnMode string // Mode to return to after combat ends ("" = exploration default)
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

		// Spell Casting State
		InSpellMode:     false,
		SelectedSpellID: "",
		HasCastSpell:    false,

		// Artifact Activation State
		InArtifactMode:           false,
		SelectedArtifactBehavior: "",

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

	// Clear spell casting state
	ts.InSpellMode = false
	ts.SelectedSpellID = ""
	ts.HasCastSpell = false

	// Clear artifact activation state
	ts.InArtifactMode = false
	ts.SelectedArtifactBehavior = ""

	// Clear encounter tracking
	ts.TriggeredEncounterID = ecs.EntityID(0)

	// Clear post-combat routing
	ts.PostCombatReturnMode = ""
}

// OverworldState holds UI-specific state for the overworld context
// This contains ONLY transient UI selection and visualization state
//
// IMPORTANT: This is UI STATE ONLY - do not cache computed game data here
// - UI state: Camera position, selections, display toggles
// - Game state: Tick data, threats, travel (stored in ECS)
type OverworldState struct {
	CameraX        int
	CameraY        int
	SelectedNodeID ecs.EntityID // Currently selected node (threat or friendly)

	ShowInfluence bool // Toggle influence visualization

	// Commander UI state
	SelectedCommanderID ecs.EntityID             // Currently selected commander
	InMoveMode          bool                     // Movement overlay showing
	ValidMoveTiles      []coords.LogicalPosition // Cached valid tiles for movement
}

// NewOverworldState creates a default overworld state
func NewOverworldState() *OverworldState {
	return &OverworldState{
		CameraX:             0,
		CameraY:             0,
		SelectedNodeID:      0,
		ShowInfluence:       false,
		SelectedCommanderID: 0,
		InMoveMode:          false,
		ValidMoveTiles:      nil,
	}
}

// ExitMoveMode clears move mode state
func (os *OverworldState) ExitMoveMode() {
	os.InMoveMode = false
	os.ValidMoveTiles = nil
}

// ClearSelection clears the currently selected node
func (os *OverworldState) ClearSelection() {
	os.SelectedNodeID = 0
}

// HasSelection returns true if a node is currently selected
func (os *OverworldState) HasSelection() bool {
	return os.SelectedNodeID != 0
}
