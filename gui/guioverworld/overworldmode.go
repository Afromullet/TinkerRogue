package guioverworld

import (
	"fmt"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/encounter"
	"game_main/world/coords"
	"game_main/world/overworld"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// OverworldMode handles the overworld UI (threat visualization, tick controls)
type OverworldMode struct {
	framework.BaseMode // Embed common mode infrastructure

	// UI State
	state *OverworldState

	// Renderer
	renderer *OverworldRenderer

	// Services
	encounterService *encounter.EncounterService

	// Widget references (populated from panel registry)
	threatInfoText  *widget.TextArea
	tickStatusText  *widget.TextArea
	eventLogText    *widget.TextArea
	threatStatsText *widget.TextArea

	// Initialization tracking
	initialized bool
}

func NewOverworldMode(modeManager *framework.UIModeManager, encounterService *encounter.EncounterService) *OverworldMode {
	om := &OverworldMode{
		state:            NewOverworldState(),
		encounterService: encounterService,
	}
	om.SetModeName("overworld")
	om.SetReturnMode("") // No simple return mode - uses context switching
	om.ModeManager = modeManager
	om.SetSelf(om) // Required for panel registry building
	return om
}

func (om *OverworldMode) Initialize(ctx *framework.UIContext) error {
	// Build base UI using ModeBuilder
	err := framework.NewModeBuilder(&om.BaseMode, framework.ModeConfig{
		ModeName:   "overworld",
		ReturnMode: "", // Uses context switching instead of simple mode transition
		// Hotkeys handled in HandleInput (Space, P, I are custom actions, not mode transitions)
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := om.BuildPanels(
		OverworldPanelTickControls,
		OverworldPanelThreatInfo,
		OverworldPanelTickStatus,
		OverworldPanelEventLog,
		OverworldPanelThreatStats,
	); err != nil {
		return err
	}

	// Initialize widget references
	om.initializeWidgetReferences()

	// Create renderer (need to cast GameMap interface to *worldmap.GameMap)
	gameMap, ok := ctx.GameMap.(*worldmap.GameMap)
	if !ok {
		return fmt.Errorf("GameMap is not *worldmap.GameMap")
	}
	om.renderer = NewOverworldRenderer(ctx.ECSManager, om.state, gameMap, ctx.TileSize)

	// Ensure tick state exists
	tickState := overworld.GetTickState(ctx.ECSManager)
	if tickState == nil {
		overworld.CreateTickStateEntity(ctx.ECSManager)
		om.logEvent("Tick state initialized")
	}

	om.initialized = true
	om.logEvent("Overworld mode initialized")
	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (om *OverworldMode) initializeWidgetReferences() {
	om.threatInfoText = GetOverworldThreatInfo(om.Panels)
	om.tickStatusText = GetOverworldTickStatus(om.Panels)
	om.eventLogText = GetOverworldEventLog(om.Panels)
	om.threatStatsText = GetOverworldThreatStats(om.Panels)
}

func (om *OverworldMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Overworld Mode")

	// Ensure recording is active when entering overworld
	ctx := overworld.GetContext()
	if ctx.Recorder != nil && ctx.Recorder.IsEnabled() {
		tickState := overworld.GetTickState(om.Context.ECSManager)
		if tickState != nil && ctx.Recorder.EventCount() == 0 {
			// Start new recording session if recorder is empty
			overworld.StartRecordingSession(tickState.CurrentTick)
		}
	}

	// Refresh UI displays
	om.refreshThreatInfo()
	om.refreshTickStatus()
	om.refreshThreatStats()

	return nil
}

func (om *OverworldMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Overworld Mode")

	// Export overworld log when leaving overworld mode
	ctx := overworld.GetContext()
	if ctx.Recorder != nil && ctx.Recorder.IsEnabled() {
		tickState := overworld.GetTickState(om.Context.ECSManager)

		// Only export if game isn't over (victory/defeat already exported)
		if tickState != nil && !tickState.IsGameOver {
			// Only export if we actually have events recorded
			if ctx.Recorder.EventCount() > 0 {
				tickMsg := fmt.Sprintf("tick %d", tickState.CurrentTick)

				// Export with "Session Paused" outcome (game continues, not victory/defeat)
				if err := overworld.FinalizeRecording("Session Paused", fmt.Sprintf("Left overworld at %s", tickMsg)); err != nil {
					fmt.Printf("WARNING: Failed to export overworld log on exit: %v\n", err)
				}

				// Clear recording for next session (will restart on next Enter)
				overworld.ClearRecording()
			}
		}
	}

	// Clear selection when leaving
	om.state.ClearSelection()

	return nil
}

func (om *OverworldMode) Update(deltaTime float64) error {
	// Update tick status display every frame
	om.refreshTickStatus()

	// Update threat stats
	om.refreshThreatStats()

	// Update threat info if selection active
	if om.state.HasSelection() {
		om.refreshThreatInfo()
	}

	return nil
}

func (om *OverworldMode) Render(screen *ebiten.Image) {
	// Render overworld visualization (threat nodes, influence, etc.)
	if om.renderer != nil {
		om.renderer.Render(screen)
	}
}

func (om *OverworldMode) HandleInput(inputState *framework.InputState) bool {
	// Handle ESC key for context switch
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if om.Context.ModeCoordinator != nil {
			if err := om.Context.ModeCoordinator.EnterBattleMap("exploration"); err != nil {
				fmt.Printf("ERROR: Failed to return to battle map: %v\n", err)
			}
			return true
		}
	}

	// Handle common input (registered hotkeys)
	if om.HandleCommonInput(inputState) {
		return true
	}

	// Handle custom hotkeys
	if inputState.KeysJustPressed[ebiten.KeySpace] {
		om.handleAdvanceTick()
		return true
	}

	if inputState.KeysJustPressed[ebiten.KeyI] {
		om.handleToggleInfluence()
		return true
	}

	// Handle 'E' key to engage selected threat
	if inputState.KeysJustPressed[ebiten.KeyE] {
		if om.state.HasSelection() {
			om.handleEngageThreat()
			return true
		}
	}

	// Handle mouse clicks for threat selection
	if inputState.MousePressed && inputState.MouseButton == ebiten.MouseButtonLeft {
		threatID := om.renderer.GetThreatAtPosition(inputState.MouseX, inputState.MouseY)
		if threatID != 0 {
			om.state.SelectedThreatID = threatID
			om.refreshThreatInfo()
			om.logEvent(fmt.Sprintf("Selected threat %d (Press E to engage)", threatID))
			return true
		} else {
			// Click on empty space clears selection
			om.state.ClearSelection()
			om.refreshThreatInfo()
		}
	}

	return false
}

// Button click handlers

func (om *OverworldMode) handleAdvanceTick() {
	err := overworld.AdvanceTick(om.Context.ECSManager)
	if err != nil {
		om.logEvent(fmt.Sprintf("ERROR: %v", err))
		return
	}

	tickState := overworld.GetTickState(om.Context.ECSManager)
	if tickState != nil {
		om.logEvent(fmt.Sprintf("Tick advanced to %d", tickState.CurrentTick))
	}

	om.refreshTickStatus()
	om.refreshThreatStats()
}

func (om *OverworldMode) handleToggleInfluence() {
	om.state.ShowInfluence = !om.state.ShowInfluence

	if om.state.ShowInfluence {
		om.logEvent("Influence zones visible")
	} else {
		om.logEvent("Influence zones hidden")
	}
}

func (om *OverworldMode) handleEngageThreat() {
	// Validate selection
	if !om.state.HasSelection() {
		om.logEvent("No threat selected")
		return
	}

	// Get threat entity
	threatEntity := om.Context.ECSManager.FindEntityByID(om.state.SelectedThreatID)
	if threatEntity == nil {
		om.logEvent("ERROR: Threat entity not found")
		return
	}

	// Validate threat data
	threatData := common.GetComponentType[*overworld.ThreatNodeData](threatEntity, overworld.ThreatNodeComponent)
	if threatData == nil {
		om.logEvent("ERROR: Invalid threat entity")
		return
	}

	// Get threat position
	posData := common.GetComponentType[*coords.LogicalPosition](threatEntity, common.PositionComponent)
	if posData == nil {
		om.logEvent("ERROR: Threat has no position")
		return
	}

	// Create encounter entity from threat
	om.logEvent(fmt.Sprintf("Engaging threat %d...", om.state.SelectedThreatID))

	encounterID, err := overworld.TriggerCombatFromThreat(om.Context.ECSManager, threatEntity, *posData)
	if err != nil {
		om.logEvent(fmt.Sprintf("ERROR: Failed to create encounter: %v", err))
		return
	}

	// Start encounter (spawns enemies, tracks state, and transitions to combat)
	threatName := fmt.Sprintf("%s (Level %d)", threatData.ThreatType.String(), threatData.Intensity)
	playerEntityID := ecs.EntityID(0)
	if om.Context.PlayerData != nil {
		playerEntityID = om.Context.PlayerData.PlayerEntityID
	}
	if err := om.encounterService.StartEncounter(encounterID, threatEntity.GetID(), threatName, *posData, playerEntityID); err != nil {
		om.logEvent(fmt.Sprintf("ERROR: Failed to start encounter: %v", err))
	}
}

// UI refresh functions

func (om *OverworldMode) refreshThreatInfo() {
	if om.threatInfoText == nil {
		return
	}

	if !om.state.HasSelection() {
		om.threatInfoText.SetText("Select a threat to view details")
		return
	}

	threat := om.Context.ECSManager.FindEntityByID(om.state.SelectedThreatID)
	infoText := FormatThreatInfo(threat, om.Context.ECSManager)
	om.threatInfoText.SetText(infoText)
}

func (om *OverworldMode) refreshTickStatus() {
	if om.tickStatusText == nil {
		return
	}

	tickState := overworld.GetTickState(om.Context.ECSManager)
	if tickState == nil {
		om.tickStatusText.SetText("Tick: ??? | Status: ERROR")
		return
	}

	statusText := "Ready"
	if tickState.IsGameOver {
		statusText = "Game Over"
	}

	om.tickStatusText.SetText(fmt.Sprintf(
		"Tick: %d | Status: %s",
		tickState.CurrentTick,
		statusText,
	))
}

func (om *OverworldMode) refreshThreatStats() {
	if om.threatStatsText == nil {
		return
	}

	count := overworld.CountThreatNodes(om.Context.ECSManager)
	avgIntensity := overworld.CalculateAverageIntensity(om.Context.ECSManager)

	om.threatStatsText.SetText(fmt.Sprintf(
		"Threats: %d | Avg Intensity: %.1f",
		count,
		avgIntensity,
	))
}

func (om *OverworldMode) logEvent(message string) {
	if om.eventLogText == nil {
		return
	}

	// Append to existing log
	currentText := om.eventLogText.GetText()
	newText := message + "\n" + currentText

	// Keep only last 10 lines (approximately 200 chars per line)
	maxChars := 2000
	if len(newText) > maxChars {
		newText = newText[:maxChars]
	}

	om.eventLogText.SetText(newText)
}
