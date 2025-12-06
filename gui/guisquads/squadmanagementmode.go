package guisquads

import (
	"fmt"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/guiresources"
	"game_main/gui/widgets"
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
	// Initialize common mode infrastructure (required for queries field)
	smm.InitializeBase(ctx)

	// Initialize command history with refresh callback
	smm.InitializeCommandHistory(smm.refreshAfterUndoRedo)

	// Register hotkeys for mode transitions (Overworld context only)
	smm.RegisterHotkey(ebiten.KeyB, "squad_builder")
	smm.RegisterHotkey(ebiten.KeyF, "formation_editor")
	smm.RegisterHotkey(ebiten.KeyP, "unit_purchase")
	smm.RegisterHotkey(ebiten.KeyE, "squad_editor")

	// Override root container with anchor layout (consistent with combat mode)
	smm.RootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	smm.GetEbitenUI().Container = smm.RootContainer

	// Container for the current squad panel (will be populated in Enter)
	// Calculate responsive size
	panelWidth := int(float64(smm.Layout.ScreenWidth) * widgets.SquadMgmtPanelWidth)
	panelHeight := int(float64(smm.Layout.ScreenHeight) * widgets.SquadMgmtPanelHeight)

	smm.panelContainer = widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		),
	})

	// Apply anchor layout positioning - top-center
	topPad := int(float64(smm.Layout.ScreenHeight) * widgets.PaddingStandard)
	smm.panelContainer.GetWidget().LayoutData = gui.AnchorCenterStart(topPad)

	smm.RootContainer.AddChild(smm.panelContainer)

	// Navigation container (Previous/Next buttons + squad counter)
	// Calculate responsive size
	navWidth := int(float64(smm.Layout.ScreenWidth) * widgets.SquadMgmtNavWidth)
	navHeight := int(float64(smm.Layout.ScreenHeight) * widgets.SquadMgmtNavHeight)

	smm.navigationContainer = widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:  navWidth,
		MinHeight: navHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(20),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(smm.Layout, widgets.PaddingExtraSmall)),
		),
	})

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

	// Position below panelContainer
	navTopOffset := int(float64(smm.Layout.ScreenHeight) * (widgets.SquadMgmtPanelHeight + widgets.PaddingStandard*2))
	smm.navigationContainer.GetWidget().LayoutData = gui.AnchorCenterStart(navTopOffset)

	smm.RootContainer.AddChild(smm.navigationContainer)

	// Command buttons container (Disband, Merge, Undo, Redo)
	// Calculate responsive size
	cmdWidth := int(float64(smm.Layout.ScreenWidth) * widgets.SquadMgmtCmdWidth)
	cmdHeight := int(float64(smm.Layout.ScreenHeight) * widgets.SquadMgmtCmdHeight)

	smm.commandContainer = widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:  cmdWidth,
		MinHeight: cmdHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(smm.Layout, widgets.PaddingExtraSmall)),
		),
	})

	// Disband Squad button
	disbandBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Disband Squad",
		OnClick: func() {
			smm.onDisbandSquad()
		},
	})
	smm.commandContainer.AddChild(disbandBtn)

	// Merge Squads button
	mergeBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Merge Squads",
		OnClick: func() {
			smm.onMergeSquads()
		},
	})
	smm.commandContainer.AddChild(mergeBtn)

	// Undo/Redo buttons from CommandHistory
	smm.commandContainer.AddChild(smm.CommandHistory.CreateUndoButton())
	smm.commandContainer.AddChild(smm.CommandHistory.CreateRedoButton())

	// Position below navigationContainer
	cmdTopOffset := int(float64(smm.Layout.ScreenHeight) * (widgets.SquadMgmtPanelHeight + widgets.SquadMgmtNavHeight + widgets.PaddingStandard*3))
	smm.commandContainer.GetWidget().LayoutData = gui.AnchorCenterStart(cmdTopOffset)

	smm.RootContainer.AddChild(smm.commandContainer)

	// Status label for command results (use BaseMode.SetStatus to update)
	smm.StatusLabel = widgets.CreateSmallLabel("")

	// Position below commandContainer
	statusTopOffset := int(float64(smm.Layout.ScreenHeight) * (widgets.SquadMgmtPanelHeight + widgets.SquadMgmtNavHeight + widgets.SquadMgmtCmdHeight + widgets.PaddingStandard*4))
	smm.StatusLabel.GetWidget().LayoutData = gui.AnchorCenterStart(statusTopOffset)

	smm.RootContainer.AddChild(smm.StatusLabel)

	// Build action buttons (bottom-center) using action button group helper
	actionButtonSpecs := []widgets.ButtonSpec{
		{
			Text: "Battle Map (ESC)",
			OnClick: func() {
				if smm.Context.ModeCoordinator != nil {
					smm.Context.ModeCoordinator.EnterBattleMap("exploration")
				}
			},
		},
		{
			Text: "Squad Builder (B)",
			OnClick: func() {
				if builderMode, exists := smm.ModeManager.GetMode("squad_builder"); exists {
					smm.ModeManager.RequestTransition(builderMode, "Open Squad Builder")
				}
			},
		},
		{
			Text: "Formation (F)",
			OnClick: func() {
				if formationMode, exists := smm.ModeManager.GetMode("formation_editor"); exists {
					smm.ModeManager.RequestTransition(formationMode, "Open Formation Editor")
				}
			},
		},
		{
			Text: "Buy Units (P)",
			OnClick: func() {
				if purchaseMode, exists := smm.ModeManager.GetMode("unit_purchase"); exists {
					smm.ModeManager.RequestTransition(purchaseMode, "Open Unit Purchase")
				}
			},
		},
		{
			Text: "Edit Squad (E)",
			OnClick: func() {
				if editorMode, exists := smm.ModeManager.GetMode("squad_editor"); exists {
					smm.ModeManager.RequestTransition(editorMode, "Open Squad Editor")
				}
			},
		},
	}
	actionButtonContainer := gui.CreateActionButtonGroup(smm.PanelBuilders, widgets.BottomCenter(), actionButtonSpecs)
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
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(smm.Layout, widgets.PaddingTight)),
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

	// Use helper to create unit list with fixed height to prevent layout jumping
	return widgets.CreateUnitList(widgets.UnitListConfig{
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
	squadName := squads.GetSquadName(currentSquadID, smm.Queries.ECSManager)

	// Show confirmation dialog
	dialog := widgets.CreateConfirmationDialog(widgets.DialogConfig{
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
	currentSquadName := squads.GetSquadName(currentSquadID, smm.Queries.ECSManager)

	// Build list of other squads to merge with
	otherSquads := make([]string, 0)
	otherSquadIDs := make([]ecs.EntityID, 0)
	for i, squadID := range smm.allSquadIDs {
		if i != smm.currentSquadIndex {
			squadName := squads.GetSquadName(squadID, smm.Queries.ECSManager)
			otherSquads = append(otherSquads, squadName)
			otherSquadIDs = append(otherSquadIDs, squadID)
		}
	}

	// Create selection dialog container
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(guiresources.PanelRes.Image),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(smm.Layout, widgets.PaddingStandard)),
		)),
	)

	// Title
	titleLabel := widgets.CreateLargeLabel("Merge Squads")
	contentContainer.AddChild(titleLabel)

	// Message
	messageLabel := widgets.CreateSmallLabel(fmt.Sprintf("Select squad to merge INTO '%s':", currentSquadName))
	contentContainer.AddChild(messageLabel)

	// Squad selection list (using helper for simple string list)
	var squadList *widget.List
	squadList = widgets.CreateSimpleStringList(widgets.SimpleStringListConfig{
		Entries:       otherSquads,
		ScreenWidth:   400,  // Fixed width for dialog
		ScreenHeight:  200,  // Fixed height for dialog
		WidthPercent:  1.0,  // Use full specified dimensions
		HeightPercent: 1.0,
	})
	contentContainer.AddChild(squadList)

	// Reference to window for closing
	var window *widget.Window

	// Button container
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(15),
		)),
	)

	// Merge button
	mergeBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Merge",
		OnClick: func() {
			selectedEntry := squadList.SelectedEntry()
			if selectedEntry == nil {
				smm.SetStatus("No target squad selected")
				return
			}

			// Find the selected squad ID
			selectedIndex := -1
			for i, entry := range otherSquads {
				if entry == selectedEntry {
					selectedIndex = i
					break
				}
			}

			if selectedIndex == -1 {
				smm.SetStatus("Error finding selected squad")
				return
			}

			targetSquadID := otherSquadIDs[selectedIndex]
			targetSquadName := squads.GetSquadName(targetSquadID, smm.Queries.ECSManager)

			// Close selection dialog
			if window != nil {
				window.Close()
			}

			// Show confirmation dialog
			confirmDialog := widgets.CreateConfirmationDialog(widgets.DialogConfig{
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
	})
	buttonContainer.AddChild(mergeBtn)

	// Cancel button
	cancelBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Cancel",
		OnClick: func() {
			smm.SetStatus("Merge cancelled")
			if window != nil {
				window.Close()
			}
		},
	})
	buttonContainer.AddChild(cancelBtn)

	contentContainer.AddChild(buttonContainer)

	// Create window
	window = widget.NewWindow(
		widget.WindowOpts.Contents(contentContainer),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.MinSize(500, 400),
	)

	smm.GetEbitenUI().AddWindow(window)
}

// refreshAfterUndoRedo is called after successful undo/redo operations
func (smm *SquadManagementMode) refreshAfterUndoRedo() {
	// Refresh squad list (squads might have been created/destroyed)
	smm.allSquadIDs = squads.FindAllSquads(smm.Queries.ECSManager)

	// Adjust index if needed
	if smm.currentSquadIndex >= len(smm.allSquadIDs) && len(smm.allSquadIDs) > 0 {
		smm.currentSquadIndex = 0
	}

	smm.refreshCurrentSquad()
	smm.updateNavigationButtons()
}

