package guiraid

import (
	"fmt"
	"strings"

	"game_main/gui/framework"
	"game_main/mind/combatpipeline"
	"game_main/mind/raid"
	"game_main/tactical/squads"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// DeployPanel controls the pre-encounter squad deployment interface.
type DeployPanel struct {
	mode *RaidMode
	room *raid.RoomData

	// Cached widget references (populated once in initWidgets)
	titleLabel     *widget.Text
	squadListLabel *widget.Text
	autoDeployBtn  *widget.Button
	startBattleBtn *widget.Button
	backBtn        *widget.Button
}

// NewDeployPanel creates a new deployment panel controller.
func NewDeployPanel(mode *RaidMode) *DeployPanel {
	dp := &DeployPanel{mode: mode}
	dp.initWidgets()
	dp.wireButtons()
	return dp
}

// initWidgets extracts widget references from the panel registry once.
func (dp *DeployPanel) initWidgets() {
	dp.titleLabel = framework.GetPanelWidget[*widget.Text](dp.mode.Panels, RaidPanelDeploy, "titleLabel")
	dp.squadListLabel = framework.GetPanelWidget[*widget.Text](dp.mode.Panels, RaidPanelDeploy, "squadListLabel")
	dp.autoDeployBtn = framework.GetPanelWidget[*widget.Button](dp.mode.Panels, RaidPanelDeploy, "autoDeployBtn")
	dp.startBattleBtn = framework.GetPanelWidget[*widget.Button](dp.mode.Panels, RaidPanelDeploy, "startBattleBtn")
	dp.backBtn = framework.GetPanelWidget[*widget.Button](dp.mode.Panels, RaidPanelDeploy, "backBtn")
}

// wireButtons connects button callbacks.
func (dp *DeployPanel) wireButtons() {
	if dp.autoDeployBtn != nil {
		dp.autoDeployBtn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				dp.autoDeploy()
			}),
		)
	}

	if dp.startBattleBtn != nil {
		dp.startBattleBtn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				dp.mode.OnDeployConfirmed()
			}),
		)
	}

	if dp.backBtn != nil {
		dp.backBtn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				dp.mode.showPanel(PanelFloorMap)
			}),
		)
	}
}

// Refresh updates the deploy panel with current squad info.
func (dp *DeployPanel) Refresh(raidState *raid.RaidStateData, room *raid.RoomData) {
	dp.room = room

	manager := dp.mode.Context.ECSManager

	// Update title
	if dp.titleLabel != nil {
		garrisonCount := len(room.GarrisonSquadIDs)
		dp.titleLabel.Label = fmt.Sprintf("Deploy Squads — %s (Room %d) — %d garrison squads",
			room.RoomType, room.NodeID, garrisonCount)
	}

	// Build squad list
	queries := dp.mode.Queries
	var lines []string
	for _, squadID := range raidState.PlayerSquadIDs {
		squadName := queries.GetSquadName(squadID)

		aliveCount := combatpipeline.CountLivingUnitsInSquad(manager, squadID)
		totalCount := len(squads.GetUnitIDsInSquad(squadID, manager))

		status := "Ready"
		if squads.IsSquadDestroyed(squadID, manager) {
			status = "Destroyed"
		}

		line := fmt.Sprintf("  %s — HP: %d%% | Units: %d/%d | %s",
			squadName,
			int(squads.GetSquadHealthPercent(squadID, manager)*100),
			aliveCount,
			totalCount,
			status,
		)
		lines = append(lines, line)
	}

	if dp.squadListLabel != nil {
		if len(lines) == 0 {
			dp.squadListLabel.Label = "  No squads available."
		} else {
			dp.squadListLabel.Label = strings.Join(lines, "\n")
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
	if inputState.ActionActive(framework.ActionConfirm) {
		dp.mode.OnDeployConfirmed()
		return true
	}
	if inputState.ActionActive(framework.ActionDeployBack) {
		dp.mode.showPanel(PanelFloorMap)
		return true
	}
	return false
}

// Render draws deployment visuals.
func (dp *DeployPanel) Render(screen *ebiten.Image) {
	// Widget rendering is handled by ebitenui.
}
