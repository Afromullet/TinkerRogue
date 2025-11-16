package gui

import (
	"github.com/ebitenui/ebitenui/widget"
)

// PanelSpec defines a reusable panel configuration specification.
// This allows panels to be defined once and used consistently across factories.
type PanelSpec struct {
	Name     string            // Descriptive name for the panel
	Position PanelOption       // Position on screen (TopCenter, LeftCenter, etc.)
	Width    float64           // Width as percentage of screen width
	Height   float64           // Height as percentage of screen height
	Padding  float64           // Padding as percentage of screen size
	Layout   PanelOption       // Layout strategy (RowLayout, HorizontalRowLayout, etc.)
	Custom   *widget.Insets    // Optional: Override padding with custom insets
}

// StandardPanels defines common panel configurations used across the GUI package.
// This is the single source of truth for panel layouts - modify here to update everywhere.
//
// Currently used by:
// - CombatUIFactory: turn_order, faction_info, squad_list, squad_detail, action_buttons
// - ExplorationMode: stats_panel, message_log, quick_inventory
// - InfoMode: options_list
//
// Note: SquadBuilderUIFactory doesn't use this system as it creates widgets (List, TextArea)
// with custom manual layout positioning rather than using BuildPanel patterns.
var StandardPanels = map[string]PanelSpec{
	// ============================================
	// Combat UI Panels
	// ============================================
	"turn_order": {
		Name:     "Turn Order",
		Position: TopCenter(),
		Width:    PanelWidthWide,
		Height:   PanelHeightTiny,
		Padding:  PaddingTight,
		Layout:   HorizontalRowLayout(),
	},
	"faction_info": {
		Name:     "Faction Info",
		Position: TopLeft(),
		Width:    PanelWidthNarrow,
		Height:   PanelHeightSmall,
		Padding:  PaddingTight,
		Layout:   RowLayout(),
	},
	"squad_list": {
		Name:     "Squad List",
		Position: LeftCenter(),
		Width:    PanelWidthNarrow,
		Height:   PanelHeightHalf,
		Padding:  PaddingTight,
		Layout:   RowLayout(),
	},
	"squad_detail": {
		Name:     "Squad Detail",
		Position: LeftBottom(),
		Width:    PanelWidthNarrow,
		Height:   PanelHeightQuarter,
		Padding:  PaddingTight,
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
		Width:    PanelWidthNarrow,
		Height:   PanelHeightSmall,
		Padding:  PaddingTight,
		Layout:   RowLayout(),
	},
	"message_log": {
		Name:     "Message Log",
		Position: BottomRight(),
		Width:    PanelWidthNarrow,
		Height:   0.15,
		Padding:  PaddingTight,
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
		Width:    PanelWidthMedium,
		Height:   PanelHeightHalf,
		Padding:  0, // Uses default padding from BuildPanel
		Layout:   RowLayout(),
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
//	AddPanelSpec("custom_panel", PanelSpec{
//	    Position: TopCenter(),
//	    Width:    PanelWidthWide,
//	    Height:   PanelHeightSmall,
//	    Padding:  PaddingTight,
//	    Layout:   RowLayout(),
//	})
func AddPanelSpec(name string, spec PanelSpec) {
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
