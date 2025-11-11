package gui

import (
	"fmt"
	"image/color"

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
	panel := cuf.panelBuilders.BuildPanel(
		TopCenter(),
		Size(0.4, 0.08),
		Padding(0.01),
		HorizontalRowLayout(),
	)
	return panel
}

// CreateFactionInfoPanel builds the faction information panel
func (cuf *CombatUIFactory) CreateFactionInfoPanel() *widget.Container {
	panel := cuf.panelBuilders.BuildPanel(
		TopLeft(),
		Size(0.15, 0.12),
		Padding(0.01),
		RowLayout(),
	)
	return panel
}

// CreateSquadListPanel builds the squad list panel
func (cuf *CombatUIFactory) CreateSquadListPanel() *widget.Container {
	panel := cuf.panelBuilders.BuildPanel(
		LeftCenter(),
		Size(0.15, 0.5),
		Padding(0.01),
		RowLayout(),
	)

	listLabel := CreateTextWithConfig(TextConfig{
		Text:     "Your Squads:",
		FontFace: SmallFace,
		Color:    color.White,
	})
	panel.AddChild(listLabel)

	return panel
}

// CreateSquadDetailPanel builds the squad detail panel
func (cuf *CombatUIFactory) CreateSquadDetailPanel() *widget.Container {
	panel := cuf.panelBuilders.BuildPanel(
		LeftBottom(),
		Size(0.15, 0.25),
		CustomPadding(widget.Insets{
			Left:   int(float64(cuf.layout.ScreenWidth) * 0.01),
			Bottom: int(float64(cuf.layout.ScreenHeight) * 0.15),
		}),
		RowLayout(),
	)

	return panel
}

// CreateLogPanel builds the combat log panel
func (cuf *CombatUIFactory) CreateLogPanel() (*widget.Container, *widget.TextArea) {
	logContainer := cuf.panelBuilders.BuildPanel(
		RightCenter(),
		Size(0.2, 0.85),
		Padding(0.01),
		AnchorLayout(),
	)

	// Create combat log text area
	logWidth := int(float64(cuf.layout.ScreenWidth) * 0.2)
	logHeight := cuf.layout.ScreenHeight - int(float64(cuf.layout.ScreenHeight)*0.15)
	logArea := CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  logWidth - 20,
		MinHeight: logHeight - 20,
		FontColor: color.White,
	})
	logArea.SetText("Combat started!\n")
	logContainer.AddChild(logArea)

	return logContainer, logArea
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
	actionButtons := cuf.panelBuilders.BuildPanel(
		BottomCenter(),
		HorizontalRowLayout(),
		CustomPadding(widget.Insets{
			Bottom: int(float64(cuf.layout.ScreenHeight) * 0.08),
		}),
	)

	actionButtons.AddChild(attackButton)
	actionButtons.AddChild(moveButton)
	actionButtons.AddChild(endTurnBtn)
	actionButtons.AddChild(fleeBtn)

	return actionButtons
}

// CreateTurnOrderLabel creates a text label for turn order display
func (cuf *CombatUIFactory) CreateTurnOrderLabel(text string) *widget.Text {
	return CreateTextWithConfig(TextConfig{
		Text:     text,
		FontFace: LargeFace,
		Color:    color.White,
	})
}

// CreateFactionInfoText creates a text label for faction info display
func (cuf *CombatUIFactory) CreateFactionInfoText(text string) *widget.Text {
	return CreateTextWithConfig(TextConfig{
		Text:     text,
		FontFace: SmallFace,
		Color:    color.White,
	})
}

// CreateSquadDetailText creates a text label for squad detail display
func (cuf *CombatUIFactory) CreateSquadDetailText(text string) *widget.Text {
	return CreateTextWithConfig(TextConfig{
		Text:     text,
		FontFace: SmallFace,
		Color:    color.White,
	})
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
