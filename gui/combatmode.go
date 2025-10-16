package gui

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat
type CombatMode struct {
	ui          *ebitenui.UI
	context     *UIContext
	layout      *LayoutConfig
	modeManager *UIModeManager

	rootContainer  *widget.Container
	turnOrderPanel *widget.Container
	combatLogArea  *widget.TextArea
	actionButtons  *widget.Container

	combatLog []string // Store combat messages
}

func NewCombatMode(modeManager *UIModeManager) *CombatMode {
	return &CombatMode{
		modeManager: modeManager,
		combatLog:   make([]string, 0, 100),
	}
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
	cm.context = ctx
	cm.layout = NewLayoutConfig(ctx)

	cm.ui = &ebitenui.UI{}
	cm.rootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	cm.ui.Container = cm.rootContainer

	// Build combat UI layout
	cm.buildTurnOrderPanel()
	cm.buildCombatLog()
	cm.buildActionButtons()

	return nil
}

func (cm *CombatMode) buildTurnOrderPanel() {
	// Top-center turn order display
	_, _, width, height := cm.layout.TopCenterPanel()

	cm.turnOrderPanel = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(width, height),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				Padding: widget.Insets{
					Top: int(float64(cm.layout.ScreenHeight) * 0.01),
				},
			}),
		),
	)

	// Placeholder for turn order (will be populated dynamically)
	turnOrderLabel := widget.NewText(
		widget.TextOpts.Text("Turn Order: [Squad 1] -> [Squad 2] -> ...", SmallFace, color.White),
	)
	cm.turnOrderPanel.AddChild(turnOrderLabel)

	cm.rootContainer.AddChild(cm.turnOrderPanel)
}

func (cm *CombatMode) buildCombatLog() {
	// Right side combat log
	_, _, width, height := cm.layout.RightSidePanel()

	logConfig := TextAreaConfig{
		MinWidth:  width - 20,
		MinHeight: height - 20,
		FontColor: color.White,
	}
	cm.combatLogArea = CreateTextAreaWithConfig(logConfig)
	cm.combatLogArea.SetText("Combat started!\n")

	logContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(width, height),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding: widget.Insets{
					Right: int(float64(cm.layout.ScreenWidth) * 0.01),
				},
			}),
		),
	)
	logContainer.AddChild(cm.combatLogArea)

	cm.rootContainer.AddChild(logContainer)
}

func (cm *CombatMode) buildActionButtons() {
	// Bottom-center action buttons
	cm.actionButtons = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding: widget.Insets{
					Bottom: int(float64(cm.layout.ScreenHeight) * 0.08),
				},
			}),
		),
	)

	// Attack button
	attackBtn := CreateButton("Attack")
	attackBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			cm.addCombatLog("Attack action selected")
		}),
	)
	cm.actionButtons.AddChild(attackBtn)

	// Defend button
	defendBtn := CreateButton("Defend")
	defendBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			cm.addCombatLog("Defend action selected")
		}),
	)
	cm.actionButtons.AddChild(defendBtn)

	// End Turn button
	endTurnBtn := CreateButton("End Turn")
	endTurnBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			cm.addCombatLog("Turn ended")
		}),
	)
	cm.actionButtons.AddChild(endTurnBtn)

	// Flee button
	fleeBtn := CreateButton("Flee (ESC)")
	fleeBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if exploreMode, exists := cm.modeManager.GetMode("exploration"); exists {
				cm.modeManager.RequestTransition(exploreMode, "Fled from combat")
			}
		}),
	)
	cm.actionButtons.AddChild(fleeBtn)

	cm.rootContainer.AddChild(cm.actionButtons)
}

func (cm *CombatMode) addCombatLog(message string) {
	cm.combatLog = append(cm.combatLog, message)

	// Update text area with all messages
	logText := ""
	for _, msg := range cm.combatLog {
		logText += msg + "\n"
	}
	cm.combatLogArea.SetText(logText)
}

func (cm *CombatMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Combat Mode")
	cm.addCombatLog("=== COMBAT STARTED ===")
	return nil
}

func (cm *CombatMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Combat Mode")
	// Clear combat log for next battle
	cm.combatLog = cm.combatLog[:0]
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	// Update turn order, check for combat end, etc.
	return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	// Custom rendering for combat effects (optional)
}

func (cm *CombatMode) HandleInput(inputState *InputState) bool {
	// ESC to flee combat
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if exploreMode, exists := cm.modeManager.GetMode("exploration"); exists {
			cm.modeManager.RequestTransition(exploreMode, "ESC pressed - fled combat")
			return true
		}
	}

	// Number keys for quick actions (1-4)
	if inputState.KeysJustPressed[ebiten.Key1] {
		cm.addCombatLog("Quick action 1")
		return true
	}

	return false
}

func (cm *CombatMode) GetEbitenUI() *ebitenui.UI {
	return cm.ui
}

func (cm *CombatMode) GetModeName() string {
	return "combat"
}
