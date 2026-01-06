package builders

import (
	"fmt"
	"game_main/common"
	"game_main/gui/widgets"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// ============================================
// SQUAD LIST HELPERS
// ============================================

// SquadListConfig provides configuration for creating squad selection lists
type SquadListConfig struct {
	SquadIDs      []ecs.EntityID
	Manager       *common.EntityManager
	OnSelect      func(squadID ecs.EntityID)
	ScreenWidth   int
	ScreenHeight  int
	WidthPercent  float64 // Default 0.3
	HeightPercent float64 // Default 0.3
}

// CreateSquadList creates a list widget for selecting squads
// Automatically formats squad names and handles selection
func CreateSquadList(config SquadListConfig) *widget.List {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.3
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.3
	}

	// Calculate dimensions
	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	// Build entries with squad names
	entries := make([]interface{}, 0, len(config.SquadIDs))
	for _, squadID := range config.SquadIDs {
		squadName := squads.GetSquadName(squadID, config.Manager)
		entries = append(entries, squadName)
	}

	return CreateListWithConfig(ListConfig{
		Entries:   entries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		OnEntrySelected: func(e interface{}) {
			if config.OnSelect != nil {
				// Find the squad ID by matching the name
				selectedName := e.(string)
				for i, squadID := range config.SquadIDs {
					if i < len(entries) && entries[i].(string) == selectedName {
						config.OnSelect(squadID)
						return
					}
				}
			}
		},
	})
}

// CreateCachedSquadList creates a cached list widget for selecting squads
// IMPORTANT: Call MarkDirty() when squad names/count changes
func CreateCachedSquadList(config SquadListConfig) *widgets.CachedListWrapper {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.3
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.3
	}

	// Calculate dimensions
	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	// Build entries with squad names
	entries := make([]interface{}, 0, len(config.SquadIDs))
	for _, squadID := range config.SquadIDs {
		squadName := squads.GetSquadName(squadID, config.Manager)
		entries = append(entries, squadName)
	}

	return CreateCachedList(ListConfig{
		Entries:   entries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		OnEntrySelected: func(e interface{}) {
			if config.OnSelect != nil {
				// Find the squad ID by matching the name
				selectedName := e.(string)
				for i, squadID := range config.SquadIDs {
					if i < len(entries) && entries[i].(string) == selectedName {
						config.OnSelect(squadID)
						return
					}
				}
			}
		},
	})
}

// ============================================
// UNIT LIST HELPERS
// ============================================

// UnitListConfig provides configuration for creating unit lists with health display
type UnitListConfig struct {
	UnitIDs       []ecs.EntityID
	Manager       *common.EntityManager
	ScreenWidth   int
	ScreenHeight  int
	WidthPercent  float64 // Default 0.4
	HeightPercent float64 // Default 0.5
}

// CreateUnitList creates a list widget displaying units with name and health
// TODO, this not not currently being used. It should be used instead of CreateUnitList, but I need to test that to make sure the cache
// Is updated correctly
func CreateUnitList(config UnitListConfig) *widget.List {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.4
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.5
	}

	// Calculate dimensions
	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	// Build entries with unit info
	entries := make([]interface{}, 0, len(config.UnitIDs))
	for _, unitID := range config.UnitIDs {
		// Get unit attributes
		if attrRaw, ok := config.Manager.GetComponent(unitID, common.AttributeComponent); ok {
			attr := attrRaw.(*common.Attributes)

			// Get unit name
			nameStr := "Unknown"
			if nameRaw, ok := config.Manager.GetComponent(unitID, common.NameComponent); ok {
				name := nameRaw.(*common.Name)
				nameStr = name.NameStr
			}

			// Store UnitIdentity object instead of string
			entries = append(entries, squads.UnitIdentity{
				ID:        unitID,
				Name:      nameStr,
				CurrentHP: attr.CurrentHealth,
				MaxHP:     attr.MaxHealth,
			})
		}
	}

	return CreateListWithConfig(ListConfig{
		Entries: entries,
		EntryLabelFunc: func(e interface{}) string {
			identity := e.(squads.UnitIdentity)
			return fmt.Sprintf("%s - HP: %d/%d", identity.Name, identity.CurrentHP, identity.MaxHP)
		},
		MinWidth:  listWidth,
		MinHeight: listHeight,
	})
}

// CreateCachedUnitList creates a cached list widget displaying units with name and health
// IMPORTANT: Call MarkDirty() when unit data changes (e.g., after adding/removing units, combat)
// TODO, this not not currently being used. It should be used instead of CreateUnitList, but I need to test that to make sure the cache
// Is updated correctly
func CreateCachedUnitList(config UnitListConfig) *widgets.CachedListWrapper {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.4
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.5
	}

	// Calculate dimensions
	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	// Build entries with unit info
	entries := make([]interface{}, 0, len(config.UnitIDs))
	for _, unitID := range config.UnitIDs {
		// Get unit attributes
		if attrRaw, ok := config.Manager.GetComponent(unitID, common.AttributeComponent); ok {
			attr := attrRaw.(*common.Attributes)

			// Get unit name
			nameStr := "Unknown"
			if nameRaw, ok := config.Manager.GetComponent(unitID, common.NameComponent); ok {
				name := nameRaw.(*common.Name)
				nameStr = name.NameStr
			}

			// Store UnitIdentity object instead of string
			entries = append(entries, squads.UnitIdentity{
				ID:        unitID,
				Name:      nameStr,
				CurrentHP: attr.CurrentHealth,
				MaxHP:     attr.MaxHealth,
			})
		}
	}

	return CreateCachedList(ListConfig{
		Entries: entries,
		EntryLabelFunc: func(e interface{}) string {
			identity := e.(squads.UnitIdentity)
			return fmt.Sprintf("%s - HP: %d/%d", identity.Name, identity.CurrentHP, identity.MaxHP)
		},
		MinWidth:  listWidth,
		MinHeight: listHeight,
	})
}

// ============================================
// SIMPLE STRING LIST HELPERS
// ============================================

// SimpleStringListConfig provides configuration for simple string lists
type SimpleStringListConfig struct {
	Entries       []string
	OnSelect      func(selected string)
	ScreenWidth   int
	ScreenHeight  int
	WidthPercent  float64 // Default 0.3
	HeightPercent float64 // Default 0.3
	LayoutData    interface{}
}

// CreateSimpleStringList creates a list widget for selecting from string entries
func CreateSimpleStringList(config SimpleStringListConfig) *widget.List {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.3
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.3
	}

	// Calculate dimensions
	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	// Convert strings to interface{} slice
	entries := make([]interface{}, len(config.Entries))
	for i, s := range config.Entries {
		entries[i] = s
	}

	listConfig := ListConfig{
		Entries:   entries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		LayoutData: config.LayoutData,
	}

	if config.OnSelect != nil {
		listConfig.OnEntrySelected = func(e interface{}) {
			config.OnSelect(e.(string))
		}
	}

	return CreateListWithConfig(listConfig)
}

// ============================================
// INVENTORY LIST HELPERS
// ============================================

// InventoryListConfig provides configuration for creating inventory item lists
type InventoryListConfig struct {
	EntryLabelFunc func(interface{}) string // Optional custom formatter
	OnSelect       func(entry interface{})  // Callback for when item is selected
	ScreenWidth    int
	ScreenHeight   int
	WidthPercent   float64 // Default 0.5
	HeightPercent  float64 // Default 0.7
	LayoutData     interface{}
}

// CreateInventoryList creates a list widget for inventory items
// Provides flexible formatting via EntryLabelFunc
func CreateInventoryList(config InventoryListConfig) *widget.List {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.5
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.7
	}

	// Default formatter if not provided
	if config.EntryLabelFunc == nil {
		config.EntryLabelFunc = func(e interface{}) string {
			return fmt.Sprintf("%v", e)
		}
	}

	// Calculate dimensions
	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	listConfig := ListConfig{
		Entries:        []interface{}{}, // Will be populated externally
		MinWidth:       listWidth,
		MinHeight:      listHeight,
		EntryLabelFunc: config.EntryLabelFunc,
		LayoutData:     config.LayoutData,
	}

	if config.OnSelect != nil {
		listConfig.OnEntrySelected = config.OnSelect
	}

	return CreateListWithConfig(listConfig)
}
