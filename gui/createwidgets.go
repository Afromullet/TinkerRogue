package gui

import (
	"image"
	"image/color"
	_ "image/png"

	"github.com/ebitenui/ebitenui/widget"
)

// Text window to display the item properties of the selected items to the player
func CreateTextArea(minSizeX, minSizeY int) *widget.TextArea {
	// construct a textarea
	return widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(

			widget.ContainerOpts.WidgetOpts(

				//Set the minimum size for the widget
				widget.WidgetOpts.MinSize(minSizeX, minSizeX),
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

func CreateButton(text string) *widget.Button {

	// construct a button
	button := widget.NewButton(
		// set general widget options
		widget.ButtonOpts.WidgetOpts(
			// instruct the container's anchor layout to center the button both horizontally and vertically
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),

			widget.WidgetOpts.MinSize(100, 100),
		),

		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text(text, largeFace, &widget.ButtonTextColor{
			Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
		}),

		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:   30,
			Right:  30,
			Top:    30,
			Bottom: 30,
		}),
	)

	return button

}

func SetContainerLocation(w *widget.Container, x, y int) {

	r := image.Rect(0, 0, 0, 0)
	r = r.Add(image.Point{x, y})

	w.SetLocation(r)

}

type StringDisplay interface {
	DisplayString()
}
