package guisquads

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for squad editor mode
const (
	SquadEditorPanelCommanderSelector framework.PanelType = "squadeditor_commander_selector"
	SquadEditorPanelSquadSelector     framework.PanelType = "squadeditor_squad_selector"
	SquadEditorPanelGridEditor        framework.PanelType = "squadeditor_grid_editor"
	SquadEditorPanelUnitRoster framework.PanelType = "squadeditor_unit_roster"
)

func init() {
	// Register commander selector panel (prev/next commander + name label)
	framework.RegisterPanel(SquadEditorPanelCommanderSelector, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			selectorWidth := int(float64(layout.ScreenWidth) * 0.4)
			selectorHeight := int(float64(layout.ScreenHeight) * specs.CommanderSelectorHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  selectorWidth,
				MinHeight: selectorHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(15),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			topPad := int(float64(layout.ScreenHeight) * specs.PaddingExtraSmall)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

			// Previous commander button
			prevBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "< Prev Cmdr",
				OnClick: func() {
					sem.showPreviousCommander()
				},
			})
			result.Container.AddChild(prevBtn)

			// Commander name label
			cmdrLabel := builders.CreateSmallLabel("Commander: ---")
			result.Container.AddChild(cmdrLabel)

			// Next commander button
			nextBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Next Cmdr >",
				OnClick: func() {
					sem.showNextCommander()
				},
			})
			result.Container.AddChild(nextBtn)

			result.Custom["commanderPrevBtn"] = prevBtn
			result.Custom["commanderNextBtn"] = nextBtn
			result.Custom["commanderLabel"] = cmdrLabel

			return nil
		},
	})

	// Register squad selector panel (includes nav buttons at bottom)
	framework.RegisterPanel(SquadEditorPanelSquadSelector, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorSquadListWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorSquadListHeight)

			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(listWidth, listHeight)),
			)

			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * (specs.CommanderSelectorHeight + specs.PaddingStandard))
			result.Container.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)

			titleLabel := builders.CreateSmallLabel("Select Squad:")
			result.Container.AddChild(titleLabel)

			// Squad list - will be populated in Enter()
			squadList := builders.CreateSquadList(builders.SquadListConfig{
				SquadIDs:      []ecs.EntityID{},
				Manager:       sem.Context.ECSManager,
				ScreenWidth:   layout.ScreenWidth,
				ScreenHeight:  layout.ScreenHeight,
				WidthPercent:  0.2,
				HeightPercent: 0.4,
				OnSelect: func(squadID ecs.EntityID) {
					sem.onSquadSelected(squadID)
				},
			})
			result.Container.AddChild(squadList)

			result.Custom["squadList"] = squadList

			return nil
		},
	})

	// Register grid editor panel (formation grid + attack pattern grid)
	framework.RegisterPanel(SquadEditorPanelGridEditor, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)

			// Wrapper container holds formation grid + attack pattern section
			wrapper := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionCenter,
					VerticalPosition:   widget.AnchorLayoutPositionCenter,
				})),
			)

			// Formation grid (interactive)
			formationContainer, gridCells := sem.PanelBuilders.BuildGridEditor(builders.GridEditorConfig{
				OnCellClick: func(row, col int) {
					sem.onGridCellClicked(row, col)
				},
			})
			wrapper.AddChild(formationContainer)

			// Attack pattern label (hidden by default)
			attackLabel := builders.CreateSmallLabel("Attack Pattern")
			attackLabel.GetWidget().Visibility = widget.Visibility_Hide
			wrapper.AddChild(attackLabel)

			// Attack pattern grid (read-only, hidden by default)
			attackGridContainer, attackGridCells := sem.PanelBuilders.BuildGridEditor(builders.GridEditorConfig{})
			attackGridContainer.GetWidget().Visibility = widget.Visibility_Hide
			wrapper.AddChild(attackGridContainer)

			result.Container = wrapper
			result.Custom["gridCells"] = gridCells
			result.Custom["attackLabel"] = attackLabel
			result.Custom["attackGridCells"] = attackGridCells
			result.Custom["attackGridContainer"] = attackGridContainer

			return nil
		},
	})

	// Register combined unit list + roster panel with tab switching
	framework.RegisterPanel(SquadEditorPanelUnitRoster, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorUnitListWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorUnitListHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * (specs.CommanderSelectorHeight + specs.PaddingStandard))
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topOffset)

			// Tab row: horizontal container with tab buttons
			tabRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)
			tabRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Squad Units",
				OnClick: func() { sem.switchTab("units") },
			}))
			tabRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Roster",
				OnClick: func() { sem.switchTab("roster") },
			}))
			result.Container.AddChild(tabRow)

			// === Unit content sub-container ===
			unitContent := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			unitContent.AddChild(builders.CreateSmallLabel("Squad Units:"))

			unitList := builders.CreateUnitList(builders.UnitListConfig{
				UnitIDs:       []ecs.EntityID{},
				Manager:       sem.Context.ECSManager,
				ScreenWidth:   400,
				ScreenHeight:  300,
				WidthPercent:  1.0,
				HeightPercent: 1.0,
			})
			unitContent.AddChild(unitList)

			unitContent.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Remove Selected Unit",
				OnClick: func() { sem.onRemoveUnit() },
			}))
			unitContent.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Make Leader",
				OnClick: func() { sem.onMakeLeader() },
			}))
			unitContent.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "View Unit",
				OnClick: func() { sem.onViewUnit() },
			}))

			result.Container.AddChild(unitContent)

			// === Roster content sub-container (starts hidden) ===
			rosterContent := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			rosterContent.AddChild(builders.CreateSmallLabel("Available Units (Roster):"))

			rosterList := builders.CreateSimpleStringList(builders.SimpleStringListConfig{
				Entries:       []string{},
				ScreenWidth:   400,
				ScreenHeight:  200,
				WidthPercent:  1.0,
				HeightPercent: 1.0,
			})
			rosterContent.AddChild(rosterList)

			rosterContent.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Add to Squad",
				OnClick: func() { sem.onAddUnitFromRoster() },
			}))

			rosterContent.GetWidget().Visibility = widget.Visibility_Hide
			result.Container.AddChild(rosterContent)

			// Store references
			result.Custom["unitList"] = unitList
			result.Custom["rosterList"] = rosterList
			result.Custom["unitContent"] = unitContent
			result.Custom["rosterContent"] = rosterContent

			return nil
		},
	})
}

