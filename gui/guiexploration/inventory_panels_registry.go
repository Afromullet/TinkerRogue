package guiexploration

import (
	"fmt"
	"image/color"

	"game_main/gear"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for inventory mode
const (
	InventoryPanelFilterButtons framework.PanelType = "inventory_filter_buttons"
	InventoryPanelItemList      framework.PanelType = "inventory_item_list"
	InventoryPanelDetail        framework.PanelType = "inventory_detail"
	InventoryPanelActionButtons framework.PanelType = "inventory_action_buttons"
)

func init() {
	// Register filter buttons panel
	framework.RegisterPanel(InventoryPanelFilterButtons, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			im := mode.(*InventoryMode)

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

			result.Container = filterButtons
			result.Custom["filterButtons"] = filterButtons

			return nil
		},
	})

	// Register item list panel
	framework.RegisterPanel(InventoryPanelItemList, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			im := mode.(*InventoryMode)

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

			// Create item list component to manage refresh logic
			itemListComponent := NewItemListComponent(
				itemList,
				im.Queries,
				im.Context.ECSManager,
				im.Context.PlayerData.PlayerEntityID,
			)

			// Wrap in container with proper LayoutData
			result.Container = widget.NewContainer(
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
			result.Container.AddChild(itemList)

			result.Custom["itemList"] = itemList
			result.Custom["itemListComponent"] = itemListComponent

			return nil
		},
	})

	// Register detail panel
	framework.RegisterPanel(InventoryPanelDetail, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			im := mode.(*InventoryMode)
			layout := im.Layout

			// Right side detail panel (35% width, 60% height)
			panelWidth := int(float64(layout.ScreenWidth) * 0.35)
			panelHeight := int(float64(layout.ScreenHeight) * 0.6)

			detailPanel := builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  panelWidth,
				MinHeight: panelHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(10),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingTight)),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			detailPanel.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)

			// Detail text area - cached for performance
			detailTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  panelWidth - 30,
				MinHeight: panelHeight - 30,
				FontColor: color.White,
			})
			detailTextArea.SetText("Select an item to view details")
			detailPanel.AddChild(detailTextArea)

			result.Container = detailPanel
			result.Custom["detailPanel"] = detailPanel
			result.Custom["detailTextArea"] = detailTextArea

			return nil
		},
	})

	// Register action buttons panel
	framework.RegisterPanel(InventoryPanelActionButtons, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			im := mode.(*InventoryMode)
			layout := im.Layout

			spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
			anchorLayout := builders.AnchorCenterEnd(bottomPad)

			result.Container = builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons: []builders.ButtonSpec{
					{Text: "Close (ESC)", OnClick: func() {
						if exploreMode, exists := im.ModeManager.GetMode("exploration"); exists {
							im.ModeManager.RequestTransition(exploreMode, "Close Inventory")
						}
					}},
				},
				Direction:  widget.DirectionHorizontal,
				Spacing:    spacing,
				Padding:    builders.NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall),
				LayoutData: &anchorLayout,
			})

			return nil
		},
	})
}

// Helper functions to retrieve widgets from panel registry

func GetInventoryFilterButtons(panels *framework.PanelRegistry) *widget.Container {
	if result := panels.Get(InventoryPanelFilterButtons); result != nil {
		if container, ok := result.Custom["filterButtons"].(*widget.Container); ok {
			return container
		}
	}
	return nil
}

func GetInventoryItemList(panels *framework.PanelRegistry) *widget.List {
	if result := panels.Get(InventoryPanelItemList); result != nil {
		if list, ok := result.Custom["itemList"].(*widget.List); ok {
			return list
		}
	}
	return nil
}

func GetInventoryItemListComponent(panels *framework.PanelRegistry) *ItemListComponent {
	if result := panels.Get(InventoryPanelItemList); result != nil {
		if component, ok := result.Custom["itemListComponent"].(*ItemListComponent); ok {
			return component
		}
	}
	return nil
}

func GetInventoryDetailPanel(panels *framework.PanelRegistry) *widget.Container {
	if result := panels.Get(InventoryPanelDetail); result != nil {
		if panel, ok := result.Custom["detailPanel"].(*widget.Container); ok {
			return panel
		}
	}
	return nil
}

func GetInventoryDetailTextArea(panels *framework.PanelRegistry) *widgets.CachedTextAreaWrapper {
	if result := panels.Get(InventoryPanelDetail); result != nil {
		if area, ok := result.Custom["detailTextArea"].(*widgets.CachedTextAreaWrapper); ok {
			return area
		}
	}
	return nil
}
