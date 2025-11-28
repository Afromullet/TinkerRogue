package guicombat

import (
	"game_main/gui"
	"game_main/gui/guicomponents"
	"game_main/gui/widgets"

	"fmt"

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
	return widgets.CreateStandardPanel(cuf.panelBuilders, "turn_order")
}

// CreateFactionInfoPanel builds the faction information panel
func (cuf *CombatUIFactory) CreateFactionInfoPanel() *widget.Container {
	return widgets.CreateStandardPanel(cuf.panelBuilders, "faction_info")
}

// CreateSquadListPanel builds the squad list panel
func (cuf *CombatUIFactory) CreateSquadListPanel() *widget.Container {
	panel := widgets.CreateStandardPanel(cuf.panelBuilders, "squad_list")

	listLabel := widgets.CreateSmallLabel("Your Squads:")
	panel.AddChild(listLabel)

	return panel
}

// CreateSquadDetailPanel builds the squad detail panel
func (cuf *CombatUIFactory) CreateSquadDetailPanel() *widget.Container {
	return widgets.CreateStandardPanel(cuf.panelBuilders, "squad_detail")
}

// CreateLogPanel builds the combat log panel using standard specification
func (cuf *CombatUIFactory) CreateLogPanel() (*widget.Container, *widget.TextArea) {
	return gui.CreateStandardDetailPanel(
		cuf.panelBuilders,
		cuf.layout,
		"combat_log",
		"Combat started!\n",
	)
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
	// Build action buttons using helper (consolidates positioning + button creation)
	return gui.CreateActionButtonGroup(
		cuf.panelBuilders,
		widgets.BottomCenter(),
		[]widgets.ButtonSpec{
			{
				Text:    "Attack (A)",
				OnClick: onAttack,
			},
			{
				Text:    "Move (M)",
				OnClick: onMove,
			},
			{
				Text:    "Undo (Ctrl+Z)",
				OnClick: onUndo,
			},
			{
				Text:    "Redo (Ctrl+Y)",
				OnClick: onRedo,
			},
			{
				Text:    "End Turn (Space)",
				OnClick: onEndTurn,
			},
			{
				Text:    "Flee (ESC)",
				OnClick: onFlee,
			},
		},
	)
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
