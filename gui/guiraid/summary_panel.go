package guiraid

import (
	"fmt"
	"strings"

	"game_main/gui/framework"
	"game_main/mind/raid"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SummaryPanel controls the post-encounter summary display.
type SummaryPanel struct {
	mode *RaidMode
}

// NewSummaryPanel creates a new summary panel controller.
func NewSummaryPanel(mode *RaidMode) *SummaryPanel {
	sp := &SummaryPanel{mode: mode}
	sp.wireButtons()
	return sp
}

// wireButtons connects the continue button callback.
func (sp *SummaryPanel) wireButtons() {
	panel := sp.mode.Panels.Get(RaidPanelSummary)
	if panel == nil {
		return
	}

	if btn, ok := panel.Custom["continueBtn"].(*widget.Button); ok {
		btn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				sp.mode.OnSummaryDismissed()
			}),
		)
	}
}

// Refresh updates the summary panel with encounter results.
func (sp *SummaryPanel) Refresh(result *raid.RaidEncounterResult) {
	panel := sp.mode.Panels.Get(RaidPanelSummary)
	if panel == nil || result == nil {
		return
	}

	// Build summary text
	var lines []string
	lines = append(lines, fmt.Sprintf("Room Cleared: %s (%s)", result.RoomName, result.RoomType))
	lines = append(lines, "")

	if result.UnitsLost > 0 {
		lines = append(lines, fmt.Sprintf("Units Lost: %d", result.UnitsLost))
	} else {
		lines = append(lines, "Units Lost: None")
	}

	lines = append(lines, fmt.Sprintf("Alert Level: %d", result.AlertLevel))

	if result.RewardText != "" {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Reward: %s", result.RewardText))
	}

	if summaryText, ok := panel.Custom["summaryText"].(*widget.Text); ok {
		summaryText.Label = strings.Join(lines, "\n")
	}
}

// HandleInput processes summary panel input.
func (sp *SummaryPanel) HandleInput(inputState *framework.InputState) bool {
	if inputState.KeysJustPressed[ebiten.KeyEnter] || inputState.KeysJustPressed[ebiten.KeySpace] {
		sp.mode.OnSummaryDismissed()
		return true
	}
	return false
}

// Render draws summary visuals.
func (sp *SummaryPanel) Render(screen *ebiten.Image) {
	// Widget rendering is handled by ebitenui.
}
