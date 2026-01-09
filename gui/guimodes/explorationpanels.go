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

// CreateSquadManagementActionButtons builds the squad management mode buttons container (no panel wrapper, like combat mode)
func (epf *ExplorationPanelFactory) CreateSquadManagementActionButtons(
	onBattleMap func(),
	onSquadBuilder func(),
	onBuyUnits func(),
	onEditSquad func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(epf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(epf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Battle Map (ESC)", OnClick: onBattleMap},
			{Text: "Squad Builder (B)", OnClick: onSquadBuilder},
			{Text: "Buy Units (P)", OnClick: onBuyUnits},
			{Text: "Edit Squad (E)", OnClick: onEditSquad},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(epf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateUnitPurchaseActionButtons builds the unit purchase mode buttons container (no panel wrapper, like combat mode)
func (epf *ExplorationPanelFactory) CreateUnitPurchaseActionButtons(
	onBuyUnit func(),
	onUndo func(),
	onRedo func(),
	onBack func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(epf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(epf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Buy Unit", OnClick: onBuyUnit},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "Back (ESC)", OnClick: onBack},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(epf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadEditorActionButtons builds the squad editor mode buttons container (no panel wrapper, like combat mode)
func (epf *ExplorationPanelFactory) CreateSquadEditorActionButtons(
	onRenameSquad func(),
	onUndo func(),
	onRedo func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(epf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(epf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Rename Squad", OnClick: onRenameSquad},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "Close (ESC)", OnClick: onClose},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(epf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadDeploymentActionButtons builds the squad deployment mode buttons container (no panel wrapper, like combat mode)
func (epf *ExplorationPanelFactory) CreateSquadDeploymentActionButtons(
	onClearAll func(),
	onStartCombat func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(epf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(epf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Clear All", OnClick: onClearAll},
			{Text: "Start Combat", OnClick: onStartCombat},
			{Text: "Close (ESC)", OnClick: onClose},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(epf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}
