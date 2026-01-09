package guicombat

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/guicomponents"
	"game_main/gui/guiresources"
	"game_main/gui/specs"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// CombatPanelFactory builds combat-specific UI panels and components
type CombatPanelFactory struct {
	queries       *guicomponents.GUIQueries
	panelBuilders *builders.PanelBuilders
	layout        *specs.LayoutConfig
}

// NewCombatPanelFactory creates a factory for combat UI components
func NewCombatPanelFactory(queries *guicomponents.GUIQueries, panelBuilders *builders.PanelBuilders, layout *specs.LayoutConfig) *CombatPanelFactory {
	return &CombatPanelFactory{
		queries:       queries,
		panelBuilders: panelBuilders,
		layout:        layout,
	}
}

// CreateCombatTurnOrderPanel builds the turn order display panel
func (cpf *CombatPanelFactory) CreateCombatTurnOrderPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cpf.layout.ScreenWidth) * specs.CombatTurnOrderWidth)
	panelHeight := int(float64(cpf.layout.ScreenHeight) * specs.CombatTurnOrderHeight)

	// Create panel with horizontal row layout
	panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(cpf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	topPad := int(float64(cpf.layout.ScreenHeight) * specs.PaddingTight)
	panel.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

	return panel
}

// CreateCombatFactionInfoPanel builds the faction information panel
func (cpf *CombatPanelFactory) CreateCombatFactionInfoPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cpf.layout.ScreenWidth) * specs.CombatFactionInfoWidth)
	panelHeight := int(float64(cpf.layout.ScreenHeight) * specs.CombatFactionInfoHeight)

	// Create panel with vertical row layout
	panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(cpf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	leftPad := int(float64(cpf.layout.ScreenWidth) * specs.PaddingTight)
	topPad := int(float64(cpf.layout.ScreenHeight) * specs.PaddingTight)
	panel.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topPad)

	return panel
}

// CreateCombatSquadListPanel builds the squad list panel
func (cpf *CombatPanelFactory) CreateCombatSquadListPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cpf.layout.ScreenWidth) * specs.CombatSquadListWidth)
	panelHeight := int(float64(cpf.layout.ScreenHeight) * specs.CombatSquadListHeight)

	// Create panel with vertical row layout
	panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(cpf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below FactionInfo panel (which is 10% height + padding)
	leftPad := int(float64(cpf.layout.ScreenWidth) * specs.PaddingTight)
	topOffset := int(float64(cpf.layout.ScreenHeight) * (specs.CombatFactionInfoHeight + specs.PaddingTight))
	panel.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)

	// Add label
	listLabel := builders.CreateSmallLabel("Your Squads:")
	panel.AddChild(listLabel)

	return panel
}

// CreateCombatSquadDetailPanel builds the squad detail panel
func (cpf *CombatPanelFactory) CreateCombatSquadDetailPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cpf.layout.ScreenWidth) * specs.CombatSquadDetailWidth)
	panelHeight := int(float64(cpf.layout.ScreenHeight) * specs.CombatSquadDetailHeight)

	// Create panel with vertical row layout
	panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(cpf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below SquadList panel (FactionInfo 10% + SquadList 35% + 3 padding gaps)
	leftPad := int(float64(cpf.layout.ScreenWidth) * specs.PaddingTight)
	topOffset := int(float64(cpf.layout.ScreenHeight) * (specs.CombatFactionInfoHeight + specs.CombatSquadListHeight + specs.PaddingTight*3))
	panel.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)

	return panel
}

// CreateCombatLogPanel builds the combat log panel using standard specification
func (cpf *CombatPanelFactory) CreateCombatLogPanel() (*widget.Container, *widgets.CachedTextAreaWrapper) {
	// Calculate responsive size
	panelWidth := int(float64(cpf.layout.ScreenWidth) * specs.CombatLogWidth)
	panelHeight := int(float64(cpf.layout.ScreenHeight) * specs.CombatLogHeight)

	// Create panel with anchor layout (to hold textarea)
	panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout:     widget.NewAnchorLayout(),
	})

	// Apply anchor layout positioning to panel
	// Position above action buttons (button height 8% + bottom offset 8% + padding)
	rightPad := int(float64(cpf.layout.ScreenWidth) * specs.PaddingTight)
	bottomOffset := int(float64(cpf.layout.ScreenHeight) * (specs.CombatActionButtonHeight + specs.BottomButtonOffset + specs.PaddingTight))
	panel.GetWidget().LayoutData = builders.AnchorEndEnd(rightPad, bottomOffset)

	// Create cached textarea to fit within panel - only re-renders when combat log updates
	textArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})

	textArea.SetText("Combat started!\n") // SetText calls MarkDirty() internally
	panel.AddChild(textArea)              // The wrapper implements the necessary widget interfaces

	return panel, textArea
}

// CreateCombatActionButtons builds the action buttons container
func (cpf *CombatPanelFactory) CreateCombatActionButtons(
	onAttack func(),
	onMove func(),
	onUndo func(),
	onRedo func(),
	onEndTurn func(),
	onFlee func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(cpf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(cpf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Attack (A)", OnClick: onAttack},
			{Text: "Move (M)", OnClick: onMove},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "End Turn (Space)", OnClick: onEndTurn},
			{Text: "Flee (ESC)", OnClick: onFlee},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(cpf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// GetFormattedSquadDetails returns formatted squad details as string
func (cpf *CombatPanelFactory) GetFormattedSquadDetails(squadID interface{}) string {
	// This is a helper that formats squad info for display
	// The actual formatting is delegated to the calling code
	return "Select a squad\nto view details"
}

// GetFormattedFactionInfo returns formatted faction info as string
func (cpf *CombatPanelFactory) GetFormattedFactionInfo(factionInfo interface{}) string {
	// This is a helper that formats faction info for display
	if fi, ok := factionInfo.(*guicomponents.FactionInfo); ok {
		infoText := fmt.Sprintf("%s\n", fi.Name)
		infoText += fmt.Sprintf("Squads: %d/%d\n", fi.AliveSquadCount, len(fi.SquadIDs))
		infoText += fmt.Sprintf("Mana: %d/%d", fi.CurrentMana, fi.MaxMana)
		return infoText
	}
	return "Faction Info"
}
