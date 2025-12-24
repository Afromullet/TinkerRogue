package guimodes

import (
	"fmt"
	"image/color"

	"game_main/world/coords"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/builders"
	"game_main/gui/specs"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// InfoMode displays entity/tile information when player right-clicks
type InfoMode struct {
	gui.BaseMode

	// UI Components
	optionsList    *widget.List
	detailTextArea *widgets.CachedTextAreaWrapper // Cached for performance

	// State
	inspectPosition coords.LogicalPosition
	selectedOption  string
}

// NewInfoMode creates a new info inspection mode
func NewInfoMode(modeManager *core.UIModeManager) *InfoMode {
	mode := &InfoMode{}
	mode.SetModeName("info_inspect")
	mode.SetReturnMode("exploration") // ESC returns to exploration
	mode.ModeManager = modeManager
	return mode
}

// Initialize sets up the info mode UI
func (im *InfoMode) Initialize(ctx *core.UIContext) error {
	err := gui.NewModeBuilder(&im.BaseMode, gui.ModeConfig{
		ModeName:   "info_inspect",
		ReturnMode: "exploration",

		Panels: []gui.PanelSpec{
			{CustomBuild: im.buildOptionsList},
			{CustomBuild: im.buildDetailPanel},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (im *InfoMode) buildOptionsList() *widget.Container {
	// Create options list (center-left)
	optionsList := builders.CreateListWithConfig(builders.ListConfig{
		Entries: []interface{}{"Look at Creature", "Look at Tile"},
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		OnEntrySelected: func(entry interface{}) {
			if option, ok := entry.(string); ok {
				im.handleOptionSelected(option)
			}
		},
		MinWidth:  int(float64(im.Layout.ScreenWidth) * 0.3),
		MinHeight: int(float64(im.Layout.ScreenHeight) * 0.3),
	})

	im.optionsList = optionsList

	// Wrap in container with LayoutData (center-left positioning)
	leftPad := int(float64(im.Layout.ScreenWidth) * specs.PaddingStandard)
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Left: leftPad,
			},
		})),
	)
	container.AddChild(optionsList)
	return container
}

func (im *InfoMode) buildDetailPanel() *widget.Container {
	// Right side detail panel (35% width, 60% height)
	panelWidth := int(float64(im.Layout.ScreenWidth) * 0.35)
	panelHeight := int(float64(im.Layout.ScreenHeight) * 0.6)

	detailPanel := builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(im.Layout, specs.PaddingTight)),
		),
	})

	rightPad := int(float64(im.Layout.ScreenWidth) * specs.PaddingStandard)
	detailPanel.GetWidget().LayoutData = gui.AnchorEndCenter(rightPad)

	// Detail text area - cached for performance
	detailTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
		MinWidth:  panelWidth - 30,
		MinHeight: panelHeight - 30,
		FontColor: color.White,
	})
	detailTextArea.SetText("Select an option to inspect") // SetText calls MarkDirty() internally
	detailPanel.AddChild(detailTextArea)

	im.detailTextArea = detailTextArea

	return detailPanel
}

// Enter is called when switching to this mode
func (im *InfoMode) Enter(fromMode core.UIMode) error {
	// Refresh detail display for current position
	im.refreshDetailDisplay()
	return nil
}

// Exit is called when switching from this mode
func (im *InfoMode) Exit(toMode core.UIMode) error {
	return nil
}

// HandleInput processes input for info mode
func (im *InfoMode) HandleInput(inputState *core.InputState) bool {
	// ESC handled by gui.BaseMode.HandleCommonInput
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
	// Use GUIQueries abstraction instead of direct ECS access
	creatureInfo := im.Queries.GetCreatureAtPosition(im.inspectPosition)

	if creatureInfo == nil {
		im.detailTextArea.SetText("No creature at this position")
		return
	}

	// Build display text from CreatureInfo
	entityType := "CREATURE"
	if creatureInfo.IsPlayer {
		entityType = "PLAYER"
	} else if creatureInfo.IsMonster {
		entityType = "MONSTER"
	}

	// Check if we have attribute data (MaxHP > 0 indicates full data)
	if creatureInfo.MaxHP == 0 {
		im.detailTextArea.SetText(fmt.Sprintf("=== %s ===\n\nName: %s\n\nNo attribute data available", entityType, creatureInfo.Name))
		return
	}

	details := fmt.Sprintf(
		"=== %s ===\n\nName: %s\n\nHP: %d/%d\nSTR: %d\nDEX: %d\nMAG: %d\nLDR: %d\nARM: %d\nWPN: %d\n",
		entityType,
		creatureInfo.Name,
		creatureInfo.CurrentHP,
		creatureInfo.MaxHP,
		creatureInfo.Strength,
		creatureInfo.Dexterity,
		creatureInfo.Magic,
		creatureInfo.Leadership,
		creatureInfo.Armor,
		creatureInfo.Weapon,
	)

	im.detailTextArea.SetText(details)
}

// displayTileInfo shows information about the tile at inspect position
func (im *InfoMode) displayTileInfo() {
	// Use GUIQueries abstraction instead of hardcoded values
	tileInfo := im.Queries.GetTileInfo(im.inspectPosition)

	// Build display text from TileInfo
	details := fmt.Sprintf(
		"=== TILE ===\n\nPosition: (%d, %d)\n\nType: %s\nMovement Cost: %d\nWalkable: %v\n",
		tileInfo.Position.X,
		tileInfo.Position.Y,
		tileInfo.TileType,
		tileInfo.MovementCost,
		tileInfo.IsWalkable,
	)

	// Add entity info if present
	if tileInfo.HasEntity {
		details += fmt.Sprintf("\nEntity Present: ID %d", tileInfo.EntityID)
	}

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
