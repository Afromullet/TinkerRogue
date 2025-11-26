package guisquads

import (
	"fmt"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/widgets"
	"game_main/squads"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// UnitPurchaseMode allows players to buy units and add them to their roster
type UnitPurchaseMode struct {
	gui.BaseMode // Embed common mode infrastructure

	purchaseService  *squads.UnitPurchaseService
	unitList         *widget.List
	detailPanel      *widget.Container
	detailTextArea   *widget.TextArea
	statsTextArea    *widget.TextArea
	goldLabel        *widget.Text
	rosterLabel      *widget.Text
	buyButton        *widget.Button
	viewStatsButton  *widget.Button
	selectedTemplate *squads.UnitTemplate
	selectedIndex    int
}

func NewUnitPurchaseMode(modeManager *core.UIModeManager) *UnitPurchaseMode {
	mode := &UnitPurchaseMode{
		selectedIndex: -1,
	}
	mode.SetModeName("unit_purchase")
	mode.ModeManager = modeManager
	return mode
}

func (upm *UnitPurchaseMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	upm.InitializeBase(ctx)

	// Create purchase service
	upm.purchaseService = squads.NewUnitPurchaseService(ctx.ECSManager)

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
	// Left side unit list (40% width)
	listWidth := int(float64(upm.Layout.ScreenWidth) * 0.4)
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
	// Right side detail panel
	panelWidth := int(float64(upm.Layout.ScreenWidth) * 0.45)
	panelHeight := int(float64(upm.Layout.ScreenHeight) * 0.7)

	upm.detailPanel = widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 15, Right: 15, Top: 15, Bottom: 15,
			}),
		),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Right: int(float64(upm.Layout.ScreenWidth) * widgets.PaddingStandard),
				Top:   int(float64(upm.Layout.ScreenHeight) * 0.1),
			},
		},
	})

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
	// Top-right resource display
	resourcePanel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:  300,
		MinHeight: 100,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 15, Right: 15, Top: 15, Bottom: 15,
			}),
		),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			Padding: widget.Insets{
				Right: int(float64(upm.Layout.ScreenWidth) * widgets.PaddingStandard),
				Top:   20,
			},
		},
	})

	// Gold label
	upm.goldLabel = widgets.CreateSmallLabel("Gold: 0")
	resourcePanel.AddChild(upm.goldLabel)

	// Roster label
	upm.rosterLabel = widgets.CreateSmallLabel("Roster: 0/0")
	resourcePanel.AddChild(upm.rosterLabel)

	upm.RootContainer.AddChild(resourcePanel)
}

func (upm *UnitPurchaseMode) buildActionButtons() {
	actionButtonContainer := gui.CreateBottomCenterButtonContainer(upm.PanelBuilders)

	// Buy button
	upm.buyButton = widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Buy Unit",
		OnClick: func() {
			upm.purchaseUnit()
		},
	})
	upm.buyButton.GetWidget().Disabled = true
	actionButtonContainer.AddChild(upm.buyButton)

	// Close button
	closeBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Back (ESC)",
		OnClick: func() {
			if squadMgmtMode, exists := upm.ModeManager.GetMode("squad_management"); exists {
				upm.ModeManager.RequestTransition(squadMgmtMode, "Close Unit Purchase")
			}
		},
	})
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
		"Target Mode: %s\n"+
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
		upm.getTargetModeName(upm.selectedTemplate.TargetMode),
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

	// Use service for entire transaction - handles validation, creation, and rollback atomically
	result := upm.purchaseService.PurchaseUnit(playerID, *upm.selectedTemplate)

	if !result.Success {
		// Display error message to user
		fmt.Printf("Purchase failed: %s\n", result.Error)
		return
	}

	// Success - display confirmation
	fmt.Printf("Purchased unit: %s for %d gold\n", result.UnitName, result.CostPaid)
	fmt.Printf("Remaining gold: %d, Roster: %d/%d\n", result.RemainingGold, result.RosterCount, result.RosterCapacity)

	// Refresh UI display
	upm.refreshResourceDisplay()
	upm.updateDetailPanel()
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

func (upm *UnitPurchaseMode) getTargetModeName(mode squads.TargetMode) string {
	switch mode {
	case squads.TargetModeRowBased:
		return "Row-based"
	case squads.TargetModeCellBased:
		return "Cell-based"
	default:
		return "Unknown"
	}
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
	// Handle common input (ESC key)
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if squadMgmtMode, exists := upm.ModeManager.GetMode("squad_management"); exists {
			upm.ModeManager.RequestTransition(squadMgmtMode, "Close Unit Purchase")
			return true
		}
	}

	return false
}
