package gui

import (
	"game_main/graphics"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
)

// Don't think I really need this, since all it has is a text area.
type PlayerStatsUI struct {
	StatUIContainer *widget.Container
	StatsTextArea   *widget.TextArea //Displays the properties of the selected items

}

// Text window to display the item properties of the selected items to the player
func (statsUI *PlayerStatsUI) CreateStatsTextArea() *widget.TextArea {

	//

	xSize := graphics.StatsUIOffset              //Only here for consistency. Used to fill up the X dimension of the GUI part
	ySize := graphics.ScreenInfo.LevelHeight / 4 //The GUI takes up 1/4th of the level height
	// construct a textarea

	//return CreateTextArea(xSize, ySize)

	return widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(

			widget.ContainerOpts.WidgetOpts(

				widget.WidgetOpts.MinSize(xSize, ySize),
			),
		),
		//widget.TextAreaOpts.ContainerOpts(),
		//Set gap between scrollbar and text
		widget.TextAreaOpts.ControlWidgetSpacing(2),
		//Tell the textarea to display bbcodes
		widget.TextAreaOpts.ProcessBBCode(true),
		//Set the font color
		widget.TextAreaOpts.FontColor(color.White),
		//Set the font face (size) to use
		widget.TextAreaOpts.FontFace(TextAreaRes.face),

		//Tell the TextArea to show the vertical scrollbar
		//widget.TextAreaOpts.ShowVerticalScrollbar(),
		//Set padding between edge of the widget and where the text is drawn
		widget.TextAreaOpts.TextPadding(TextAreaRes.entryPadding),
		//This sets the background images for the scroll container

		widget.TextAreaOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(ListRes.image)),

		//This sets the images to use for the sliders
		widget.TextAreaOpts.SliderOpts(
			widget.SliderOpts.Images(ListRes.track, ListRes.handle),
			widget.SliderOpts.MinHandleSize(ListRes.handleSize),
			widget.SliderOpts.TrackPadding(ListRes.trackPadding),
		),
	)

}

func (statsUI *PlayerStatsUI) CreateStatsUI() {
	// construct a new container that serves as the root of the UI hierarchy

	// construct a new container that serves as the root of the UI hierarchy
	statsUI.StatUIContainer = widget.NewContainer(

		//widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(3000, 100)),

		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(30)),
		)),
	)

	statsUI.StatsTextArea = statsUI.CreateStatsTextArea()

	//statsUI.StatsTextArea..

	statsUI.StatUIContainer.AddChild(statsUI.StatsTextArea)

	/*
		r := image.Rect(500, 500, 500, 500)
		r = r.Add(image.Point{2000, 2000})
		statsUI.StatsTextArea.SetLocation(r)
	*/

	//statsUI.StatsTextArea2 = CreateTextArea()

}
