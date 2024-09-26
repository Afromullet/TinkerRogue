package gui

import (
	"fmt"
	"image"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

type InfoUI struct {
	RootContainer *widget.Container //Holds all of the GUI elements
	RootWindow    *widget.Window    //Window to hold the root container content
}

func CreateInfoUI() InfoUI {

	infoUI := InfoUI{}
	// Holds the widget that displays the selected items to the player
	infoUI.RootContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

	infoUI.RootWindow = widget.NewWindow(

		widget.WindowOpts.Contents(infoUI.RootContainer),

		//	widget.WindowOpts.TitleBar(titleContainer, 25),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.Resizeable(),
		widget.WindowOpts.MinSize(500, 500),

		widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Moving")
		}),
		//Set the callback that triggers when a resize is complete
		widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Resized")
		}),
	)

	return infoUI

}

func (info *InfoUI) InfoSelectionWindow(ui *ebitenui.UI, cursorX, cursorY int) {

	x, y := info.RootWindow.Contents.PreferredSize()

	r := image.Rect(0, 0, x, y)
	r = r.Add(image.Point{cursorX, cursorY})
	info.RootWindow.SetLocation(r)

	ui.AddWindow(info.RootWindow)
}
