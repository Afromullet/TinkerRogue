package framework

import (
	"github.com/bytearena/ecs"
)

// OverworldState holds persistent UI state for the overworld context
// This data is saved when entering battle map and restored when returning
// TODO. None of the state is currently used. Determine whether we still need them
type OverworldState struct {
	// Squad management state
	SelectedSquadID ecs.EntityID   // Currently selgui/modebuilder.goected squad in squad management
	SquadListScroll int            // Scroll position in squad list
	SquadIDs        []ecs.EntityID // All available squads (for quick access)

	// Formation editor state
	EditingSquadID    ecs.EntityID // Squad being edited in formation editor
	FormationDirty    bool         // Whether formation has unsaved changes
	SelectedFormation string       // Name of selected formation preset

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
	InAttackMode bool // Whether attack overlay is showing
	InMoveMode   bool // Whether movement overlay is showing

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
		InAttackMode: false,
		InMoveMode:   false,

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
		"encounterID":    bms.TriggeredEncounterID,
	}
}
