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

	// Cached widget references (populated once in initWidgets)
	titleLabel  *widget.Text
	summaryText *widget.Text
	continueBtn *widget.Button
}

// NewSummaryPanel creates a new summary panel controller.
func NewSummaryPanel(mode *RaidMode) *SummaryPanel {
	sp := &SummaryPanel{mode: mode}
	sp.initWidgets()
	sp.wireButtons()
	return sp
}

// initWidgets extracts widget references from the panel registry once.
func (sp *SummaryPanel) initWidgets() {
	sp.titleLabel = framework.GetPanelWidget[*widget.Text](sp.mode.Panels, RaidPanelSummary, "titleLabel")
	sp.summaryText = framework.GetPanelWidget[*widget.Text](sp.mode.Panels, RaidPanelSummary, "summaryText")
	sp.continueBtn = framework.GetPanelWidget[*widget.Button](sp.mode.Panels, RaidPanelSummary, "continueBtn")
}

// wireButtons connects the continue button callback.
func (sp *SummaryPanel) wireButtons() {
	if sp.continueBtn != nil {
		sp.continueBtn.Configure(
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				sp.mode.OnSummaryDismissed()
			}),
		)
	}
}

// Refresh updates the summary panel with encounter results.
func (sp *SummaryPanel) Refresh(result *raid.RaidEncounterResult) {
	if result == nil {
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

	if sp.summaryText != nil {
		sp.summaryText.Label = strings.Join(lines, "\n")
	}
}

// HandleInput processes summary panel input.
func (sp *SummaryPanel) HandleInput(inputState *framework.InputState) bool {
	if inputState.ActionActive(framework.ActionConfirm) || inputState.ActionActive(framework.ActionDismiss) {
		sp.mode.OnSummaryDismissed()
		return true
	}
	return false
}

// Render draws summary visuals.
func (sp *SummaryPanel) Render(screen *ebiten.Image) {
	// Widget rendering is handled by ebitenui.
}
