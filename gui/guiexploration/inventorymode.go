package guiexploration

import (
	"fmt"

	"game_main/gear"
	"game_main/gui/framework"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// InventoryMode provides full-screen inventory browsing and management
type InventoryMode struct {
	framework.BaseMode // Embed common mode infrastructure

	inventoryService *gear.InventoryService

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	itemList          *widget.List
	itemListComponent *ItemListComponent
	detailPanel       *widget.Container
	detailTextArea    *widgets.CachedTextAreaWrapper
	filterButtons     *widget.Container

	currentFilter string // "all", "throwables", "equipment", "consumables"
}

func NewInventoryMode(modeManager *framework.UIModeManager) *InventoryMode {
	mode := &InventoryMode{
		currentFilter: "all",
	}
	mode.SetModeName("inventory")
	mode.SetReturnMode("exploration")
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (im *InventoryMode) Initialize(ctx *framework.UIContext) error {
	// Create inventory service first (needed by filter/list builders)
	im.inventoryService = gear.NewInventoryService(ctx.ECSManager)

	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&im.BaseMode, framework.ModeConfig{
		ModeName:   "inventory",
		ReturnMode: "exploration",
		Hotkeys: []framework.HotkeySpec{
			{Key: ebiten.KeyI, TargetMode: "exploration"},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := im.BuildPanels(
		InventoryPanelFilterButtons,
		InventoryPanelItemList,
		InventoryPanelDetail,
		InventoryPanelActionButtons,
	); err != nil {
		return err
	}

	// Initialize widget references from registry
	im.initializeWidgetReferences()

	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (im *InventoryMode) initializeWidgetReferences() {
	im.filterButtons = GetInventoryFilterButtons(im.Panels)
	im.itemList = GetInventoryItemList(im.Panels)
	im.itemListComponent = GetInventoryItemListComponent(im.Panels)
	im.detailPanel = GetInventoryDetailPanel(im.Panels)
	im.detailTextArea = GetInventoryDetailTextArea(im.Panels)
}

// handleItemSelection processes item selection from the inventory list
func (im *InventoryMode) handleItemSelection(selectedEntry interface{}) {
	// Handle InventoryListEntry type
	if entry, ok := selectedEntry.(gear.InventoryListEntry); ok {
		fmt.Printf("Selected item: %s (index %d)\n", entry.Name, entry.Index)

		// If in throwables mode, prepare the throwable
		if im.currentFilter == "Throwables" && im.Context.PlayerData != nil {
			im.handleThrowableSelection(entry)
		} else {
			im.detailTextArea.SetText(fmt.Sprintf("Selected: %s x%d", entry.Name, entry.Count))
		}
	} else if str, ok := selectedEntry.(string); ok {
		// Handle string messages
		im.detailTextArea.SetText(str)
	}
}

func (im *InventoryMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Inventory Mode")

	// Initialize item list with current filter (defaults to "All")
	// This also handles refreshing if the inventory was modified
	if im.itemListComponent != nil {
		// Ensure filter and list are in sync
		im.itemListComponent.SetFilter(im.currentFilter)
	}

	return nil
}

func (im *InventoryMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Inventory Mode")
	return nil
}

func (im *InventoryMode) Update(deltaTime float64) error {
	return nil
}

func (im *InventoryMode) Render(screen *ebiten.Image) {
	// No custom rendering
}

func (im *InventoryMode) HandleInput(inputState *framework.InputState) bool {
	// Handle common input (ESC key)
	if im.HandleCommonInput(inputState) {
		return true
	}

	// I key to close (inventory-specific hotkey)
	if inputState.KeysJustPressed[ebiten.KeyI] {
		if exploreMode, exists := im.ModeManager.GetMode("exploration"); exists {
			im.ModeManager.RequestTransition(exploreMode, "Close Inventory")
			return true
		}
	}

	return false
}

// handleThrowableSelection handles selecting a throwable item using the service layer
// This replaces direct ECS manipulation in the UI code
func (im *InventoryMode) handleThrowableSelection(entry gear.InventoryListEntry) {
	// Use service to validate and prepare throwable selection
	result := im.inventoryService.SelectThrowable(im.Context.PlayerData.PlayerEntityID, entry.Index)

	if !result.Success {
		// Display error message
		im.detailTextArea.SetText(fmt.Sprintf("Cannot select throwable: %s", result.Error))
		fmt.Printf("Throwable selection failed: %s\n", result.Error)
		return
	}

	// Update player data with throwable selection (UI state only)
	im.Context.PlayerData.Throwables.SelectedThrowableID = result.ItemEntityID
	im.Context.PlayerData.Throwables.ThrowableItemEntityID = result.ItemEntityID
	im.Context.PlayerData.Throwables.ThrowableItemIndex = result.ItemIndex
	im.Context.PlayerData.InputStates.IsThrowing = true

	// Display item details with effects
	effectStr := ""
	for _, effectName := range result.EffectDescriptions {
		effectStr += fmt.Sprintf("%s\n", effectName)
	}

	detailText := fmt.Sprintf("Selected: %s\n\n%s", result.ItemName, effectStr)
	im.detailTextArea.SetText(detailText)

	fmt.Printf("Throwable prepared: %s\n", result.ItemName)

	// Close inventory and return to exploration
	if exploreMode, exists := im.ModeManager.GetMode("exploration"); exists {
		im.ModeManager.RequestTransition(exploreMode, "Throwable selected")
	}
}
