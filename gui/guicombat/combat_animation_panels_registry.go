package guicombat

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/visual/rendering"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for combat animation mode
const (
	CombatAnimationPanelPrompt framework.PanelType = "combat_animation_prompt"
)

func init() {
	// Register prompt panel
	framework.RegisterPanel(CombatAnimationPanelPrompt, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			cam := mode.(*CombatAnimationMode)

			// Create squad renderer (needs Queries which is set by ModeBuilder)
			if cam.squadRenderer == nil {
				cam.squadRenderer = rendering.NewSquadCombatRenderer(cam.Queries)
			}

			// Create prompt label (centered at bottom)
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
