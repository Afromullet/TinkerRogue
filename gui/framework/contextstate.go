package framework

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// BattleMapState holds UI state for the battle map context
// This contains ONLY transient UI selection and mode state used during combat
//
// IMPORTANT: This is UI STATE ONLY - do not cache computed game data here
// - UI state: User selections, mode flags, display preferences
// - Game state: Combat data, positions, stats (stored in ECS)
// - Computed data: Calculate on-demand from ECS via services
type BattleMapState struct {
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

// NewBattleMapState creates a default battle map state
func NewBattleMapState() *BattleMapState {
	return &BattleMapState{
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

// Reset clears the battle map state (for starting a new battle)
func (bms *BattleMapState) Reset() {
	// Clear UI selection state
	bms.SelectedSquadID = ecs.EntityID(0)
	bms.SelectedTargetID = ecs.EntityID(0)

	// Clear UI mode flags
	bms.InAttackMode = false
	bms.InMoveMode = false
	bms.ShowHealthBars = false

	// Clear encounter tracking
	bms.TriggeredEncounterID = ecs.EntityID(0)
}

// DebugMap returns state as a map for structured logging
func (bms *BattleMapState) DebugMap() map[string]interface{} {
	return map[string]interface{}{
		"selectedSquad":  bms.SelectedSquadID,
		"selectedTarget": bms.SelectedTargetID,
		"inAttackMode":   bms.InAttackMode,
		"inMoveMode":     bms.InMoveMode,
		"showHealthBars": bms.ShowHealthBars,
		"encounterID":    bms.TriggeredEncounterID,
	}
}

// OverworldState holds UI-specific state for the overworld context
// This contains ONLY transient UI selection and visualization state
//
// IMPORTANT: This is UI STATE ONLY - do not cache computed game data here
// - UI state: Camera position, selections, display toggles
// - Game state: Tick data, threats, travel (stored in ECS)
type OverworldState struct {
	CameraX          int
	CameraY          int
	SelectedThreatID ecs.EntityID            // Selected threat node for inspection
	HoveredPosition  *coords.LogicalPosition // Current mouse hover position
	ShowInfluence    bool                    // Toggle influence visualization
	IsAutoTraveling  bool                    // Auto-advance ticks during travel
}

// NewOverworldState creates a default overworld state
func NewOverworldState() *OverworldState {
	return &OverworldState{
		CameraX:          0,
		CameraY:          0,
		SelectedThreatID: 0,
		HoveredPosition:  nil,
		ShowInfluence:    false,
	}
}

// ClearSelection clears the currently selected threat
func (os *OverworldState) ClearSelection() {
	os.SelectedThreatID = 0
}

// HasSelection returns true if a threat is currently selected
func (os *OverworldState) HasSelection() bool {
	return os.SelectedThreatID != 0
}
