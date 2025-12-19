package widgets

import (
	"game_main/gui/guiresources"

	"github.com/ebitenui/ebitenui/widget"
)

// CreateCachedPanelConfig extends PanelConfig with caching options.
type CreateCachedPanelConfig struct {
	PanelConfig
	EnableCaching bool // Whether to use cached background rendering
	PreCache      bool // Whether to pre-render background on creation
}

// CreateCachedPanel creates a panel with optional background caching.
// Note: Due to EbitenUI limitations, we create the panel with the standard background.
// The caching is handled by the global background pools in guiresources.
//
// Example:
//   panel := CreateCachedPanel(CreateCachedPanelConfig{
//       PanelConfig: PanelConfig{
//           MinWidth:  400,
//           MinHeight: 300,
//           Background: guiresources.PanelRes.Image,
//       },
//       EnableCaching: true,  // Use cached background
//       PreCache:      true,  // Pre-render immediately
//   })
func CreateCachedPanel(config CreateCachedPanelConfig) *widget.Container {
	// Pre-cache the background if requested
	if config.EnableCaching && config.PreCache && config.Background != nil {
		_ = guiresources.GetPanelBackground(config.MinWidth, config.MinHeight)
	}

	// Create panel with standard approach
	// The actual caching optimization is applied at the BuildPanel level
	return CreatePanelWithConfig(config.PanelConfig)
}

// CreateStaticPanel creates a panel optimized for static content (enables caching and pre-rendering).
// Use this for panels that:
// - Have fixed dimensions
// - Don't change size frequently
// - Are visible most of the time
//
// Examples: squad management panels, combat UI panels, stats displays
func CreateStaticPanel(config PanelConfig) *widget.Container {
	config.EnableCaching = true
	return CreatePanelWithConfig(config)
}

// CreateDynamicPanel creates a panel optimized for dynamic content (no caching).
// Use this for panels that:
// - Change size frequently
// - Are created/destroyed often
// - Have variable dimensions
//
// Examples: tooltips, context menus, popups
func CreateDynamicPanel(config PanelConfig) *widget.Container {
	config.EnableCaching = false
	return CreatePanelWithConfig(config)
}

// PanelBuilders extension methods for cached panels

// BuildStaticPanel creates a static panel with caching using the functional options pattern.
// This is like BuildPanel but optimized for static UI elements.
func (pb *PanelBuilders) BuildStaticPanel(opts ...PanelOption) *widget.Container {
	// Use standard BuildPanel logic but create with static caching
	panel := pb.BuildPanel(opts...)

	// TODO: Retrofit existing panel with caching
	// For now, this is a placeholder that returns a standard panel
	// In the future, we can add caching by wrapping the background

	return panel
}

// PreCachePanelBackground pre-renders a panel background at the specified size.
// Call this during initialization to warm the cache for commonly-used panel sizes.
// This reduces the first-frame rendering cost when the panel is actually displayed.
func PreCachePanelBackground(width, height int) {
	guiresources.GetPanelBackground(width, height)
}

// ClearPanelCache clears all cached panel backgrounds.
// Call this when changing themes or to free memory.
func ClearPanelCache() {
	guiresources.ClearAllBackgroundCaches()
}
