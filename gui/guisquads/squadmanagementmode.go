package guisquads

import (
	"fmt"
	"game_main/common"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/guiresources"
	"game_main/gui/widgets"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadManagementMode shows one squad at a time with navigation controls
type SquadManagementMode struct {
	gui.BaseMode // Embed common mode infrastructure

	currentSquadIndex   int                                // Index of currently displayed squad
	allSquadIDs         []ecs.EntityID                     // All available squad IDs
	currentPanel        *SquadPanel                        // Currently displayed panel
	panelContainer      *widget.Container                  // Container for the current squad panel
	navigationContainer *widget.Container                  // Container for navigation buttons
	closeButton         *widget.Button
	prevButton          *widget.Button
	nextButton          *widget.Button
	squadCounterLabel   *widget.Text // Shows "Squad 1 of 3"
}

// SquadPanel represents a single squad's UI panel
type SquadPanel struct {
	container    *widget.Container
	squadID      ecs.EntityID
	gridDisplay  *widget.TextArea // Shows 3x3 grid visualization
	statsDisplay *widget.TextArea // Shows squad stats
	unitList     *widget.List     // Shows individual units
}

func NewSquadManagementMode(modeManager *core.UIModeManager) *SquadManagementMode {
	mode := &SquadManagementMode{
		currentSquadIndex: 0,
		allSquadIDs:       make([]ecs.EntityID, 0),
	}
	mode.SetModeName("squad_management")
	mode.ModeManager = modeManager
	return mode
}

func (smm *SquadManagementMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure (required for queries field)
	smm.InitializeBase(ctx)

	// Register hotkeys for mode transitions (Overworld context only)
	smm.RegisterHotkey(ebiten.KeyB, "squad_builder")
	smm.RegisterHotkey(ebiten.KeyF, "formation_editor")
	smm.RegisterHotkey(ebiten.KeyP, "unit_purchase")

	// Override root container with vertical layout for single squad panel + navigation
	smm.RootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 20, Right: 20, Top: 20, Bottom: 20,
			}),
		)),
	)
	smm.GetEbitenUI().Container = smm.RootContainer

	// Container for the current squad panel (will be populated in Enter)
	smm.panelContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		)),
	)
	smm.RootContainer.AddChild(smm.panelContainer)

	// Navigation container (Previous/Next buttons + squad counter)
	smm.navigationContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(20),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 10, Right: 10, Top: 10, Bottom: 10,
			}),
		)),
	)

	// Previous button
	smm.prevButton = widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "< Previous",
		OnClick: func() {
			smm.showPreviousSquad()
		},
	})
	smm.navigationContainer.AddChild(smm.prevButton)

	// Squad counter label
	smm.squadCounterLabel = widgets.CreateSmallLabel("Squad 1 of 1")
	smm.navigationContainer.AddChild(smm.squadCounterLabel)

	// Next button
	smm.nextButton = widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Next >",
		OnClick: func() {
			smm.showNextSquad()
		},
	})
	smm.navigationContainer.AddChild(smm.nextButton)

	smm.RootContainer.AddChild(smm.navigationContainer)

	// Build action buttons (bottom-center) using helper
	actionButtonContainer := gui.CreateBottomCenterButtonContainer(smm.PanelBuilders)

	// Return to Battle Map button
	battleMapBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Battle Map (ESC)",
		OnClick: func() {
			if smm.Context.ModeCoordinator != nil {
				smm.Context.ModeCoordinator.EnterBattleMap("exploration")
			}
		},
	})
	actionButtonContainer.AddChild(battleMapBtn)

	// Squad Builder button
	builderBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Squad Builder (B)",
		OnClick: func() {
			if builderMode, exists := smm.ModeManager.GetMode("squad_builder"); exists {
				smm.ModeManager.RequestTransition(builderMode, "Open Squad Builder")
			}
		},
	})
	actionButtonContainer.AddChild(builderBtn)

	// Formation Editor button
	formationBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Formation (F)",
		OnClick: func() {
			if formationMode, exists := smm.ModeManager.GetMode("formation_editor"); exists {
				smm.ModeManager.RequestTransition(formationMode, "Open Formation Editor")
			}
		},
	})
	actionButtonContainer.AddChild(formationBtn)

	// Unit Purchase button
	purchaseBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Buy Units (P)",
		OnClick: func() {
			if purchaseMode, exists := smm.ModeManager.GetMode("unit_purchase"); exists {
				smm.ModeManager.RequestTransition(purchaseMode, "Open Unit Purchase")
			}
		},
	})
	actionButtonContainer.AddChild(purchaseBtn)

	smm.GetEbitenUI().Container.AddChild(actionButtonContainer)

	return nil
}

func (smm *SquadManagementMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Management Mode")

	// Get all squad IDs from ECS
	smm.allSquadIDs = squads.FindAllSquads(smm.Queries.ECSManager)

	// Reset to first squad if we have any
	if len(smm.allSquadIDs) > 0 {
		smm.currentSquadIndex = 0
		smm.refreshCurrentSquad()
	} else {
		// No squads available - show message
		smm.clearPanel()
		noSquadsLabel := widgets.CreateLargeLabel("No squads available")
		smm.panelContainer.AddChild(noSquadsLabel)
	}

	smm.updateNavigationButtons()
	return nil
}

func (smm *SquadManagementMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Squad Management Mode")

	// Clear current panel
	smm.clearPanel()
	smm.currentPanel = nil

	return nil
}

// refreshCurrentSquad clears the panel and displays the current squad
func (smm *SquadManagementMode) refreshCurrentSquad() {
	smm.clearPanel()

	if len(smm.allSquadIDs) == 0 {
		return
	}

	// Get current squad ID
	squadID := smm.allSquadIDs[smm.currentSquadIndex]

	// Create and display panel
	smm.currentPanel = smm.createSquadPanel(squadID)
	smm.panelContainer.AddChild(smm.currentPanel.container)

	// Update squad counter label
	counterText := fmt.Sprintf("Squad %d of %d", smm.currentSquadIndex+1, len(smm.allSquadIDs))
	smm.squadCounterLabel.Label = counterText
}

// clearPanel removes all children from the panel container
func (smm *SquadManagementMode) clearPanel() {
	smm.panelContainer.RemoveChildren()
}

// showPreviousSquad cycles to the previous squad (wraps around)
func (smm *SquadManagementMode) showPreviousSquad() {
	if len(smm.allSquadIDs) == 0 {
		return
	}

	smm.currentSquadIndex--
	if smm.currentSquadIndex < 0 {
		smm.currentSquadIndex = len(smm.allSquadIDs) - 1
	}

	smm.refreshCurrentSquad()
	smm.updateNavigationButtons()
}

// showNextSquad cycles to the next squad (wraps around)
func (smm *SquadManagementMode) showNextSquad() {
	if len(smm.allSquadIDs) == 0 {
		return
	}

	smm.currentSquadIndex++
	if smm.currentSquadIndex >= len(smm.allSquadIDs) {
		smm.currentSquadIndex = 0
	}

	smm.refreshCurrentSquad()
	smm.updateNavigationButtons()
}

// updateNavigationButtons enables/disables navigation buttons based on squad count
func (smm *SquadManagementMode) updateNavigationButtons() {
	hasMultipleSquads := len(smm.allSquadIDs) > 1

	if smm.prevButton != nil {
		smm.prevButton.GetWidget().Disabled = !hasMultipleSquads
	}

	if smm.nextButton != nil {
		smm.nextButton.GetWidget().Disabled = !hasMultipleSquads
	}
}

func (smm *SquadManagementMode) createSquadPanel(squadID ecs.EntityID) *SquadPanel {
	panel := &SquadPanel{
		squadID: squadID,
	}

	// Container for this squad's panel
	panel.container = widgets.CreatePanelWithConfig(widgets.PanelConfig{
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 15, Right: 15, Top: 15, Bottom: 15,
			}),
		),
	})

	// Squad name label - use unified query service
	squadName := squads.GetSquadName(squadID, smm.Queries.ECSManager)
	nameLabel := widgets.CreateLargeLabel(fmt.Sprintf("Squad: %s", squadName))
	panel.container.AddChild(nameLabel)

	// 3x3 grid visualization
	gridVisualization := squads.VisualizeSquad(squadID, smm.Queries.ECSManager)
	gridConfig := widgets.TextAreaConfig{
		MinWidth:  300,
		MinHeight: 200,
		FontColor: color.White,
	}
	panel.gridDisplay = widgets.CreateTextAreaWithConfig(gridConfig)
	panel.gridDisplay.SetText(gridVisualization)
	panel.container.AddChild(panel.gridDisplay)

	// Squad stats display
	statsConfig := widgets.TextAreaConfig{
		MinWidth:  300,
		MinHeight: 100,
		FontColor: color.White,
	}
	panel.statsDisplay = widgets.CreateTextAreaWithConfig(statsConfig)
	panel.statsDisplay.SetText(smm.getSquadStats(squadID))
	panel.container.AddChild(panel.statsDisplay)

	// Unit list (clickable for details)
	panel.unitList = smm.createUnitList(squadID)
	panel.container.AddChild(panel.unitList)

	return panel
}

func (smm *SquadManagementMode) createUnitList(squadID ecs.EntityID) *widget.List {
	// Get all units in this squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, smm.Queries.ECSManager)

	// Create list entries
	entries := make([]interface{}, 0, len(unitIDs))
	for _, unitID := range unitIDs {
		// Get unit attributes (units use common.Attributes, not separate UnitData)
		if attrRaw, ok := smm.Context.ECSManager.GetComponent(unitID, common.AttributeComponent); ok {
			attr := attrRaw.(*common.Attributes)
			// Get unit name
			nameStr := "Unknown"
			if nameRaw, ok := smm.Context.ECSManager.GetComponent(unitID, common.NameComponent); ok {
				name := nameRaw.(*common.Name)
				nameStr = name.NameStr
			}
			entries = append(entries, fmt.Sprintf("%s - HP: %d/%d", nameStr, attr.CurrentHealth, attr.MaxHealth))
		}
	}

	// Create list widget using exported resources
	list := widgets.CreateListWithConfig(widgets.ListConfig{
		Entries: entries,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
	})

	return list
}

func (smm *SquadManagementMode) getSquadStats(squadID ecs.EntityID) string {
	// Use unified query service to get squad stats
	squadInfo := smm.Queries.GetSquadInfo(squadID)
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

func (smm *SquadManagementMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key)
	if smm.HandleCommonInput(inputState) {
		return true
	}

	// E key hotkey is now handled by gui.BaseMode.HandleCommonInput via RegisterHotkey
	return false
}
