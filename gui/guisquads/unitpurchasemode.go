package guisquads

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guimodes"

	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/tactical/squadcommands"
	"game_main/tactical/squads"
	"game_main/tactical/squadservices"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// UnitPurchaseMode allows players to buy units and add them to their roster
type UnitPurchaseMode struct {
	framework.BaseMode // Embed common mode infrastructure

	purchaseService *squadservices.UnitPurchaseService
	unitList        *widgets.CachedListWrapper
	detailPanel     *widget.Container
	detailTextArea  *widgets.CachedTextAreaWrapper // Cached for performance
	statsTextArea   *widgets.CachedTextAreaWrapper // Cached for performance
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
	return mode
}

func (upm *UnitPurchaseMode) Initialize(ctx *framework.UIContext) error {
	// Create purchase service first (needed by UI builders)
	upm.purchaseService = squadservices.NewUnitPurchaseService(ctx.ECSManager)

	return framework.NewModeBuilder(&upm.BaseMode, framework.ModeConfig{
		ModeName:   "unit_purchase",
		ReturnMode: "squad_management",

		Panels: []framework.ModePanelConfig{
			{CustomBuild: upm.buildResourceDisplay},
			{CustomBuild: upm.buildUnitList},
			{CustomBuild: upm.buildDetailPanel},
			{CustomBuild: upm.buildActionButtons},
		},

		Commands:  true,
		OnRefresh: upm.refreshAfterUndoRedo,
	}).Build(ctx)
}

func (upm *UnitPurchaseMode) buildUnitList() *widget.Container {
	// Left side unit list (35% width to prevent overlap with 25% top-center resource display)
	listWidth := int(float64(upm.Layout.ScreenWidth) * specs.UnitPurchaseListWidth)
	listHeight := int(float64(upm.Layout.ScreenHeight) * specs.UnitPurchaseListHeight)

	baseList := builders.CreateListWithConfig(builders.ListConfig{
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
	})

	// Wrap with caching for performance (~90% render reduction for static lists)
	upm.unitList = widgets.NewCachedListWrapper(baseList)

	// Position below resource panel using Start-Start anchor (left-top)
	leftPad := int(float64(upm.Layout.ScreenWidth) * specs.PaddingStandard)
	topOffset := int(float64(upm.Layout.ScreenHeight) * (specs.UnitPurchaseResourceHeight + specs.PaddingStandard*2))

	// Wrap in container with LayoutData
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(builders.AnchorStartStart(leftPad, topOffset))),
	)
	// Add the underlying list to maintain interaction functionality
	container.AddChild(baseList)
	return container
}

func (upm *UnitPurchaseMode) buildDetailPanel() *widget.Container {
	// Right side detail panel (35% width to prevent overlap with 25% top-center resource display)
	panelWidth := int(float64(upm.Layout.ScreenWidth) * 0.35)
	panelHeight := int(float64(upm.Layout.ScreenHeight) * 0.6)

	upm.detailPanel = builders.CreateStaticPanel(builders.ContainerConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(upm.Layout, specs.PaddingTight)),
		),
	})

	rightPad := int(float64(upm.Layout.ScreenWidth) * specs.PaddingStandard)
	upm.detailPanel.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)

	// Basic info text area (cached - only re-renders when selection changes)
	upm.detailTextArea = builders.CreateCachedTextArea(builders.TextAreaConfig{
		MinWidth:  panelWidth - 30,
		MinHeight: 100,
		FontColor: color.White,
	})
	upm.detailTextArea.SetText("Select a unit to view details") // SetText calls MarkDirty() internally
	upm.detailPanel.AddChild(upm.detailTextArea)

	// View Stats button
	upm.viewStatsButton = builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "View Stats",
		OnClick: func() {
			upm.showStats()
		},
	})
	upm.viewStatsButton.GetWidget().Disabled = true
	upm.detailPanel.AddChild(upm.viewStatsButton)

	// Stats text area (hidden by default, cached - only re-renders when stats viewed)
	upm.statsTextArea = builders.CreateCachedTextArea(builders.TextAreaConfig{
		MinWidth:  panelWidth - 30,
		MinHeight: 300,
		FontColor: color.White,
	})
	upm.statsTextArea.GetWidget().Visibility = widget.Visibility_Hide
	upm.detailPanel.AddChild(upm.statsTextArea)

	return upm.detailPanel
}

func (upm *UnitPurchaseMode) buildResourceDisplay() *widget.Container {
	// Top-center resource display (responsive sizing)
	panelWidth := int(float64(upm.Layout.ScreenWidth) * 0.25)
	panelHeight := int(float64(upm.Layout.ScreenHeight) * 0.08)

	resourcePanel := builders.CreateStaticPanel(builders.ContainerConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(upm.Layout, specs.PaddingTight)),
		),
	})

	topPad := int(float64(upm.Layout.ScreenHeight) * 0.02)
	resourcePanel.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

	// Gold label
	upm.goldLabel = builders.CreateSmallLabel("Gold: 0")
	resourcePanel.AddChild(upm.goldLabel)

	// Roster label
	upm.rosterLabel = builders.CreateSmallLabel("Roster: 0/0")
	resourcePanel.AddChild(upm.rosterLabel)

	return resourcePanel
}

func (upm *UnitPurchaseMode) buildActionButtons() *widget.Container {
	// Create UI factory
	panelFactory := guimodes.NewExplorationPanelFactory(upm.PanelBuilders, upm.Layout)

	// Create button callbacks (no panel wrapper - like combat mode)
	actionButtonContainer := panelFactory.CreateUnitPurchaseActionButtons(
		// Buy Unit
		func() {
			upm.purchaseUnit()
		},
		// Undo
		func() {
			upm.CommandHistory.Undo()
		},
		// Redo
		func() {
			upm.CommandHistory.Redo()
		},
		// Back
		func() {
			if mode, exists := upm.ModeManager.GetMode("squad_management"); exists {
				upm.ModeManager.RequestTransition(mode, "Back button pressed")
			}
		},
	)

	// Store buy button reference for enable/disable control (update after creation)
	// Note: We need to reference the button from the container's children
	if children := actionButtonContainer.Children(); len(children) > 0 {
		if btn, ok := children[0].(*widget.Button); ok {
			upm.buyButton = btn
			upm.buyButton.GetWidget().Disabled = true
		}
	}

	return actionButtonContainer
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
