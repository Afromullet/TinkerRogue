package guiinspect

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/tactical/squads"
	"strings"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// InspectPanelType is the panel type constant for the inspect formation grid.
const InspectPanelType framework.PanelType = "combat_inspect_grid"

// BuildPanel constructs the inspect formation grid widget tree.
// Called from the guicombat panel registry; the registration wrapper handles
// combat-mode-specific concerns (sub-menu registration).
func BuildPanel(result *framework.PanelResult, pb *builders.PanelBuilders) {
	result.Container = pb.BuildPanel(
		builders.RightCenter(),
		builders.Size(specs.CombatInspectPanelWidth, specs.CombatInspectPanelHeight),
		builders.Padding(specs.PaddingTight),
		builders.RowLayout(),
	)

	// Squad name title label
	squadNameLabel := builders.CreateSmallLabel("Squad Formation")
	result.Container.AddChild(squadNameLabel)
	result.Custom["squadNameLabel"] = squadNameLabel

	// 3x3 formation grid (read-only, no OnCellClick)
	gridContainer, gridCells := pb.BuildGridEditor(builders.GridEditorConfig{})
	result.Container.AddChild(gridContainer)
	result.Custom["gridCells"] = gridCells

	// Attack pattern label
	attackLabel := builders.CreateSmallLabel("Attack Pattern")
	result.Container.AddChild(attackLabel)

	// 3x3 attack pattern grid (read-only)
	attackGridContainer, attackGridCells := pb.BuildGridEditor(builders.GridEditorConfig{})
	result.Container.AddChild(attackGridContainer)
	result.Custom["attackGridCells"] = attackGridCells

	// Hidden by default
	result.Container.GetWidget().Visibility = widget.Visibility_Hide
}

// InspectPanelController manages the squad formation inspect panel display.
// Owns the widget references and all grid population logic.
type InspectPanelController struct {
	queries          *framework.GUIQueries
	squadNameLabel   *widget.Text
	gridCells        [3][3]*widget.Button
	attackGridCells  [3][3]*widget.Button
	panelContainer   *widget.Container
}

// NewInspectPanelController creates a new inspect panel controller.
func NewInspectPanelController(queries *framework.GUIQueries) *InspectPanelController {
	return &InspectPanelController{
		queries: queries,
	}
}

// SetWidgets sets widget references after panel construction.
func (ip *InspectPanelController) SetWidgets(nameLabel *widget.Text, gridCells [3][3]*widget.Button, attackGridCells [3][3]*widget.Button, container *widget.Container) {
	ip.squadNameLabel = nameLabel
	ip.gridCells = gridCells
	ip.attackGridCells = attackGridCells
	ip.panelContainer = container
}

// PopulateGrid fills the inspect grid with formation data for the given squad.
func (ip *InspectPanelController) PopulateGrid(squadID ecs.EntityID) {
	if ip.squadNameLabel == nil {
		return
	}

	// Set squad name in title
	squadInfo := ip.queries.GetSquadInfo(squadID)
	if squadInfo != nil {
		ip.squadNameLabel.Label = squadInfo.Name
	} else {
		ip.squadNameLabel.Label = "Squad"
	}

	// Clear all grid cells
	ip.ClearGrid()

	// Get unit IDs in this squad
	unitIDs := ip.queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		info := ip.queries.GetUnitGridInfo(unitID)
		if info == nil {
			continue
		}

		nameStr := info.Name

		var cellText string
		if !info.IsAlive {
			cellText = fmt.Sprintf("%s\n[DEAD]", nameStr)
		} else {
			if info.IsLeader {
				nameStr = "[L] " + nameStr
			}
			if info.MaxHP > 0 {
				cellText = fmt.Sprintf("%s\n%d/%d HP", nameStr, info.CurrentHP, info.MaxHP)
			} else {
				cellText = nameStr
			}
		}

		if info.AnchorRow >= 0 && info.AnchorRow < 3 && info.AnchorCol >= 0 && info.AnchorCol < 3 {
			ip.gridCells[info.AnchorRow][info.AnchorCol].Text().Label = cellText
		}
	}

	ip.populateAttackPatternGrid(squadID)
	ip.Show()
}

// populateAttackPatternGrid fills the attack pattern grid showing which defender
// cells each unit would target, assuming a full 3x3 enemy grid.
func (ip *InspectPanelController) populateAttackPatternGrid(squadID ecs.EntityID) {
	manager := ip.queries.ECSManager
	pattern := squads.ComputeGenericAttackPattern(squadID, manager)
	PopulateAttackGridCells(ip.attackGridCells, pattern)
}

// PopulateAttackGridCells fills a 3x3 button grid with attack pattern data.
// Reusable by any mode that needs attack pattern visualization.
func PopulateAttackGridCells(gridCells [3][3]*widget.Button, pattern [3][3]squads.AttackPatternCell) {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := pattern[row][col]
			var cellText string
			if len(cell.UnitNames) == 0 {
				cellText = ""
			} else if len(cell.UnitNames) <= 2 {
				cellText = strings.Join(cell.UnitNames, "\n")
			} else {
				cellText = strings.Join(cell.UnitNames[:2], "\n") + fmt.Sprintf("\n+%d more", len(cell.UnitNames)-2)
			}
			gridCells[row][col].Text().Label = cellText
		}
	}
}

// ClearGrid clears all grid cell labels.
func (ip *InspectPanelController) ClearGrid() {
	ClearGridCells(ip.gridCells)
	ClearGridCells(ip.attackGridCells)
}

// ClearGridCells clears all labels in a 3x3 button grid.
func ClearGridCells(gridCells [3][3]*widget.Button) {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			gridCells[row][col].Text().Label = ""
		}
	}
}

// Show makes the inspect panel visible.
func (ip *InspectPanelController) Show() {
	if ip.panelContainer != nil {
		ip.panelContainer.GetWidget().Visibility = widget.Visibility_Show
	}
}

// Hide hides the inspect panel.
func (ip *InspectPanelController) Hide() {
	if ip.panelContainer != nil {
		ip.panelContainer.GetWidget().Visibility = widget.Visibility_Hide
	}
}
