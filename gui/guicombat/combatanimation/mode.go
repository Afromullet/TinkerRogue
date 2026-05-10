package combatanimation

import (
	"image/color"

	"game_main/gui/framework"
	"game_main/visual/combatrender"

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
	squadRenderer *combatrender.SquadCombatRenderer

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

	// Cached grid background image (regenerated only when cellSize changes)
	cachedGridImage    *ebiten.Image
	cachedGridCellSize int

	// Input action map
	actionMap *framework.ActionMap
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

// SetOnComplete sets the callback to execute when animation completes
func (cam *CombatAnimationMode) SetOnComplete(callback func()) {
	cam.onAnimationComplete = callback
}

// SetAutoPlay enables auto-play mode (for AI attacks)
func (cam *CombatAnimationMode) SetAutoPlay(autoPlay bool) {
	cam.autoPlay = autoPlay
}

// ResetForNextAttack resets animation state to replay another attack
func (cam *CombatAnimationMode) ResetForNextAttack() {
	cam.animationPhase = PhaseIdle
	cam.animationTimer = 0
	cam.flashTimer = 0

	for defenderID := range cam.defenderFlashIndex {
		cam.defenderFlashIndex[defenderID] = 0
	}
}

// Initialize sets up the combat animation mode
func (cam *CombatAnimationMode) Initialize(ctx *framework.UIContext) error {
	cam.screenWidth = ctx.ScreenWidth
	cam.screenHeight = ctx.ScreenHeight
	cam.calculateLayout()

	err := framework.NewModeBuilder(&cam.BaseMode, framework.ModeConfig{
		ModeName:   "combat_animation",
		ReturnMode: "combat",
	}).Build(ctx)

	if err != nil {
		return err
	}

	cam.actionMap = framework.DefaultCombatAnimationBindings()

	if err := cam.BuildPanels(CombatAnimationPanelPrompt); err != nil {
		return err
	}

	cam.promptLabel = GetCombatAnimationPromptLabel(cam.Panels)

	return nil
}

// calculateLayout computes the positions and sizes for rendering
func (cam *CombatAnimationMode) calculateLayout() {
	cam.cellSize = cam.screenWidth / 12
	if cam.cellSize > 96 {
		cam.cellSize = 96
	}
	if cam.cellSize < 32 {
		cam.cellSize = 32
	}

	cam.gridWidth = 3 * cam.cellSize
	cam.gridHeight = 3 * cam.cellSize

	gap := cam.screenWidth / 8
	centerY := (cam.screenHeight - cam.gridHeight) / 2
	centerX := cam.screenWidth / 2

	cam.attackerX = centerX - gap/2 - cam.gridWidth
	cam.attackerY = centerY

	cam.defenderX = centerX + gap/2
	cam.defenderY = centerY
}

// Enter is called when switching to this mode
func (cam *CombatAnimationMode) Enter(fromMode framework.UIMode) error {
	cam.animationPhase = PhaseIdle
	cam.animationTimer = 0
	cam.promptLabel.Label = ""
	return nil
}

// Exit is called when switching from this mode
func (cam *CombatAnimationMode) Exit(toMode framework.UIMode) error {
	cam.attackerColors = nil
	cam.defenderColorList = nil
	cam.defenderFlashIndex = nil
	cam.flashTimer = 0
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
		cam.flashTimer += deltaTime

		if cam.flashTimer >= DefenderFlashDuration {
			cam.flashTimer = 0
			for defenderID := range cam.defenderColorList {
				cam.defenderFlashIndex[defenderID]++
				if cam.defenderFlashIndex[defenderID] >= len(cam.defenderColorList[defenderID]) {
					cam.defenderFlashIndex[defenderID] = 0
				}
			}
		}

		if cam.animationTimer >= AttackingDuration {
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
		if cam.onAnimationComplete != nil {
			callback := cam.onAnimationComplete
			cam.onAnimationComplete = nil
			callback()
		} else {
			if combatMode, exists := cam.ModeManager.GetMode("combat"); exists {
				cam.ModeManager.RequestTransition(combatMode, "Animation complete")
			}
		}
	}

	return nil
}

// Render draws the combat animation scene
func (cam *CombatAnimationMode) Render(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 30, B: 40, A: 255})

	cam.drawGridBackground(screen, cam.attackerX, cam.attackerY)
	cam.drawGridBackground(screen, cam.defenderX, cam.defenderY)

	switch cam.animationPhase {
	case PhaseIdle:
		cam.squadRenderer.RenderSquad(screen, cam.attackerSquadID, cam.attackerX, cam.attackerY, cam.cellSize, false)
		cam.squadRenderer.RenderSquad(screen, cam.defenderSquadID, cam.defenderX, cam.defenderY, cam.cellSize, true)

	case PhaseAttacking:
		cam.renderSquadWithUnitColors(screen, cam.attackerSquadID, cam.attackerX, cam.attackerY, cam.cellSize, false, true)
		cam.renderSquadWithUnitColors(screen, cam.defenderSquadID, cam.defenderX, cam.defenderY, cam.cellSize, true, false)

	case PhaseWaiting, PhaseComplete:
		cam.squadRenderer.RenderSquad(screen, cam.attackerSquadID, cam.attackerX, cam.attackerY, cam.cellSize, false)
		cam.squadRenderer.RenderSquad(screen, cam.defenderSquadID, cam.defenderX, cam.defenderY, cam.cellSize, true)
	}

	cam.drawSquadNames(screen)
}

// GetActionMap returns the action map for combat animation mode.
func (cam *CombatAnimationMode) GetActionMap() *framework.ActionMap {
	return cam.actionMap
}

// HandleInput handles input for the combat animation mode
func (cam *CombatAnimationMode) HandleInput(inputState *framework.InputState) bool {
	if cam.animationPhase == PhaseWaiting {
		if inputState.ActionActive(framework.ActionReplayAnimation) {
			cam.animationPhase = PhaseIdle
			cam.animationTimer = 0
			cam.flashTimer = 0
			for defenderID := range cam.defenderFlashIndex {
				cam.defenderFlashIndex[defenderID] = 0
			}
			cam.promptLabel.Label = ""
			return true
		}

		if inputState.AnyKeyJustPressed() || inputState.ActionActive(framework.ActionMouseClick) {
			cam.animationPhase = PhaseComplete
			return true
		}
	}

	if inputState.ActionActive(framework.ActionCancel) {
		cam.animationPhase = PhaseComplete
		return true
	}

	return false
}
