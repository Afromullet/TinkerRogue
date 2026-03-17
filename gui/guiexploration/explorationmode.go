package guiexploration

import (
	"fmt"

	"game_main/gui/framework"
	"game_main/mind/encounter"
	"game_main/tactical/combat"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	framework.BaseMode // Embed common mode infrastructure

	initialized bool

	// Sub-menu controller (manages debug sub-menu visibility)
	subMenus *framework.SubMenuController

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	messageLog *widget.TextArea

	// Input action map (includes camera bindings for exploration)
	actionMap *framework.ActionMap

	// Combat start callback (injected by moderegistry for debug encounters)
	startCombat func(starter combat.CombatStarter) (*combat.CombatStartResult, error)
}

func NewExplorationMode(modeManager *framework.UIModeManager, startCombat func(starter combat.CombatStarter) (*combat.CombatStartResult, error)) *ExplorationMode {
	mode := &ExplorationMode{
		startCombat: startCombat,
	}
	mode.SetModeName("exploration")
	mode.SetReturnMode("") // No return mode - exploration is the main mode
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

// GetActionMap implements framework.ActionMapProvider.
func (em *ExplorationMode) GetActionMap() *framework.ActionMap {
	return em.actionMap
}

func (em *ExplorationMode) Initialize(ctx *framework.UIContext) error {
	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&em.BaseMode, framework.ModeConfig{
		ModeName:   "exploration",
		ReturnMode: "", // No return mode - exploration is the main mode

	}).Build(ctx)

	if err != nil {
		return err
	}

	// Initialize action map: merge camera bindings for exploration mode
	em.actionMap = framework.DefaultCameraBindings()

	// Initialize sub-menu controller before building panels (panels register with it)
	em.subMenus = framework.NewSubMenuController()

	// Build panels from registry (debug menu must be built before action buttons)
	if err := em.BuildPanels(
		ExplorationPanelDebugMenu,
		ExplorationPanelMessageLog,
		ExplorationPanelActionButtons,
	); err != nil {
		return err
	}

	// Initialize widget references from registry
	em.initializeWidgetReferences()

	em.initialized = true
	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (em *ExplorationMode) initializeWidgetReferences() {
	em.messageLog = GetExplorationMessageLog(em.Panels)
}

func (em *ExplorationMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Exploration Mode")

	return nil
}

func (em *ExplorationMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Exploration Mode")
	return nil
}

func (em *ExplorationMode) Update(deltaTime float64) error {
	// Update message log if new messages
	// Update stats if player data changed
	// (Minimal updates - most updates happen in Enter/Exit)
	return nil
}

func (em *ExplorationMode) Render(screen *ebiten.Image) {
	// No custom rendering needed - ebitenui handles everything
	// Could add overlays here (threat ranges, movement paths, etc.)
}

// startMultiFactionBattle creates a debug encounter with the player + 2 AI factions.
func (em *ExplorationMode) startMultiFactionBattle() {
	if em.startCombat == nil {
		fmt.Println("ERROR: startCombat callback not configured")
		return
	}

	playerEntityID := em.Context.PlayerData.PlayerEntityID
	playerPos := *em.Context.PlayerData.Pos

	encounterID, err := encounter.TriggerRandomEncounter(em.Context.ECSManager, 1)
	if err != nil {
		fmt.Printf("ERROR: Failed to create encounter: %v\n", err)
		return
	}

	starter := &encounter.MultiFactionCombatStarter{
		EncounterID:   encounterID,
		PlayerPos:     playerPos,
		RosterOwnerID: playerEntityID,
		FactionCount:  2, // 2 AI factions + player = 3 total
	}

	if _, err := em.startCombat(starter); err != nil {
		fmt.Printf("ERROR: Failed to start multi-faction battle: %v\n", err)
		return
	}
}

func (em *ExplorationMode) HandleInput(inputState *framework.InputState) bool {
	// Handle common input first (ESC key, registered hotkeys)
	if em.HandleCommonInput(inputState) {
		return true
	}

	return false
}
