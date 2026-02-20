package guiraid

import (
	"fmt"

	"game_main/gui/framework"
	"game_main/mind/raid"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/world/worldmap"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// RaidMode implements framework.UIMode for the garrison raid interface.
// Coordinates floor map, deployment, and summary sub-panels.
type RaidMode struct {
	framework.BaseMode

	state      *RaidUIState
	raidRunner *raid.RaidRunner

	// Sub-panel controllers
	floorPanel   *FloorMapPanel
	deployPanel  *DeployPanel
	summaryPanel *SummaryPanel
}

// NewRaidMode creates a new raid mode.
func NewRaidMode(modeManager *framework.UIModeManager) *RaidMode {
	rm := &RaidMode{
		state: NewRaidUIState(),
	}
	rm.SetModeName("raid")
	rm.SetReturnMode("exploration")
	rm.SetSelf(rm)
	rm.ModeManager = modeManager
	return rm
}

// SetRaidRunner injects the raid runner after construction.
func (rm *RaidMode) SetRaidRunner(runner *raid.RaidRunner) {
	rm.raidRunner = runner
}

// Initialize sets up the raid mode's UI infrastructure.
func (rm *RaidMode) Initialize(ctx *framework.UIContext) error {
	err := framework.NewModeBuilder(&rm.BaseMode, framework.ModeConfig{
		ModeName:    "raid",
		ReturnMode:  "exploration",
		StatusLabel: true,
	}).Build(ctx)
	if err != nil {
		return err
	}

	// Build panels from registry
	if err := rm.BuildPanels(
		RaidPanelFloorMap,
		RaidPanelDeploy,
		RaidPanelSummary,
	); err != nil {
		return err
	}

	// Initialize sub-panel controllers
	rm.floorPanel = NewFloorMapPanel(rm)
	rm.deployPanel = NewDeployPanel(rm)
	rm.summaryPanel = NewSummaryPanel(rm)

	return nil
}

// Enter is called when switching to raid mode.
func (rm *RaidMode) Enter(fromMode framework.UIMode) error {
	// Detect returning from combat — show encounter summary
	if fromMode != nil && fromMode.GetModeName() == "combat" {
		if rm.raidRunner != nil && rm.raidRunner.LastEncounterResult != nil {
			rm.OnCombatComplete(rm.raidRunner.LastEncounterResult)
			rm.raidRunner.LastEncounterResult = nil
			return nil
		}
	}

	// Auto-start a raid if none is active (debug entry point)
	if rm.raidRunner != nil && !rm.raidRunner.IsActive() {
		if err := rm.autoStartRaid(); err != nil {
			rm.SetStatus(fmt.Sprintf("Failed to start raid: %v", err))
			rm.showPanel(PanelFloorMap)
			return nil
		}
	}

	rm.showPanel(PanelFloorMap)
	rm.updateFloorMapDisplay()
	return nil
}

// autoStartRaid starts a new raid using the player's current commander and squads.
func (rm *RaidMode) autoStartRaid() error {
	manager := rm.Context.ECSManager
	playerID := rm.Context.PlayerData.PlayerEntityID

	// Find the player's commander
	roster := commander.GetPlayerCommanderRoster(playerID, manager)
	if roster == nil || len(roster.CommanderIDs) == 0 {
		return fmt.Errorf("no commanders available")
	}
	commanderID := roster.CommanderIDs[0]

	// Get the commander's squads
	squadRoster := squads.GetPlayerSquadRoster(commanderID, manager)
	if squadRoster == nil || len(squadRoster.OwnedSquads) == 0 {
		return fmt.Errorf("no squads available")
	}

	// Limit squads to max allowed for raids
	raidSquads := squadRoster.OwnedSquads
	maxSquads := raid.MaxPlayerSquads()
	if len(raidSquads) > maxSquads {
		raidSquads = raidSquads[:maxSquads]
	}

	if err := rm.raidRunner.StartRaid(commanderID, raidSquads, raid.DefaultFloorCount()); err != nil {
		return err
	}

	// Enter floor 1
	if err := rm.raidRunner.EnterFloor(1); err != nil {
		return err
	}

	fmt.Printf("RaidMode: Auto-started raid with %d squads\n", len(raidSquads))
	return nil
}

// Update is called every frame.
func (rm *RaidMode) Update(deltaTime float64) error {
	return nil
}

// HandleInput processes raid-specific input.
func (rm *RaidMode) HandleInput(inputState *framework.InputState) bool {
	if rm.HandleCommonInput(inputState) {
		return true
	}

	switch rm.state.CurrentPanel {
	case PanelFloorMap:
		return rm.floorPanel.HandleInput(inputState)
	case PanelDeploy:
		return rm.deployPanel.HandleInput(inputState)
	case PanelSummary:
		return rm.summaryPanel.HandleInput(inputState)
	}

	return false
}

// Render draws the current raid panel.
func (rm *RaidMode) Render(screen *ebiten.Image) {
	switch rm.state.CurrentPanel {
	case PanelFloorMap:
		rm.floorPanel.Render(screen)
	case PanelDeploy:
		rm.deployPanel.Render(screen)
	case PanelSummary:
		rm.summaryPanel.Render(screen)
	}
}

// showPanel switches the active sub-panel.
func (rm *RaidMode) showPanel(panel RaidPanel) {
	rm.state.CurrentPanel = panel

	// Toggle panel visibility
	floorContainer := rm.GetPanelContainer(RaidPanelFloorMap)
	deployContainer := rm.GetPanelContainer(RaidPanelDeploy)
	summaryContainer := rm.GetPanelContainer(RaidPanelSummary)

	if floorContainer != nil {
		floorContainer.GetWidget().Visibility = visibilityFor(panel == PanelFloorMap)
	}
	if deployContainer != nil {
		deployContainer.GetWidget().Visibility = visibilityFor(panel == PanelDeploy)
	}
	if summaryContainer != nil {
		summaryContainer.GetWidget().Visibility = visibilityFor(panel == PanelSummary)
	}
}

func visibilityFor(visible bool) widget.Visibility {
	if visible {
		return widget.Visibility_Show
	}
	return widget.Visibility_Hide_Blocking
}

// updateFloorMapDisplay refreshes the floor map panel content.
func (rm *RaidMode) updateFloorMapDisplay() {
	if rm.floorPanel == nil {
		return
	}

	raidState := raid.GetRaidState(rm.Context.ECSManager)
	if raidState == nil {
		rm.SetStatus("No active raid")
		return
	}

	rm.floorPanel.Refresh(raidState)
	rm.SetStatus(fmt.Sprintf("Floor %d/%d", raidState.CurrentFloor, raidState.TotalFloors))
}

// OnRoomSelected handles room selection from the floor map panel.
func (rm *RaidMode) OnRoomSelected(nodeID int) {
	rm.state.SelectedRoomID = nodeID

	raidState := raid.GetRaidState(rm.Context.ECSManager)
	if raidState == nil {
		return
	}

	room := raid.GetRoomData(rm.Context.ECSManager, nodeID, raidState.CurrentFloor)
	if room == nil {
		return
	}

	// Non-combat rooms are handled directly by the runner
	switch room.RoomType {
	case worldmap.GarrisonRoomRestRoom, worldmap.GarrisonRoomStairs:
		if err := rm.raidRunner.SelectRoom(nodeID); err != nil {
			rm.SetStatus(fmt.Sprintf("Error: %v", err))
			return
		}
		rm.updateFloorMapDisplay()
		return
	}

	// Combat room — show deployment panel
	rm.showPanel(PanelDeploy)
	rm.deployPanel.Refresh(raidState, room)
}

// OnDeployConfirmed starts the encounter after squad deployment is confirmed.
func (rm *RaidMode) OnDeployConfirmed() {
	if rm.raidRunner == nil {
		return
	}

	// Auto-deploy if no deployment has been set
	deployment := raid.GetDeployment(rm.Context.ECSManager)
	if deployment == nil || len(deployment.DeployedSquadIDs) == 0 {
		_, err := raid.AutoDeploy(rm.Context.ECSManager)
		if err != nil {
			rm.SetStatus(fmt.Sprintf("Deploy failed: %v", err))
			rm.showPanel(PanelFloorMap)
			return
		}
	}

	// Trigger the actual combat encounter
	if err := rm.raidRunner.TriggerRaidEncounter(rm.state.SelectedRoomID); err != nil {
		rm.SetStatus(fmt.Sprintf("Error: %v", err))
		rm.showPanel(PanelFloorMap)
		return
	}

	// Combat mode transition is handled by TriggerRaidEncounter → EncounterService.BeginRaidCombat
}

// OnCombatComplete is called when combat ends and we return to raid mode.
func (rm *RaidMode) OnCombatComplete(result *raid.RaidEncounterResult) {
	rm.state.SummaryData = result
	rm.state.ShowingSummary = true
	rm.showPanel(PanelSummary)
	rm.summaryPanel.Refresh(result)
}

// OnSummaryDismissed returns to the floor map after viewing the summary.
func (rm *RaidMode) OnSummaryDismissed() {
	rm.state.ShowingSummary = false
	rm.state.SummaryData = nil
	rm.showPanel(PanelFloorMap)
	rm.updateFloorMapDisplay()

	// Check if floor is complete — advance to next floor automatically
	raidState := raid.GetRaidState(rm.Context.ECSManager)
	if raidState != nil {
		floorState := raid.GetFloorState(rm.Context.ECSManager, raidState.CurrentFloor)
		if floorState != nil && floorState.IsComplete {
			if raidState.CurrentFloor < raidState.TotalFloors {
				if err := rm.raidRunner.AdvanceFloor(); err != nil {
					rm.SetStatus(fmt.Sprintf("Floor advance failed: %v", err))
				} else {
					rm.updateFloorMapDisplay()
					rm.SetStatus(fmt.Sprintf("Advanced to floor %d/%d", raidState.CurrentFloor, raidState.TotalFloors))
				}
			} else {
				rm.SetStatus("All floors cleared! Raid complete!")
			}
		}
	}
}

// Exit clears transient UI state when leaving raid mode.
func (rm *RaidMode) Exit(toMode framework.UIMode) error {
	rm.state.SelectedRoomID = 0
	rm.state.ShowingSummary = false
	rm.state.SummaryData = nil
	return nil
}
