package guisquads

import (
	"fmt"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/widgets"
	"game_main/squads"
	"game_main/squads/squadcommands"
	"game_main/squads/squadservices"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// UnitPurchaseMode allows players to buy units and add them to their roster
type UnitPurchaseMode struct {
	gui.BaseMode // Embed common mode infrastructure

	purchaseService *squadservices.UnitPurchaseService
	unitList        *widget.List
	detailPanel     *widget.Container
	detailTextArea  *widget.TextArea
	statsTextArea   *widget.TextArea
	goldLabel       *widget.Text
	rosterLabel     *widget.Text
	buyButton       *widget.Button
	viewStatsButton *widget.Button

	selectedTemplate *squads.UnitTemplate
	selectedIndex    int
}

func NewUnitPurchaseMode(modeManager *core.UIModeManager) *UnitPurchaseMode {
	mode := &UnitPurchaseMode{
		selectedIndex: -1,
	}
	mode.SetModeName("unit_purchase")
	mode.SetReturnMode("squad_management") // ESC returns to squad management
	mode.ModeManager = modeManager
	return mode
}

func (upm *UnitPurchaseMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	upm.InitializeBase(ctx)

	// Create purchase service
	upm.purchaseService = squadservices.NewUnitPurchaseService(ctx.ECSManager)

	// Initialize command history with refresh callback
	upm.InitializeCommandHistory(upm.refreshAfterUndoRedo)

	// Build unit list (left side)
	upm.buildUnitList()

	// Build detail panel (right side)
	upm.buildDetailPanel()

	// Build action buttons (bottom-center)
	upm.buildActionButtons()

	// Build resource display (top-right)
	upm.buildResourceDisplay()

	return nil
}

func (upm *UnitPurchaseMode) buildUnitList() {
	// Left side unit list (35% width to prevent overlap with 25% top-center resource display)
	listWidth := int(float64(upm.Layout.ScreenWidth) * 0.35)
	listHeight := int(float64(upm.Layout.ScreenHeight) * 0.7)

	upm.unitList = widgets.CreateListWithConfig(widgets.ListConfig{
		Entries:   []interface{}{}, // Will be populated in Enter
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			if template, ok := e.(*squads.UnitTemplate); ok {
				// Use service to get owned count
				totalOwned, available := upm.purchaseService.GetUnitOwnedCount(
					upm.Context.PlayerData.PlayerEntityID,
					template.Name,
				)
				if totalOwned > 0 {
					return fmt.Sprintf("%s (Owned: %d, Available: %d)", template.Name, totalOwned, available)
				}
				return fmt.Sprintf("%s (Owned: 0)", template.Name)
			}
			return fmt.Sprintf("%v", e)
		},
		OnEntrySelected: func(selectedEntry interface{}) {
			if template, ok := selectedEntry.(*squads.UnitTemplate); ok {
				upm.selectedTemplate = template
				upm.updateDetailPanel()
			}
		},
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Left: int(float64(upm.Layout.ScreenWidth) * widgets.PaddingStandard),
				Top:  int(float64(upm.Layout.ScreenHeight) * 0.1),
			},
		},
	})

	upm.RootContainer.AddChild(upm.unitList)
}

func (upm *UnitPurchaseMode) buildDetailPanel() {
	// Right side detail panel (35% width to prevent overlap with 25% top-center resource display)
	panelWidth := int(float64(upm.Layout.ScreenWidth) * 0.35)
	panelHeight := int(float64(upm.Layout.ScreenHeight) * 0.6)

	upm.detailPanel = widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(upm.Layout, widgets.PaddingTight)),
		),
	})

	rightPad := int(float64(upm.Layout.ScreenWidth) * widgets.PaddingStandard)
	upm.detailPanel.GetWidget().LayoutData = gui.AnchorEndCenter(rightPad)

	// Basic info text area
	upm.detailTextArea = widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
		MinWidth:  panelWidth - 30,
		MinHeight: 100,
		FontColor: color.White,
	})
	upm.detailTextArea.SetText("Select a unit to view details")
	upm.detailPanel.AddChild(upm.detailTextArea)

	// View Stats button
	upm.viewStatsButton = widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "View Stats",
		OnClick: func() {
			upm.showStats()
		},
	})
	upm.viewStatsButton.GetWidget().Disabled = true
	upm.detailPanel.AddChild(upm.viewStatsButton)

	// Stats text area (hidden by default)
	upm.statsTextArea = widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
		MinWidth:  panelWidth - 30,
		MinHeight: 300,
		FontColor: color.White,
	})
	upm.statsTextArea.GetWidget().Visibility = widget.Visibility_Hide
	upm.detailPanel.AddChild(upm.statsTextArea)

	upm.RootContainer.AddChild(upm.detailPanel)
}

func (upm *UnitPurchaseMode) buildResourceDisplay() {
	// Top-center resource display (responsive sizing)
	panelWidth := int(float64(upm.Layout.ScreenWidth) * 0.25)
	panelHeight := int(float64(upm.Layout.ScreenHeight) * 0.08)

	resourcePanel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(upm.Layout, widgets.PaddingTight)),
		),
	})

	topPad := int(float64(upm.Layout.ScreenHeight) * 0.02)
	resourcePanel.GetWidget().LayoutData = gui.AnchorCenterStart(topPad)

	// Gold label
	upm.goldLabel = widgets.CreateSmallLabel("Gold: 0")
	resourcePanel.AddChild(upm.goldLabel)

	// Roster label
	upm.rosterLabel = widgets.CreateSmallLabel("Roster: 0/0")
	resourcePanel.AddChild(upm.rosterLabel)

	upm.RootContainer.AddChild(resourcePanel)
}

func (upm *UnitPurchaseMode) buildActionButtons() {
	// Create positioned container using helper (reduces layout boilerplate)
	buttonSpecs := []widgets.ButtonSpec{
		{
			Text: "Buy Unit",
			OnClick: func() {
				upm.purchaseUnit()
			},
		},
	}
	actionButtonContainer := gui.CreateActionButtonGroup(upm.PanelBuilders, widgets.BottomCenter(), buttonSpecs)

	// Store reference to buy button for later enable/disable control
	// (Note: Container doesn't expose children directly, so we keep our reference)
	upm.buyButton = widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Buy Unit",
		OnClick: func() {
			upm.purchaseUnit()
		},
	})
	upm.buyButton.GetWidget().Disabled = true

	// Clear the helper-created buttons and add our managed buttons
	actionButtonContainer.RemoveChildren()
	actionButtonContainer.AddChild(upm.buyButton)

	// Undo/Redo buttons from CommandHistory
	actionButtonContainer.AddChild(upm.CommandHistory.CreateUndoButton())
	actionButtonContainer.AddChild(upm.CommandHistory.CreateRedoButton())

	// Close button using standard pattern
	closeBtn := gui.CreateCloseButton(upm.ModeManager, "squad_management", "Back (ESC)")
	actionButtonContainer.AddChild(closeBtn)

	upm.RootContainer.AddChild(actionButtonContainer)
}

func (upm *UnitPurchaseMode) updateDetailPanel() {
	if upm.selectedTemplate == nil {
		upm.detailTextArea.SetText("Select a unit to view details")
		upm.viewStatsButton.GetWidget().Disabled = true
		upm.buyButton.GetWidget().Disabled = true
		return
	}

	// Use service to get cost
	cost := upm.purchaseService.GetUnitCost(*upm.selectedTemplate)
	info := fmt.Sprintf("Unit: %s\nCost: %d Gold\n\nRole: %s\nSize: %dx%d",
		upm.selectedTemplate.Name,
		cost,
		upm.getRoleName(upm.selectedTemplate.Role),
		upm.selectedTemplate.GridWidth,
		upm.selectedTemplate.GridHeight)

	upm.detailTextArea.SetText(info)

	// Enable buttons
	upm.viewStatsButton.GetWidget().Disabled = false

	// Use service to validate if player can purchase
	validation := upm.purchaseService.CanPurchaseUnit(
		upm.Context.PlayerData.PlayerEntityID,
		*upm.selectedTemplate,
	)
	upm.buyButton.GetWidget().Disabled = !validation.CanPurchase

	// Hide stats when selection changes
	upm.statsTextArea.GetWidget().Visibility = widget.Visibility_Hide
}

func (upm *UnitPurchaseMode) showStats() {
	if upm.selectedTemplate == nil {
		return
	}

	attr := upm.selectedTemplate.Attributes
	stats := fmt.Sprintf("=== STATS ===\n"+
		"HP: %d\n"+
		"Strength: %d\n"+
		"Dexterity: %d\n"+
		"Magic: %d\n"+
		"Leadership: %d\n"+
		"Armor: %d\n"+
		"Weapon: %d\n\n"+
		"=== COMBAT ===\n"+
		"Attack Range: %d\n"+
		"Movement Speed: %d\n"+
		"Target Cells: %d\n"+
		"Cover Value: %.2f\n"+
		"Cover Range: %d",
		attr.GetMaxHealth(),
		attr.Strength,
		attr.Dexterity,
		attr.Magic,
		attr.Leadership,
		attr.Armor,
		attr.Weapon,
		upm.selectedTemplate.AttackRange,
		upm.selectedTemplate.MovementSpeed,
		len(upm.selectedTemplate.TargetCells),
		upm.selectedTemplate.CoverValue,
		upm.selectedTemplate.CoverRange)

	upm.statsTextArea.SetText(stats)
	upm.statsTextArea.GetWidget().Visibility = widget.Visibility_Show
}

func (upm *UnitPurchaseMode) purchaseUnit() {
	if upm.selectedTemplate == nil {
		return
	}

	playerID := upm.Context.PlayerData.PlayerEntityID

	// Create and execute purchase command
	cmd := squadcommands.NewPurchaseUnitCommand(
		upm.Queries.ECSManager,
		upm.purchaseService,
		playerID,
		*upm.selectedTemplate,
	)

	upm.CommandHistory.Execute(cmd)
}

func (upm *UnitPurchaseMode) refreshUnitList() {
	// Repopulate unit list to update owned/available counts
	entries := make([]interface{}, 0, len(squads.Units))
	for i := range squads.Units {
		entries = append(entries, &squads.Units[i])
	}
	upm.unitList.SetEntries(entries)
}

func (upm *UnitPurchaseMode) refreshResourceDisplay() {
	playerID := upm.Context.PlayerData.PlayerEntityID

	// Use service to get player purchase info
	info := upm.purchaseService.GetPlayerPurchaseInfo(playerID)

	upm.goldLabel.Label = fmt.Sprintf("Gold: %d", info.Gold)
	upm.rosterLabel.Label = fmt.Sprintf("Roster: %d/%d", info.RosterCount, info.RosterCapacity)
}

func (upm *UnitPurchaseMode) getRoleName(role squads.UnitRole) string {
	switch role {
	case squads.RoleTank:
		return "Tank"
	case squads.RoleDPS:
		return "DPS"
	case squads.RoleSupport:
		return "Support"
	default:
		return "Unknown"
	}
}


// refreshAfterUndoRedo is called after successful undo/redo operations
func (upm *UnitPurchaseMode) refreshAfterUndoRedo() {
	upm.refreshUnitList()
	upm.refreshResourceDisplay()
	upm.updateDetailPanel()
}


func (upm *UnitPurchaseMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Unit Purchase Mode")

	// Populate unit list with all available templates
	entries := make([]interface{}, 0, len(squads.Units))
	for i := range squads.Units {
		entries = append(entries, &squads.Units[i])
	}
	upm.unitList.SetEntries(entries)

	// Refresh resource display
	upm.refreshResourceDisplay()

	// Clear selection
	upm.selectedTemplate = nil
	upm.selectedIndex = -1
	upm.updateDetailPanel()

	return nil
}

func (upm *UnitPurchaseMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Unit Purchase Mode")
	return nil
}

func (upm *UnitPurchaseMode) Update(deltaTime float64) error {
	return nil
}

func (upm *UnitPurchaseMode) Render(screen *ebiten.Image) {
	// No custom rendering
}

func (upm *UnitPurchaseMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input first (ESC key, registered hotkeys)
	if upm.HandleCommonInput(inputState) {
		return true
	}

	// Handle undo/redo input (Ctrl+Z, Ctrl+Y)
	if upm.CommandHistory.HandleInput(inputState) {
		return true
	}

	return false
}
