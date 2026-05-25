package guicombat

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guiartifacts"
	combatpanels "game_main/gui/guicombat/combatbase"
	"game_main/gui/guiinspect"
	"game_main/gui/specs"
	"game_main/gui/widgetresources"
	"game_main/templates"

	"github.com/ebitenui/ebitenui/widget"
)

// createCombatSubMenu creates a vertical sub-menu panel, registers it with the controller, and returns it.
func createCombatSubMenu(cm *CombatMode, name string, buttons []builders.ButtonConfig) *widget.Container {
	spacing := int(float64(cm.Layout.ScreenWidth) * specs.PaddingTight)
	subMenuBottomPad := int(float64(cm.Layout.ScreenHeight) * (specs.BottomButtonOffset + specs.CombatSubMenuOffset))
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
	cm.subMenus.Register(name, panel)
	return panel
}

func init() {
	// Register debug sub-menu panel
	framework.RegisterPanel(combatpanels.CombatPanelDebugMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cm := mode.(*CombatMode)

			result.Container = createCombatSubMenu(cm, "debug", []builders.ButtonConfig{
				{Text: "Kill Squad", OnClick: func() {
					cm.inputHandler.EnterDebugKillMode()
					cm.subMenus.CloseAll()
				}},
			})
			return nil
		},
	})

	// Register magic sub-menu panel (groups Cast Spell + Artifact)
	framework.RegisterPanel(combatpanels.CombatPanelMagicMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cm := mode.(*CombatMode)
			result.Container = createCombatSubMenu(cm, "magic", []builders.ButtonConfig{
				{Text: "Cast Spell (S)", OnClick: func() {
					cm.subMenus.CloseAll()
					cm.handleSpellClick()
				}},
				{Text: "Artifact (D)", OnClick: func() {
					cm.subMenus.CloseAll()
					cm.handleArtifactClick()
				}},
			})
			return nil
		},
	})

	// Register all combat panels
	framework.RegisterPanel(combatpanels.CombatPanelTurnOrder, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatTurnOrderWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatTurnOrderHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(10),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			topPad := int(float64(layout.ScreenHeight) * specs.PaddingTight)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

			// Add label (stored in result for type-safe access)
			result.TextLabel = builders.CreateLargeLabel("Initializing combat...")
			result.Container.AddChild(result.TextLabel)

			return nil
		},
	})

	framework.RegisterPanel(combatpanels.CombatPanelFactionInfo, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatFactionInfoWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatFactionInfoHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			topPad := int(float64(layout.ScreenHeight) * specs.PaddingTight)
			result.Container.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topPad)

			result.TextLabel = builders.CreateSmallLabel("Faction Info")
			result.Container.AddChild(result.TextLabel)

			return nil
		},
	})

	framework.RegisterPanel(combatpanels.CombatPanelSquadDetail, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatSquadDetailWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatSquadDetailHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			topOffset := int(float64(layout.ScreenHeight) * (specs.CombatFactionInfoHeight + specs.PaddingTight))
			result.Container.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)

			result.TextLabel = builders.CreateSmallLabel("Select a squad\nto view details")
			result.Container.AddChild(result.TextLabel)

			return nil
		},
	})

	framework.RegisterPanel(combatpanels.CombatPanelLayerStatus, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatLayerStatusWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatLayerStatusHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(3),
					widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(5)),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingExtraSmall)
			topPad := int(float64(layout.ScreenHeight) * specs.PaddingExtraSmall)
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topPad)

			result.TextLabel = builders.CreateSmallLabel("")
			result.Container.AddChild(result.TextLabel)

			// Hide initially
			result.Container.GetWidget().Visibility = widget.Visibility_Hide

			return nil
		},
	})

	// Register spell selection panel (right side, shown during spell mode)
	framework.RegisterPanel(combatpanels.CombatPanelSpellSelection, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cm := mode.(*CombatMode)

			panel := builders.BuildSelectionPanel(cm.Layout, builders.SelectionPanelConfig{
				PanelWidthFrac:    specs.CombatSpellPanelWidth,
				PanelHeightFrac:   specs.CombatSpellPanelHeight,
				ListHeightFrac:    specs.CombatSpellListHeight,
				DetailHeightFrac:  specs.CombatSpellDetailHeight,
				StatusLabelText:   "Mana: 0/0",
				DetailPlaceholder: "Select a spell to view details",
				ActionButtonText:  "Cast",
				CancelButtonText:  "Cancel (ESC)",
				EntryLabelFunc: func(e interface{}) string {
					if spell, ok := e.(*templates.SpellDefinition); ok {
						return fmt.Sprintf("%s (%d MP)", spell.Name, spell.ManaCost)
					}
					return "???"
				},
				OnEntrySelected: func(e interface{}) {
					if spell, ok := e.(*templates.SpellDefinition); ok {
						cm.spellPanel.OnSpellSelected(spell)
					}
				},
				OnActionClicked: func() { cm.spellPanel.OnCastClicked() },
				OnCancelClicked: func() { cm.spellPanel.OnCancelClicked() },
			})

			result.Container = panel.Container
			result.Custom["manaLabel"] = panel.StatusLabel
			result.Custom["spellList"] = panel.List
			result.Custom["detailArea"] = panel.Detail
			result.Custom["castButton"] = panel.ActionButton

			cm.subMenus.Register("spell", result.Container)
			return nil
		},
	})

	// Register artifact activation panel (right side, shown during artifact mode)
	framework.RegisterPanel(combatpanels.CombatPanelArtifactSelection, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cm := mode.(*CombatMode)

			panel := builders.BuildSelectionPanel(cm.Layout, builders.SelectionPanelConfig{
				PanelWidthFrac:    specs.CombatArtifactPanelWidth,
				PanelHeightFrac:   specs.CombatArtifactPanelHeight,
				ListHeightFrac:    specs.CombatArtifactListHeight,
				DetailHeightFrac:  specs.CombatArtifactDetailHeight,
				DetailPlaceholder: "Select an artifact to view details",
				ActionButtonText:  "Activate",
				CancelButtonText:  "Cancel (ESC)",
				EntryLabelFunc: func(e interface{}) string {
					if opt, ok := e.(*guiartifacts.ArtifactOption); ok {
						chargeStr := "Ready"
						if !opt.Available {
							chargeStr = "Spent"
						}
						return fmt.Sprintf("%s [%s]", opt.Name, chargeStr)
					}
					return "???"
				},
				OnEntrySelected: func(e interface{}) {
					if opt, ok := e.(*guiartifacts.ArtifactOption); ok {
						cm.artifactPanel.OnArtifactSelected(opt)
					}
				},
				OnActionClicked: func() { cm.artifactPanel.OnActivateClicked() },
				OnCancelClicked: func() { cm.artifactPanel.OnCancelClicked() },
			})

			result.Container = panel.Container
			result.Custom["artifactList"] = panel.List
			result.Custom["detailArea"] = panel.Detail
			result.Custom["activateButton"] = panel.ActionButton

			cm.subMenus.Register("artifact", result.Container)
			return nil
		},
	})

	// Register inspect formation grid panel (delegates construction to guiinspect)
	framework.RegisterPanel(guiinspect.InspectPanelType, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cm := mode.(*CombatMode)
			guiinspect.BuildPanel(result, cm.PanelBuilders)
			cm.subMenus.Register("inspect", result.Container)
			return nil
		},
	})
}
