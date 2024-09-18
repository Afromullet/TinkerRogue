package gui

import (
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// Don't think I really need this, since all it has is a text area.
type PlayerStatsUI struct {
	StatsTextArea *widget.TextArea //Displays the properties of the selected items

}

func (statsUI *PlayerStatsUI) CreateStatsUI() {
	statsUI.CreateStatsTextArea()

}

// Text window to display the item properties of the selected items to the player
func (statsUI *PlayerStatsUI) CreateStatsTextArea() {
	// construct a textarea
	statsUI.StatsTextArea = widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(

			widget.ContainerOpts.WidgetOpts(

				widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionEnd,
					VerticalPosition:   widget.AnchorLayoutPositionStart,
					StretchHorizontal:  false,
					StretchVertical:    false,
				}),

				//Set the minimum size for the widget
				widget.WidgetOpts.MinSize(300, 100),
			),
		),
		//widget.TextAreaOpts.ContainerOpts(),
		//Set gap between scrollbar and text
		widget.TextAreaOpts.ControlWidgetSpacing(2),
		//Tell the textarea to display bbcodes
		widget.TextAreaOpts.ProcessBBCode(true),
		//Set the font color
		widget.TextAreaOpts.FontColor(color.Black),
		//Set the font face (size) to use
		widget.TextAreaOpts.FontFace(face),

		//Tell the TextArea to show the vertical scrollbar
		widget.TextAreaOpts.ShowVerticalScrollbar(),
		//Set padding between edge of the widget and where the text is drawn
		widget.TextAreaOpts.TextPadding(widget.NewInsetsSimple(10)),
		//This sets the background images for the scroll container
		widget.TextAreaOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
				Idle: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Mask: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
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
