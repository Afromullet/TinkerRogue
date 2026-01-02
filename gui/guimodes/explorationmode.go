package guimodes

import (
	"fmt"

	"game_main/common"
	"game_main/gui"
	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/world/encounter"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	gui.BaseMode // Embed common mode infrastructure

	initialized bool

	// UI Components (ebitenui widgets)
	messageLog     *widget.TextArea
	quickInventory *widget.Container
}

func NewExplorationMode(modeManager *core.UIModeManager) *ExplorationMode {
	mode := &ExplorationMode{}
	mode.SetModeName("exploration")
	mode.ModeManager = modeManager
	return mode
}

func (em *ExplorationMode) Initialize(ctx *core.UIContext) error {
	// Use ModeBuilder for declarative initialization (reduces 60+ lines to ~30)
	err := gui.NewModeBuilder(&em.BaseMode, gui.ModeConfig{
		ModeName:   "exploration",
		ReturnMode: "", // No return mode - exploration is the main mode

		// Register hotkeys for mode transitions (Battle Map context only)
		Hotkeys: []gui.HotkeySpec{
			{Key: ebiten.KeyI, TargetMode: "inventory"},
			{Key: ebiten.KeyC, TargetMode: "combat"},
			{Key: ebiten.KeyD, TargetMode: "squad_deployment"},
			// Note: 'E' key for squads requires context switch - handled in button
		},

		// Build panels
		Panels: []gui.PanelSpec{
			{
				// Message log panel (bottom-right) - now uses typed panel
				PanelType:  builders.PanelTypeDetail,
				SpecName:   "message_log",
				DetailText: "",
			},
			{
				// Quick inventory panel (custom build)
				CustomBuild: em.buildQuickInventory,
			},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Get reference to message log TextArea from typed panel
	if w, ok := em.PanelWidgets["message_log"]; ok {
		if textArea, ok := w.(*widget.TextArea); ok {
			em.messageLog = textArea
		}
	}

	em.initialized = true
	return nil
}

func (em *ExplorationMode) buildQuickInventory() *widget.Container {
	// Create UI factory
	uiFactory := gui.NewUIComponentFactory(em.Queries, em.PanelBuilders, em.Layout)

	// Create button callbacks (no panel wrapper - like combat mode)
	quickInventory := uiFactory.CreateExplorationActionButtons(
		// Throwables
		func() {
			if mode, exists := em.ModeManager.GetMode("inventory"); exists {
				em.ModeManager.RequestTransition(mode, "Throwables clicked")
			}
		},
		// Squads (switches to Overworld context)
		func() {
			if em.Context.ModeCoordinator != nil {
				if err := em.Context.ModeCoordinator.ReturnToOverworld("squad_management"); err != nil {
					fmt.Printf("ERROR: Failed to return to overworld: %v\n", err)
				}
			}
		},
		// Inventory
		func() {
			if mode, exists := em.ModeManager.GetMode("inventory"); exists {
				em.ModeManager.RequestTransition(mode, "Inventory clicked")
			}
		},
		// Deploy
		func() {
			if mode, exists := em.ModeManager.GetMode("squad_deployment"); exists {
				em.ModeManager.RequestTransition(mode, "Deploy clicked")
			}
		},
		// Combat
		func() {
			if mode, exists := em.ModeManager.GetMode("combat"); exists {
				em.ModeManager.RequestTransition(mode, "Combat clicked")
			}
		},
	)

	// Store reference and return
	em.quickInventory = quickInventory
	return quickInventory
}

func (em *ExplorationMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Exploration Mode")

	return nil
}

func (em *ExplorationMode) Exit(toMode core.UIMode) error {
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

func (em *ExplorationMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input first (ESC key, registered hotkeys like I/C/D)
	if em.HandleCommonInput(inputState) {
		return true
	}

	return false
}

// triggerCombat transitions to combat mode when an encounter is triggered
func (em *ExplorationMode) triggerCombat(encounterID ecs.EntityID) {
	// Store encounter ID in BattleMapState for combat mode to use
	if em.Context.ModeCoordinator != nil {
		battleMapState := em.Context.ModeCoordinator.GetBattleMapState()
		battleMapState.TriggeredEncounterID = encounterID
	}

	// Log encounter details
	entity := em.Context.ECSManager.FindEntityByID(encounterID)
	if entity != nil {
		encounterData := common.GetComponentType[*encounter.OverworldEncounterData](
			entity,
			encounter.OverworldEncounterComponent,
		)
		if encounterData != nil {
			fmt.Printf("Triggering combat encounter: %s (Level %d)\n",
				encounterData.Name, encounterData.Level)
		}
	}

	// Transition to combat mode
	if em.Context.ModeCoordinator != nil {
		em.Context.ModeCoordinator.EnterBattleMap("combat")
	}
}
