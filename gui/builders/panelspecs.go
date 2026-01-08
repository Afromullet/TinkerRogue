package builders

import (
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// PanelLayoutSpec defines a reusable panel configuration specification.
// This allows panels to be defined once and used consistently across factories.
type PanelLayoutSpec struct {
	Name     string                      // Descriptive name for the panel
	Position PanelOption        // Position on screen (TopCenter, LeftCenter, etc.)
	Width    float64                     // Width as percentage of screen width
	Height   float64                     // Height as percentage of screen height
	Padding  float64                     // Padding as percentage of screen size
	Layout   PanelOption        // Layout strategy (RowLayout, HorizontalRowLayout, etc.)
	Custom   *widget.Insets              // Optional: Override padding with custom insets
}

// StandardPanels defines common panel configurations used across the GUI package.
// This is the single source of truth for panel layouts - modify here to update everywhere.
//
// Currently used by:
// - UIComponentFactory: Combat and Squad Builder components use custom panel creation
// - CombatMode: action buttons
// - ExplorationMode: stats_panel, message_log, quick_inventory
// - InfoMode: options_list, info_detail
// - InventoryMode: inventory_detail
//
// Note: UIComponentFactory creates widgets (List, TextArea) with custom manual layout
// positioning rather than using BuildPanel patterns for some complex components.
var StandardPanels = map[string]PanelLayoutSpec{
	// ============================================
	// Combat UI Panels
	// ============================================
	"turn_order": {
		Name:     "Turn Order",
		Position: TopCenter(),
		Width:    specs.PanelWidthWide,
		Height:   specs.PanelHeightTiny,
		Padding:  specs.PaddingTight,
		Layout:   HorizontalRowLayout(),
	},
	"faction_info": {
		Name:     "Faction Info",
		Position: TopLeft(),
		Width:    specs.PanelWidthNarrow,
		Height:   specs.PanelHeightSmall,
		Padding:  specs.PaddingTight,
		Layout:   RowLayout(),
	},
	"squad_list": {
		Name:     "Squad List",
		Position: LeftCenter(),
		Width:    specs.PanelWidthNarrow,
		Height:   specs.PanelHeightHalf,
		Padding:  specs.PaddingTight,
		Layout:   RowLayout(),
	},
	"squad_detail": {
		Name:     "Squad Detail",
		Position: LeftBottom(),
		Width:    specs.PanelWidthNarrow,
		Height:   specs.PanelHeightQuarter,
		Padding:  specs.PaddingTight,
		Layout:   RowLayout(),
	},
	"action_buttons": {
		Name:     "Action Buttons",
		Position: BottomCenter(),
		Width:    0.5, // Will be sized by container
		Height:   0.08,
		Padding:  0, // Custom padding applied at runtime
		Layout:   HorizontalRowLayout(),
	},

	// ============================================
	// Exploration Mode Panels
	// ============================================
	"stats_panel": {
		Name:     "Stats Display",
		Position: TopRight(),
		Width:    specs.PanelWidthNarrow,
		Height:   specs.PanelHeightSmall,
		Padding:  specs.PaddingTight,
		Layout:   RowLayout(),
	},
	"message_log": {
		Name:     "Message Log",
		Position: BottomRight(),
		Width:    specs.PanelWidthNarrow,
		Height:   0.15,
		Padding:  specs.PaddingTight,
		Layout:   RowLayout(),
	},
	"quick_inventory": {
		Name:     "Quick Inventory",
		Position: BottomCenter(),
		Width:    0.5,
		Height:   0.08,
		Padding:  0, // Custom padding applied at runtime
		Layout:   HorizontalRowLayout(),
	},

	// ============================================
	// Info/Inspection Mode Panels
	// ============================================
	"options_list": {
		Name:     "Options List",
		Position: Center(),
		Width:    0.25, // Reduced from specs.PanelWidthMedium (0.30) to 0.25 to eliminate 5% overlap with info_detail (creates 5% gap)
		Height:   specs.PanelHeightHalf,
		Padding:  0, // Uses default padding from BuildPanel
		Layout:   RowLayout(),
	},

	// ============================================
	// Detail Panels (TextArea Containers)
	// ============================================
	"inventory_detail": {
		Name:     "Inventory Detail",
		Position: RightCenter(),
		Width:    specs.PanelWidthExtraWide,
		Height:   specs.PanelHeightTall,
		Padding:  specs.PaddingStandard,
		Layout:   AnchorLayout(),
	},
	"info_detail": {
		Name:     "Info Detail",
		Position: RightCenter(),
		Width:    0.4,
		Height:   0.6,
		Padding:  0.01,
		Layout:   AnchorLayout(),
	},
	"combat_log": {
		Name:     "Combat Log",
		Position: BottomRight(), // Position at bottom-right with 24% width to avoid overlapping with 50% bottom-center buttons (1% gap: 75% vs 76%)
		Width:    0.24, // Reduced from 0.45 to 0.24 to eliminate overlap (action_buttons end at 75%, this starts at 76%)
		Height:   specs.CombatLogHeight,
		Padding:  specs.PaddingTight,
		Layout:   AnchorLayout(),
	},
}

// CreateStandardPanel builds a panel from a specification by name.
// If the spec doesn't exist, it returns nil and should be handled by the caller.
//
// Example usage:
//
//	panel := CreateStandardPanel(panelBuilders, "turn_order")
func CreateStandardPanel(pb *PanelBuilders, specName string) *widget.Container {
	spec, exists := StandardPanels[specName]
	if !exists {
		return nil
	}

	// Build options slice
	opts := []PanelOption{
		spec.Position,
		Size(spec.Width, spec.Height),
		spec.Layout,
	}

	// Add padding option
	if spec.Custom != nil {
		opts = append(opts, CustomPadding(*spec.Custom))
	} else {
		opts = append(opts, Padding(spec.Padding))
	}

	return pb.BuildPanel(opts...)
}

// CreateStandardPanelWithOptions builds a panel from a specification and additional options.
// This allows for overriding or extending the standard spec.
//
// Example usage:
//
//	panel := CreateStandardPanelWithOptions(pb, "faction_info", WithTitle("Faction Status"))
func CreateStandardPanelWithOptions(pb *PanelBuilders, specName string, additionalOpts ...PanelOption) *widget.Container {
	spec, exists := StandardPanels[specName]
	if !exists {
		return nil
	}

	// Build options slice
	opts := []PanelOption{
		spec.Position,
		Size(spec.Width, spec.Height),
		spec.Layout,
	}

	// Add padding option
	if spec.Custom != nil {
		opts = append(opts, CustomPadding(*spec.Custom))
	} else {
		opts = append(opts, Padding(spec.Padding))
	}

	// Append additional options (these override spec options)
	opts = append(opts, additionalOpts...)

	return pb.BuildPanel(opts...)
}

// AddPanelSpec adds or updates a panel specification in the StandardPanels map.
// This allows dynamic panel registration for custom modes.
//
// Example usage:
//
//	AddPanelSpec("custom_panel", PanelLayoutSpec{
//	    Position: TopCenter(),
//	    Width:    specs.PanelWidthWide,
//	    Height:   specs.PanelHeightSmall,
//	    Padding:  specs.PaddingTight,
//	    Layout:   RowLayout(),
//	})
func AddPanelSpec(name string, spec PanelLayoutSpec) {
	spec.Name = name
	StandardPanels[name] = spec
}

// ListPanelSpecs returns a list of all available panel specification names.
// Useful for debugging and documentation.
func ListPanelSpecs() []string {
	specs := make([]string, 0, len(StandardPanels))
	for name := range StandardPanels {
		specs = append(specs, name)
	}
	return specs
}
