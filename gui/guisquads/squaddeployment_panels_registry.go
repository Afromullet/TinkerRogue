package guisquads

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/gui/widgets"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for squad deployment mode
const (
	SquadDeploymentPanelInstruction   framework.PanelType = "squaddeployment_instruction"
	SquadDeploymentPanelSquadList     framework.PanelType = "squaddeployment_squad_list"
	SquadDeploymentPanelDetailPanel   framework.PanelType = "squaddeployment_detail_panel"
	SquadDeploymentPanelActionButtons framework.PanelType = "squaddeployment_action_buttons"
)

func init() {
	// Register instruction text panel
	framework.RegisterPanel(SquadDeploymentPanelInstruction, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sdm := mode.(*SquadDeploymentMode)
			layout := sdm.Layout

			instructionText := builders.CreateSmallLabel("Select a squad from the list, then click on the map to place it")
			topPad := int(float64(layout.ScreenHeight) * specs.PaddingStandard)

			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(builders.AnchorCenterStart(topPad))),
			)
			result.Container.AddChild(instructionText)

			result.Custom["instructionText"] = instructionText

			return nil
		},
	})

	// Register squad list panel
	framework.RegisterPanel(SquadDeploymentPanelSquadList, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sdm := mode.(*SquadDeploymentMode)
			layout := sdm.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadDeployListWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadDeployListHeight)

			baseList := builders.CreateListWithConfig(builders.ListConfig{
				Entries:   []interface{}{}, // Will be populated in Enter
				MinWidth:  listWidth,
				MinHeight: listHeight,
				EntryLabelFunc: func(e interface{}) string {
					if squadID, ok := e.(ecs.EntityID); ok {
						squadName := sdm.Queries.SquadCache.GetSquadName(squadID)
						unitCount := len(sdm.Queries.SquadCache.GetUnitIDsInSquad(squadID))

						// Check if squad has been placed
						allPositions := sdm.deploymentService.GetAllSquadPositions()
						if pos, hasPosition := allPositions[squadID]; hasPosition {
							return fmt.Sprintf("%s (%d units) - Placed at (%d, %d)", squadName, unitCount, pos.X, pos.Y)
						}
						return fmt.Sprintf("%s (%d units)", squadName, unitCount)
					}
					return fmt.Sprintf("%v", e)
				},
				OnEntrySelected: func(selectedEntry interface{}) {
					if squadID, ok := selectedEntry.(ecs.EntityID); ok {
						sdm.selectedSquadID = squadID
						sdm.isPlacingSquad = true
						sdm.updateInstructionText()
						sdm.updateDetailPanel()
					}
				},
			})

			// Wrap with caching for performance
			squadList := widgets.NewCachedListWrapper(baseList)

			// Position below instruction text
			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * (specs.PaddingStandard * 3))

			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(builders.AnchorStartStart(leftPad, topOffset))),
			)
			result.Container.AddChild(baseList)

			result.Custom["squadList"] = squadList
			result.Custom["baseList"] = baseList

			return nil
		},
	})

	// Register detail panel
	framework.RegisterPanel(SquadDeploymentPanelDetailPanel, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sdm := mode.(*SquadDeploymentMode)
			layout := sdm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.35)
			panelHeight := int(float64(layout.ScreenHeight) * 0.6)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  panelWidth,
				MinHeight: panelHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(10),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingTight)),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			result.Container.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)

			// Detail text area - cached for performance
			detailTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  panelWidth - 30,
				MinHeight: panelHeight - 30,
				FontColor: color.White,
			})
			detailTextArea.SetText("Select a squad to view details")
			result.Container.AddChild(detailTextArea)

			result.Custom["detailTextArea"] = detailTextArea

			return nil
		},
	})

	// Register action buttons panel
	framework.RegisterPanel(SquadDeploymentPanelActionButtons, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sdm := mode.(*SquadDeploymentMode)

			result.Container = builders.CreateBottomActionBar(sdm.Layout, []builders.ButtonSpec{
				{Text: "Clear All", OnClick: func() { sdm.clearAllSquadPositions() }},
				{Text: "Start Combat", OnClick: func() {
					if combatMode, exists := sdm.ModeManager.GetMode("combat"); exists {
						sdm.ModeManager.RequestTransition(combatMode, "Squads deployed, starting combat")
					}
				}},
				{Text: "Close (ESC)", OnClick: func() {
					if mode, exists := sdm.ModeManager.GetMode("exploration"); exists {
						sdm.ModeManager.RequestTransition(mode, "Close button pressed")
					}
				}},
			})

			return nil
		},
	})
}

