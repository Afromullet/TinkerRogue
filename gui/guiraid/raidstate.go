package guiraid

import "game_main/mind/raid"

// RaidPanel identifies which sub-panel is currently displayed.
type RaidPanel int

const (
	PanelFloorMap RaidPanel = iota
	PanelDeploy
	PanelSummary
)

// RaidUIState holds UI-only state for the raid mode. No game logic.
type RaidUIState struct {
	SelectedRoomID int
	HoveredRoomID  int
	CurrentPanel   RaidPanel
	ShowingSummary bool
	SummaryData    *raid.RaidEncounterResult
}

// NewRaidUIState creates a fresh raid UI state.
func NewRaidUIState() *RaidUIState {
	return &RaidUIState{
		SelectedRoomID: -1,
		HoveredRoomID:  -1,
		CurrentPanel:   PanelFloorMap,
	}
}
