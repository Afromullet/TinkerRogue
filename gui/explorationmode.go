package gui

import (
	"fmt"
	"game_main/coords"
	"game_main/graphics"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	BaseMode // Embed common mode infrastructure

	initialized bool

	// UI Components (ebitenui widgets)
	statsPanel          *widget.Container
	statsTextArea       *widget.TextArea
	statsComponent      *StatsDisplayComponent
	messageLog          *widget.TextArea
	quickInventory      *widget.Container
}

func NewExplorationMode(modeManager *UIModeManager) *ExplorationMode {
	return &ExplorationMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "exploration",
			returnMode:  "exploration",
		},
	}
}

func (em *ExplorationMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure
	em.InitializeBase(ctx)

	// Register hotkeys for mode transitions
	em.RegisterHotkey(ebiten.KeyE, "squad_management")
	em.RegisterHotkey(ebiten.KeyI, "inventory")
	em.RegisterHotkey(ebiten.KeyB, "squad_builder")
	em.RegisterHotkey(ebiten.KeyC, "combat")

	// Build stats panel (top-right) using helper
	em.statsPanel, em.statsTextArea = CreateDetailPanel(
		em.panelBuilders,
		em.layout,
		TopRight(),
		PanelWidthNarrow, PanelHeightSmall, PaddingTight,
		em.context.PlayerData.PlayerAttributes(em.context.ECSManager).DisplayString(),
	)
	em.rootContainer.AddChild(em.statsPanel)

	// Create stats display component to manage refresh logic
	em.statsComponent = NewStatsDisplayComponent(em.statsTextArea, nil)
	// Use default formatter which displays player attributes

	// Build message log (bottom-right) using helper
	logContainer, messageLog := CreateDetailPanel(
		em.panelBuilders,
		em.layout,
		BottomRight(),
		PanelWidthNarrow, 0.15, PaddingTight,
		"",
	)
	em.messageLog = messageLog
	em.rootContainer.AddChild(logContainer)

	// Build exploration-specific UI layout
	em.buildQuickInventory()

	em.initialized = true
	return nil
}

func (em *ExplorationMode) buildQuickInventory() {
	// Use BuildPanel for bottom-center button container
	em.quickInventory = em.panelBuilders.BuildPanel(
		BottomCenter(),
		HorizontalRowLayout(),
		CustomPadding(widget.Insets{
			Bottom: int(float64(em.layout.ScreenHeight) * BottomButtonOffset),
		}),
	)

	// Throwables button
	throwableBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Throwables",
		OnClick: func() {
			// Transition to inventory mode
			if invMode, exists := em.modeManager.GetMode("inventory"); exists {
				em.modeManager.RequestTransition(invMode, "Open Throwables")
			}
		},
	})
	em.quickInventory.AddChild(throwableBtn)

	// Inventory button
	inventoryBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Inventory (I)",
		OnClick: func() {
			if invMode, exists := em.modeManager.GetMode("inventory"); exists {
				em.modeManager.RequestTransition(invMode, "Open Inventory")
			}
		},
	})
	em.quickInventory.AddChild(inventoryBtn)

	// Squad button
	squadBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Squads (E)",
		OnClick: func() {
			if squadMode, exists := em.modeManager.GetMode("squad_management"); exists {
				em.modeManager.RequestTransition(squadMode, "Open Squad Management")
			}
		},
	})
	em.quickInventory.AddChild(squadBtn)

	// Squad Builder button
	builderBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Builder (B)",
		OnClick: func() {
			if builderMode, exists := em.modeManager.GetMode("squad_builder"); exists {
				em.modeManager.RequestTransition(builderMode, "Open Squad Builder")
			}
		},
	})
	em.quickInventory.AddChild(builderBtn)

	// Squad Deployment button
	deployBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Deploy (D)",
		OnClick: func() {
			if deployMode, exists := em.modeManager.GetMode("squad_deployment"); exists {
				em.modeManager.RequestTransition(deployMode, "Open Squad Deployment")
			}
		},
	})
	em.quickInventory.AddChild(deployBtn)

	// Combat button
	combatBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Combat (C)",
		OnClick: func() {
			if combatMode, exists := em.modeManager.GetMode("combat"); exists {
				em.modeManager.RequestTransition(combatMode, "Enter Combat")
			}
		},
	})
	em.quickInventory.AddChild(combatBtn)

	em.rootContainer.AddChild(em.quickInventory)
}

func (em *ExplorationMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Exploration Mode")

	// Refresh player stats using component
	if em.statsComponent != nil && em.context.PlayerData != nil {
		em.statsComponent.RefreshStats(em.context.PlayerData, em.context.ECSManager)
	}

	return nil
}

func (em *ExplorationMode) Exit(toMode UIMode) error {
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

func (em *ExplorationMode) HandleInput(inputState *InputState) bool {
	// Handle right-click to open info mode
	if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
		// Only open if not in other input modes
		if !inputState.PlayerInputStates.IsThrowing {
			// Convert mouse position to logical position
			playerPos := *em.context.PlayerData.Pos
			manager := coords.NewCoordinateManager(graphics.ScreenInfo)
			viewport := coords.NewViewport(manager, playerPos)
			clickedPos := viewport.ScreenToLogical(inputState.MouseX, inputState.MouseY)

			// Transition to info mode with position
			if infoMode, exists := em.modeManager.GetMode("info_inspect"); exists {
				if infoModeTyped, ok := infoMode.(*InfoMode); ok {
					infoModeTyped.SetInspectPosition(clickedPos)
					em.modeManager.RequestTransition(infoMode, "Right-click inspection")
				}
			}
			return true
		}
	}

	// Mode transition hotkeys are now handled by BaseMode.HandleCommonInput via RegisterHotkey
	return false
}
