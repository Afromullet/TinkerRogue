package guisquads

import (
	"game_main/gui/builders"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// SquadPanelFactory builds squad management UI components (action buttons, etc.)
type SquadPanelFactory struct {
	panelBuilders *builders.PanelBuilders
	layout        *specs.LayoutConfig
}

// NewSquadPanelFactory creates a factory for squad management UI components
func NewSquadPanelFactory(panelBuilders *builders.PanelBuilders, layout *specs.LayoutConfig) *SquadPanelFactory {
	return &SquadPanelFactory{
		panelBuilders: panelBuilders,
		layout:        layout,
	}
}

// CreateSquadManagementActionButtons builds the squad management mode buttons container
func (spf *SquadPanelFactory) CreateSquadManagementActionButtons(
	onBattleMap func(),
	onSquadBuilder func(),
	onBuyUnits func(),
	onEditSquad func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(spf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(spf.layout.ScreenHeight) * specs.BottomButtonOffset)
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
		Padding:    builders.NewResponsiveHorizontalPadding(spf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateUnitPurchaseActionButtons builds the unit purchase mode buttons container
func (spf *SquadPanelFactory) CreateUnitPurchaseActionButtons(
	onBuyUnit func(),
	onUndo func(),
	onRedo func(),
	onBack func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(spf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(spf.layout.ScreenHeight) * specs.BottomButtonOffset)
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
		Padding:    builders.NewResponsiveHorizontalPadding(spf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadEditorActionButtons builds the squad editor mode buttons container
func (spf *SquadPanelFactory) CreateSquadEditorActionButtons(
	onRenameSquad func(),
	onUndo func(),
	onRedo func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(spf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(spf.layout.ScreenHeight) * specs.BottomButtonOffset)
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
		Padding:    builders.NewResponsiveHorizontalPadding(spf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadDeploymentActionButtons builds the squad deployment mode buttons container
func (spf *SquadPanelFactory) CreateSquadDeploymentActionButtons(
	onClearAll func(),
	onStartCombat func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(spf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(spf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Clear All", OnClick: onClearAll},
			{Text: "Start Combat", OnClick: onStartCombat},
			{Text: "Close (ESC)", OnClick: onClose},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(spf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}
