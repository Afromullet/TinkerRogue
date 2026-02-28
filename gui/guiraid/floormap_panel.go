package guiraid

import (
	"fmt"

	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/mind/raid"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// FloorMapPanel controls the floor map display within the raid mode.
type FloorMapPanel struct {
	mode     *RaidMode
	renderer *FloorMapRenderer

	// Cached widget references (populated once in initWidgets)
	titleLabel *widget.Text
	alertLabel *widget.Text
	retreatBtn *widget.Button
}

// NewFloorMapPanel creates a new floor map panel controller.
func NewFloorMapPanel(mode *RaidMode) *FloorMapPanel {
	fp := &FloorMapPanel{
		mode:     mode,
		renderer: NewFloorMapRenderer(),
	}
	fp.initWidgets()
	fp.wireButtons()
	return fp
}

// initWidgets extracts widget references from the panel registry once.
func (fp *FloorMapPanel) initWidgets() {
	fp.titleLabel = framework.GetPanelWidget[*widget.Text](fp.mode.Panels, RaidPanelFloorMap, "titleLabel")
	fp.alertLabel = framework.GetPanelWidget[*widget.Text](fp.mode.Panels, RaidPanelFloorMap, "alertLabel")
	fp.retreatBtn = framework.GetPanelWidget[*widget.Button](fp.mode.Panels, RaidPanelFloorMap, "retreatBtn")
}

// wireButtons connects the retreat button callback.
func (fp *FloorMapPanel) wireButtons() {
	if fp.retreatBtn != nil {
		fp.retreatBtn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				if fp.mode.raidRunner != nil {
					if err := fp.mode.raidRunner.Retreat(); err != nil {
						fp.mode.SetStatus(fmt.Sprintf("Retreat failed: %v", err))
						return
					}
					fp.mode.ModeManager.SetMode("exploration")
				}
			}),
		)
	}
}

// Refresh updates the floor map display with current room data.
func (fp *FloorMapPanel) Refresh(raidState *raid.RaidStateData) {
	manager := fp.mode.Context.ECSManager

	// Update title
	if fp.titleLabel != nil {
		fp.titleLabel.Label = fmt.Sprintf("Garrison Raid — Floor %d/%d", raidState.CurrentFloor, raidState.TotalFloors)
	}

	// Update alert level
	alertData := raid.GetAlertData(manager, raidState.CurrentFloor)
	if fp.alertLabel != nil {
		if alertData != nil {
			levelCfg := raid.GetAlertLevel(alertData.CurrentLevel)
			levelName := "Unknown"
			if levelCfg != nil {
				levelName = levelCfg.Name
			}
			fp.alertLabel.Label = fmt.Sprintf("Alert Level: %d (%s) | Encounters: %d",
				alertData.CurrentLevel, levelName, alertData.EncounterCount)
		}
	}

	// Compute card layout for the renderer
	rooms := raid.GetAllRoomsForFloor(manager, raidState.CurrentFloor)

	// Compute drawing area from screen dimensions minus space for title/alert/buttons
	layout := fp.mode.Layout
	panelW := int(float64(layout.ScreenWidth) * specs.RaidFloorMapWidth)
	panelH := int(float64(layout.ScreenHeight) * specs.RaidFloorMapHeight)
	panelX := (layout.ScreenWidth - panelW) / 2
	panelY := (layout.ScreenHeight - panelH) / 2

	// Leave room for title+alert at top (~80px) and button row at bottom (~60px)
	areaX := panelX + 20
	areaY := panelY + 80
	areaW := panelW - 40
	areaH := panelH - 140

	fp.renderer.ComputeLayout(rooms, areaX, areaY, areaW, areaH)

	// Sync selected room highlight
	fp.renderer.SetSelected(fp.mode.state.SelectedRoomID)
}

// Update advances animations each frame.
func (fp *FloorMapPanel) Update(deltaTime float64) {
	fp.renderer.Update(deltaTime)
}

// HandleInput processes input for the floor map panel.
func (fp *FloorMapPanel) HandleInput(inputState *framework.InputState) bool {
	// Update hover state
	hoveredID := fp.renderer.UpdateHover(inputState.MouseX, inputState.MouseY)
	fp.mode.state.HoveredRoomID = hoveredID

	// Click detection on accessible cards
	if inputState.ActionActive(framework.ActionMouseClick) {
		room := fp.renderer.HitTest(inputState.MouseX, inputState.MouseY)
		if room != nil && room.IsAccessible && !room.IsCleared {
			fp.mode.OnRoomSelected(room.NodeID)
			return true
		}
	}

	// Number keys for quick room selection
	accessibleRooms := raid.GetAccessibleRooms(fp.mode.Context.ECSManager,
		fp.getCurrentFloor())

	roomActions := []framework.InputAction{
		framework.ActionSelectRoom1, framework.ActionSelectRoom2, framework.ActionSelectRoom3,
		framework.ActionSelectRoom4, framework.ActionSelectRoom5, framework.ActionSelectRoom6,
		framework.ActionSelectRoom7, framework.ActionSelectRoom8, framework.ActionSelectRoom9,
	}
	for i, nodeID := range accessibleRooms {
		if i >= len(roomActions) {
			break
		}
		if inputState.ActionActive(roomActions[i]) {
			fp.mode.OnRoomSelected(nodeID)
			return true
		}
	}

	return false
}

// Render draws floor map visuals (card grid overlay).
func (fp *FloorMapPanel) Render(screen *ebiten.Image) {
	fp.renderer.Render(screen)
}

func (fp *FloorMapPanel) getCurrentFloor() int {
	raidState := raid.GetRaidState(fp.mode.Context.ECSManager)
	if raidState == nil {
		return 1
	}
	return raidState.CurrentFloor
}
