package gui

import (
	"fmt"
	"game_main/common"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadManagementMode shows all squads with detailed information
type SquadManagementMode struct {
	BaseMode // Embed common mode infrastructure

	squadPanels       []*SquadPanel // One panel per squad
	closeButton       *widget.Button
	squadPanelComponent *PanelListComponent // Component managing squad panel refresh
}

// SquadPanel represents a single squad's UI panel
type SquadPanel struct {
	container    *widget.Container
	squadID      ecs.EntityID
	gridDisplay  *widget.TextArea // Shows 3x3 grid visualization
	statsDisplay *widget.TextArea // Shows squad stats
	unitList     *widget.List     // Shows individual units
}

func NewSquadManagementMode(modeManager *UIModeManager) *SquadManagementMode {
	return &SquadManagementMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "squad_management",
			returnMode:  "exploration",
		},
		squadPanels: make([]*SquadPanel, 0),
	}
}

func (smm *SquadManagementMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure (required for queries field)
	smm.InitializeBase(ctx)

	// Register hotkey for mode transition (back to exploration)
	smm.RegisterHotkey(ebiten.KeyE, "exploration")

	// Override root container with grid layout for multiple squad panels
	smm.ui = &ebitenui.UI{}
	smm.rootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2), // 2 squads per row
			widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true, true}),
			widget.GridLayoutOpts.Spacing(10, 10),
			widget.GridLayoutOpts.Padding(widget.Insets{
				Left: 20, Right: 20, Top: 20, Bottom: 80, // Extra bottom for close button
			}),
		)),
	)
	smm.ui.Container = smm.rootContainer

	// Build close button (bottom-center) using helper
	closeButtonContainer := CreateBottomCenterButtonContainer(smm.panelBuilders)
	closeBtn := CreateCloseButton(smm.modeManager, "exploration", "Close (ESC)")
	closeButtonContainer.AddChild(closeBtn)
	smm.ui.Container.AddChild(closeButtonContainer)

	// Initialize panel list component to manage squad panels
	smm.squadPanelComponent = NewPanelListComponent(
		smm.rootContainer,
		smm.queries,
		smm.panelBuilders,
		func(queries *GUIQueries, squadID ecs.EntityID) *widget.Container {
			panel := smm.createSquadPanel(squadID)
			return panel.container
		},
		func(squadID ecs.EntityID) bool {
			// Show all squads
			return true
		},
	)

	return nil
}

func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Squad Management Mode")

	// Refresh squad panels using component
	smm.squadPanelComponent.Refresh()

	return nil
}

func (smm *SquadManagementMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Squad Management Mode")

	// Clear panels using component
	smm.squadPanelComponent.Clear()

	return nil
}

func (smm *SquadManagementMode) createSquadPanel(squadID ecs.EntityID) *SquadPanel {
	panel := &SquadPanel{
		squadID: squadID,
	}

	// Container for this squad's panel
	panel.container = CreatePanelWithConfig(PanelConfig{
		Background: PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 15, Right: 15, Top: 15, Bottom: 15,
			}),
		),
	})

	// Squad name label - use unified query service
	squadName := smm.queries.GetSquadName(squadID)
	nameLabel := CreateLargeLabel(fmt.Sprintf("Squad: %s", squadName))
	panel.container.AddChild(nameLabel)

	// 3x3 grid visualization (using squad system's VisualizeSquad function)
	gridVisualization := squads.VisualizeSquad(squadID, smm.context.ECSManager)
	gridConfig := TextAreaConfig{
		MinWidth:  300,
		MinHeight: 200,
		FontColor: color.White,
	}
	panel.gridDisplay = CreateTextAreaWithConfig(gridConfig)
	panel.gridDisplay.SetText(gridVisualization)
	panel.container.AddChild(panel.gridDisplay)

	// Squad stats display
	statsConfig := TextAreaConfig{
		MinWidth:  300,
		MinHeight: 100,
		FontColor: color.White,
	}
	panel.statsDisplay = CreateTextAreaWithConfig(statsConfig)
	panel.statsDisplay.SetText(smm.getSquadStats(squadID))
	panel.container.AddChild(panel.statsDisplay)

	// Unit list (clickable for details)
	panel.unitList = smm.createUnitList(squadID)
	panel.container.AddChild(panel.unitList)

	return panel
}

func (smm *SquadManagementMode) createUnitList(squadID ecs.EntityID) *widget.List {
	// Get all units in this squad (using squad system query)
	unitIDs := squads.GetUnitIDsInSquad(squadID, smm.context.ECSManager)

	// Create list entries
	entries := make([]interface{}, 0, len(unitIDs))
	for _, unitID := range unitIDs {
		// Get unit attributes (units use common.Attributes, not separate UnitData)
		if attrRaw, ok := smm.context.ECSManager.GetComponent(unitID, common.AttributeComponent); ok {
			attr := attrRaw.(*common.Attributes)
			// Get unit name
			nameStr := "Unknown"
			if nameRaw, ok := smm.context.ECSManager.GetComponent(unitID, common.NameComponent); ok {
				name := nameRaw.(*common.Name)
				nameStr = name.NameStr
			}
			entries = append(entries, fmt.Sprintf("%s - HP: %d/%d", nameStr, attr.CurrentHealth, attr.MaxHealth))
		}
	}

	// Create list widget using exported resources
	list := CreateListWithConfig(ListConfig{
		Entries: entries,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
	})

	return list
}

func (smm *SquadManagementMode) getSquadStats(squadID ecs.EntityID) string {
	// Use unified query service to get squad stats
	squadInfo := smm.queries.GetSquadInfo(squadID)
	if squadInfo == nil {
		return "Squad not found"
	}

	return fmt.Sprintf("Units: %d\nTotal HP: %d/%d\nMorale: N/A", squadInfo.TotalUnits, squadInfo.CurrentHP, squadInfo.MaxHP)
}

func (smm *SquadManagementMode) Update(deltaTime float64) error {
	// Could refresh squad data periodically
	// For now, data is static until mode is re-entered
	return nil
}

func (smm *SquadManagementMode) Render(screen *ebiten.Image) {
	// No custom rendering - ebitenui draws everything
}

func (smm *SquadManagementMode) HandleInput(inputState *InputState) bool {
	// Handle common input (ESC key)
	if smm.HandleCommonInput(inputState) {
		return true
	}

	// E key hotkey is now handled by BaseMode.HandleCommonInput via RegisterHotkey
	return false
}

