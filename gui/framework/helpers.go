// Package framework provides UI and mode system for the game
package framework

import (
	"game_main/gui/builders"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// CreateFilterButtonContainer creates a filter button container with consistent styling.
// Eliminates repetitive panel building for filter buttons across multiple modes.
// Returns an empty container ready for buttons to be added to it.
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
//   - alignment: Panel position (e.g., builders.TopLeft(), builders.TopRight())
//
// TODO, this will be removed in the future
func CreateFilterButtonContainer(panelBuilders *builders.PanelBuilders, alignment builders.PanelOption) *widget.Container {
	return panelBuilders.BuildPanel(
		alignment,
		builders.Padding(specs.PaddingStandard),
		builders.HorizontalRowLayout(),
	)
}
