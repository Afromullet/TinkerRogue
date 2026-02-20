package guiraid

import (
	"fmt"
	"strings"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/mind/raid"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// FloorMapPanel controls the floor map display within the raid mode.
type FloorMapPanel struct {
	mode        *RaidMode
	roomButtons []*widget.Button

	// Cached widget references (populated once in initWidgets)
	titleLabel    *widget.Text
	roomListLabel *widget.Text
	alertLabel    *widget.Text
	retreatBtn    *widget.Button
}

// NewFloorMapPanel creates a new floor map panel controller.
func NewFloorMapPanel(mode *RaidMode) *FloorMapPanel {
	fp := &FloorMapPanel{mode: mode}
	fp.initWidgets()
	fp.wireButtons()
	return fp
}

// initWidgets extracts widget references from the panel registry once.
func (fp *FloorMapPanel) initWidgets() {
	fp.titleLabel = framework.GetPanelWidget[*widget.Text](fp.mode.Panels, RaidPanelFloorMap, "titleLabel")
	fp.roomListLabel = framework.GetPanelWidget[*widget.Text](fp.mode.Panels, RaidPanelFloorMap, "roomListLabel")
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

	// Build room list text
	rooms := raid.GetAllRoomsForFloor(manager, raidState.CurrentFloor)
	var lines []string
	for _, room := range rooms {
		status := "Locked"
		if room.IsCleared {
			status = "Cleared"
		} else if room.IsAccessible {
			status = "Accessible"
		}

		garrisonInfo := ""
		if len(room.GarrisonSquadIDs) > 0 && !room.IsCleared {
			garrisonInfo = fmt.Sprintf(" [%d squads]", len(room.GarrisonSquadIDs))
		}

		line := fmt.Sprintf("  Room %d: %s (%s)%s",
			room.NodeID, room.RoomType, status, garrisonInfo)

		if room.OnCriticalPath {
			line += " *"
		}

		lines = append(lines, line)
	}

	if fp.roomListLabel != nil {
		fp.roomListLabel.Label = strings.Join(lines, "\n")
	}

	// Rebuild room buttons for accessible rooms
	fp.rebuildRoomButtons(rooms)
}

// rebuildRoomButtons creates clickable buttons for accessible rooms.
func (fp *FloorMapPanel) rebuildRoomButtons(rooms []*raid.RoomData) {
	panel := fp.mode.Panels.Get(RaidPanelFloorMap)
	if panel == nil {
		return
	}

	// Remove old room buttons
	for _, btn := range fp.roomButtons {
		panel.Container.RemoveChild(btn)
	}
	fp.roomButtons = nil

	// Add buttons for accessible, uncleared rooms
	for _, room := range rooms {
		if !room.IsAccessible || room.IsCleared {
			continue
		}

		roomID := room.NodeID
		roomType := room.RoomType

		btn := builders.CreateButtonWithConfig(builders.ButtonConfig{
			Text: fmt.Sprintf("Enter %s (Room %d)", roomType, roomID),
			OnClick: func() {
				fp.mode.OnRoomSelected(roomID)
			},
		})

		panel.Container.AddChild(btn)
		fp.roomButtons = append(fp.roomButtons, btn)
	}
}

// HandleInput processes input for the floor map panel.
func (fp *FloorMapPanel) HandleInput(inputState *framework.InputState) bool {
	// Number keys for quick room selection
	accessibleRooms := raid.GetAccessibleRooms(fp.mode.Context.ECSManager,
		fp.getCurrentFloor())

	for i, nodeID := range accessibleRooms {
		key := ebiten.Key(int(ebiten.Key1) + i)
		if i >= 9 {
			break
		}
		if inputState.KeysJustPressed[key] {
			fp.mode.OnRoomSelected(nodeID)
			return true
		}
	}

	return false
}

// Render draws floor map visuals (room graph overlay).
func (fp *FloorMapPanel) Render(screen *ebiten.Image) {
	// The panel registry handles widget rendering via ebitenui.
	// Custom rendering (room graph lines) could be added here later.
}

func (fp *FloorMapPanel) getCurrentFloor() int {
	raidState := raid.GetRaidState(fp.mode.Context.ECSManager)
	if raidState == nil {
		return 1
	}
	return raidState.CurrentFloor
}
