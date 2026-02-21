package guistartmenu

import (
	"fmt"
	"image/color"

	"game_main/config"
	"game_main/gui/builders"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// GameModeChoice represents the player's selection from the start menu.
type GameModeChoice int

const (
	ModeNone GameModeChoice = iota
	ModeOverworld
	ModeRoguelike
)

// StartMenu is a self-contained pre-game menu screen.
type StartMenu struct {
	ui              *ebitenui.UI
	selected        GameModeChoice
	showSettings    bool
	settingsUI      *ebitenui.UI
	settingsMessage string // Message to show after saving (e.g., "Restart to apply")
}

// NewStartMenu builds the start menu UI.
func NewStartMenu() *StartMenu {
	sm := &StartMenu{}

	// Center column: title + buttons, stacked vertically
	centerColumn := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(20),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	// Title
	title := builders.CreateLargeLabel("TinkerRogue")
	centerColumn.AddChild(title)

	// Overworld button
	overworldBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Start Overworld Mode",
		OnClick: func() {
			sm.selected = ModeOverworld
		},
	})
	centerColumn.AddChild(overworldBtn)

	// Roguelike button
	roguelikeBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Start Roguelike Mode",
		OnClick: func() {
			sm.selected = ModeRoguelike
		},
	})
	centerColumn.AddChild(roguelikeBtn)

	// Settings button
	settingsBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Settings",
		OnClick: func() {
			sm.showSettings = true
			sm.settingsUI = buildSettingsUI(sm)
		},
	})
	centerColumn.AddChild(settingsBtn)

	// Root container fills the screen and centers the column
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	rootContainer.AddChild(centerColumn)

	sm.ui = &ebitenui.UI{Container: rootContainer}
	return sm
}

// buildSettingsUI creates the settings panel with resolution selection.
func buildSettingsUI(sm *StartMenu) *ebitenui.UI {
	// Build resolution preset labels
	presetLabels := make([]string, len(config.ResolutionPresets))
	currentIdx := 0
	for i, preset := range config.ResolutionPresets {
		label := preset.Label
		if config.CurrentSettings != nil &&
			preset.Width == config.CurrentSettings.ResolutionWidth &&
			preset.Height == config.CurrentSettings.ResolutionHeight {
			label += " (current)"
			currentIdx = i
		}
		presetLabels[i] = label
	}

	// Convert to []interface{} for the list
	entries := make([]interface{}, len(presetLabels))
	for i, l := range presetLabels {
		entries[i] = l
	}

	centerColumn := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	// Title
	title := builders.CreateLargeLabel("Resolution Settings")
	centerColumn.AddChild(title)

	// Current resolution display
	currentLabel := builders.CreateSmallLabel(
		fmt.Sprintf("Current: %dx%d", config.CurrentSettings.ResolutionWidth, config.CurrentSettings.ResolutionHeight),
	)
	centerColumn.AddChild(currentLabel)

	// Resolution list
	resList := builders.CreateListWithConfig(builders.ListConfig{
		Entries:   entries,
		MinWidth:  400,
		MinHeight: 300,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
	})

	// Select the current resolution entry
	if currentIdx < len(entries) {
		resList.SetSelectedEntry(entries[currentIdx])
	}

	centerColumn.AddChild(resList)

	// Status label for messages
	statusLabel := builders.CreateSmallLabel("")
	centerColumn.AddChild(statusLabel)

	// Button row
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(15),
		)),
	)

	// Apply button
	applyBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Apply",
		OnClick: func() {
			selected := resList.SelectedEntry()
			if selected == nil {
				return
			}
			// Find which preset was selected
			selectedStr := selected.(string)
			for i, label := range presetLabels {
				if label == selectedStr {
					preset := config.ResolutionPresets[i]
					config.CurrentSettings.ResolutionWidth = preset.Width
					config.CurrentSettings.ResolutionHeight = preset.Height
					if err := config.SaveUserSettings("settings.json"); err != nil {
						statusLabel.Label = fmt.Sprintf("Error: %v", err)
					} else {
						statusLabel.Label = fmt.Sprintf("Saved %dx%d - Restart to apply", preset.Width, preset.Height)
					}
					break
				}
			}
		},
	})
	buttonContainer.AddChild(applyBtn)

	// Back button
	backBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Back",
		OnClick: func() {
			sm.showSettings = false
		},
	})
	buttonContainer.AddChild(backBtn)

	centerColumn.AddChild(buttonContainer)

	// Root container
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	rootContainer.AddChild(centerColumn)

	return &ebitenui.UI{Container: rootContainer}
}

// Update processes UI input for the start menu.
func (sm *StartMenu) Update() {
	if sm.showSettings && sm.settingsUI != nil {
		sm.settingsUI.Update()
		return
	}
	sm.ui.Update()
}

// Draw renders the start menu to the screen.
func (sm *StartMenu) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	if sm.showSettings && sm.settingsUI != nil {
		sm.settingsUI.Draw(screen)
		return
	}
	sm.ui.Draw(screen)
}

// GetSelection returns the player's menu choice (ModeNone if nothing selected yet).
func (sm *StartMenu) GetSelection() GameModeChoice {
	return sm.selected
}
