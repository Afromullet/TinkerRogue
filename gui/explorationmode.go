package gui

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	ui          *ebitenui.UI
	context     *UIContext
	layout      *LayoutConfig
	initialized bool

	// UI Components (ebitenui widgets)
	rootContainer  *widget.Container
	statsPanel     *widget.Container
	statsTextArea  *widget.TextArea
	messageLog     *widget.TextArea
	quickInventory *widget.Container
	infoWindow     *InfoUI

	// Mode manager reference (for transitions)
	modeManager *UIModeManager
}

func NewExplorationMode(modeManager *UIModeManager) *ExplorationMode {
	return &ExplorationMode{
		modeManager: modeManager,
	}
}

func (em *ExplorationMode) Initialize(ctx *UIContext) error {
	em.context = ctx
	em.layout = NewLayoutConfig(ctx)

	// Create ebitenui root
	em.ui = &ebitenui.UI{}
	em.rootContainer = widget.NewContainer()
	em.ui.Container = em.rootContainer

	// Build exploration-specific UI layout
	em.buildStatsPanel()
	em.buildMessageLog()
	em.buildQuickInventory()
	em.buildInfoWindow()

	em.initialized = true
	return nil
}

func (em *ExplorationMode) buildStatsPanel() {
	// Get responsive position
	x, y, width, height := em.layout.TopRightPanel()

	// Stats panel (top-right corner)
	em.statsPanel = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{
				Left: 10, Right: 10, Top: 10, Bottom: 10,
			}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(width, height),
		),
	)

	// Stats text area
	statsConfig := TextAreaConfig{
		MinWidth:  width - 20,
		MinHeight: height - 20,
		FontColor: color.White,
	}
	em.statsTextArea = CreateTextAreaWithConfig(statsConfig)
	em.statsTextArea.SetText(em.context.PlayerData.PlayerAttributes().DisplayString())

	em.statsPanel.AddChild(em.statsTextArea)

	// Position using responsive layout
	SetContainerLocation(em.statsPanel, x, y)

	em.rootContainer.AddChild(em.statsPanel)
}

func (em *ExplorationMode) buildMessageLog() {
	// Get responsive position
	x, y, width, height := em.layout.BottomRightPanel()

	// Message log (bottom-right corner)
	logConfig := TextAreaConfig{
		MinWidth:  width - 20,
		MinHeight: height - 20,
		FontColor: color.White,
	}
	em.messageLog = CreateTextAreaWithConfig(logConfig)

	logContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(width, height),
		),
	)
	logContainer.AddChild(em.messageLog)

	// Position using responsive layout
	SetContainerLocation(logContainer, x, y)

	em.rootContainer.AddChild(logContainer)
}

func (em *ExplorationMode) buildQuickInventory() {
	// Get responsive position
	x, y := em.layout.BottomCenterButtons()

	// Quick inventory buttons (bottom-center)
	em.quickInventory = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
		)),
	)

	// Throwables button
	throwableBtn := CreateButton("Throwables")
	throwableBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			// Transition to inventory mode (throwables)
			if invMode, exists := em.modeManager.GetMode("inventory"); exists {

				//TODO remove this in the future. Just here for testing
				// Set the initial filter to "Throwables" before transitioning
				if inventoryMode, ok := invMode.(*InventoryMode); ok {
					inventoryMode.SetInitialFilter("Throwables")
				}
				em.modeManager.RequestTransition(invMode, "Open Throwables")
			}
		}),
	)
	em.quickInventory.AddChild(throwableBtn)

	// Inventory button
	inventoryBtn := CreateButton("Inventory (I)")
	inventoryBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if invMode, exists := em.modeManager.GetMode("inventory"); exists {
				em.modeManager.RequestTransition(invMode, "Open Inventory")
			}
		}),
	)
	em.quickInventory.AddChild(inventoryBtn)

	// Squad button
	squadBtn := CreateButton("Squads (E)")
	squadBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if squadMode, exists := em.modeManager.GetMode("squad_management"); exists {
				em.modeManager.RequestTransition(squadMode, "Open Squad Management")
			}
		}),
	)
	em.quickInventory.AddChild(squadBtn)

	// Squad Builder button
	builderBtn := CreateButton("Builder (B)")
	builderBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if builderMode, exists := em.modeManager.GetMode("squad_builder"); exists {
				em.modeManager.RequestTransition(builderMode, "Open Squad Builder")
			}
		}),
	)
	em.quickInventory.AddChild(builderBtn)

	// Position using responsive layout
	SetContainerLocation(em.quickInventory, x, y)

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
	// Handle right-click info window
	if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
		// Only open if not in other input modes
		if !inputState.PlayerInputStates.IsThrowing {
			em.infoWindow.InfoSelectionWindow(inputState.MouseX, inputState.MouseY)
			inputState.PlayerInputStates.InfoMeuOpen = true
			return true
		}
	}

	// Handle info window closing
	if inputState.PlayerInputStates.InfoMeuOpen {
		if inputState.KeysJustPressed[ebiten.KeyEscape] {
			em.infoWindow.CloseWindows()
			inputState.PlayerInputStates.InfoMeuOpen = false
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

	return false // Input not consumed, let game logic handle
}

func (em *ExplorationMode) GetEbitenUI() *ebitenui.UI {
	return em.ui
}

func (em *ExplorationMode) GetModeName() string {
	return "exploration"
}
