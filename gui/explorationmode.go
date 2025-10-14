package gui

import (
	"fmt"
	"image"
	"image/color"

	"game_main/common"
	"game_main/gear"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	ui            *ebitenui.UI
	context       *UIContext
	layout        *LayoutConfig
	initialized   bool

	// UI Components
	rootContainer     *widget.Container
	statsPanel        *widget.Container
	statsTextArea     *widget.TextArea
	messageLog        *widget.TextArea
	quickInventory    *widget.Container
	infoWindow        *InfoUI
	throwablesWindow  *widget.Window
	throwablesList    *widget.List

	// Mode manager reference
	modeManager *UIModeManager
}

// NewExplorationMode creates a new exploration mode
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
			fmt.Println("Throwables button clicked")
			em.openThrowablesWindow()
		}),
	)
	em.quickInventory.AddChild(throwableBtn)

	// Squad button
	squadBtn := CreateButton("Squads (E)")
	squadBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			// Transition to squad management mode (when implemented)
			fmt.Println("Squads button clicked")
		}),
	)
	em.quickInventory.AddChild(squadBtn)

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
	// Update stats if player data changed
	if em.context.PlayerData != nil && em.statsTextArea != nil {
		em.statsTextArea.SetText(em.context.PlayerData.PlayerAttributes().DisplayString())
	}
	return nil
}

func (em *ExplorationMode) Render(screen *ebiten.Image) {
	// No custom rendering needed - ebitenui handles everything
}

func (em *ExplorationMode) HandleInput(inputState *InputState) bool {
	// Handle right-click info window
	if inputState.MouseButton == ebiten.MouseButtonRight && inputState.MousePressed {
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

	// Check for mode transition hotkeys (when other modes are implemented)
	if inputState.KeysJustPressed[ebiten.KeyE] {
		fmt.Println("E key pressed - squad management mode not yet implemented")
		return true
	}

	if inputState.KeysJustPressed[ebiten.KeyI] {
		fmt.Println("I key pressed - inventory mode not yet implemented")
		return true
	}

	return false // Input not consumed, let game logic handle
}

func (em *ExplorationMode) GetEbitenUI() *ebitenui.UI {
	return em.ui
}

func (em *ExplorationMode) GetModeName() string {
	return "exploration"
}

// openThrowablesWindow opens a modal window showing throwable items
func (em *ExplorationMode) openThrowablesWindow() {
	if em.throwablesWindow != nil {
		// Window already exists, just show it
		em.ui.AddWindow(em.throwablesWindow)
		return
	}

	// Create throwables list window
	em.buildThrowablesWindow()
	em.ui.AddWindow(em.throwablesWindow)
}

// buildThrowablesWindow creates the throwables selection window
func (em *ExplorationMode) buildThrowablesWindow() {
	// Get throwable item entities from player inventory
	throwableEntities := em.getThrowableItemEntities()

	// Create list entries
	entries := make([]interface{}, 0, len(throwableEntities))
	for i, entity := range throwableEntities {
		// Get item name from NameComponent
		nameComp := common.GetComponentType[*common.Name](entity, common.NameComponent)
		itemName := fmt.Sprintf("%d. %s", i+1, nameComp.NameStr)
		entries = append(entries, itemName)
	}

	if len(entries) == 0 {
		entries = append(entries, "No throwable items in inventory")
	}

	// Create list widget
	em.throwablesList = widget.NewList(
		widget.ListOpts.Entries(entries),
		widget.ListOpts.EntryLabelFunc(func(e interface{}) string {
			return e.(string)
		}),
		widget.ListOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(ListRes.image),
		),
		widget.ListOpts.SliderOpts(
			widget.SliderOpts.Images(ListRes.track, ListRes.handle),
		),
		widget.ListOpts.EntryColor(ListRes.entry),
		widget.ListOpts.EntryFontFace(ListRes.face),
		widget.ListOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(300, 400),
			),
		),
	)

	// Handle item selection
	em.throwablesList.EntrySelectedEvent.AddHandler(func(args interface{}) {
		a := args.(*widget.ListEntrySelectedEventArgs)
		selectedIndex := a.Entry

		// Find the index in the original entries list
		for i, entry := range entries {
			if entry == selectedIndex {
				if i < len(throwableEntities) {
					// Prepare the throwable
					em.context.PlayerData.Throwables.PrepareThrowable(throwableEntities[i], i)
					em.context.PlayerData.InputStates.IsThrowing = true

					// Get item name for logging
					nameComp := common.GetComponentType[*common.Name](throwableEntities[i], common.NameComponent)
					fmt.Printf("Selected throwable: %s\n", nameComp.NameStr)
				}
				break
			}
		}

		// Close the window
		em.throwablesWindow.Close()
	})

	// Create container for the list
	listContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 15, Right: 15, Top: 15, Bottom: 15}),
		)),
	)

	// Add title
	titleText := widget.NewText(
		widget.TextOpts.Text("Select Throwable Item", LargeFace, color.White),
	)
	listContainer.AddChild(titleText)
	listContainer.AddChild(em.throwablesList)

	// Create window
	em.throwablesWindow = widget.NewWindow(
		widget.WindowOpts.Contents(listContainer),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.MinSize(350, 500),
	)

	// Center the window using rect construction
	x, y, width, height := em.layout.CenterWindow(0.3, 0.6)
	r := image.Rect(x, y, x+width, y+height)
	em.throwablesWindow.SetLocation(r)
}

// getThrowableItemEntities returns all throwable item entities from player inventory
func (em *ExplorationMode) getThrowableItemEntities() []*ecs.Entity {
	if em.context.PlayerData == nil || em.context.PlayerData.Inventory == nil {
		return []*ecs.Entity{}
	}

	inv := em.context.PlayerData.Inventory
	throwables := make([]*ecs.Entity, 0)

	for _, itemEntity := range inv.InventoryContent {
		item := gear.GetItem(itemEntity)
		if item != nil && item.GetThrowableAction() != nil {
			throwables = append(throwables, itemEntity)
		}
	}

	return throwables
}
