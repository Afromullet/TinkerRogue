package builders

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
)

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
