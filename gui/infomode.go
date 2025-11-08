package gui

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
)

// InfoMode displays entity/tile information when player right-clicks
type InfoMode struct {
	BaseMode

	// UI Components
	optionsList    *widget.List
	detailTextArea *widget.TextArea

	// State
	inspectPosition coords.LogicalPosition
	selectedOption  string
}

// NewInfoMode creates a new info inspection mode
func NewInfoMode(modeManager *UIModeManager) *InfoMode {
	return &InfoMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "info_inspect",
			returnMode:  "exploration", // ESC returns to exploration
		},
	}
}

// Initialize sets up the info mode UI
func (im *InfoMode) Initialize(ctx *UIContext) error {
	im.InitializeBase(ctx)

	// Build options panel using modern pattern
	optionsPanel := im.panelBuilders.BuildPanel(
		Center(),
		Size(0.3, 0.4),
		RowLayout(),
	)

	// Options list
	options := []interface{}{"Look at Creature", "Look at Tile"}
	im.optionsList = CreateListWithConfig(ListConfig{
		Entries: options,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		OnEntrySelected: func(entry interface{}) {
			if option, ok := entry.(string); ok {
				im.handleOptionSelected(option)
			}
		},
	})
	optionsPanel.AddChild(im.optionsList)
	im.rootContainer.AddChild(optionsPanel)

	// Detail panel using modern pattern
	detailPanel := im.panelBuilders.BuildPanel(
		RightCenter(),
		Size(0.4, 0.6),
		AnchorLayout(),
	)

	im.detailTextArea = CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  int(float64(im.layout.ScreenWidth) * 0.38),
		MinHeight: int(float64(im.layout.ScreenHeight) * 0.58),
		FontColor: color.White,
	})
	im.detailTextArea.SetText("Select an option to inspect")
	detailPanel.AddChild(im.detailTextArea)
	im.rootContainer.AddChild(detailPanel)

	return nil
}

// Enter is called when switching to this mode
func (im *InfoMode) Enter(fromMode UIMode) error {
	// Refresh detail display for current position
	im.refreshDetailDisplay()
	return nil
}

// Exit is called when switching from this mode
func (im *InfoMode) Exit(toMode UIMode) error {
	return nil
}

// HandleInput processes input for info mode
func (im *InfoMode) HandleInput(inputState *InputState) bool {
	// ESC handled by BaseMode.HandleCommonInput
	if im.HandleCommonInput(inputState) {
		return true
	}
	return false
}

// handleOptionSelected processes when user selects an option from the list
func (im *InfoMode) handleOptionSelected(option string) {
	im.selectedOption = option

	switch option {
	case "Look at Creature":
		im.displayCreatureInfo()
	case "Look at Tile":
		im.displayTileInfo()
	}
}

// displayCreatureInfo shows information about the creature at inspect position
func (im *InfoMode) displayCreatureInfo() {
	creature := common.GetCreatureAtPosition(im.context.ECSManager, &im.inspectPosition)

	if creature == nil {
		im.detailTextArea.SetText("No creature at this position")
		return
	}

	// Get creature details
	name := "Unknown"
	if nameComp, ok := im.context.ECSManager.GetComponent(creature.GetID(), common.NameComponent); ok {
		if nameData, ok := nameComp.(*common.Name); ok {
			name = nameData.NameStr
		}
	}

	// Get attributes
	attrs := common.GetComponentType[*common.Attributes](creature, common.AttributeComponent)
	if attrs == nil {
		im.detailTextArea.SetText(fmt.Sprintf("=== CREATURE ===\n\nName: %s\n\nNo attribute data available", name))
		return
	}

	details := fmt.Sprintf(
		"=== CREATURE ===\n\nName: %s\n\nHP: %d/%d\nSTR: %d\nDEX: %d\nMAG: %d\n",
		name,
		attrs.CurrentHealth,
		attrs.MaxHealth,
		attrs.Strength,
		attrs.Dexterity,
		attrs.Magic,
	)

	im.detailTextArea.SetText(details)
}

// displayTileInfo shows information about the tile at inspect position
func (im *InfoMode) displayTileInfo() {
	// Query tile properties at position
	// Note: This is a simplified implementation - extend based on your tile system
	details := fmt.Sprintf(
		"=== TILE ===\n\nPosition: (%d, %d)\n\nType: Floor\nMovement Cost: 1\n",
		im.inspectPosition.X,
		im.inspectPosition.Y,
	)

	im.detailTextArea.SetText(details)
}

// SetInspectPosition is called from ExplorationMode before transitioning
func (im *InfoMode) SetInspectPosition(pos coords.LogicalPosition) {
	im.inspectPosition = pos
}

// refreshDetailDisplay updates the display if an option was previously selected
func (im *InfoMode) refreshDetailDisplay() {
	if im.selectedOption != "" {
		im.handleOptionSelected(im.selectedOption)
	}
}
