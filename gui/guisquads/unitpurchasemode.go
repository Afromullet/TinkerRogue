package guisquads

import (
	"fmt"
	"game_main/common"
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
				// Show unit name with owned count
				roster := squads.GetPlayerRoster(upm.Context.PlayerData.PlayerEntityID, upm.Context.ECSManager)
				if roster != nil {
					entry, exists := roster.Units[template.Name]
					if exists {
						available := roster.GetAvailableCount(template.Name)
						return fmt.Sprintf("%s (Owned: %d, Available: %d)", template.Name, entry.TotalOwned, available)
					}
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

	// Show basic info
	cost := upm.getUnitCost(upm.selectedTemplate.Name)
	info := fmt.Sprintf("Unit: %s\nCost: %d Gold\n\nRole: %s\nSize: %dx%d",
		upm.selectedTemplate.Name,
		cost,
		upm.getRoleName(upm.selectedTemplate.Role),
		upm.selectedTemplate.GridWidth,
		upm.selectedTemplate.GridHeight)

	upm.detailTextArea.SetText(info)

	// Enable buttons
	upm.viewStatsButton.GetWidget().Disabled = false

	// Enable buy button only if player can afford
	resources := common.GetPlayerResources(upm.Context.PlayerData.PlayerEntityID, upm.Context.ECSManager)
	roster := squads.GetPlayerRoster(upm.Context.PlayerData.PlayerEntityID, upm.Context.ECSManager)
	canBuy := resources != nil && resources.CanAfford(cost) && roster != nil && roster.CanAddUnit()
	upm.buyButton.GetWidget().Disabled = !canBuy

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
	resources := common.GetPlayerResources(playerID, upm.Context.ECSManager)
	roster := squads.GetPlayerRoster(playerID, upm.Context.ECSManager)

	if resources == nil || roster == nil {
		fmt.Println("Error: Player resources or roster not found")
		return
	}

	cost := upm.getUnitCost(upm.selectedTemplate.Name)

	// Check if can afford and has space
	if !resources.CanAfford(cost) {
		fmt.Printf("Cannot afford unit: need %d gold, have %d\n", cost, resources.Gold)
		return
	}

	if !roster.CanAddUnit() {
		fmt.Println("Roster is full")
		return
	}

	// Create unit entity from template
	unitEntity, err := squads.CreateUnitEntity(upm.Context.ECSManager, *upm.selectedTemplate)
	if err != nil {
		fmt.Printf("Failed to create unit: %v\n", err)
		return
	}

	unitID := unitEntity.GetID()

	// Add to roster
	if err := roster.AddUnit(unitID, upm.selectedTemplate.Name); err != nil {
		fmt.Printf("Failed to add unit to roster: %v\n", err)
		// Clean up entity11
		upm.Context.ECSManager.World.DisposeEntities(unitEntity)
		return
	}

	// Spend gold
	if err := resources.SpendGold(cost); err != nil {
		fmt.Printf("Failed to spend gold: %v\n", err)
		// Rollback roster addition
		roster.RemoveUnit(unitID)
		upm.Context.ECSManager.World.DisposeEntities(unitEntity)
		return
	}

	fmt.Printf("Purchased unit: %s for %d gold\n", upm.selectedTemplate.Name, cost)

	// Refresh display
	upm.refreshResourceDisplay()
	upm.updateDetailPanel()
}

func (upm *UnitPurchaseMode) refreshResourceDisplay() {
	playerID := upm.Context.PlayerData.PlayerEntityID
	resources := common.GetPlayerResources(playerID, upm.Context.ECSManager)
	roster := squads.GetPlayerRoster(playerID, upm.Context.ECSManager)

	if resources != nil {
		upm.goldLabel.Label = fmt.Sprintf("Gold: %d", resources.Gold)
	}

	if roster != nil {
		current, max := roster.GetUnitCount()
		upm.rosterLabel.Label = fmt.Sprintf("Roster: %d/%d", current, max)
	}
}

func (upm *UnitPurchaseMode) getUnitCost(unitName string) int {
	// Simple cost formula based on unit name hash for now
	// TODO: Add cost to UnitTemplate or JSON data
	baseCost := 100
	for _, c := range unitName {
		baseCost += int(c) % 50
	}
	return baseCost
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
