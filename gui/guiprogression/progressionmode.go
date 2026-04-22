// Package guiprogression provides a standalone UI mode for the permanent
// progression library: Arcana Points for spells, Skill Points for perks.
// The panel is commander-scoped and operates on a specific Commander entity's
// ProgressionData component (set via SetCommanderID before entering this mode).
package guiprogression

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/tactical/powers/progression"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ProgressionMode lets the player spend a commander's Arcana Points and Skill
// Points to permanently unlock spells and perks for that commander. Opened
// from the squad editor; the caller must call SetCommanderID before the
// transition so Enter can refresh against the right commander's data.
type ProgressionMode struct {
	framework.BaseMode

	controller *progressionController

	// Commander whose progression this mode edits. Set by SetCommanderID.
	activeCommander ecs.EntityID

	// Input action map
	actionMap *framework.ActionMap
}

// SetCommanderID selects which commander's progression is displayed. Call
// before RequestTransition into this mode.
func (pm *ProgressionMode) SetCommanderID(commanderID ecs.EntityID) {
	pm.activeCommander = commanderID
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

// activeCommanderID returns the commander entity ID whose progression is being
// edited. 0 if SetCommanderID was never called.
func (pm *ProgressionMode) activeCommanderID() ecs.EntityID {
	return pm.activeCommander
}

// onAddSkillPoint grants 1 Skill Point to the active commander and refreshes the UI (debug).
func (pm *ProgressionMode) onAddSkillPoint() {
	progression.AddSkillPoints(pm.activeCommanderID(), 1, pm.Context.ECSManager)
	pm.controller.refresh()
}

// onAddArcanaPoint grants 1 Arcana Point to the active commander and refreshes the UI (debug).
func (pm *ProgressionMode) onAddArcanaPoint() {
	progression.AddArcanaPoints(pm.activeCommanderID(), 1, pm.Context.ECSManager)
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
