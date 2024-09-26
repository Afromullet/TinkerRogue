package gui

import (
	"game_main/graphics"
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// Don't think I really need this, since all it has is a text area.
type PlayerMessageUI struct {
	msgUIContainer     *widget.Container
	msgTextArea        *widget.TextArea //Displays the properties of the selected items
	currentNumMessages int
}

func (msgUI *PlayerMessageUI) AppendText(s string) {

	msgUI.msgTextArea.AppendText(s)

}

func (msgUI *PlayerMessageUI) SetText(s string) {

	msgUI.msgTextArea.SetText(s)

}

func (msgUI *PlayerMessageUI) SetTextWithArray(msg []string) {

	for _, s := range msg {
		msgUI.msgTextArea.AppendText(s + "\n")
	}

}

func (msgUI *PlayerMessageUI) ResetText() {

	msgUI.msgTextArea.SetText("")

}

// Text window to display the item properties of the selected items to the player
func (msgUI *PlayerMessageUI) CreateMsgTextArea() *widget.TextArea {

	//

	xSize := graphics.StatsUIOffset   //Only here for consistency. Used to fill up the X dimension of the GUI part
	ySize := graphics.LevelHeight / 4 //The GUI takes up 1/4th of the level height
	// construct a textarea
	return widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(

			widget.ContainerOpts.WidgetOpts(

				/*
					Commented out as I'm trying to figure out how to position windows
						widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
							HorizontalPosition: widget.AnchorLayoutPositionEnd,
							VerticalPosition:   widget.AnchorLayoutPositionStart,
							StretchHorizontal:  false,
							StretchVertical:    false,
						}),
				*/

				//Set the minimum size for the widget

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
		widget.TextAreaOpts.FontFace(largeFace),

		//Tell the TextArea to show the vertical scrollbar
		widget.TextAreaOpts.ShowVerticalScrollbar(),
		//Set padding between edge of the widget and where the text is drawn
		widget.TextAreaOpts.TextPadding(widget.NewInsetsSimple(10)),
		//This sets the background images for the scroll container
		widget.TextAreaOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
				Idle: defaultWidgetColor,
				Mask: defaultWidgetColor,
			}),
		),
		//This sets the images to use for the sliders
		widget.TextAreaOpts.SliderOpts(
			widget.SliderOpts.Images(
				// Set the track images
				&widget.SliderTrackImage{
					Idle:  e_image.NewNineSliceColor(color.NRGBA{200, 200, 200, 255}),
					Hover: e_image.NewNineSliceColor(color.NRGBA{200, 200, 200, 255}),
				},
				// Set the handle images
				&widget.ButtonImage{
					Idle:    e_image.NewNineSliceColor(color.NRGBA{255, 100, 100, 255}),
					Hover:   e_image.NewNineSliceColor(color.NRGBA{255, 100, 100, 255}),
					Pressed: e_image.NewNineSliceColor(color.NRGBA{255, 100, 100, 255}),
				},
			),
		),
	)

}

func (msgUI *PlayerMessageUI) CreatMsgUI() {
	// construct a new container that serves as the root of the UI hierarchy

	// construct a new container that serves as the root of the UI hierarchy
	msgUI.msgUIContainer = widget.NewContainer(

		//widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(3000, 100)),

		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(30)),
		)),
	)

	msgUI.msgTextArea = msgUI.CreateMsgTextArea()

	//statsUI.StatsTextArea..

	msgUI.msgUIContainer.AddChild(msgUI.msgTextArea)

	/*
		r := image.Rect(500, 500, 500, 500)
		r = r.Add(image.Point{2000, 2000})
		statsUI.StatsTextArea.SetLocation(r)
	*/

	//statsUI.StatsTextArea2 = CreateTextArea()

}
