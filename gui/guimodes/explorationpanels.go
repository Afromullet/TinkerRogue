package guimodes

import (
	"game_main/gui/builders"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// ExplorationPanelFactory builds exploration and squad management UI components
type ExplorationPanelFactory struct {
	panelBuilders *builders.PanelBuilders
	layout        *specs.LayoutConfig
}

// NewExplorationPanelFactory creates a factory for exploration UI components
func NewExplorationPanelFactory(panelBuilders *builders.PanelBuilders, layout *specs.LayoutConfig) *ExplorationPanelFactory {
	return &ExplorationPanelFactory{
		panelBuilders: panelBuilders,
		layout:        layout,
	}
}

// CreateExplorationActionButtons builds the exploration mode buttons container (no panel wrapper, like combat mode)
func (epf *ExplorationPanelFactory) CreateExplorationActionButtons(
	onThrowables func(),
	onSquads func(),
	onInventory func(),
	onDeploy func(),
	onCombat func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(epf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(epf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Throwables", OnClick: onThrowables},
			{Text: "Squads (E)", OnClick: onSquads},
			{Text: "Inventory (I)", OnClick: onInventory},
			{Text: "Deploy (D)", OnClick: onDeploy},
			{Text: "Combat (C)", OnClick: onCombat},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(epf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}
