// Package guiprogression provides a standalone UI mode for the permanent
// progression library: Arcana Points for spells, Skill Points for perks.
// The panel is player-scoped and operates on the active Player entity's
// ProgressionData component.
package guiprogression

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/tactical/powers/progression"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ProgressionMode lets the player spend Arcana Points and Skill Points to
// permanently unlock spells and perks. Opened from the squad editor.
type ProgressionMode struct {
	framework.BaseMode

	controller *progressionController

	// Input action map
	actionMap *framework.ActionMap
}

func NewProgressionMode(modeManager *framework.UIModeManager) *ProgressionMode {
	mode := &ProgressionMode{}
	mode.SetModeName("progression_manager")
	mode.SetReturnMode("squad_editor")
	mode.ModeManager = modeManager
	mode.SetSelf(mode)
	return mode
}

// GetActionMap implements framework.ActionMapProvider.
func (pm *ProgressionMode) GetActionMap() *framework.ActionMap {
	return pm.actionMap
}

// activePlayerID returns the currently active Player entity's ID (0 if unavailable).
func (pm *ProgressionMode) activePlayerID() ecs.EntityID {
	if pm.Context == nil || pm.Context.PlayerData == nil {
		return 0
	}
	return pm.Context.PlayerData.PlayerEntityID
}

// onAddSkillPoint grants 1 Skill Point and refreshes the UI (debug).
func (pm *ProgressionMode) onAddSkillPoint() {
	progression.AddSkillPoints(pm.activePlayerID(), 1, pm.Context.ECSManager)
	pm.controller.refresh()
}

// onAddArcanaPoint grants 1 Arcana Point and refreshes the UI (debug).
func (pm *ProgressionMode) onAddArcanaPoint() {
	progression.AddArcanaPoints(pm.activePlayerID(), 1, pm.Context.ECSManager)
	pm.controller.refresh()
}

func (pm *ProgressionMode) Initialize(ctx *framework.UIContext) error {
	if err := framework.NewModeBuilder(&pm.BaseMode, framework.ModeConfig{
		ModeName:    "progression_manager",
		ReturnMode:  "squad_editor",
		StatusLabel: true,
	}).Build(ctx); err != nil {
		return err
	}

	pm.actionMap = framework.DefaultProgressionBindings()

	if err := pm.BuildPanels(ProgressionPanelHeader, ProgressionPanelPerks, ProgressionPanelSpells); err != nil {
		return err
	}

	pm.controller = newProgressionController(pm)
	pm.controller.initWidgets()

	pm.RootContainer.AddChild(pm.buildNavigationActions())
	return nil
}

func (pm *ProgressionMode) Enter(fromMode framework.UIMode) error {
	pm.controller.refresh()
	return nil
}

func (pm *ProgressionMode) Exit(toMode framework.UIMode) error {
	return nil
}

func (pm *ProgressionMode) Update(deltaTime float64) error {
	return nil
}

func (pm *ProgressionMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (pm *ProgressionMode) HandleInput(inputState *framework.InputState) bool {
	return pm.HandleCommonInput(inputState)
}

func (pm *ProgressionMode) buildNavigationActions() *widget.Container {
	return builders.CreateRightActionBar(pm.Layout, []builders.ButtonSpec{
		{Text: "Back (ESC)", OnClick: func() {
			if returnMode, exists := pm.ModeManager.GetMode(pm.GetReturnMode()); exists {
				pm.ModeManager.RequestTransition(returnMode, "Back button pressed")
			}
		}},
	})
}
