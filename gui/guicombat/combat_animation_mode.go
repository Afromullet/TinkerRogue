package guicombat

import (
	"fmt"
	"image/color"
	"math"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/tactical/combat"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// AnimationPhase represents the current state of the combat animation
type AnimationPhase int

const (
	PhaseIdle      AnimationPhase = iota // Display both squads statically
	PhaseAttacking                       // Show attack animation effect
	PhaseWaiting                         // Wait for player input
	PhaseComplete                        // Animation done, execute callback
)

// Animation timing constants (in seconds)
const (
	IdleDuration          = 1.0
	AttackingDuration     = 2.0
	DefenderFlashDuration = 0.4 // seconds per color cycle
)

// createColorScale is a helper to create an ebiten ColorScale from RGB multipliers
func createColorScale(r, g, b float32) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.SetR(r)
	cs.SetG(g)
	cs.SetB(b)
	cs.SetA(1.0)
	return cs
}

// attackColorPalette defines the 9 distinct colors for attacking units
// Colors are carefully chosen for maximum contrast and visibility
var attackColorPalette = []ebiten.ColorScale{
	createColorScale(2.0, 0.3, 0.3), // 0: Vivid Red
	createColorScale(0.3, 2.0, 0.3), // 1: Vivid Green
	createColorScale(0.3, 0.3, 2.0), // 2: Vivid Blue
	createColorScale(2.0, 2.0, 0.0), // 3: Bright Yellow
	createColorScale(2.0, 0.0, 2.0), // 4: Vivid Magenta
	createColorScale(0.0, 2.0, 2.0), // 5: Vivid Cyan
	createColorScale(2.0, 1.0, 0.0), // 6: Bright Orange
	createColorScale(1.0, 0.0, 2.0), // 7: Deep Purple
	createColorScale(2.0, 0.3, 1.0), // 8: Hot Pink
}

// CombatAnimationMode displays a full-screen battle scene during combat.
// Shows both squads side-by-side with units at their grid positions.
type CombatAnimationMode struct {
	framework.BaseMode

	// Combat participants
	attackerSquadID ecs.EntityID
	defenderSquadID ecs.EntityID

	// Animation state
	animationPhase AnimationPhase
	animationTimer float64

	// Callback to execute after animation
	onAnimationComplete func()

	// Auto-play mode (for AI attacks - skips waiting for user input)
	autoPlay bool

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

	// Color animation state
	attackerColors     map[ecs.EntityID]ebiten.ColorScale
	defenderColorList  map[ecs.EntityID][]ebiten.ColorScale
	defenderFlashIndex map[ecs.EntityID]int
	flashTimer         float64
}

// NewCombatAnimationMode creates a new combat animation mode
func NewCombatAnimationMode(modeManager *framework.UIModeManager) *CombatAnimationMode {
	cam := &CombatAnimationMode{
		animationPhase: PhaseIdle,
	}
	cam.SetModeName("combat_animation")
	cam.SetReturnMode("combat") // ESC returns to combat mode
	cam.ModeManager = modeManager
	cam.SetSelf(cam) // Required for panel registry building
	return cam
}

// SetCombatants sets the attacker and defender squads for the animation
func (cam *CombatAnimationMode) SetCombatants(attackerID, defenderID ecs.EntityID) {
	cam.attackerSquadID = attackerID
	cam.defenderSquadID = defenderID

	// Initialize color maps
	cam.attackerColors = make(map[ecs.EntityID]ebiten.ColorScale)
	cam.defenderFlashIndex = make(map[ecs.EntityID]int)
	cam.defenderColorList = make(map[ecs.EntityID][]ebiten.ColorScale)
	cam.flashTimer = 0

	// Assign colors to attacking units
	combatSys := combat.NewCombatActionSystem(cam.Queries.ECSManager, cam.Queries.CombatCache)
	attackingUnits := combatSys.GetAttackingUnits(attackerID, defenderID)

	for i, attackerUnitID := range attackingUnits {
		colorIdx := i % len(attackColorPalette)
		cam.attackerColors[attackerUnitID] = attackColorPalette[colorIdx]
	}

	// Pre-compute defender color lists
	cam.computeDefenderColorLists(attackingUnits, defenderID)

	fmt.Printf("[DEBUG] SetCombatants: attacker=%d, defender=%d, attacking_units=%d\n", attackerID, defenderID, len(attackingUnits))
}

// SetOnComplete sets the callback to execute when animation completes
func (cam *CombatAnimationMode) SetOnComplete(callback func()) {
	cam.onAnimationComplete = callback
}

// SetAutoPlay enables auto-play mode (for AI attacks)
// When enabled, animation completes automatically without waiting for user input
func (cam *CombatAnimationMode) SetAutoPlay(autoPlay bool) {
	cam.autoPlay = autoPlay
}

// ResetForNextAttack resets animation state to replay another attack
// Used for chaining multiple AI attacks without mode transitions
func (cam *CombatAnimationMode) ResetForNextAttack() {
	cam.animationPhase = PhaseIdle
	cam.animationTimer = 0
	cam.flashTimer = 0

	// Reset defender flash indices
	for defenderID := range cam.defenderFlashIndex {
		cam.defenderFlashIndex[defenderID] = 0
	}
}

// computeDefenderColorLists maps which attacker colors should target each defender
func (cam *CombatAnimationMode) computeDefenderColorLists(
	attackingUnits []ecs.EntityID,
	defenderSquadID ecs.EntityID,
) {
	// Build map: defender → list of attacking unit colors
	defenderToAttackers := make(map[ecs.EntityID][]ecs.EntityID)

	for _, attackerID := range attackingUnits {
		// Get this attacker's target cells
		targetRowData := common.GetComponentTypeByID[*squads.TargetRowData](
			cam.Queries.ECSManager, attackerID, squads.TargetRowComponent,
		)
		if targetRowData == nil {
			continue
		}

		// Find defenders using new targeting system
		targets := squads.SelectTargetUnits(
			attackerID, defenderSquadID, cam.Queries.ECSManager,
		)

		for _, defenderID := range targets {
			defenderToAttackers[defenderID] = append(
				defenderToAttackers[defenderID], attackerID,
			)
		}
	}

	// Convert to color lists
	for defenderID, attackerList := range defenderToAttackers {
		var colorList []ebiten.ColorScale
		for _, attackerID := range attackerList {
			if color, exists := cam.attackerColors[attackerID]; exists {
				colorList = append(colorList, color)
			}
		}
		cam.defenderColorList[defenderID] = colorList
		cam.defenderFlashIndex[defenderID] = 0
	}
}

// Initialize sets up the combat animation mode
func (cam *CombatAnimationMode) Initialize(ctx *framework.UIContext) error {
	// Store screen dimensions and calculate layout before ModeBuilder
	cam.screenWidth = ctx.ScreenWidth
	cam.screenHeight = ctx.ScreenHeight
	cam.calculateLayout()

	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&cam.BaseMode, framework.ModeConfig{
		ModeName:   "combat_animation",
		ReturnMode: "combat",
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := cam.BuildPanels(CombatAnimationPanelPrompt); err != nil {
		return err
	}

	// Initialize widget references from registry
	cam.promptLabel = GetCombatAnimationPromptLabel(cam.Panels)

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
func (cam *CombatAnimationMode) Enter(fromMode framework.UIMode) error {
	// Reset animation state
	cam.animationPhase = PhaseIdle
	cam.animationTimer = 0

	// Update prompt based on auto-play mode
	if cam.autoPlay {
		cam.promptLabel.Label = "" // No prompt in auto-play mode
	} else {
		cam.promptLabel.Label = ""
	}

	return nil
}

// Exit is called when switching from this mode
func (cam *CombatAnimationMode) Exit(toMode framework.UIMode) error {
	// Clear color state
	cam.attackerColors = nil
	cam.defenderColorList = nil
	cam.defenderFlashIndex = nil
	cam.flashTimer = 0

	// Reset auto-play for next use
	cam.autoPlay = false

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
		// Update flash timer for defenders
		cam.flashTimer += deltaTime

		if cam.flashTimer >= DefenderFlashDuration {
			cam.flashTimer = 0
			// Advance all defenders to next color
			for defenderID := range cam.defenderColorList {
				cam.defenderFlashIndex[defenderID]++
				// Wrap around
				if cam.defenderFlashIndex[defenderID] >= len(cam.defenderColorList[defenderID]) {
					cam.defenderFlashIndex[defenderID] = 0
				}
			}
		}

		if cam.animationTimer >= AttackingDuration {
			// Auto-play mode: skip waiting and go straight to complete
			if cam.autoPlay {
				cam.animationPhase = PhaseComplete
			} else {
				cam.animationPhase = PhaseWaiting
				cam.promptLabel.Label = "Press SPACE to replay or any other key to continue..."
			}
			cam.animationTimer = 0
		}

	case PhaseWaiting:
		// Handled in HandleInput

	case PhaseComplete:
		// Execute callback
		if cam.onAnimationComplete != nil {
			callback := cam.onAnimationComplete
			cam.onAnimationComplete = nil // Clear callback to prevent re-execution

			// Don't automatically transition - let callback decide
			callback()
		} else {
			// No callback - return to combat mode (manual player attack)
			if combatMode, exists := cam.ModeManager.GetMode("combat"); exists {
				cam.ModeManager.RequestTransition(combatMode, "Animation complete")
			}
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
		// Render attackers with individual colors
		cam.renderSquadWithUnitColors(
			screen, cam.attackerSquadID, cam.attackerX, cam.attackerY,
			cam.cellSize, false, true, // isAttacker = true
		)

		// Render defenders with sequential flash
		cam.renderSquadWithUnitColors(
			screen, cam.defenderSquadID, cam.defenderX, cam.defenderY,
			cam.cellSize, true, false, // isAttacker = false
		)

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
	attackerName := cam.Queries.SquadCache.GetSquadName(cam.attackerSquadID)
	defenderName := cam.Queries.SquadCache.GetSquadName(cam.defenderSquadID)

	// For now, we'll rely on ebitenui to render text via the prompt label
	// Squad names could be drawn here using text rendering, but we'll keep it simple
	// The prompt label shows the current state
	_ = attackerName
	_ = defenderName
}

// getAttackHighlightColor returns a pulsing color for an attacking unit
func (cam *CombatAnimationMode) getAttackHighlightColor(unitID ecs.EntityID) *ebiten.ColorScale {
	baseColor, exists := cam.attackerColors[unitID]
	if !exists {
		// Fallback to default red pulse if unit has no assigned color
		return cam.getDefaultAttackPulse()
	}

	// Create pulsing effect using sine wave (slower: 1 full cycle per 3 seconds)
	pulse := float32(0.5 + 0.5*math.Sin(cam.animationTimer*2.0))

	colorScale := ebiten.ColorScale{}
	colorScale.SetR(baseColor.R() + pulse*0.3)
	colorScale.SetG(baseColor.G() + pulse*0.3)
	colorScale.SetB(baseColor.B() + pulse*0.3)
	colorScale.SetA(1.0)

	return &colorScale
}

// getDefaultAttackPulse returns a fallback red pulsing color
func (cam *CombatAnimationMode) getDefaultAttackPulse() *ebiten.ColorScale {
	pulse := float32(0.5 + 0.5*float64(cam.animationTimer/AttackingDuration))
	colorScale := &ebiten.ColorScale{}
	colorScale.SetR(1.0 + pulse*0.5)
	colorScale.SetG(1.0 - pulse*0.3)
	colorScale.SetB(1.0 - pulse*0.5)
	colorScale.SetA(1.0)
	return colorScale
}

// getDefenderHighlightColor returns the color for a defending unit to flash
func (cam *CombatAnimationMode) getDefenderHighlightColor(unitID ecs.EntityID) *ebiten.ColorScale {
	colorList, exists := cam.defenderColorList[unitID]
	if !exists || len(colorList) == 0 {
		return nil // Not targeted, no highlight
	}

	currentIndex := cam.defenderFlashIndex[unitID]
	if currentIndex >= len(colorList) {
		currentIndex = 0
	}

	return &colorList[currentIndex]
}

// renderSquadWithUnitColors renders a squad where each unit has its own color
func (cam *CombatAnimationMode) renderSquadWithUnitColors(
	screen *ebiten.Image, squadID ecs.EntityID,
	baseX, baseY, cellSize int, facingLeft bool, isAttacker bool,
) {
	unitIDs := cam.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		var colorScale *ebiten.ColorScale

		if isAttacker {
			colorScale = cam.getAttackHighlightColor(unitID)
		} else {
			colorScale = cam.getDefenderHighlightColor(unitID)
		}

		if colorScale != nil {
			cam.squadRenderer.RenderUnitWithColor(
				screen, unitID, baseX, baseY, cellSize, facingLeft, colorScale,
			)
		} else {
			cam.squadRenderer.RenderUnit(
				screen, unitID, baseX, baseY, cellSize, facingLeft,
			)
		}
	}
}

// HandleInput handles input for the combat animation mode
func (cam *CombatAnimationMode) HandleInput(inputState *framework.InputState) bool {
	// In waiting phase, Space replays the animation, any other key dismisses
	if cam.animationPhase == PhaseWaiting {
		// Space to replay animation
		if inputState.KeysJustPressed[ebiten.KeySpace] {
			cam.animationPhase = PhaseIdle
			cam.animationTimer = 0
			cam.flashTimer = 0
			// Reset defender flash indices to restart the color cycling
			for defenderID := range cam.defenderFlashIndex {
				cam.defenderFlashIndex[defenderID] = 0
			}
			cam.promptLabel.Label = ""
			return true
		}

		// Check for any other key press to dismiss
		for key, pressed := range inputState.KeysJustPressed {
			if pressed && key != ebiten.KeySpace {
				cam.animationPhase = PhaseComplete
				return true
			}
		}

		// Also accept mouse click to dismiss
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
