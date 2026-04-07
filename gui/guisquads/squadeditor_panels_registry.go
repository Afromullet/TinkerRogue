package guisquads

import (
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/tactical/powers/perks"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for squad editor mode
const (
	SquadEditorPanelSquadSelector framework.PanelType = "squadeditor_squad_selector"
	SquadEditorPanelGridEditor    framework.PanelType = "squadeditor_grid_editor"
	SquadEditorPanelUnitList      framework.PanelType = "squadeditor_unit_list"
	SquadEditorPanelRoster        framework.PanelType = "squadeditor_roster"
	SquadEditorPanelPerks         framework.PanelType = "squadeditor_perks"
	SquadEditorPanelSquadInfoMenu    framework.PanelType = "squadeditor_squad_info_menu"
	SquadEditorPanelPatternsMenu    framework.PanelType = "squadeditor_patterns_menu"
)

// createSquadEditorSubMenu creates a vertical sub-menu panel positioned above the action bar.
// Follows the same pattern as combat mode's createCombatSubMenu.
func createSquadEditorSubMenu(sem *SquadEditorMode, name string, buttons []builders.ButtonConfig) *widget.Container {
	spacing := int(float64(sem.Layout.ScreenWidth) * specs.PaddingTight)
	subMenuBottomPad := int(float64(sem.Layout.ScreenHeight) * (specs.BottomButtonOffset + specs.SquadEditorSubMenuOffset))
	anchorLayout := builders.AnchorCenterEnd(subMenuBottomPad)

	panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(spacing),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(10)),
		),
		LayoutData: anchorLayout,
	})
	for _, btn := range buttons {
		panel.AddChild(builders.CreateButtonWithConfig(btn))
	}
	panel.GetWidget().Visibility = widget.Visibility_Hide
	sem.subMenus.Register(name, panel)
	return panel
}

func init() {
	// Register squad info sub-menu (groups Units, Roster, Perks)
	framework.RegisterPanel(SquadEditorPanelSquadInfoMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			result.Container = createSquadEditorSubMenu(sem, "squad_info", []builders.ButtonConfig{
				{Text: "Units (U)", OnClick: func() {
					sem.subMenus.CloseAll()
					sem.subMenus.Toggle("units")()
				}},
				{Text: "Roster (R)", OnClick: func() {
					sem.subMenus.CloseAll()
					sem.subMenus.Toggle("roster")()
				}},
				{Text: "Perks (K)", OnClick: func() {
					sem.subMenus.CloseAll()
					sem.subMenus.Toggle("perks")()
					if sem.subMenus.IsActive("perks") {
						sem.perkPanel.refreshPerkPanel()
					}
				}},
			})
			return nil
		},
	})

	// Register patterns sub-menu (groups Attack Pattern + Support Pattern)
	framework.RegisterPanel(SquadEditorPanelPatternsMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			result.Container = createSquadEditorSubMenu(sem, "patterns", []builders.ButtonConfig{
				{Text: "Atk Pattern (V)", OnClick: func() {
					sem.subMenus.CloseAll()
					sem.toggleAttackPattern()
				}},
				{Text: "Support Pattern (B)", OnClick: func() {
					sem.subMenus.CloseAll()
					sem.toggleSupportPattern()
				}},
			})
			return nil
		},
	})

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
			squadList := CreateSquadList(SquadListConfig{
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

			// Support pattern label (hidden by default)
			supportLabel := builders.CreateSmallLabel("Support Pattern")
			supportLabel.GetWidget().Visibility = widget.Visibility_Hide
			wrapper.AddChild(supportLabel)

			// Support pattern grid (read-only, hidden by default)
			supportGridContainer, supportGridCells := sem.PanelBuilders.BuildGridEditor(builders.GridEditorConfig{})
			supportGridContainer.GetWidget().Visibility = widget.Visibility_Hide
			wrapper.AddChild(supportGridContainer)

			result.Container = wrapper
			result.Custom["gridCells"] = gridCells
			result.Custom["attackLabel"] = attackLabel
			result.Custom["attackGridCells"] = attackGridCells
			result.Custom["attackGridContainer"] = attackGridContainer
			result.Custom["supportLabel"] = supportLabel
			result.Custom["supportGridCells"] = supportGridCells
			result.Custom["supportGridContainer"] = supportGridContainer

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

			unitList := CreateUnitList(UnitListConfig{
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

	// Register perk panel (right side, hidden by default, managed by SubMenuController)
	framework.RegisterPanel(SquadEditorPanelPerks, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			sem := mode.(*SquadEditorMode)
			layout := sem.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorPerkPanelWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorPerkPanelHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  panelWidth,
				MinHeight: panelHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * specs.PaddingStandard)
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topOffset)

			// Slot count header
			slotLabel := builders.CreateSmallLabel("Perks (0/3)")
			result.Container.AddChild(slotLabel)

			// Equipped perks section
			result.Container.AddChild(builders.CreateSmallLabel("Equipped:"))

			equippedList := builders.CreateListWithConfig(builders.ListConfig{
				Entries:   []interface{}{},
				MinWidth:  panelWidth - 20,
				MinHeight: 120,
				EntryLabelFunc: func(e interface{}) string {
					def := e.(*perks.PerkDefinition)
					return def.Name
				},
				OnEntrySelected: func(e interface{}) {
					if sem.perkPanel != nil {
						sem.perkPanel.onEquippedSelected(e.(*perks.PerkDefinition))
					}
				},
			})
			result.Container.AddChild(equippedList)

			unequipBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Unequip",
				OnClick: func() {
					if sem.perkPanel != nil {
						sem.perkPanel.onUnequipClicked()
					}
				},
			})
			unequipBtn.GetWidget().Disabled = true
			result.Container.AddChild(unequipBtn)

			// Available perks section
			result.Container.AddChild(builders.CreateSmallLabel("Available:"))

			availableList := builders.CreateListWithConfig(builders.ListConfig{
				Entries:   []interface{}{},
				MinWidth:  panelWidth - 20,
				MinHeight: 150,
				EntryLabelFunc: func(e interface{}) string {
					def := e.(*perks.PerkDefinition)
					return def.Name
				},
				OnEntrySelected: func(e interface{}) {
					if sem.perkPanel != nil {
						sem.perkPanel.onAvailableSelected(e.(*perks.PerkDefinition))
					}
				},
			})
			result.Container.AddChild(availableList)

			equipBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Equip",
				OnClick: func() {
					if sem.perkPanel != nil {
						sem.perkPanel.onEquipClicked()
					}
				},
			})
			equipBtn.GetWidget().Disabled = true
			result.Container.AddChild(equipBtn)

			// Detail area
			detailArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  panelWidth - 20,
				MinHeight: 120,
				FontColor: color.White,
			})
			detailArea.SetText("Select a perk to view details.")
			result.Container.AddChild(detailArea)

			// Hidden by default, registered with SubMenuController
			result.Container.GetWidget().Visibility = widget.Visibility_Hide
			sem.subMenus.Register("perks", result.Container)

			result.Custom["equippedList"] = equippedList
			result.Custom["availableList"] = availableList
			result.Custom["detailArea"] = detailArea
			result.Custom["equipBtn"] = equipBtn
			result.Custom["unequipBtn"] = unequipBtn
			result.Custom["slotLabel"] = slotLabel

			return nil
		},
	})
}
