package gui

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	BaseMode // Embed common mode infrastructure

	initialized bool

	// UI Components (ebitenui widgets)
	statsPanel     *widget.Container
	statsTextArea  *widget.TextArea
	messageLog     *widget.TextArea
	quickInventory *widget.Container
	infoWindow     *InfoUI
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

	// Build stats panel (top-right) using BuildPanel
	em.statsPanel = em.panelBuilders.BuildPanel(
		TopRight(),
		Size(0.15, 0.2),
		Padding(0.01),
		AnchorLayout(),
	)

	// Create stats text area inside panel
	panelWidth := int(float64(em.layout.ScreenWidth) * 0.15)
	panelHeight := int(float64(em.layout.ScreenHeight) * 0.2)
	em.statsTextArea = CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})
	em.statsTextArea.SetText(em.context.PlayerData.PlayerAttributes().DisplayString())
	em.statsPanel.AddChild(em.statsTextArea)
	em.rootContainer.AddChild(em.statsPanel)

	// Build message log (bottom-right) using BuildPanel
	logContainer := em.panelBuilders.BuildPanel(
		BottomRight(),
		Size(0.15, 0.15),
		Padding(0.01),
		AnchorLayout(),
	)

	// Create message log text area inside panel
	logWidth := int(float64(em.layout.ScreenWidth) * 0.15)
	logHeight := int(float64(em.layout.ScreenHeight) * 0.15)
	em.messageLog = CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  logWidth - 20,
		MinHeight: logHeight - 20,
		FontColor: color.White,
	})
	logContainer.AddChild(em.messageLog)
	em.rootContainer.AddChild(logContainer)

	// Build exploration-specific UI layout
	em.buildQuickInventory()
	em.buildInfoWindow()

	em.initialized = true
	return nil
}

func (em *ExplorationMode) buildQuickInventory() {
	// Use BuildPanel for bottom-center button container
	em.quickInventory = em.panelBuilders.BuildPanel(
		BottomCenter(),
		HorizontalRowLayout(),
		CustomPadding(widget.Insets{
			Bottom: int(float64(em.layout.ScreenHeight) * 0.08),
		}),
	)

	// Throwables button
	throwableBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Throwables",
		OnClick: func() {
			// Transition to inventory mode (throwables)
			if invMode, exists := em.modeManager.GetMode("inventory"); exists {
				//TODO remove this in the future. Just here for testing
				// Set the initial filter to "Throwables" before transitioning
				if inventoryMode, ok := invMode.(*InventoryMode); ok {
					inventoryMode.SetInitialFilter("Throwables")
				}
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

func (em *ExplorationMode) buildInfoWindow() {
	// Create info window (right-click inspection)
	infoUI := CreateInfoUI(em.context.ECSManager, em.ui)
	em.infoWindow = &infoUI
}

func (em *ExplorationMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Exploration Mode")

	// Refresh player stats
	if em.context.PlayerData != nil && em.statsTextArea != nil {
		em.statsTextArea.SetText(em.context.PlayerData.PlayerAttributes().DisplayString())
	}

	return nil
}

func (em *ExplorationMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Exploration Mode")

	// Close any open info windows
	if em.infoWindow != nil {
		em.infoWindow.CloseWindows()
	}

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
	// Handle info window closing first (higher priority than ESC navigation)
	if inputState.PlayerInputStates.InfoMeuOpen {
		if inputState.KeysJustPressed[ebiten.KeyEscape] {
			em.infoWindow.CloseWindows()
			inputState.PlayerInputStates.InfoMeuOpen = false
			return true
		}
	}

	// Handle right-click info window
	if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
		// Only open if not in other input modes
		if !inputState.PlayerInputStates.IsThrowing {
			em.infoWindow.InfoSelectionWindow(inputState.MouseX, inputState.MouseY)
			inputState.PlayerInputStates.InfoMeuOpen = true
			return true
		}
	}

	// Check for mode transition hotkeys
	if inputState.KeysJustPressed[ebiten.KeyE] {
		// Open squad management
		if squadMode, exists := em.modeManager.GetMode("squad_management"); exists {
			em.modeManager.RequestTransition(squadMode, "E key pressed")
			return true
		}
	}

	if inputState.KeysJustPressed[ebiten.KeyI] {
		// Open full inventory
		if invMode, exists := em.modeManager.GetMode("inventory"); exists {
			em.modeManager.RequestTransition(invMode, "I key pressed")
			return true
		}
	}

	if inputState.KeysJustPressed[ebiten.KeyB] {
		// Open squad builder
		if builderMode, exists := em.modeManager.GetMode("squad_builder"); exists {
			em.modeManager.RequestTransition(builderMode, "B key pressed")
			return true
		}
	}

	if inputState.KeysJustPressed[ebiten.KeyC] {
		// Enter combat mode
		if combatMode, exists := em.modeManager.GetMode("combat"); exists {
			em.modeManager.RequestTransition(combatMode, "C key pressed")
			return true
		}
	}

	return false // Input not consumed, let game logic handle
}
