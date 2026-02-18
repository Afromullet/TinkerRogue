package guioverworld

import (
	"fmt"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/encounter"
	"game_main/overworld/core"
	"game_main/overworld/threat"
	"game_main/overworld/tick"
	"game_main/tactical/commander"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// OverworldMode handles the overworld UI (threat visualization, tick controls)
type OverworldMode struct {
	framework.BaseMode // Embed common mode infrastructure

	// UI State (owned by GameModeCoordinator, shared across overworld modes)
	state *framework.OverworldState

	// Renderer
	renderer *OverworldRenderer

	// Services
	encounterService *encounter.EncounterService

	// Handlers
	actionHandler *OverworldActionHandler
	inputHandler  *OverworldInputHandler

	// Sub-menu controller (manages debug/node sub-menu visibility)
	subMenus *framework.SubMenuController

	// Widget references (populated from panel registry)
	resourcesText   *widget.TextArea
	threatInfoText  *widget.TextArea
	tickStatusText  *widget.TextArea
	eventLogText    *widget.TextArea
	threatStatsText *widget.TextArea

	// Dirty tracking for event-driven refresh
	lastTick         int64
	lastSelectedNode ecs.EntityID

	// Initialization tracking
	initialized bool
}

func NewOverworldMode(modeManager *framework.UIModeManager, encounterService *encounter.EncounterService) *OverworldMode {
	om := &OverworldMode{
		encounterService: encounterService,
	}
	om.SetModeName("overworld")
	om.SetReturnMode("") // No simple return mode - uses context switching
	om.ModeManager = modeManager
	om.SetSelf(om) // Required for panel registry building
	return om
}

func (om *OverworldMode) Initialize(ctx *framework.UIContext) error {
	// Get persistent state from coordinator
	om.state = ctx.ModeCoordinator.GetOverworldState()

	// Build base UI using ModeBuilder
	err := framework.NewModeBuilder(&om.BaseMode, framework.ModeConfig{
		ModeName:   "overworld",
		ReturnMode: "", // Uses context switching instead of simple mode transition
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Initialize sub-menu controller before building panels (panels register with it)
	om.subMenus = framework.NewSubMenuController()

	// Build panels from registry (sub-menu panels must be built before tick controls)
	if err := om.BuildPanels(
		OverworldPanelDebugMenu,
		OverworldPanelNodeMenu,
		OverworldPanelTickControls,
		OverworldPanelResources,
		OverworldPanelThreatInfo,
		OverworldPanelTickStatus,
		OverworldPanelEventLog,
		OverworldPanelThreatStats,
	); err != nil {
		return err
	}

	// Initialize widget references
	om.initializeWidgetReferences()

	om.renderer = NewOverworldRenderer(ctx.ECSManager, om.state, ctx.GameMap, ctx.TileSize, ctx)

	// Create commander movement system
	cmdMovement := commander.NewCommanderMovementSystem(ctx.ECSManager, common.GlobalPositionSystem)

	// Create deps + handlers
	deps := &OverworldModeDeps{
		State:             om.state,
		Manager:           ctx.ECSManager,
		PlayerData:        ctx.PlayerData,
		EncounterService:  om.encounterService,
		Renderer:          om.renderer,
		ModeManager:       om.ModeManager,
		ModeCoordinator:   ctx.ModeCoordinator,
		CommanderMovement: cmdMovement,
		LogEvent:          om.logEvent,
		RefreshPanels:     om.refreshAllPanels,
	}
	om.actionHandler = NewOverworldActionHandler(deps)
	om.inputHandler = NewOverworldInputHandler(om.actionHandler, deps, om.GetEbitenUI())

	// Ensure tick state exists
	tickState := core.GetTickState(ctx.ECSManager)
	if tickState == nil {
		tick.CreateTickStateEntity(ctx.ECSManager)
		om.logEvent("Tick state initialized")
	}

	om.initialized = true
	om.logEvent("Overworld mode initialized")
	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (om *OverworldMode) initializeWidgetReferences() {
	om.resourcesText = GetOverworldResources(om.Panels)
	om.threatInfoText = GetOverworldThreatInfo(om.Panels)
	om.tickStatusText = GetOverworldTickStatus(om.Panels)
	om.eventLogText = GetOverworldEventLog(om.Panels)
	om.threatStatsText = GetOverworldThreatStats(om.Panels)
}

func (om *OverworldMode) Enter(fromMode framework.UIMode) error {
	om.startRecordingIfNeeded()

	// Auto-select first commander if none selected
	if om.state.SelectedCommanderID == 0 && om.Context.PlayerData != nil {
		commanders := commander.GetAllCommanders(om.Context.PlayerData.PlayerEntityID, om.Context.ECSManager)
		if len(commanders) > 0 {
			om.state.SelectedCommanderID = commanders[0]
		}
	}

	// Refresh UI displays
	om.refreshAllPanels()

	return nil
}

func (om *OverworldMode) Exit(toMode framework.UIMode) error {
	om.exportRecordingIfNeeded()

	// Clear selection when leaving
	om.state.ClearSelection()

	return nil
}

func (om *OverworldMode) Update(deltaTime float64) error {
	// Dirty-check: only refresh tick/stats panels when tick changes
	tickState := core.GetTickState(om.Context.ECSManager)
	if tickState != nil && tickState.CurrentTick != om.lastTick {
		om.lastTick = tickState.CurrentTick
		om.refreshResources()
		om.refreshTickStatus()
		om.refreshThreatStats()
	}

	// Dirty-check: only refresh threat info when selection changes
	if om.state.SelectedNodeID != om.lastSelectedNode {
		om.lastSelectedNode = om.state.SelectedNodeID
		om.refreshThreatInfo()
	}

	return nil
}

func (om *OverworldMode) Render(screen *ebiten.Image) {
	if om.renderer != nil {
		om.renderer.Render(screen)
	}
}

func (om *OverworldMode) HandleInput(inputState *framework.InputState) bool {
	return om.inputHandler.HandleInput(inputState)
}

// UI refresh functions

func (om *OverworldMode) refreshAllPanels() {
	om.refreshResources()
	om.refreshThreatInfo()
	om.refreshTickStatus()
	om.refreshThreatStats()
}

func (om *OverworldMode) refreshResources() {
	if om.resourcesText == nil {
		return
	}

	stockpile := common.GetResourceStockpile(om.Context.PlayerData.PlayerEntityID, om.Context.ECSManager)
	if stockpile == nil {
		om.resourcesText.SetText("Resources: N/A")
		return
	}

	om.resourcesText.SetText(fmt.Sprintf(
		"Gold: %d\nIron: %d\nWood: %d\nStone: %d",
		stockpile.Gold, stockpile.Iron, stockpile.Wood, stockpile.Stone,
	))
}

func (om *OverworldMode) refreshThreatInfo() {
	if om.threatInfoText == nil {
		return
	}

	if !om.state.HasSelection() {
		om.threatInfoText.SetText("Select a threat to view details")
		return
	}

	nodeEntity := om.Context.ECSManager.FindEntityByID(om.state.SelectedNodeID)
	infoText := FormatThreatInfo(nodeEntity, om.Context.ECSManager)
	om.threatInfoText.SetText(infoText)
}

func (om *OverworldMode) refreshTickStatus() {
	if om.tickStatusText == nil {
		return
	}

	tickState := core.GetTickState(om.Context.ECSManager)
	if tickState == nil {
		om.tickStatusText.SetText("Tick: ??? | Status: ERROR")
		return
	}

	statusText := "Ready"
	if tickState.IsGameOver {
		statusText = "Game Over"
	}

	turnState := commander.GetOverworldTurnState(om.Context.ECSManager)
	turnStr := ""
	if turnState != nil {
		turnStr = fmt.Sprintf("Turn: %d | ", turnState.CurrentTurn)
	}

	om.tickStatusText.SetText(fmt.Sprintf(
		"%sTick: %d | Status: %s",
		turnStr,
		tickState.CurrentTick,
		statusText,
	))
}

func (om *OverworldMode) refreshThreatStats() {
	if om.threatStatsText == nil {
		return
	}

	count := threat.CountThreatNodes(om.Context.ECSManager)
	avgIntensity := threat.CalculateAverageIntensity(om.Context.ECSManager)

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

	currentText := om.eventLogText.GetText()
	newText := message + "\n" + currentText

	maxChars := 2000
	if len(newText) > maxChars {
		newText = newText[:maxChars]
	}

	om.eventLogText.SetText(newText)
}

// Recording helpers

func (om *OverworldMode) startRecordingIfNeeded() {
	ctx := core.GetContext()
	if ctx.Recorder != nil && ctx.Recorder.IsEnabled() {
		tickState := core.GetTickState(om.Context.ECSManager)
		if tickState != nil && ctx.Recorder.EventCount() == 0 {
			core.StartRecordingSession(tickState.CurrentTick)
		}
	}
}

func (om *OverworldMode) exportRecordingIfNeeded() {
	ctx := core.GetContext()
	if ctx.Recorder == nil || !ctx.Recorder.IsEnabled() {
		return
	}

	tickState := core.GetTickState(om.Context.ECSManager)
	if tickState == nil || tickState.IsGameOver {
		return
	}

	if ctx.Recorder.EventCount() == 0 {
		return
	}

	tickMsg := fmt.Sprintf("tick %d", tickState.CurrentTick)
	if err := core.FinalizeRecording("Session Paused", fmt.Sprintf("Left overworld at %s", tickMsg)); err != nil {
		fmt.Printf("WARNING: Failed to export overworld log on exit: %v\n", err)
	}
	core.ClearRecording()
}
