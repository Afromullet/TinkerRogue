package guicombat

import (
	"fmt"
	"image/color"

	"game_main/gui"
	"game_main/gui/guicomponents"
	"game_main/gui/guiresources"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// CombatUIFactory builds combat UI panels and widgets
type CombatUIFactory struct {
	queries       *guicomponents.GUIQueries
	panelBuilders *widgets.PanelBuilders
	layout        *widgets.LayoutConfig
	width, height int
}

// NewCombatUIFactory creates a new combat UI factory
func NewCombatUIFactory(queries *guicomponents.GUIQueries, panelBuilders *widgets.PanelBuilders, layout *widgets.LayoutConfig) *CombatUIFactory {
	return &CombatUIFactory{
		queries:       queries,
		panelBuilders: panelBuilders,
		layout:        layout,
		width:         layout.ScreenWidth,
		height:        layout.ScreenHeight,
	}
}

// CreateTurnOrderPanel builds the turn order display panel
func (cuf *CombatUIFactory) CreateTurnOrderPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cuf.layout.ScreenWidth) * widgets.CombatTurnOrderWidth)
	panelHeight := int(float64(cuf.layout.ScreenHeight) * widgets.CombatTurnOrderHeight)

	// Create panel with horizontal row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(cuf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	topPad := int(float64(cuf.layout.ScreenHeight) * widgets.PaddingTight)
	panel.GetWidget().LayoutData = gui.AnchorCenterStart(topPad)

	return panel
}

// CreateFactionInfoPanel builds the faction information panel
func (cuf *CombatUIFactory) CreateFactionInfoPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cuf.layout.ScreenWidth) * widgets.CombatFactionInfoWidth)
	panelHeight := int(float64(cuf.layout.ScreenHeight) * widgets.CombatFactionInfoHeight)

	// Create panel with vertical row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(cuf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	leftPad := int(float64(cuf.layout.ScreenWidth) * widgets.PaddingTight)
	topPad := int(float64(cuf.layout.ScreenHeight) * widgets.PaddingTight)
	panel.GetWidget().LayoutData = gui.AnchorStartStart(leftPad, topPad)

	return panel
}

// CreateSquadListPanel builds the squad list panel
func (cuf *CombatUIFactory) CreateSquadListPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cuf.layout.ScreenWidth) * widgets.CombatSquadListWidth)
	panelHeight := int(float64(cuf.layout.ScreenHeight) * widgets.CombatSquadListHeight)

	// Create panel with vertical row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(cuf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below FactionInfo panel (which is 10% height + padding)
	leftPad := int(float64(cuf.layout.ScreenWidth) * widgets.PaddingTight)
	topOffset := int(float64(cuf.layout.ScreenHeight) * (widgets.CombatFactionInfoHeight + widgets.PaddingTight))
	panel.GetWidget().LayoutData = gui.AnchorStartStart(leftPad, topOffset)

	// Add label
	listLabel := widgets.CreateSmallLabel("Your Squads:")
	panel.AddChild(listLabel)

	return panel
}

// CreateSquadDetailPanel builds the squad detail panel
func (cuf *CombatUIFactory) CreateSquadDetailPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(cuf.layout.ScreenWidth) * widgets.CombatSquadDetailWidth)
	panelHeight := int(float64(cuf.layout.ScreenHeight) * widgets.CombatSquadDetailHeight)

	// Create panel with vertical row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(cuf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below SquadList panel (FactionInfo 10% + SquadList 35% + 3 padding gaps)
	leftPad := int(float64(cuf.layout.ScreenWidth) * widgets.PaddingTight)
	topOffset := int(float64(cuf.layout.ScreenHeight) * (widgets.CombatFactionInfoHeight + widgets.CombatSquadListHeight + widgets.PaddingTight*3))
	panel.GetWidget().LayoutData = gui.AnchorStartStart(leftPad, topOffset)

	return panel
}

// CreateLogPanel builds the combat log panel using standard specification
func (cuf *CombatUIFactory) CreateLogPanel() (*widget.Container, *widget.TextArea) {
	// Calculate responsive size
	panelWidth := int(float64(cuf.layout.ScreenWidth) * widgets.CombatLogWidth)
	panelHeight := int(float64(cuf.layout.ScreenHeight) * widgets.CombatLogHeight)

	// Create panel with anchor layout (to hold textarea)
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout:     widget.NewAnchorLayout(),
	})

	// Apply anchor layout positioning to panel
	// Position above action buttons (button height 8% + bottom offset 8% + padding)
	rightPad := int(float64(cuf.layout.ScreenWidth) * widgets.PaddingTight)
	bottomOffset := int(float64(cuf.layout.ScreenHeight) * (widgets.CombatActionButtonHeight + widgets.BottomButtonOffset + widgets.PaddingTight))
	panel.GetWidget().LayoutData = gui.AnchorEndEnd(rightPad, bottomOffset)

	// Create textarea to fit within panel
	textArea := widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})

	textArea.SetText("Combat started!\n")
	panel.AddChild(textArea)

	return panel, textArea
}

// CreateActionButtons builds the action buttons container
func (cuf *CombatUIFactory) CreateActionButtons(
	onAttack func(),
	onMove func(),
	onUndo func(),
	onRedo func(),
	onEndTurn func(),
	onFlee func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(cuf.layout.ScreenWidth) * widgets.PaddingTight)

	// Create button group using widgets.CreateButtonGroup with LayoutData
	bottomPad := int(float64(cuf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := gui.AnchorCenterEnd(bottomPad)

	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{Text: "Attack (A)", OnClick: onAttack},
			{Text: "Move (M)", OnClick: onMove},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "End Turn (Space)", OnClick: onEndTurn},
			{Text: "Flee (ESC)", OnClick: onFlee},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    gui.NewResponsiveHorizontalPadding(cuf.layout, widgets.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// GetFormattedSquadDetails returns formatted squad details as string
func (cuf *CombatUIFactory) GetFormattedSquadDetails(squadID interface{}) string {
	// This is a helper that formats squad info for display
	// The actual formatting is delegated to the calling code
	return "Select a squad\nto view details"
}

// GetFormattedFactionInfo returns formatted faction info as string
func (cuf *CombatUIFactory) GetFormattedFactionInfo(factionInfo interface{}) string {
	// This is a helper that formats faction info for display
	if fi, ok := factionInfo.(*guicomponents.FactionInfo); ok {
		infoText := fmt.Sprintf("%s\n", fi.Name)
		infoText += fmt.Sprintf("Squads: %d/%d\n", fi.AliveSquadCount, len(fi.SquadIDs))
		infoText += fmt.Sprintf("Mana: %d/%d", fi.CurrentMana, fi.MaxMana)
		return infoText
	}
	return "Faction Info"
}
