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
	SquadEditorPanelSquadSelector framework.PanelType = "squadeditor_squad_selector"
	SquadEditorPanelGridEditor    framework.PanelType = "squadeditor_grid_editor"
	SquadEditorPanelUnitList      framework.PanelType = "squadeditor_unit_list"
	SquadEditorPanelRoster        framework.PanelType = "squadeditor_roster"
)

func init() {
	// Register squad selector panel with commander row header and squad operation buttons
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
			topPad := int(float64(layout.ScreenHeight) * specs.PaddingStandard)
			result.Container.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topPad)

			// Commander row: [< Prev] "Commander: Name" [Next >]
			cmdrRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			prevBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "<",
				OnClick: func() { sem.showPreviousCommander() },
			})
			cmdrRow.AddChild(prevBtn)

			cmdrLabel := builders.CreateSmallLabel("Commander: ---")
			cmdrRow.AddChild(cmdrLabel)

			nextBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    ">",
				OnClick: func() { sem.showNextCommander() },
			})
			cmdrRow.AddChild(nextBtn)

			result.Container.AddChild(cmdrRow)

			// Squad list title
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

			// Squad operation buttons
			newSquadBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "New Squad (N)",
				OnClick: func() { sem.onNewSquad() },
			})
			result.Container.AddChild(newSquadBtn)

			renameBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Rename",
				OnClick: func() { sem.onRenameSquad() },
			})
			result.Container.AddChild(renameBtn)

			result.Custom["squadList"] = squadList
			result.Custom["commanderPrevBtn"] = prevBtn
			result.Custom["commanderNextBtn"] = nextBtn
			result.Custom["commanderLabel"] = cmdrLabel

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

	// Register unit list panel (right side, hidden by default, managed by SubMenuController)
	framework.RegisterPanel(SquadEditorPanelUnitList, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorUnitPanelWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorUnitPanelHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * specs.PaddingStandard)
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topOffset)

			result.Container.AddChild(builders.CreateSmallLabel("Squad Units:"))

			unitList := builders.CreateUnitList(builders.UnitListConfig{
				UnitIDs:       []ecs.EntityID{},
				Manager:       sem.Context.ECSManager,
				ScreenWidth:   listWidth,
				ScreenHeight:  listHeight,
				WidthPercent:  1.0,
				HeightPercent: 0.50,
			})
			result.Container.AddChild(unitList)

			result.Container.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Remove Selected Unit",
				OnClick: func() { sem.onRemoveUnit() },
			}))
			result.Container.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Make Leader",
				OnClick: func() { sem.onMakeLeader() },
			}))
			result.Container.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "View Unit",
				OnClick: func() { sem.onViewUnit() },
			}))

			// Hidden by default, registered with SubMenuController
			result.Container.GetWidget().Visibility = widget.Visibility_Hide
			sem.subMenus.Register("units", result.Container)

			result.Custom["unitList"] = unitList

			return nil
		},
	})

	// Register roster panel (right side, hidden by default, managed by SubMenuController)
	framework.RegisterPanel(SquadEditorPanelRoster, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorRosterPanelWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorRosterPanelHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * specs.PaddingStandard)
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topOffset)

			result.Container.AddChild(builders.CreateSmallLabel("Available Units (Roster):"))

			rosterList := builders.CreateSimpleStringList(builders.SimpleStringListConfig{
				Entries:       []string{},
				ScreenWidth:   400,
				ScreenHeight:  200,
				WidthPercent:  1.0,
				HeightPercent: 1.0,
			})
			result.Container.AddChild(rosterList)

			result.Container.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Add to Squad",
				OnClick: func() { sem.onAddUnitFromRoster() },
			}))

			// Hidden by default, registered with SubMenuController
			result.Container.GetWidget().Visibility = widget.Visibility_Hide
			sem.subMenus.Register("roster", result.Container)

			result.Custom["rosterList"] = rosterList

			return nil
		},
	})
}
