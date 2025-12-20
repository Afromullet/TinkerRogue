package guisquads

import (
	"fmt"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/guiresources"
	"game_main/gui/builders"
	"game_main/gui/specs"
	"game_main/squads"
	"game_main/squads/squadcommands"
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
	navigationContainer *widget.Container // Container for navigation buttons
	commandContainer    *widget.Container // Container for command buttons
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
	err := gui.NewModeBuilder(&smm.BaseMode, gui.ModeConfig{
		ModeName:   "squad_management",
		ReturnMode: "", // Context switch handled separately

		Hotkeys: []gui.HotkeySpec{
			{Key: ebiten.KeyB, TargetMode: "squad_builder"},
			{Key: ebiten.KeyF, TargetMode: "formation_editor"},
			{Key: ebiten.KeyP, TargetMode: "unit_purchase"},
			{Key: ebiten.KeyE, TargetMode: "squad_editor"},
		},

		Panels: []gui.PanelSpec{
			{CustomBuild: smm.buildSquadPanel},
			{CustomBuild: smm.buildNavigationPanel},
			{CustomBuild: smm.buildCommandPanel},
			{CustomBuild: smm.buildActionButtons}, // Build after Context is available
		},

		StatusLabel: true,
		Commands:    true,
		OnRefresh:   smm.refreshAfterUndoRedo,
	}).Build(ctx)

	return err
}

func (smm *SquadManagementMode) buildSquadPanel() *widget.Container {
	// Container for the current squad panel (will be populated in Enter)
	panelWidth := int(float64(smm.Layout.ScreenWidth) * specs.SquadMgmtPanelWidth)
	panelHeight := int(float64(smm.Layout.ScreenHeight) * specs.SquadMgmtPanelHeight)

	panelContainer := builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		),
	})

	// Apply anchor layout positioning - top-center
	topPad := int(float64(smm.Layout.ScreenHeight) * specs.PaddingStandard)
	panelContainer.GetWidget().LayoutData = gui.AnchorCenterStart(topPad)

	smm.panelContainer = panelContainer
	return panelContainer
}

func (smm *SquadManagementMode) buildNavigationPanel() *widget.Container {
	// Navigation container (Previous/Next buttons + squad counter)
	navWidth := int(float64(smm.Layout.ScreenWidth) * specs.SquadMgmtNavWidth)
	navHeight := int(float64(smm.Layout.ScreenHeight) * specs.SquadMgmtNavHeight)

	navigationContainer := builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  navWidth,
		MinHeight: navHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(20),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(smm.Layout, specs.PaddingExtraSmall)),
		),
	})

	// Previous button
	smm.prevButton = builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "< Previous",
		OnClick: func() {
			smm.showPreviousSquad()
		},
	})
	navigationContainer.AddChild(smm.prevButton)

	// Squad counter label
	smm.squadCounterLabel = builders.CreateSmallLabel("Squad 1 of 1")
	navigationContainer.AddChild(smm.squadCounterLabel)

	// Next button
	smm.nextButton = builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Next >",
		OnClick: func() {
			smm.showNextSquad()
		},
	})
	navigationContainer.AddChild(smm.nextButton)

	// Position below panelContainer
	navTopOffset := int(float64(smm.Layout.ScreenHeight) * (specs.SquadMgmtPanelHeight + specs.PaddingStandard*2))
	navigationContainer.GetWidget().LayoutData = gui.AnchorCenterStart(navTopOffset)

	smm.navigationContainer = navigationContainer
	return navigationContainer
}

func (smm *SquadManagementMode) buildCommandPanel() *widget.Container {
	// Command buttons container (Disband, Merge, Undo, Redo)
	cmdWidth := int(float64(smm.Layout.ScreenWidth) * specs.SquadMgmtCmdWidth)
	cmdHeight := int(float64(smm.Layout.ScreenHeight) * specs.SquadMgmtCmdHeight)

	commandContainer := builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  cmdWidth,
		MinHeight: cmdHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(smm.Layout, specs.PaddingExtraSmall)),
		),
	})

	// Disband Squad button
	disbandBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Disband Squad",
		OnClick: func() {
			smm.onDisbandSquad()
		},
	})
	commandContainer.AddChild(disbandBtn)

	// Merge Squads button
	mergeBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Merge Squads",
		OnClick: func() {
			smm.onMergeSquads()
		},
	})
	commandContainer.AddChild(mergeBtn)

	// Undo/Redo buttons from CommandHistory (will be available after Initialize)
	commandContainer.AddChild(smm.CommandHistory.CreateUndoButton())
	commandContainer.AddChild(smm.CommandHistory.CreateRedoButton())

	// Position below navigationContainer
	cmdTopOffset := int(float64(smm.Layout.ScreenHeight) * (specs.SquadMgmtPanelHeight + specs.SquadMgmtNavHeight + specs.PaddingStandard*3))
	commandContainer.GetWidget().LayoutData = gui.AnchorCenterStart(cmdTopOffset)

	smm.commandContainer = commandContainer
	return commandContainer
}

func (smm *SquadManagementMode) buildActionButtons() *widget.Container {
	// Create UI factory
	uiFactory := gui.NewUIComponentFactory(smm.Queries, smm.PanelBuilders, smm.Layout)

	// Create button callbacks (no panel wrapper - like combat mode)
	buttonContainer := uiFactory.CreateSquadManagementActionButtons(
		// Battle Map (ESC)
		func() {
			if smm.Context.ModeCoordinator != nil {
				if err := smm.Context.ModeCoordinator.EnterBattleMap("exploration"); err != nil {
					fmt.Printf("ERROR: Failed to enter battle map: %v\n", err)
				}
			}
		},
		// Squad Builder (B)
		func() {
			if mode, exists := smm.ModeManager.GetMode("squad_builder"); exists {
				smm.ModeManager.RequestTransition(mode, "Squad Builder clicked")
			}
		},
		// Formation (F)
		func() {
			if mode, exists := smm.ModeManager.GetMode("formation_editor"); exists {
				smm.ModeManager.RequestTransition(mode, "Formation clicked")
			}
		},
		// Buy Units (P)
		func() {
			if mode, exists := smm.ModeManager.GetMode("unit_purchase"); exists {
				smm.ModeManager.RequestTransition(mode, "Buy Units clicked")
			}
		},
		// Edit Squad (E)
		func() {
			if mode, exists := smm.ModeManager.GetMode("squad_editor"); exists {
				smm.ModeManager.RequestTransition(mode, "Edit Squad clicked")
			}
		},
	)

	return buttonContainer
}

func (smm *SquadManagementMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Management Mode")

	// Get all squad IDs from ECS
	smm.allSquadIDs = smm.Queries.SquadCache.FindAllSquads()

	// Reset to first squad if we have any
	if len(smm.allSquadIDs) > 0 {
		smm.currentSquadIndex = 0
		smm.refreshCurrentSquad()
	} else {
		// No squads available - show message
		smm.clearPanel()
		noSquadsLabel := builders.CreateLargeLabel("No squads available")
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
	panel.container = builders.CreateStaticPanel(builders.PanelConfig{
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(smm.Layout, specs.PaddingTight)),
		),
	})

	// Squad name label - use unified query service
	squadName := smm.Queries.SquadCache.GetSquadName(squadID)
	nameLabel := builders.CreateLargeLabel(fmt.Sprintf("Squad: %s", squadName))
	panel.container.AddChild(nameLabel)

	// 3x3 grid visualization
	gridVisualization := squads.VisualizeSquad(squadID, smm.Queries.ECSManager)
	gridConfig := builders.TextAreaConfig{
		MinWidth:  300,
		MinHeight: 200,
		FontColor: color.White,
	}
	panel.gridDisplay = builders.CreateTextAreaWithConfig(gridConfig)
	panel.gridDisplay.SetText(gridVisualization)
	panel.container.AddChild(panel.gridDisplay)

	// Squad stats display
	statsConfig := builders.TextAreaConfig{
		MinWidth:  300,
		MinHeight: 100,
		FontColor: color.White,
	}
	panel.statsDisplay = builders.CreateTextAreaWithConfig(statsConfig)
	panel.statsDisplay.SetText(smm.getSquadStats(squadID))
	panel.container.AddChild(panel.statsDisplay)

	// Unit list (clickable for details)
	panel.unitList = smm.createUnitList(squadID)
	panel.container.AddChild(panel.unitList)

	return panel
}

func (smm *SquadManagementMode) createUnitList(squadID ecs.EntityID) *widget.List {
	// Get all units in this squad
	unitIDs := smm.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	// Use helper to create unit list with fixed height to prevent layout jumping
	return builders.CreateUnitList(builders.UnitListConfig{
		UnitIDs:       unitIDs,
		Manager:       smm.Queries.ECSManager,
		ScreenWidth:   400,  // Fixed width
		ScreenHeight:  1000, // Fixed height (set HeightPercent to get 250px)
		WidthPercent:  1.0,  // Use full fixed width
		HeightPercent: 0.25, // 250px fixed height (1000 * 0.25)
	})
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

	// Handle undo/redo input (Ctrl+Z, Ctrl+Y)
	if smm.CommandHistory.HandleInput(inputState) {
		return true
	}

	// E key hotkey is now handled by gui.BaseMode.HandleCommonInput via RegisterHotkey
	return false
}

// onDisbandSquad shows confirmation dialog then executes DisbandSquadCommand for the current squad
func (smm *SquadManagementMode) onDisbandSquad() {
	if len(smm.allSquadIDs) == 0 {
		smm.SetStatus("No squad selected")
		return
	}

	currentSquadID := smm.allSquadIDs[smm.currentSquadIndex]
	squadName := smm.Queries.SquadCache.GetSquadName(currentSquadID)

	// Show confirmation dialog
	dialog := builders.CreateConfirmationDialog(builders.DialogConfig{
		Title:   "Confirm Disband",
		Message: fmt.Sprintf("Disband squad '%s'? This will return all units to the roster.\n\nYou can undo this action with Ctrl+Z.", squadName),
		OnConfirm: func() {
			// Create and execute disband command
			cmd := squadcommands.NewDisbandSquadCommand(
				smm.Queries.ECSManager,
				smm.Context.PlayerData.PlayerEntityID,
				currentSquadID,
			)

			smm.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			smm.SetStatus("Disband cancelled")
		},
	})

	smm.GetEbitenUI().AddWindow(dialog)
}

// onMergeSquads shows squad selection dialog then executes MergeSquadsCommand
func (smm *SquadManagementMode) onMergeSquads() {
	if len(smm.allSquadIDs) < 2 {
		smm.SetStatus("Need at least 2 squads to merge")
		return
	}

	currentSquadID := smm.allSquadIDs[smm.currentSquadIndex]
	currentSquadName := smm.Queries.SquadCache.GetSquadName(currentSquadID)

	// Build list of other squads to merge with
	otherSquads := make([]string, 0)
	otherSquadIDs := make([]ecs.EntityID, 0)
	for i, squadID := range smm.allSquadIDs {
		if i != smm.currentSquadIndex {
			squadName := smm.Queries.SquadCache.GetSquadName(squadID)
			otherSquads = append(otherSquads, squadName)
			otherSquadIDs = append(otherSquadIDs, squadID)
		}
	}

	// Show selection dialog using new builder - replaces 90+ lines of manual dialog creation
	selectionDialog := builders.CreateSelectionDialog(builders.SelectionDialogConfig{
		Title:            "Merge Squads",
		Message:          fmt.Sprintf("Select squad to merge INTO '%s':", currentSquadName),
		SelectionEntries: otherSquads,
		OnSelect: func(selected string) {
			// Find the selected squad ID
			selectedIndex := -1
			for i, entry := range otherSquads {
				if entry == selected {
					selectedIndex = i
					break
				}
			}

			if selectedIndex == -1 {
				smm.SetStatus("Error finding selected squad")
				return
			}

			targetSquadID := otherSquadIDs[selectedIndex]
			targetSquadName := smm.Queries.SquadCache.GetSquadName(targetSquadID)

			// Show confirmation dialog
			confirmDialog := builders.CreateConfirmationDialog(builders.DialogConfig{
				Title:   "Confirm Merge",
				Message: fmt.Sprintf("Merge '%s' INTO '%s'?\n\n'%s' will be disbanded and all units moved to '%s'.\n\nYou can undo this action with Ctrl+Z.", currentSquadName, targetSquadName, currentSquadName, targetSquadName),
				OnConfirm: func() {
					// Create and execute merge command
					cmd := squadcommands.NewMergeSquadsCommand(
						smm.Queries.ECSManager,
						smm.Context.PlayerData.PlayerEntityID,
						currentSquadID,
						targetSquadID,
					)

					smm.CommandHistory.Execute(cmd)
				},
				OnCancel: func() {
					smm.SetStatus("Merge cancelled")
				},
			})

			smm.GetEbitenUI().AddWindow(confirmDialog)
		},
		OnCancel: func() {
			smm.SetStatus("Merge cancelled")
		},
	})

	smm.GetEbitenUI().AddWindow(selectionDialog)
}

// refreshAfterUndoRedo is called after successful undo/redo operations
func (smm *SquadManagementMode) refreshAfterUndoRedo() {
	// Refresh squad list (squads might have been created/destroyed)
	smm.allSquadIDs = smm.Queries.SquadCache.FindAllSquads()

	// Adjust index if needed
	if smm.currentSquadIndex >= len(smm.allSquadIDs) && len(smm.allSquadIDs) > 0 {
		smm.currentSquadIndex = 0
	}

	smm.refreshCurrentSquad()
	smm.updateNavigationButtons()
}

