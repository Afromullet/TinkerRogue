package guistartmenu

import (
	"image/color"

	"game_main/gui/builders"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// GameModeChoice represents the player's selection from the start menu.
type GameModeChoice int

const (
	ModeNone      GameModeChoice = iota
	ModeOverworld
	ModeRoguelike
)

// StartMenu is a self-contained pre-game menu screen.
type StartMenu struct {
	ui       *ebitenui.UI
	selected GameModeChoice
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

	// Root container fills the screen and centers the column
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	rootContainer.AddChild(centerColumn)

	sm.ui = &ebitenui.UI{Container: rootContainer}
	return sm
}

// Update processes UI input for the start menu.
func (sm *StartMenu) Update() {
	sm.ui.Update()
}

// Draw renders the start menu to the screen.
func (sm *StartMenu) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	sm.ui.Draw(screen)
}

// GetSelection returns the player's menu choice (ModeNone if nothing selected yet).
func (sm *StartMenu) GetSelection() GameModeChoice {
	return sm.selected
}
