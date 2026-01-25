package guioverworld

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// OverworldState holds UI-specific state (not game state)
// Follows TinkerRogue pattern: UI state separate from ECS game state
type OverworldState struct {
	CameraX          int
	CameraY          int
	SelectedThreatID ecs.EntityID            // Selected threat node for inspection
	HoveredPosition  *coords.LogicalPosition // Current mouse hover position
	ShowInfluence    bool                    // Toggle influence visualization
}

// NewOverworldState creates a new UI state instance
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
