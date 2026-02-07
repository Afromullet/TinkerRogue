package guisquads

import (
	"fmt"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/tactical/squadcommands"
	"game_main/tactical/squads"
	"game_main/tactical/squadservices"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// UnitPurchaseMode allows players to buy units and add them to their roster
type UnitPurchaseMode struct {
	framework.BaseMode // Embed common mode infrastructure

	purchaseService *squadservices.UnitPurchaseService

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	unitList        *widgets.CachedListWrapper
	detailTextArea  *widgets.CachedTextAreaWrapper
	statsTextArea   *widgets.CachedTextAreaWrapper
	goldLabel       *widget.Text
	rosterLabel     *widget.Text
	buyButton       *widget.Button
	viewStatsButton *widget.Button

	selectedTemplate *squads.UnitTemplate
	selectedIndex    int
}

func NewUnitPurchaseMode(modeManager *framework.UIModeManager) *UnitPurchaseMode {
	mode := &UnitPurchaseMode{
		selectedIndex: -1,
	}
	mode.SetModeName("unit_purchase")
	mode.SetReturnMode("squad_management") // ESC returns to squad management
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (upm *UnitPurchaseMode) Initialize(ctx *framework.UIContext) error {
	// Create purchase service first (needed by UI builders)
	upm.purchaseService = squadservices.NewUnitPurchaseService(ctx.ECSManager)

	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&upm.BaseMode, framework.ModeConfig{
		ModeName:   "unit_purchase",
		ReturnMode: "squad_management",
		Commands:   true,
		OnRefresh:  upm.refreshAfterUndoRedo,
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := upm.BuildPanels(
		UnitPurchasePanelResourceDisplay,
		UnitPurchasePanelUnitList,
		UnitPurchasePanelDetailPanel,
		UnitPurchasePanelActionButtons,
	); err != nil {
		return err
	}

	// Initialize widget references from registry
	upm.initializeWidgetReferences()

	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (upm *UnitPurchaseMode) initializeWidgetReferences() {
	// Resource display
	upm.goldLabel = framework.GetPanelWidget[*widget.Text](upm.Panels, UnitPurchasePanelResourceDisplay, "goldLabel")
	upm.rosterLabel = framework.GetPanelWidget[*widget.Text](upm.Panels, UnitPurchasePanelResourceDisplay, "rosterLabel")

	// Unit list
	upm.unitList = framework.GetPanelWidget[*widgets.CachedListWrapper](upm.Panels, UnitPurchasePanelUnitList, "unitList")

	// Detail panel
	upm.detailTextArea = framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](upm.Panels, UnitPurchasePanelDetailPanel, "detailTextArea")
	upm.statsTextArea = framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](upm.Panels, UnitPurchasePanelDetailPanel, "statsTextArea")
	upm.viewStatsButton = framework.GetPanelWidget[*widget.Button](upm.Panels, UnitPurchasePanelDetailPanel, "viewStatsButton")

	// Buy button
	upm.buyButton = framework.GetPanelWidget[*widget.Button](upm.Panels, UnitPurchasePanelActionButtons, "buyButton")
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
	upm.unitList.GetList().SetEntries(entries)
	upm.unitList.MarkDirty() // Trigger re-render with updated entries
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

func (upm *UnitPurchaseMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Unit Purchase Mode")

	// Populate unit list with all available templates
	entries := make([]interface{}, 0, len(squads.Units))
	for i := range squads.Units {
		entries = append(entries, &squads.Units[i])
	}
	upm.unitList.GetList().SetEntries(entries)
	upm.unitList.MarkDirty() // Trigger re-render with updated entries

	// Refresh resource display
	upm.refreshResourceDisplay()

	// Clear selection
	upm.selectedTemplate = nil
	upm.selectedIndex = -1
	upm.updateDetailPanel()

	return nil
}

func (upm *UnitPurchaseMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Unit Purchase Mode")
	return nil
}

func (upm *UnitPurchaseMode) Update(deltaTime float64) error {
	return nil
}

func (upm *UnitPurchaseMode) Render(screen *ebiten.Image) {
	// No custom rendering
}

func (upm *UnitPurchaseMode) HandleInput(inputState *framework.InputState) bool {
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
