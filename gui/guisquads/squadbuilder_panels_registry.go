package guisquads

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guiresources"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/tactical/squads"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for squad builder mode
const (
	SquadBuilderPanelNameInput     framework.PanelType = "squadbuilder_name_input"
	SquadBuilderPanelGrid          framework.PanelType = "squadbuilder_grid"
	SquadBuilderPanelRosterPalette framework.PanelType = "squadbuilder_roster_palette"
	SquadBuilderPanelCapacity      framework.PanelType = "squadbuilder_capacity"
	SquadBuilderPanelDetails       framework.PanelType = "squadbuilder_details"
	SquadBuilderPanelActionButtons framework.PanelType = "squadbuilder_action_buttons"
)

func init() {
	// Register squad name input panel
	framework.RegisterPanel(SquadBuilderPanelNameInput, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sbm := mode.(*SquadBuilderMode)
			layout := sbm.Layout

			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(10),
				)),
			)

			// Label
			nameLabel := builders.CreateLargeLabel("Squad Name:")
			result.Container.AddChild(nameLabel)

			// Text input
			squadNameInput := builders.CreateTextInputWithConfig(builders.TextInputConfig{
				MinWidth:    300,
				MinHeight:   50,
				FontFace:    guiresources.SmallFace,
				Placeholder: "Enter squad name...",
				OnChanged: func(text string) {
					sbm.currentSquadName = text
				},
			})
			result.Container.AddChild(squadNameInput)

			// Position at top center with responsive padding
			vPadding := int(float64(layout.ScreenHeight) * specs.PaddingStandard)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(vPadding)

			result.Custom["squadNameInput"] = squadNameInput

			return nil
		},
	})

	// Register grid panel
	framework.RegisterPanel(SquadBuilderPanelGrid, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sbm := mode.(*SquadBuilderMode)
			layout := sbm.Layout

			padding := int(float64(layout.ScreenWidth) * specs.PaddingTight)

			var buttons [3][3]*widget.Button
			result.Container, buttons = sbm.PanelBuilders.BuildGridEditor(builders.GridEditorConfig{
				CellTextFormat: func(row, col int) string {
					return fmt.Sprintf("Empty\n[%d,%d]", row, col)
				},
				OnCellClick: func(row, col int) {
					sbm.onCellClicked(row, col)
				},
				Padding: widget.Insets{Left: padding, Right: padding, Top: padding, Bottom: padding},
			})

			result.Custom["gridButtons"] = buttons

			return nil
		},
	})

	// Register roster palette panel
	framework.RegisterPanel(SquadBuilderPanelRosterPalette, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sbm := mode.(*SquadBuilderMode)
			layout := sbm.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadBuilderUnitListWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadBuilderUnitListHeight)
			hPadding := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			vPadding := int(float64(layout.ScreenHeight) * specs.PaddingStandard)

			baseList := builders.CreateListWithConfig(builders.ListConfig{
				Entries:   []interface{}{}, // Will be populated dynamically
				MinWidth:  listWidth,
				MinHeight: listHeight,
				EntryLabelFunc: func(e interface{}) string {
					// Handle roster entries
					if rosterEntry, ok := e.(*squads.UnitRosterEntry); ok {
						roster := squads.GetPlayerRoster(sbm.Context.PlayerData.PlayerEntityID, sbm.Queries.ECSManager)
						if roster != nil {
							available := roster.GetAvailableCount(rosterEntry.TemplateName)
							return fmt.Sprintf("%s (x%d)", rosterEntry.TemplateName, available)
						}
						return rosterEntry.TemplateName
					}
					// Handle string messages
					if str, ok := e.(string); ok {
						return str
					}
					return fmt.Sprintf("%v", e)
				},
				OnEntrySelected: func(entry interface{}) {
					if rosterEntry, ok := entry.(*squads.UnitRosterEntry); ok {
						sbm.selectedRosterEntry = rosterEntry
						sbm.updateUnitDetails()
					} else {
						// Deselect if it's a message string
						sbm.selectedRosterEntry = nil
						sbm.updateUnitDetails()
					}
				},
				LayoutData: widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionStart,
					VerticalPosition:   widget.AnchorLayoutPositionCenter,
					Padding: widget.Insets{
						Left: hPadding,
						Top:  vPadding,
					},
				},
			})

			// Wrap with caching for performance
			unitPalette := widgets.NewCachedListWrapper(baseList)

			// Wrap in container with LayoutData
			layoutData := baseList.GetWidget().LayoutData
			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(layoutData)),
			)
			result.Container.AddChild(baseList)

			result.Custom["unitPalette"] = unitPalette
			result.Custom["baseList"] = baseList

			return nil
		},
	})

	// Register capacity display panel
	framework.RegisterPanel(SquadBuilderPanelCapacity, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sbm := mode.(*SquadBuilderMode)
			layout := sbm.Layout

			displayWidth := int(float64(layout.ScreenWidth) * specs.SquadBuilderInfoWidth)
			displayHeight := int(float64(layout.ScreenHeight) * specs.SquadBuilderInfoHeight)

			capacityDisplay := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  displayWidth,
				MinHeight: displayHeight,
				FontColor: color.White,
			})
			capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No leader)")

			hPadding := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			vPadding := int(float64(layout.ScreenHeight) * specs.PaddingStackedWidget)
			capacityDisplay.GetWidget().LayoutData = builders.AnchorEndStart(hPadding, vPadding)

			// Wrap in container with LayoutData
			layoutData := capacityDisplay.GetWidget().LayoutData
			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(layoutData)),
			)
			result.Container.AddChild(capacityDisplay)

			result.Custom["capacityDisplay"] = capacityDisplay

			return nil
		},
	})

	// Register details panel
	framework.RegisterPanel(SquadBuilderPanelDetails, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sbm := mode.(*SquadBuilderMode)
			layout := sbm.Layout

			displayWidth := int(float64(layout.ScreenWidth) * specs.SquadBuilderInfoWidth)
			displayHeight := int(float64(layout.ScreenHeight) * (specs.SquadBuilderInfoHeight * 2))

			unitDetailsArea := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  displayWidth,
				MinHeight: displayHeight,
				FontColor: color.White,
			})
			unitDetailsArea.SetText("Select a unit to view details")

			hPadding := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			unitDetailsArea.GetWidget().LayoutData = builders.AnchorEndCenter(hPadding)

			// Wrap in container with LayoutData
			layoutData := unitDetailsArea.GetWidget().LayoutData
			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(layoutData)),
			)
			result.Container.AddChild(unitDetailsArea)

			result.Custom["unitDetailsArea"] = unitDetailsArea

			return nil
		},
	})

	// Register action buttons panel
	framework.RegisterPanel(SquadBuilderPanelActionButtons, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sbm := mode.(*SquadBuilderMode)
			layout := sbm.Layout

			spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			hPadding := int(float64(layout.ScreenWidth) * specs.PaddingExtraSmall)

			result.Container = builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons: []builders.ButtonSpec{
					{Text: "Create Squad", OnClick: func() { sbm.onCreateSquad() }},
					{Text: "Clear Grid", OnClick: func() { sbm.onClearGrid() }},
					{Text: "Toggle Leader (L)", OnClick: func() { sbm.onToggleLeader() }},
					{Text: "Close (ESC)", OnClick: func() { sbm.handleClose() }},
				},
				Direction: widget.DirectionHorizontal,
				Spacing:   spacing,
				Padding:   widget.Insets{Left: hPadding, Right: hPadding},
			})

			bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
			anchorLayout := builders.AnchorCenterEnd(bottomPad)
			result.Container.GetWidget().LayoutData = anchorLayout

			return nil
		},
	})
}

// Helper functions to retrieve widgets from panel registry

func GetSquadBuilderNameInput(panels *framework.PanelRegistry) *widget.TextInput {
	if result := panels.Get(SquadBuilderPanelNameInput); result != nil {
		if input, ok := result.Custom["squadNameInput"].(*widget.TextInput); ok {
			return input
		}
	}
	return nil
}

func GetSquadBuilderGridButtons(panels *framework.PanelRegistry) [3][3]*widget.Button {
	if result := panels.Get(SquadBuilderPanelGrid); result != nil {
		if buttons, ok := result.Custom["gridButtons"].([3][3]*widget.Button); ok {
			return buttons
		}
	}
	return [3][3]*widget.Button{}
}

func GetSquadBuilderUnitPalette(panels *framework.PanelRegistry) *widgets.CachedListWrapper {
	if result := panels.Get(SquadBuilderPanelRosterPalette); result != nil {
		if palette, ok := result.Custom["unitPalette"].(*widgets.CachedListWrapper); ok {
			return palette
		}
	}
	return nil
}

func GetSquadBuilderCapacityDisplay(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(SquadBuilderPanelCapacity); result != nil {
		if display, ok := result.Custom["capacityDisplay"].(*widget.TextArea); ok {
			return display
		}
	}
	return nil
}

func GetSquadBuilderUnitDetailsArea(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(SquadBuilderPanelDetails); result != nil {
		if area, ok := result.Custom["unitDetailsArea"].(*widget.TextArea); ok {
			return area
		}
	}
	return nil
}
