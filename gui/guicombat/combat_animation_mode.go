package guicombat

import (
	"fmt"
	"image/color"

	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/widgets"
	"game_main/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// AnimationPhase represents the current state of the combat animation
type AnimationPhase int

const (
	PhaseIdle       AnimationPhase = iota // Display both squads statically
	PhaseAttacking                        // Show attack animation effect
	PhaseWaiting                          // Wait for player input
	PhaseComplete                         // Animation done, execute callback
)

// Animation timing constants (in seconds)
const (
	IdleDuration      = 0.5
	AttackingDuration = 0.5
)

// CombatAnimationMode displays a full-screen battle scene during combat.
// Shows both squads side-by-side with units at their grid positions.
type CombatAnimationMode struct {
	gui.BaseMode

	// Combat participants
	attackerSquadID ecs.EntityID
	defenderSquadID ecs.EntityID

	// Animation state
	animationPhase AnimationPhase
	animationTimer float64

	// Callback to execute after animation
	onAnimationComplete func()

	// Renderer
	squadRenderer *SquadCombatRenderer

	// UI elements
	promptLabel *widget.Text

	// Layout constants (calculated on initialize)
	cellSize     int
	attackerX    int
	attackerY    int
	defenderX    int
	defenderY    int
	gridWidth    int
	gridHeight   int
	screenWidth  int
	screenHeight int
}

// NewCombatAnimationMode creates a new combat animation mode
func NewCombatAnimationMode(modeManager *core.UIModeManager) *CombatAnimationMode {
	cam := &CombatAnimationMode{
		animationPhase: PhaseIdle,
	}
	cam.SetModeName("combat_animation")
	cam.SetReturnMode("combat") // ESC returns to combat mode
	cam.ModeManager = modeManager
	return cam
}

// SetCombatants sets the attacker and defender squads for the animation
func (cam *CombatAnimationMode) SetCombatants(attackerID, defenderID ecs.EntityID) {
	cam.attackerSquadID = attackerID
	cam.defenderSquadID = defenderID
	fmt.Printf("[DEBUG] SetCombatants: attacker=%d, defender=%d\n", attackerID, defenderID)
}

// SetOnComplete sets the callback to execute when animation completes
func (cam *CombatAnimationMode) SetOnComplete(callback func()) {
	cam.onAnimationComplete = callback
}

// Initialize sets up the combat animation mode
func (cam *CombatAnimationMode) Initialize(ctx *core.UIContext) error {
	cam.InitializeBase(ctx)

	// Create squad renderer
	cam.squadRenderer = NewSquadCombatRenderer(cam.Queries)

	// Store screen dimensions
	cam.screenWidth = ctx.ScreenWidth
	cam.screenHeight = ctx.ScreenHeight

	// Calculate layout
	cam.calculateLayout()

	// Create prompt label (centered at bottom)
	cam.promptLabel = widgets.CreateLargeLabel("")
	promptContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding:            widget.NewInsetsSimple(40),
			}),
		),
	)
	promptContainer.AddChild(cam.promptLabel)
	cam.RootContainer.AddChild(promptContainer)

	return nil
}

// calculateLayout computes the positions and sizes for rendering
func (cam *CombatAnimationMode) calculateLayout() {
	// Cell size: fit 3 cells in about 1/4 of screen width, with some padding
	cam.cellSize = cam.screenWidth / 12 // Each grid is 3 cells, we want 2 grids + gap
	if cam.cellSize > 96 {
		cam.cellSize = 96 // Cap at reasonable size
	}
	if cam.cellSize < 32 {
		cam.cellSize = 32 // Minimum size
	}

	cam.gridWidth = 3 * cam.cellSize
	cam.gridHeight = 3 * cam.cellSize

	// Gap between the two grids
	gap := cam.screenWidth / 8

	// Center both grids vertically
	centerY := (cam.screenHeight - cam.gridHeight) / 2

	// Position attacker on left, defender on right
	centerX := cam.screenWidth / 2

	cam.attackerX = centerX - gap/2 - cam.gridWidth
	cam.attackerY = centerY

	cam.defenderX = centerX + gap/2
	cam.defenderY = centerY
}

// Enter is called when switching to this mode
func (cam *CombatAnimationMode) Enter(fromMode core.UIMode) error {
	// Reset animation state
	cam.animationPhase = PhaseIdle
	cam.animationTimer = 0
	cam.promptLabel.Label = ""
	return nil
}

// Exit is called when switching from this mode
func (cam *CombatAnimationMode) Exit(toMode core.UIMode) error {
	return nil
}

// Update advances the animation state
func (cam *CombatAnimationMode) Update(deltaTime float64) error {
	cam.animationTimer += deltaTime

	switch cam.animationPhase {
	case PhaseIdle:
		if cam.animationTimer >= IdleDuration {
			cam.animationPhase = PhaseAttacking
			cam.animationTimer = 0
		}

	case PhaseAttacking:
		if cam.animationTimer >= AttackingDuration {
			cam.animationPhase = PhaseWaiting
			cam.animationTimer = 0
			cam.promptLabel.Label = "Press any key to continue..."
		}

	case PhaseWaiting:
		// Handled in HandleInput

	case PhaseComplete:
		// Execute callback and return to combat mode
		if cam.onAnimationComplete != nil {
			cam.onAnimationComplete()
		}
		// Return to combat mode
		if combatMode, exists := cam.ModeManager.GetMode("combat"); exists {
			cam.ModeManager.RequestTransition(combatMode, "Animation complete")
		}
	}

	return nil
}

// Render draws the combat animation scene
func (cam *CombatAnimationMode) Render(screen *ebiten.Image) {
	// Fill screen with dark background
	screen.Fill(color.RGBA{R: 30, G: 30, B: 40, A: 255})

	// Draw grid backgrounds (subtle cell outlines)
	cam.drawGridBackground(screen, cam.attackerX, cam.attackerY)
	cam.drawGridBackground(screen, cam.defenderX, cam.defenderY)

	// Debug: verify squad IDs
	fmt.Printf("[DEBUG] Render: attackerID=%d, defenderID=%d\n", cam.attackerSquadID, cam.defenderSquadID)

	// Render squads based on animation phase
	switch cam.animationPhase {
	case PhaseIdle:
		// Static display
		cam.squadRenderer.RenderSquad(screen, cam.attackerSquadID, cam.attackerX, cam.attackerY, cam.cellSize, false)
		cam.squadRenderer.RenderSquad(screen, cam.defenderSquadID, cam.defenderX, cam.defenderY, cam.cellSize, true)

	case PhaseAttacking:
		// Highlight attacking units with a flash effect
		highlightColor := cam.getAttackHighlightColor()
		attackingUnits := cam.getAttackingUnits()

		cam.squadRenderer.RenderSquadWithHighlight(
			screen, cam.attackerSquadID, cam.attackerX, cam.attackerY, cam.cellSize, false,
			attackingUnits, highlightColor,
		)
		cam.squadRenderer.RenderSquad(screen, cam.defenderSquadID, cam.defenderX, cam.defenderY, cam.cellSize, true)

	case PhaseWaiting, PhaseComplete:
		// Static display
		cam.squadRenderer.RenderSquad(screen, cam.attackerSquadID, cam.attackerX, cam.attackerY, cam.cellSize, false)
		cam.squadRenderer.RenderSquad(screen, cam.defenderSquadID, cam.defenderX, cam.defenderY, cam.cellSize, true)
	}

	// Draw squad names above each grid
	cam.drawSquadNames(screen)
}

// drawGridBackground draws subtle cell outlines for the grid
func (cam *CombatAnimationMode) drawGridBackground(screen *ebiten.Image, baseX, baseY int) {
	gridColor := color.RGBA{R: 60, G: 60, B: 70, A: 255}

	// Draw horizontal lines
	for row := 0; row <= 3; row++ {
		y := baseY + row*cam.cellSize
		for x := baseX; x < baseX+cam.gridWidth; x++ {
			screen.Set(x, y, gridColor)
		}
	}

	// Draw vertical lines
	for col := 0; col <= 3; col++ {
		x := baseX + col*cam.cellSize
		for y := baseY; y < baseY+cam.gridHeight; y++ {
			screen.Set(x, y, gridColor)
		}
	}
}

// drawSquadNames draws the squad names above each grid
func (cam *CombatAnimationMode) drawSquadNames(screen *ebiten.Image) {
	// Get squad names
	attackerName := squads.GetSquadName(cam.attackerSquadID, cam.Queries.ECSManager)
	defenderName := squads.GetSquadName(cam.defenderSquadID, cam.Queries.ECSManager)

	// For now, we'll rely on ebitenui to render text via the prompt label
	// Squad names could be drawn here using text rendering, but we'll keep it simple
	// The prompt label shows the current state
	_ = attackerName
	_ = defenderName
}

// getAttackHighlightColor returns a pulsing highlight color for attacking units
func (cam *CombatAnimationMode) getAttackHighlightColor() *ebiten.ColorScale {
	// Create a pulsing red/orange highlight
	// Pulse based on animation timer
	pulse := float32(0.5 + 0.5*float64(cam.animationTimer/AttackingDuration))

	colorScale := &ebiten.ColorScale{}
	colorScale.SetR(1.0 + pulse*0.5) // Boost red
	colorScale.SetG(1.0 - pulse*0.3) // Reduce green slightly
	colorScale.SetB(1.0 - pulse*0.5) // Reduce blue
	colorScale.SetA(1.0)

	return colorScale
}

// getAttackingUnits returns the units that are attacking (all alive units for now)
func (cam *CombatAnimationMode) getAttackingUnits() []ecs.EntityID {
	// For now, highlight all alive units in the attacking squad
	return squads.GetUnitIDsInSquad(cam.attackerSquadID, cam.Queries.ECSManager)
}

// HandleInput handles input for the combat animation mode
func (cam *CombatAnimationMode) HandleInput(inputState *core.InputState) bool {
	// In waiting phase, any key dismisses
	if cam.animationPhase == PhaseWaiting {
		// Check for any key press
		for _, pressed := range inputState.KeysJustPressed {
			if pressed {
				cam.animationPhase = PhaseComplete
				return true
			}
		}

		// Also accept mouse click
		if inputState.MousePressed {
			cam.animationPhase = PhaseComplete
			return true
		}
	}

	// ESC can skip the animation entirely (goes straight to complete)
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		cam.animationPhase = PhaseComplete
		return true
	}

	return false
}
