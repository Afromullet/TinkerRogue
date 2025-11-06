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
	ui          *ebitenui.UI
	context     *UIContext
	layout      *LayoutConfig
	modeManager *UIModeManager

	rootContainer *widget.Container
	squadPanels   []*SquadPanel // One panel per squad
	closeButton   *widget.Button

	// Panel builders for UI composition
	panelBuilders *PanelBuilders
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
		modeManager: modeManager,
		squadPanels: make([]*SquadPanel, 0),
	}
}

func (smm *SquadManagementMode) Initialize(ctx *UIContext) error {
	smm.context = ctx
	smm.layout = NewLayoutConfig(ctx)
	smm.panelBuilders = NewPanelBuilders(smm.layout, smm.modeManager)

	// Create ebitenui root with grid layout for multiple squad panels
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

	// Build close button (bottom-center)
	smm.buildCloseButton()

	return nil
}

func (smm *SquadManagementMode) buildCloseButton() {
	// Use panel builder for close button
	closeButtonContainer := smm.panelBuilders.BuildCloseButton("exploration", "Close (ESC)")

	// Add to root (not grid layout, so it floats)
	smm.ui.Container.AddChild(closeButtonContainer)
}

func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Squad Management Mode")

	// Clear old panels
	smm.clearSquadPanels()

	// Find all squads in the game
	allSquads := smm.findAllSquads()

	// Create panel for each squad
	for _, squadID := range allSquads {
		panel := smm.createSquadPanel(squadID)
		smm.squadPanels = append(smm.squadPanels, panel)
		smm.rootContainer.AddChild(panel.container)
	}

	return nil
}

func (smm *SquadManagementMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Squad Management Mode")

	// Clean up panels (will be rebuilt on next Enter)
	smm.clearSquadPanels()

	return nil
}

func (smm *SquadManagementMode) clearSquadPanels() {
	for _, panel := range smm.squadPanels {
		smm.rootContainer.RemoveChild(panel.container)
	}
	smm.squadPanels = smm.squadPanels[:0] // Clear slice
}

func (smm *SquadManagementMode) findAllSquads() []ecs.EntityID {
	// Query ECS for all entities with SquadData component
	// Uses common.EntityManager wrapper methods
	allSquads := make([]ecs.EntityID, 0)

	// Iterate through all entities
	entityIDs := smm.context.ECSManager.GetAllEntities()
	for _, entityID := range entityIDs {
		// Check if entity has SquadData component
		if smm.context.ECSManager.HasComponent(entityID, squads.SquadComponent) {
			allSquads = append(allSquads, entityID)
		}
	}

	return allSquads
}

func (smm *SquadManagementMode) createSquadPanel(squadID ecs.EntityID) *SquadPanel {
	panel := &SquadPanel{
		squadID: squadID,
	}

	// Container for this squad's panel
	panel.container = CreatePanelWithConfig(PanelConfig{
		Background: PanelRes.image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 15, Right: 15, Top: 15, Bottom: 15,
			}),
		),
	})

	// Squad name label - get component data using common.EntityManager
	if squadDataRaw, ok := smm.context.ECSManager.GetComponent(squadID, squads.SquadComponent); ok {
		squadData := squadDataRaw.(*squads.SquadData)
		nameLabel := widget.NewText(
			widget.TextOpts.Text(fmt.Sprintf("Squad: %s", squadData.Name), LargeFace, color.White),
		)
		panel.container.AddChild(nameLabel)
	}

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
	unitIDs := squads.GetUnitIDsInSquad(squadID, smm.context.ECSManager)

	totalHP := 0
	maxHP := 0
	unitCount := len(unitIDs)

	for _, unitID := range unitIDs {
		if attrRaw, ok := smm.context.ECSManager.GetComponent(unitID, common.AttributeComponent); ok {
			attr := attrRaw.(*common.Attributes)
			totalHP += attr.CurrentHealth
			maxHP += attr.MaxHealth
		}
	}

	return fmt.Sprintf("Units: %d\nTotal HP: %d/%d\nMorale: N/A", unitCount, totalHP, maxHP)
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
	// ESC or E to close
	if inputState.KeysJustPressed[ebiten.KeyEscape] || inputState.KeysJustPressed[ebiten.KeyE] {
		if exploreMode, exists := smm.modeManager.GetMode("exploration"); exists {
			smm.modeManager.RequestTransition(exploreMode, "ESC pressed")
			return true
		}
	}

	return false
}

func (smm *SquadManagementMode) GetEbitenUI() *ebitenui.UI {
	return smm.ui
}

func (smm *SquadManagementMode) GetModeName() string {
	return "squad_management"
}
