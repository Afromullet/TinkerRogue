package gui

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
)

// CombatUIFactory builds combat UI panels and widgets
type CombatUIFactory struct {
	queries       *GUIQueries
	panelBuilders *PanelBuilders
	layout        *LayoutConfig
	width, height int
}

// NewCombatUIFactory creates a new combat UI factory
func NewCombatUIFactory(queries *GUIQueries, panelBuilders *PanelBuilders, layout *LayoutConfig) *CombatUIFactory {
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
	return CreateStandardPanel(cuf.panelBuilders, "turn_order")
}

// CreateFactionInfoPanel builds the faction information panel
func (cuf *CombatUIFactory) CreateFactionInfoPanel() *widget.Container {
	return CreateStandardPanel(cuf.panelBuilders, "faction_info")
}

// CreateSquadListPanel builds the squad list panel
func (cuf *CombatUIFactory) CreateSquadListPanel() *widget.Container {
	panel := CreateStandardPanel(cuf.panelBuilders, "squad_list")

	listLabel := CreateSmallLabel("Your Squads:")
	panel.AddChild(listLabel)

	return panel
}

// CreateSquadDetailPanel builds the squad detail panel
func (cuf *CombatUIFactory) CreateSquadDetailPanel() *widget.Container {
	return CreateStandardPanel(cuf.panelBuilders, "squad_detail")
}

// CreateLogPanel builds the combat log panel using the detail panel helper
func (cuf *CombatUIFactory) CreateLogPanel() (*widget.Container, *widget.TextArea) {
	return CreateDetailPanel(
		cuf.panelBuilders,
		cuf.layout,
		RightCenter(),
		PanelWidthExtraWide, CombatLogHeight, PaddingTight,
		"Combat started!\n",
	)
}

// CreateActionButtons builds the action buttons container
func (cuf *CombatUIFactory) CreateActionButtons(
	onAttack func(),
	onMove func(),
	onEndTurn func(),
	onFlee func(),
) *widget.Container {
	attackButton := CreateButtonWithConfig(ButtonConfig{
		Text: "Attack (A)",
		OnClick: func() {
			if onAttack != nil {
				onAttack()
			}
		},
	})

	moveButton := CreateButtonWithConfig(ButtonConfig{
		Text: "Move (M)",
		OnClick: func() {
			if onMove != nil {
				onMove()
			}
		},
	})

	endTurnBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "End Turn (Space)",
		OnClick: func() {
			if onEndTurn != nil {
				onEndTurn()
			}
		},
	})

	fleeBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Flee (ESC)",
		OnClick: func() {
			if onFlee != nil {
				onFlee()
			}
		},
	})

	// Build action buttons container
	actionButtons := CreateStandardPanelWithOptions(
		cuf.panelBuilders,
		"action_buttons",
		CustomPadding(widget.Insets{
			Bottom: int(float64(cuf.layout.ScreenHeight) * BottomButtonOffset),
		}),
	)

	actionButtons.AddChild(attackButton)
	actionButtons.AddChild(moveButton)
	actionButtons.AddChild(endTurnBtn)
	actionButtons.AddChild(fleeBtn)

	return actionButtons
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
	if fi, ok := factionInfo.(*FactionInfo); ok {
		infoText := fmt.Sprintf("%s\n", fi.Name)
		infoText += fmt.Sprintf("Squads: %d/%d\n", fi.AliveSquadCount, len(fi.SquadIDs))
		infoText += fmt.Sprintf("Mana: %d/%d", fi.CurrentMana, fi.MaxMana)
		return infoText
	}
	return "Faction Info"
}
