package guimodes

import (
	"fmt"

	"game_main/gear"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// InventoryMode provides full-screen inventory browsing and management
type InventoryMode struct {
	gui.BaseMode // Embed common mode infrastructure

	inventoryService  *gear.InventoryService
	itemList          *widget.List
	itemListComponent *guicomponents.ItemListComponent
	detailPanel       *widget.Container
	detailTextArea    *widget.TextArea
	filterButtons     *widget.Container
	closeButton       *widget.Button

	currentFilter string // "all", "throwables", "equipment", "consumables"
}

func NewInventoryMode(modeManager *core.UIModeManager) *InventoryMode {
	mode := &InventoryMode{
		currentFilter: "all",
	}
	mode.SetModeName("inventory")
	mode.ModeManager = modeManager
	return mode
}

func (im *InventoryMode) Initialize(ctx *core.UIContext) error {
	// Create inventory service first (needed by filter/list builders)
	im.inventoryService = gear.NewInventoryService(ctx.ECSManager)

	err := gui.NewModeBuilder(&im.BaseMode, gui.ModeConfig{
		ModeName:   "inventory",
		ReturnMode: "exploration",

		Hotkeys: []gui.HotkeySpec{
			{Key: ebiten.KeyI, TargetMode: "exploration"},
		},

		Panels: []gui.PanelSpec{
			{CustomBuild: im.buildFilterButtons},
			{CustomBuild: im.buildItemList},
			{
				// Detail panel (right side) - now uses typed panel
				PanelType:  widgets.PanelTypeDetail,
				SpecName:   "inventory_detail",
				DetailText: "Select an item to view details",
			},
		},

		Buttons: []gui.ButtonGroupSpec{
			{
				Position: widgets.BottomCenter(),
				Buttons: []widgets.ButtonSpec{
					gui.ModeTransitionSpec(im.ModeManager, "Close (ESC)", "exploration"),
				},
			},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Get reference to detail TextArea from typed panel
	if w, ok := im.PanelWidgets["inventory_detail"]; ok {
		if textArea, ok := w.(*widget.TextArea); ok {
			im.detailTextArea = textArea
		}
	}

	return nil
}

func (im *InventoryMode) buildFilterButtons() *widget.Container {
	// Top-left filter buttons using helper
	filterButtons := gui.CreateFilterButtonContainer(im.PanelBuilders, widgets.TopLeft())

	// Filter buttons - use component's SetFilter when clicked
	filters := []string{"All", "Throwables", "Equipment", "Consumables"}
	for _, filterName := range filters {
		filterNameCopy := filterName // Capture for closure
		btn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
			Text: filterName,
			OnClick: func() {
				// Sync both filter states before delegating to component
				im.currentFilter = filterNameCopy
				if im.itemListComponent != nil {
					im.itemListComponent.SetFilter(filterNameCopy)
				}
			},
		})
		filterButtons.AddChild(btn)
	}

	im.filterButtons = filterButtons
	return filterButtons
}

func (im *InventoryMode) buildItemList() *widget.Container {
	// Create inventory list using helper (without LayoutData in config)
	itemList := widgets.CreateInventoryList(widgets.InventoryListConfig{
		ScreenWidth:   im.Layout.ScreenWidth,
		ScreenHeight:  im.Layout.ScreenHeight,
		WidthPercent:  widgets.InventoryListWidth,
		HeightPercent: widgets.InventoryListHeight,
		OnSelect:      im.handleItemSelection,
		EntryLabelFunc: func(e interface{}) string {
			// Handle both string messages and InventoryListEntry
			switch v := e.(type) {
			case string:
				return v
			case gear.InventoryListEntry:
				return fmt.Sprintf("%s x%d", v.Name, v.Count)
			default:
				return fmt.Sprintf("%v", e)
			}
		},
	})

	im.itemList = itemList

	// Create item list component to manage refresh logic
	im.itemListComponent = guicomponents.NewItemListComponent(
		im.itemList,
		im.Queries,
		im.Context.ECSManager,
		im.Context.PlayerData.PlayerEntityID,
	)

	// Wrap in container with proper LayoutData
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Left: int(float64(im.Layout.ScreenWidth) * widgets.PaddingStandard),
				Top:  int(float64(im.Layout.ScreenHeight) * widgets.PanelHeightSmall),
			},
		})),
	)
	container.AddChild(itemList)
	return container
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

func (im *InventoryMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Inventory Mode")

	// Initialize item list with current filter (defaults to "All")
	// This also handles refreshing if the inventory was modified
	if im.itemListComponent != nil {
		// Ensure filter and list are in sync
		im.itemListComponent.SetFilter(im.currentFilter)
	}

	return nil
}

func (im *InventoryMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Inventory Mode")
	return nil
}

func (im *InventoryMode) Update(deltaTime float64) error {
	return nil
}

func (im *InventoryMode) Render(screen *ebiten.Image) {
	// No custom rendering
}

func (im *InventoryMode) HandleInput(inputState *core.InputState) bool {
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
