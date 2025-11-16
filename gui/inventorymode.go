package gui

import (
	"fmt"
	"game_main/common"
	"game_main/gear"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// InventoryMode provides full-screen inventory browsing and management
type InventoryMode struct {
	BaseMode // Embed common mode infrastructure

	itemList          *widget.List
	itemListComponent *ItemListComponent
	detailPanel       *widget.Container
	detailTextArea    *widget.TextArea
	filterButtons     *widget.Container
	closeButton       *widget.Button

	currentFilter string // "all", "throwables", "equipment", "consumables"
}

func NewInventoryMode(modeManager *UIModeManager) *InventoryMode {
	return &InventoryMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "inventory",
			returnMode:  "exploration",
		},
		currentFilter: "all",
	}
}

func (im *InventoryMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure
	im.InitializeBase(ctx)

	// Build inventory UI
	im.buildFilterButtons()
	im.buildItemList()

	// Create item list component to manage refresh logic
	im.itemListComponent = NewItemListComponent(
		im.itemList,
		im.queries,
		ctx.ECSManager,
		ctx.PlayerData.PlayerEntityID,
	)

	// Build detail panel (right side) using helper
	im.detailPanel, im.detailTextArea = CreateDetailPanel(
		im.panelBuilders,
		im.layout,
		RightCenter(),
		PanelWidthExtraWide, PanelHeightTall, PaddingStandard,
		"Select an item to view details",
	)
	im.rootContainer.AddChild(im.detailPanel)

	// Build close button (bottom-center) using helpers
	closeButtonContainer := CreateBottomCenterButtonContainer(im.panelBuilders)
	closeBtn := CreateCloseButton(im.modeManager, "exploration", "Close (ESC)")
	closeButtonContainer.AddChild(closeBtn)
	im.rootContainer.AddChild(closeButtonContainer)

	return nil
}

func (im *InventoryMode) buildFilterButtons() {
	// Top-left filter buttons using helper
	im.filterButtons = CreateFilterButtonContainer(im.panelBuilders, TopLeft())

	// Filter buttons - use component's SetFilter when clicked
	filters := []string{"All", "Throwables", "Equipment", "Consumables"}
	for _, filterName := range filters {
		filterNameCopy := filterName // Capture for closure
		btn := CreateButtonWithConfig(ButtonConfig{
			Text: filterName,
			OnClick: func() {
				// Sync both filter states before delegating to component
				im.currentFilter = filterNameCopy
				if im.itemListComponent != nil {
					im.itemListComponent.SetFilter(filterNameCopy)
				}
			},
		})
		im.filterButtons.AddChild(btn)
	}

	im.rootContainer.AddChild(im.filterButtons)
}

func (im *InventoryMode) buildItemList() {
	// Left side item list (50% width)
	listWidth := int(float64(im.layout.ScreenWidth) * InventoryListWidth)
	listHeight := int(float64(im.layout.ScreenHeight) * InventoryListHeight)

	im.itemList = CreateListWithConfig(ListConfig{
		Entries:    []interface{}{}, // Will be populated by component
		MinWidth:   listWidth,
		MinHeight:  listHeight,
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
		OnEntrySelected: func(selectedEntry interface{}) {
			// Handle InventoryListEntry type
			if entry, ok := selectedEntry.(gear.InventoryListEntry); ok {
				fmt.Printf("Selected item: %s (index %d)\n", entry.Name, entry.Index)

				// If in throwables mode, prepare the throwable
				if im.currentFilter == "Throwables" && im.context.PlayerData != nil {
					// Query inventory from player entity via ECS instead of using Inventory field
					inv := common.GetComponentTypeByID[*gear.Inventory](im.context.ECSManager, im.context.PlayerData.PlayerEntityID, gear.InventoryComponent)
					if inv == nil {
						return
					}

					// Get the item entity ID from inventory
					itemEntityID, err := gear.GetItemEntityID(inv, entry.Index)
					if err == nil && itemEntityID != 0 {
						// Prepare the throwable using EntityID (no need to find entity)
						im.context.PlayerData.Throwables.SelectedThrowableID = itemEntityID
						im.context.PlayerData.Throwables.ThrowableItemEntityID = itemEntityID

						// Get item component and setup throwing state
						item := gear.GetItemByID(im.context.ECSManager.World, itemEntityID)
						if item != nil {
							if throwableAction := item.GetThrowableAction(); throwableAction != nil {
								im.context.PlayerData.Throwables.ThrowableItemIndex = entry.Index
							}
						}

						im.context.PlayerData.InputStates.IsThrowing = true
						fmt.Printf("Throwable prepared: %s\n", entry.Name)
					}

					// Get item details
					item := gear.GetItemByID(im.context.ECSManager.World, itemEntityID)
					if item != nil {
						effectNames := gear.GetItemEffectNames(im.context.ECSManager.World, item)
						effectStr := ""
						for _, name := range effectNames {
							effectStr += fmt.Sprintf("%s\n", name)
						}
						im.detailTextArea.SetText(fmt.Sprintf("Selected: %s\n\n%s", entry.Name, effectStr))
					} else {
						im.detailTextArea.SetText(fmt.Sprintf("Selected: %s", entry.Name))
					}

					// Close inventory and return to exploration
					if exploreMode, exists := im.modeManager.GetMode("exploration"); exists {
						im.modeManager.RequestTransition(exploreMode, "Throwable selected")
					}
				} else {
					im.detailTextArea.SetText(fmt.Sprintf("Selected: %s x%d", entry.Name, entry.Count))
				}
			} else if str, ok := selectedEntry.(string); ok {
				// Handle string messages
				im.detailTextArea.SetText(str)
			}
		},
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Left: int(float64(im.layout.ScreenWidth) * PaddingStandard),
				Top:  int(float64(im.layout.ScreenHeight) * PanelHeightSmall),
			},
		},
	})

	im.rootContainer.AddChild(im.itemList)
}

func (im *InventoryMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Inventory Mode")

	// Initialize item list with current filter (defaults to "All")
	// This also handles refreshing if the inventory was modified
	if im.itemListComponent != nil {
		// Ensure filter and list are in sync
		im.itemListComponent.SetFilter(im.currentFilter)
	}

	return nil
}

func (im *InventoryMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Inventory Mode")
	return nil
}

func (im *InventoryMode) Update(deltaTime float64) error {
	return nil
}

func (im *InventoryMode) Render(screen *ebiten.Image) {
	// No custom rendering
}

func (im *InventoryMode) HandleInput(inputState *InputState) bool {
	// Handle common input (ESC key)
	if im.HandleCommonInput(inputState) {
		return true
	}

	// I key to close (inventory-specific hotkey)
	if inputState.KeysJustPressed[ebiten.KeyI] {
		if exploreMode, exists := im.modeManager.GetMode("exploration"); exists {
			im.modeManager.RequestTransition(exploreMode, "Close Inventory")
			return true
		}
	}

	return false
}
