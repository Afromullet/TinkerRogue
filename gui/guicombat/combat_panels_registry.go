package guicombat

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guiartifacts"
	"game_main/gui/guiinspect"
	"game_main/gui/specs"
	"game_main/gui/widgetresources"
	"game_main/gui/widgets"
	"game_main/templates"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for combat mode
const (
	CombatPanelTurnOrder   framework.PanelType = "combat_turn_order"
	CombatPanelFactionInfo framework.PanelType = "combat_faction_info"
	CombatPanelSquadDetail framework.PanelType = "combat_squad_detail"
	CombatPanelLayerStatus framework.PanelType = "combat_layer_status"
	CombatPanelDebugMenu            framework.PanelType = "combat_debug_menu"
	CombatPanelMagicMenu            framework.PanelType = "combat_magic_menu"
	CombatPanelSpellSelection       framework.PanelType = "combat_spell_selection"
	CombatPanelArtifactSelection    framework.PanelType = "combat_artifact_selection"
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
	framework.RegisterPanel(CombatPanelDebugMenu, framework.PanelDescriptor{
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
	framework.RegisterPanel(CombatPanelMagicMenu, framework.PanelDescriptor{
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
	framework.RegisterPanel(CombatPanelTurnOrder, framework.PanelDescriptor{
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

	framework.RegisterPanel(CombatPanelFactionInfo, framework.PanelDescriptor{
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

	framework.RegisterPanel(CombatPanelSquadDetail, framework.PanelDescriptor{
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

	framework.RegisterPanel(CombatPanelLayerStatus, framework.PanelDescriptor{
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
	framework.RegisterPanel(CombatPanelSpellSelection, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cm := mode.(*CombatMode)
			layout := cm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatSpellPanelWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatSpellPanelHeight)

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

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			result.Container.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)

			// Mana label
			manaLabel := builders.CreateSmallLabel("Mana: 0/0")
			result.Container.AddChild(manaLabel)
			result.Custom["manaLabel"] = manaLabel

			// Spell list
			listWidth := panelWidth - 20
			listHeight := int(float64(layout.ScreenHeight) * specs.CombatSpellListHeight)

			spellList := builders.CreateListWithConfig(builders.ListConfig{
				Entries:    []interface{}{},
				MinWidth:   listWidth,
				MinHeight:  listHeight,
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
			})
			cachedList := widgets.NewCachedListWrapper(spellList)
			result.Container.AddChild(spellList)
			result.Custom["spellList"] = cachedList

			// Detail text area
			detailWidth := panelWidth - 20
			detailHeight := int(float64(layout.ScreenHeight) * specs.CombatSpellDetailHeight)
			detailArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  detailWidth,
				MinHeight: detailHeight,
				FontColor: color.White,
			})
			detailArea.SetText("Select a spell to view details")
			result.Container.AddChild(detailArea)
			result.Custom["detailArea"] = detailArea

			// Cast button
			castButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Cast",
				OnClick: func() {
					cm.spellPanel.OnCastClicked()
				},
			})
			castButton.GetWidget().Disabled = true
			result.Container.AddChild(castButton)
			result.Custom["castButton"] = castButton

			// Cancel button
			cancelButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Cancel (ESC)",
				OnClick: func() {
					cm.spellPanel.OnCancelClicked()
				},
			})
			result.Container.AddChild(cancelButton)

			// Hidden by default
			result.Container.GetWidget().Visibility = widget.Visibility_Hide

			// Register with sub-menu controller
			cm.subMenus.Register("spell", result.Container)

			return nil
		},
	})

	// Register artifact activation panel (right side, shown during artifact mode)
	framework.RegisterPanel(CombatPanelArtifactSelection, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cm := mode.(*CombatMode)
			layout := cm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatArtifactPanelWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatArtifactPanelHeight)

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

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			result.Container.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)

			// Artifact list
			listWidth := panelWidth - 20
			listHeight := int(float64(layout.ScreenHeight) * specs.CombatArtifactListHeight)

			artifactList := builders.CreateListWithConfig(builders.ListConfig{
				Entries:   []interface{}{},
				MinWidth:  listWidth,
				MinHeight: listHeight,
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
			})
			cachedList := widgets.NewCachedListWrapper(artifactList)
			result.Container.AddChild(artifactList)
			result.Custom["artifactList"] = cachedList

			// Detail text area — use raw TextArea in container for proper rendering,
			// wrap it for the controller's SetText convenience.
			detailWidth := panelWidth - 20
			detailHeight := int(float64(layout.ScreenHeight) * specs.CombatArtifactDetailHeight)
			rawDetailArea := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  detailWidth,
				MinHeight: detailHeight,
				FontColor: color.White,
			})
			rawDetailArea.SetText("Select an artifact to view details")
			result.Container.AddChild(rawDetailArea)
			result.Custom["detailArea"] = widgets.NewCachedTextAreaWrapper(rawDetailArea)

			// Activate button
			activateButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Activate",
				OnClick: func() {
					cm.artifactPanel.OnActivateClicked()
				},
			})
			activateButton.GetWidget().Disabled = true
			result.Container.AddChild(activateButton)
			result.Custom["activateButton"] = activateButton

			// Cancel button
			cancelButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Cancel (ESC)",
				OnClick: func() {
					cm.artifactPanel.OnCancelClicked()
				},
			})
			result.Container.AddChild(cancelButton)

			// Hidden by default
			result.Container.GetWidget().Visibility = widget.Visibility_Hide

			// Register with sub-menu controller
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
