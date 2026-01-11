package guiexploration

import (
	"fmt"

	"game_main/common"
	"game_main/gear"
	"game_main/gui/framework"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// ItemListComponent manages an inventory list widget with filtering
type ItemListComponent struct {
	listWidget     *widget.List
	queries        *framework.GUIQueries
	ecsManager     *common.EntityManager
	playerEntityID ecs.EntityID
	currentFilter  string
}

// NewItemListComponent creates a reusable inventory list component
func NewItemListComponent(
	listWidget *widget.List,
	queries *framework.GUIQueries,
	ecsManager *common.EntityManager,
	playerEntityID ecs.EntityID,
) *ItemListComponent {
	return &ItemListComponent{
		listWidget:     listWidget,
		queries:        queries,
		ecsManager:     ecsManager,
		playerEntityID: playerEntityID,
		currentFilter:  "All",
	}
}

// SetFilter updates the current filter and refreshes the list
func (ilc *ItemListComponent) SetFilter(filter string) {
	ilc.currentFilter = filter
	ilc.Refresh()
}

// Refresh updates the list with items based on current filter
func (ilc *ItemListComponent) Refresh() {
	if ilc.listWidget == nil || ilc.ecsManager == nil {
		return
	}

	// Get inventory from player entity
	inv := common.GetComponentTypeByID[*gear.Inventory](ilc.ecsManager, ilc.playerEntityID, gear.InventoryComponent)
	if inv == nil {
		ilc.listWidget.SetEntries([]interface{}{"No inventory available"})
		return
	}

	var entries []interface{}

	// Query inventory based on current filter
	switch ilc.currentFilter {
	case "Throwables":
		// Get throwable items
		throwableEntries := gear.GetThrowableItems(ilc.ecsManager, inv, []int{})
		if len(throwableEntries) == 0 {
			entries = []interface{}{"No throwable items"}
		} else {
			entries = make([]interface{}, len(throwableEntries))
			for i, e := range throwableEntries {
				entries[i] = e
			}
		}

	case "All":
		// Get all items
		allEntries := gear.GetInventoryForDisplay(ilc.ecsManager, inv, []int{})
		if len(allEntries) == 0 {
			entries = []interface{}{"Inventory is empty"}
		} else {
			entries = make([]interface{}, len(allEntries))
			for i, e := range allEntries {
				entries[i] = e
			}
		}

	default:
		// Placeholder for other filters
		entries = []interface{}{fmt.Sprintf("Filter '%s' not yet implemented", ilc.currentFilter)}
	}

	ilc.listWidget.SetEntries(entries)
}
