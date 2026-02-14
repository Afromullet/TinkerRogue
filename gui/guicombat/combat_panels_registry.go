package guicombat

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
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
	CombatPanelCombatLog   framework.PanelType = "combat_log"
	CombatPanelLayerStatus framework.PanelType = "combat_layer_status"
	CombatPanelDebugMenu        framework.PanelType = "combat_debug_menu"
	CombatPanelSpellSelection   framework.PanelType = "combat_spell_selection"
)

// combatSubMenuController manages sub-menu visibility. Only one sub-menu can be open at a time.
type combatSubMenuController struct {
	menus  map[string]*widget.Container
	active string
}

func newCombatSubMenuController() *combatSubMenuController {
	return &combatSubMenuController{
		menus: make(map[string]*widget.Container),
	}
}

func (sc *combatSubMenuController) Register(name string, container *widget.Container) {
	sc.menus[name] = container
}

// Toggle returns a callback that toggles the named sub-menu.
// Opening one menu closes any other open menu.
func (sc *combatSubMenuController) Toggle(name string) func() {
	return func() {
		if sc.active == name {
			sc.menus[name].GetWidget().Visibility = widget.Visibility_Hide
			sc.active = ""
			return
		}
		sc.CloseAll()
		if c, ok := sc.menus[name]; ok {
			c.GetWidget().Visibility = widget.Visibility_Show
			sc.active = name
		}
	}
}

// Show opens the named sub-menu, closing any other open menu first.
func (sc *combatSubMenuController) Show(name string) {
	sc.CloseAll()
	if c, ok := sc.menus[name]; ok {
		c.GetWidget().Visibility = widget.Visibility_Show
		sc.active = name
	}
}

// IsActive returns true if the named sub-menu is currently open.
func (sc *combatSubMenuController) IsActive(name string) bool {
	return sc.active == name
}

func (sc *combatSubMenuController) CloseAll() {
	for _, c := range sc.menus {
		c.GetWidget().Visibility = widget.Visibility_Hide
	}
	sc.active = ""
}

// createCombatSubMenu creates a vertical sub-menu panel, registers it with the controller, and returns it.
func createCombatSubMenu(cm *CombatMode, name string, buttons []builders.ButtonConfig) *widget.Container {
	spacing := int(float64(cm.Layout.ScreenWidth) * specs.PaddingTight)
	subMenuBottomPad := int(float64(cm.Layout.ScreenHeight) * (specs.BottomButtonOffset + 0.15))
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

	framework.RegisterPanel(CombatPanelCombatLog, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatLogWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatLogHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout:     widget.NewAnchorLayout(),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			bottomOffset := int(float64(layout.ScreenHeight) * (specs.CombatActionButtonHeight + specs.BottomButtonOffset + specs.PaddingTight))
			result.Container.GetWidget().LayoutData = builders.AnchorEndEnd(rightPad, bottomOffset)

			// Create cached textarea
			textArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  panelWidth - 20,
				MinHeight: panelHeight - 20,
				FontColor: color.White,
			})
			textArea.SetText("Combat started!\n")
			result.Container.AddChild(textArea)

			// Store in Custom map for retrieval
			result.Custom["textArea"] = textArea

			return nil
		},
	})

	framework.RegisterPanel(CombatPanelLayerStatus, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.15)
			panelHeight := int(float64(layout.ScreenHeight) * 0.08)

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

			rightPad := int(float64(layout.ScreenWidth) * 0.01)
			topPad := int(float64(layout.ScreenHeight) * 0.01)
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
}

// GetCombatLogTextArea retrieves the combat log text area from panel registry
func GetCombatLogTextArea(panels *framework.PanelRegistry) *widgets.CachedTextAreaWrapper {
	if result := panels.Get(CombatPanelCombatLog); result != nil {
		if ta, ok := result.Custom["textArea"].(*widgets.CachedTextAreaWrapper); ok {
			return ta
		}
	}
	return nil
}
