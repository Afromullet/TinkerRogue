package guimodes

import (
	"fmt"
	"image/color"

	"game_main/gear"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// InventoryMode provides full-screen inventory browsing and management
type InventoryMode struct {
	framework.BaseMode // Embed common mode infrastructure

	inventoryService  *gear.InventoryService
	itemList          *widget.List
	itemListComponent *ItemListComponent
	detailPanel       *widget.Container
	detailTextArea    *widgets.CachedTextAreaWrapper // Cached for performance
	filterButtons     *widget.Container
	closeButton       *widget.Button

	currentFilter string // "all", "throwables", "equipment", "consumables"
}

func NewInventoryMode(modeManager *framework.UIModeManager) *InventoryMode {
	mode := &InventoryMode{
		currentFilter: "all",
	}
	mode.SetModeName("inventory")
	mode.ModeManager = modeManager
	return mode
}

func (im *InventoryMode) Initialize(ctx *framework.UIContext) error {
	// Create inventory service first (needed by filter/list builders)
	im.inventoryService = gear.NewInventoryService(ctx.ECSManager)

	err := framework.NewModeBuilder(&im.BaseMode, framework.ModeConfig{
		ModeName:   "inventory",
		ReturnMode: "exploration",

		Hotkeys: []framework.HotkeySpec{
			{Key: ebiten.KeyI, TargetMode: "exploration"},
		},

		Panels: []framework.ModePanelConfig{
			{CustomBuild: im.buildFilterButtons},
			{CustomBuild: im.buildItemList},
			{CustomBuild: im.buildDetailPanel},
		},

		Buttons: []framework.ButtonGroupSpec{
			{
				Position: builders.BottomCenter(),
				Buttons: []builders.ButtonSpec{
					framework.ModeTransitionSpec(im.ModeManager, "Close (ESC)", "exploration"),
				},
			},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (im *InventoryMode) buildFilterButtons() *widget.Container {
	// Top-left filter buttons using helper
	filterButtons := framework.CreateFilterButtonContainer(im.PanelBuilders, builders.TopLeft())

	// Filter buttons - use component's SetFilter when clicked
	filters := []string{"All", "Throwables", "Equipment", "Consumables"}
	for _, filterName := range filters {
		filterNameCopy := filterName // Capture for closure
		btn := builders.CreateButtonWithConfig(builders.ButtonConfig{
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
	itemList := builders.CreateInventoryList(builders.InventoryListConfig{
		ScreenWidth:   im.Layout.ScreenWidth,
		ScreenHeight:  im.Layout.ScreenHeight,
		WidthPercent:  specs.InventoryListWidth,
		HeightPercent: specs.InventoryListHeight,
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
	im.itemListComponent = NewItemListComponent(
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
				Left: int(float64(im.Layout.ScreenWidth) * specs.PaddingStandard),
				Top:  int(float64(im.Layout.ScreenHeight) * specs.PanelHeightSmall),
			},
		})),
	)
	container.AddChild(itemList)
	return container
}

func (im *InventoryMode) buildDetailPanel() *widget.Container {
	// Right side detail panel (35% width, 60% height)
	panelWidth := int(float64(im.Layout.ScreenWidth) * 0.35)
	panelHeight := int(float64(im.Layout.ScreenHeight) * 0.6)

	detailPanel := builders.CreateStaticPanel(builders.ContainerConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(im.Layout, specs.PaddingTight)),
		),
	})

	rightPad := int(float64(im.Layout.ScreenWidth) * specs.PaddingStandard)
	detailPanel.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)

	// Detail text area - cached for performance
	detailTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
		MinWidth:  panelWidth - 30,
		MinHeight: panelHeight - 30,
		FontColor: color.White,
	})
	detailTextArea.SetText("Select an item to view details") // SetText calls MarkDirty() internally
	detailPanel.AddChild(detailTextArea)

	im.detailPanel = detailPanel
	im.detailTextArea = detailTextArea

	return detailPanel
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
