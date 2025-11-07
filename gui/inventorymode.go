package gui

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// InventoryMode provides full-screen inventory browsing and management
type InventoryMode struct {
	BaseMode // Embed common mode infrastructure

	itemList       *widget.List
	detailPanel    *widget.Container
	detailTextArea *widget.TextArea
	filterButtons  *widget.Container
	closeButton    *widget.Button

	currentFilter string // "all", "throwables", "equipment", "consumables"
	initialFilter string // Filter to set on Enter() - allows pre-setting filter before transition
}

func NewInventoryMode(modeManager *UIModeManager) *InventoryMode {
	return &InventoryMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "inventory",
			returnMode:  "exploration",
		},
		currentFilter: "all",
		initialFilter: "",
	}
}

// TOOO remove in the future. This is here for testing
// SetInitialFilter sets the filter that will be applied when entering this mode
// Call this before transitioning to pre-set the desired filter
func (im *InventoryMode) SetInitialFilter(filter string) {
	im.initialFilter = filter
}

func (im *InventoryMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure
	im.InitializeBase(ctx)

	// Build inventory UI
	im.buildFilterButtons()
	im.buildItemList()

	// Build detail panel (right side) using BuildPanel
	im.detailPanel = im.panelBuilders.BuildPanel(
		RightCenter(),
		Size(0.45, 0.75),
		Padding(0.02),
		AnchorLayout(),
	)

	// Create detail text area inside panel
	panelWidth := int(float64(im.layout.ScreenWidth) * 0.45)
	panelHeight := int(float64(im.layout.ScreenHeight) * 0.75)
	im.detailTextArea = CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})
	im.detailTextArea.SetText("Select an item to view details")
	im.detailPanel.AddChild(im.detailTextArea)
	im.rootContainer.AddChild(im.detailPanel)

	// Build close button (bottom-center) using BuildPanel
	closeButtonContainer := im.panelBuilders.BuildPanel(
		BottomCenter(),
		CustomPadding(widget.Insets{
			Bottom: int(float64(im.layout.ScreenHeight) * 0.08),
		}),
		AnchorLayout(),
	)

	closeBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Close (ESC)",
		OnClick: func() {
			if targetMode, exists := im.modeManager.GetMode("exploration"); exists {
				im.modeManager.RequestTransition(targetMode, "Close button pressed")
			}
		},
	})
	closeButtonContainer.AddChild(closeBtn)
	im.rootContainer.AddChild(closeButtonContainer)

	return nil
}

func (im *InventoryMode) buildFilterButtons() {
	// Top-left filter buttons using BuildPanel
	im.filterButtons = im.panelBuilders.BuildPanel(
		TopLeft(),
		Padding(0.02),
		HorizontalRowLayout(),
	)

	// Filter buttons
	filters := []string{"All", "Throwables", "Equipment", "Consumables"}
	for _, filterName := range filters {
		filterNameCopy := filterName // Capture for closure
		btn := CreateButtonWithConfig(ButtonConfig{
			Text: filterName,
			OnClick: func() {
				im.currentFilter = filterNameCopy
				im.refreshItemList()
			},
		})
		im.filterButtons.AddChild(btn)
	}

	im.rootContainer.AddChild(im.filterButtons)
}

func (im *InventoryMode) buildItemList() {
	// Left side item list (50% width)
	listWidth := int(float64(im.layout.ScreenWidth) * 0.45)
	listHeight := int(float64(im.layout.ScreenHeight) * 0.75)

	im.itemList = CreateListWithConfig(ListConfig{
		Entries:    []interface{}{}, // Will be populated by refreshItemList
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
					// Type assert the inventory interface{} to *gear.Inventory
					if inv, ok := im.context.PlayerData.Inventory.(*gear.Inventory); ok {
						// Get the item entity ID from inventory
						itemEntityID, err := gear.GetItemEntityID(inv, entry.Index)
						if err == nil {
							// Find the actual entity
							itemEntity := gear.FindItemEntityByID(im.context.ECSManager.World, itemEntityID)
							if itemEntity != nil {
								// Prepare the throwable directly (no wrapper)
								im.context.PlayerData.Throwables.SelectedThrowable = itemEntity
								im.context.PlayerData.Throwables.ThrowableItemEntity = itemEntity

								// Get item component and setup throwing shape
								item := common.GetComponentType[*gear.Item](itemEntity, gear.ItemComponent)
								if item != nil {
									if throwableAction := item.GetThrowableAction(); throwableAction != nil {
										im.context.PlayerData.Throwables.ThrowableItemIndex = entry.Index
										im.context.PlayerData.Throwables.ThrowingAOEShape = throwableAction.Shape
									}
								}

								im.context.PlayerData.InputStates.IsThrowing = true
								fmt.Printf("Throwable prepared: %s\n", entry.Name)
							}
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
				Left: int(float64(im.layout.ScreenWidth) * 0.02),
				Top:  int(float64(im.layout.ScreenHeight) * 0.15),
			},
		},
	})

	im.rootContainer.AddChild(im.itemList)
}

func (im *InventoryMode) refreshItemList() {
	if im.context.PlayerData == nil || im.context.PlayerData.Inventory == nil {
		entries := []interface{}{"No inventory available"}
		im.itemList.SetEntries(entries)
		return
	}

	// Type assert the inventory interface{} to *gear.Inventory
	inv, ok := im.context.PlayerData.Inventory.(*gear.Inventory)
	if !ok {
		entries := []interface{}{"Inventory type mismatch"}
		im.itemList.SetEntries(entries)
		return
	}

	entries := []interface{}{}

	// Query inventory based on current filter
	switch im.currentFilter {
	case "Throwables":
		// Get throwable items
		throwableEntries := gear.GetThrowableItems(im.context.ECSManager.World, inv, []int{})
		if len(throwableEntries) == 0 {
			entries = append(entries, "No throwable items")
		} else {
			entries = throwableEntries
		}

	case "All":
		// Get all items
		allEntries := gear.GetInventoryForDisplay(im.context.ECSManager.World, inv, []int{})
		if len(allEntries) == 0 {
			entries = append(entries, "Inventory is empty")
		} else {
			entries = allEntries
		}

	default:
		// Placeholder for other filters
		entries = append(entries, fmt.Sprintf("Filter '%s' not yet implemented", im.currentFilter))
	}

	im.itemList.SetEntries(entries)
}

func (im *InventoryMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Inventory Mode")

	//TODO remove in the future. Here for testing
	// Apply initial filter if one was set
	if im.initialFilter != "" {
		im.currentFilter = im.initialFilter
		im.initialFilter = "" // Reset after use
	}

	im.refreshItemList()
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
