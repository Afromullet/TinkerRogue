package guiraid

import (
	"fmt"
	"strings"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/raid"
	"game_main/tactical/squads"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// DeployPanel controls the pre-encounter squad deployment interface.
type DeployPanel struct {
	mode     *RaidMode
	room     *raid.RoomData
}

// NewDeployPanel creates a new deployment panel controller.
func NewDeployPanel(mode *RaidMode) *DeployPanel {
	dp := &DeployPanel{mode: mode}
	dp.wireButtons()
	return dp
}

// wireButtons connects button callbacks.
func (dp *DeployPanel) wireButtons() {
	panel := dp.mode.Panels.Get(RaidPanelDeploy)
	if panel == nil {
		return
	}

	if btn, ok := panel.Custom["autoDeployBtn"].(*widget.Button); ok {
		btn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				dp.autoDeploy()
			}),
		)
	}

	if btn, ok := panel.Custom["startBattleBtn"].(*widget.Button); ok {
		btn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				dp.mode.OnDeployConfirmed()
			}),
		)
	}

	if btn, ok := panel.Custom["backBtn"].(*widget.Button); ok {
		btn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				dp.mode.showPanel(PanelFloorMap)
			}),
		)
	}
}

// Refresh updates the deploy panel with current squad info.
func (dp *DeployPanel) Refresh(raidState *raid.RaidStateData, room *raid.RoomData) {
	dp.room = room
	panel := dp.mode.Panels.Get(RaidPanelDeploy)
	if panel == nil {
		return
	}

	manager := dp.mode.Context.ECSManager

	// Update title
	if titleLabel, ok := panel.Custom["titleLabel"].(*widget.Text); ok {
		garrisonCount := len(room.GarrisonSquadIDs)
		titleLabel.Label = fmt.Sprintf("Deploy Squads — %s (Room %d) — %d garrison squads",
			room.RoomType, room.NodeID, garrisonCount)
	}

	// Build squad list
	var lines []string
	for _, squadID := range raidState.PlayerSquadIDs {
		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
		if squadData == nil {
			continue
		}

		aliveCount := raid.CountLivingUnits(manager, squadID)
		totalCount := len(squads.GetUnitIDsInSquad(squadID, manager))

		status := "Ready"
		if squads.IsSquadDestroyed(squadID, manager) {
			status = "Destroyed"
		}

		line := fmt.Sprintf("  %s — HP: %d%% | Morale: %d | Units: %d/%d | %s",
			squadData.Name,
			int(squads.GetSquadHealthPercent(squadID, manager)*100),
			squadData.Morale,
			aliveCount,
			totalCount,
			status,
		)
		lines = append(lines, line)
	}

	if squadListLabel, ok := panel.Custom["squadListLabel"].(*widget.Text); ok {
		if len(lines) == 0 {
			squadListLabel.Label = "  No squads available."
		} else {
			squadListLabel.Label = strings.Join(lines, "\n")
		}
	}
}

// autoDeploy runs auto-deployment and refreshes the display.
func (dp *DeployPanel) autoDeploy() {
	if dp.mode.raidRunner == nil {
		return
	}

	raidState := raid.GetRaidState(dp.mode.Context.ECSManager)
	if raidState == nil {
		return
	}

	_, err := raid.AutoDeploy(dp.mode.Context.ECSManager)
	if err != nil {
		dp.mode.SetStatus(fmt.Sprintf("Auto deploy failed: %v", err))
		return
	}

	dp.mode.SetStatus("Auto-deployed squads")
	if dp.room != nil {
		dp.Refresh(raidState, dp.room)
	}
}

// HandleInput processes deploy panel input.
func (dp *DeployPanel) HandleInput(inputState *framework.InputState) bool {
	if inputState.KeysJustPressed[ebiten.KeyEnter] {
		dp.mode.OnDeployConfirmed()
		return true
	}
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		dp.mode.showPanel(PanelFloorMap)
		return true
	}
	return false
}

// Render draws deployment visuals.
func (dp *DeployPanel) Render(screen *ebiten.Image) {
	// Widget rendering is handled by ebitenui.
}
