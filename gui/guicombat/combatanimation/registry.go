package combatanimation

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/visual/combatrender"

	"github.com/ebitenui/ebitenui/widget"
)

const (
	CombatAnimationPanelPrompt framework.PanelType = "combat_animation_prompt"
)

func init() {
	framework.RegisterPanel(CombatAnimationPanelPrompt, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cam := mode.(*CombatAnimationMode)

			if cam.squadRenderer == nil {
				cam.squadRenderer = combatrender.NewSquadCombatRenderer(cam.Queries)
			}

			promptLabel := builders.CreateLargeLabel("")

			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
						HorizontalPosition: widget.AnchorLayoutPositionCenter,
						VerticalPosition:   widget.AnchorLayoutPositionEnd,
						Padding:            widget.NewInsetsSimple(40),
					}),
				),
			)
			result.Container.AddChild(promptLabel)

			result.Custom["promptLabel"] = promptLabel

			return nil
		},
	})
}

// GetCombatAnimationPromptLabel returns the prompt label from the panel registry
func GetCombatAnimationPromptLabel(panels *framework.PanelRegistry) *widget.Text {
	if result := panels.Get(CombatAnimationPanelPrompt); result != nil {
		if label, ok := result.Custom["promptLabel"].(*widget.Text); ok {
			return label
		}
	}
	return nil
}
