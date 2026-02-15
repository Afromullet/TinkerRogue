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
	SquadEditorPanelNavigation        framework.PanelType = "squadeditor_navigation"
	SquadEditorPanelSquadSelector     framework.PanelType = "squadeditor_squad_selector"
	SquadEditorPanelGridEditor        framework.PanelType = "squadeditor_grid_editor"
	SquadEditorPanelUnitList          framework.PanelType = "squadeditor_unit_list"
	SquadEditorPanelRosterList        framework.PanelType = "squadeditor_roster_list"
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

	// Register navigation panel (Previous/Next buttons + counter)
	framework.RegisterPanel(SquadEditorPanelNavigation, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			navWidth := int(float64(layout.ScreenWidth) * 0.5)
			navHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorNavHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  navWidth,
				MinHeight: navHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(20),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			topPad := int(float64(layout.ScreenHeight) * (specs.CommanderSelectorHeight + specs.PaddingExtraSmall))
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

			// Previous button
			prevButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "< Previous",
				OnClick: func() {
					sem.showPreviousSquad()
				},
			})
			result.Container.AddChild(prevButton)

			// Squad counter label
			counterLabel := builders.CreateSmallLabel("Squad 1 of 1")
			result.Container.AddChild(counterLabel)

			// Next button
			nextButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Next >",
				OnClick: func() {
					sem.showNextSquad()
				},
			})
			result.Container.AddChild(nextButton)

			// Store widgets for later access
			result.Custom["prevButton"] = prevButton
			result.Custom["nextButton"] = nextButton
			result.Custom["counterLabel"] = counterLabel

			return nil
		},
	})

	// Register squad selector panel
	framework.RegisterPanel(SquadEditorPanelSquadSelector, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorSquadListWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorSquadListHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * (specs.CommanderSelectorHeight + specs.SquadEditorNavHeight + specs.PaddingStandard*2))
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

	// Register grid editor panel
	framework.RegisterPanel(SquadEditorPanelGridEditor, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)

			container, gridCells := sem.PanelBuilders.BuildGridEditor(builders.GridEditorConfig{
				OnCellClick: func(row, col int) {
					sem.onGridCellClicked(row, col)
				},
			})

			result.Container = container
			result.Custom["gridCells"] = gridCells

			return nil
		},
	})

	// Register unit list panel
	framework.RegisterPanel(SquadEditorPanelUnitList, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorUnitListWidth)
			listHeight := int(float64(layout.ScreenHeight) * 0.35)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * (specs.CommanderSelectorHeight + specs.SquadEditorNavHeight + specs.PaddingStandard*2))
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topOffset)

			titleLabel := builders.CreateSmallLabel("Squad Units:")
			result.Container.AddChild(titleLabel)

			// Unit list - will be populated when squad is selected
			unitList := builders.CreateUnitList(builders.UnitListConfig{
				UnitIDs:       []ecs.EntityID{},
				Manager:       sem.Context.ECSManager,
				ScreenWidth:   400,
				ScreenHeight:  300,
				WidthPercent:  1.0,
				HeightPercent: 1.0,
			})
			result.Container.AddChild(unitList)

			// Action buttons
			removeUnitBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Remove Selected Unit",
				OnClick: func() {
					sem.onRemoveUnit()
				},
			})
			result.Container.AddChild(removeUnitBtn)

			makeLeaderBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Make Leader",
				OnClick: func() {
					sem.onMakeLeader()
				},
			})
			result.Container.AddChild(makeLeaderBtn)

			result.Custom["unitList"] = unitList

			return nil
		},
	})

	// Register roster list panel
	framework.RegisterPanel(SquadEditorPanelRosterList, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorRosterListWidth)
			listHeight := int(float64(layout.ScreenHeight) * 0.35)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * (specs.CommanderSelectorHeight + specs.SquadEditorNavHeight + 0.35 + specs.PaddingStandard*3))
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topOffset)

			titleLabel := builders.CreateSmallLabel("Available Units (Roster):")
			result.Container.AddChild(titleLabel)

			// Roster list - will be populated in refreshRosterList()
			rosterList := builders.CreateSimpleStringList(builders.SimpleStringListConfig{
				Entries:       []string{},
				ScreenWidth:   400,
				ScreenHeight:  200,
				WidthPercent:  1.0,
				HeightPercent: 1.0,
			})
			result.Container.AddChild(rosterList)

			addUnitBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Add to Squad",
				OnClick: func() {
					sem.onAddUnitFromRoster()
				},
			})
			result.Container.AddChild(addUnitBtn)

			result.Custom["rosterList"] = rosterList

			return nil
		},
	})
}

