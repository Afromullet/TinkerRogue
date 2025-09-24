package gui

import (
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"game_main/monsters"

	"image"
	"image/color"

	"github.com/ebitenui/ebitenui"
	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

var LookAtCreatureOpt = "Look at Creature"
var LookAtTileOpt = "Look at Tile"

type InfoUI struct {
	InfoOptionsContainer *widget.Container //Holds all of the GUI elements for displaying the options - Look at Creature, Look at Tile
	InfoOptionsWindow    *widget.Window    //Window for the InfoOptions container
	InfoOptionList       *widget.List      //List containiner the options the user can select

	DisplayInfoContainer *widget.Container //Holds all of the GUI elements for displaying the data after selecting the option
	DisplayInfoWindow    *widget.Window    //Window to hold the root container content
	DisplayInfoTextArea  *widget.TextArea

	ecsmnager         *common.EntityManager
	baseContainer     *ebitenui.UI //Ebiten base container
	windowX, windowY  int
	removeHandlerFunc func()
}

func (info *InfoUI) InfoSelectionWindow(cursorX, cursorY int) {

	x, y := info.InfoOptionsWindow.Contents.PreferredSize()

	r := image.Rect(0, 0, x, y)

	//r = r.Add(image.Point{cursorX, cursorY})

	r = r.Add(image.Point{graphics.ScreenInfo.LevelWidth / 2, graphics.ScreenInfo.LevelHeight / 2}) //Shouldn't this be level width? todo
	info.InfoOptionsWindow.SetLocation(r)

	info.windowX = cursorX
	info.windowY = cursorY

	info.baseContainer.AddWindow(info.InfoOptionsWindow)

	addInfoListHandler(info.InfoOptionList, info.ecsmnager, info)

}

func (info *InfoUI) CreatureInfoWindow(cursorX, cursorY int) {

	x, y := info.DisplayInfoWindow.Contents.PreferredSize()

	r := image.Rect(0, 0, x, y)
	//r = r.Add(image.Point{cursorX + 50, cursorY + 50})
	r = r.Add(image.Point{graphics.ScreenInfo.LevelWidth / 2, graphics.ScreenInfo.LevelHeight / 2}) //Shouldn't this be level width? todo
	info.DisplayInfoWindow.SetLocation(r)

	info.windowX = cursorX
	info.windowY = cursorY

	info.baseContainer.AddWindow(info.DisplayInfoWindow)

}

func (info *InfoUI) CloseWindows() {

	info.InfoOptionsWindow.Close()
	info.DisplayInfoWindow.Close()

}

func CreateInfoUI(ecsmanager *common.EntityManager, ui *ebitenui.UI) InfoUI {

	infoUI := InfoUI{}
	// Holds the widget that displays the selected items to the player
	infoUI.InfoOptionsContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

	//The window will open whenever a player right clicks on a tile
	infoUI.InfoOptionsWindow = widget.NewWindow(

		widget.WindowOpts.Contents(infoUI.InfoOptionsContainer),

		//	widget.WindowOpts.TitleBar(titleContainer, 25),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.NONE),
		//widget.WindowOpts.CloseMode(widget.CLICK),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.Resizeable(),
		widget.WindowOpts.MinSize(500, 500),

		widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
			// Window moved
		}),
		//Set the callback that triggers when a resize is complete
		widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
			// Window resized
		}),
	)

	infoUI.InfoOptionList = createOptionList()
	//addInfoListHandler(infoUI.InfoOptionList, infoUI.ecsmnager, &infoUI)
	infoUI.InfoOptionsContainer.AddChild(infoUI.InfoOptionList)

	infoUI.DisplayInfoContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

	//The window will open whenever a player right clicks on a tile
	infoUI.DisplayInfoWindow = widget.NewWindow(

		widget.WindowOpts.Contents(infoUI.DisplayInfoContainer),

		//	widget.WindowOpts.TitleBar(titleContainer, 25),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.NONE),
		//widget.WindowOpts.CloseMode(widget.CLICK),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.Resizeable(),
		widget.WindowOpts.MinSize(500, 500),

		widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
			// Window moved
		}),
		//Set the callback that triggers when a resize is complete
		widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
			// Window resized
		}),
	)

	infoUI.DisplayInfoTextArea = CreateTextArea(300, 300)

	infoUI.DisplayInfoContainer.AddChild(infoUI.DisplayInfoTextArea)

	infoUI.ecsmnager = ecsmanager

	infoUI.baseContainer = ui

	return infoUI

}

func createOptionList() *widget.List {

	infoOptions := make([]any, 0)
	infoOptions = append(infoOptions, LookAtCreatureOpt)
	infoOptions = append(infoOptions, LookAtTileOpt)

	li := widget.NewList(

		widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(150, 0),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				StretchVertical:    true,
				Padding:            widget.NewInsetsSimple(50),
			}),
		)),

		// Set the entries in the list
		widget.ListOpts.Entries(infoOptions),
		widget.ListOpts.ScrollContainerOpts(
			// Set the background images/color for the list
			widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
				Idle:     e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Disabled: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Mask:     e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
			}),
		),
		widget.ListOpts.SliderOpts(
			// Set the background images/color for the background of the slider track
			widget.SliderOpts.Images(&widget.SliderTrackImage{
				Idle:  e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Hover: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
			}, buttonImage),
			widget.SliderOpts.MinHandleSize(5),
			// Set how wide the track should be
			widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(2))),
		// Hide the horizontal slider
		widget.ListOpts.HideHorizontalSlider(),
		// Set the font for the list options
		widget.ListOpts.EntryFontFace(smallFace),
		// Set the colors for the list
		widget.ListOpts.EntryColor(&widget.ListEntryColor{
			Selected:                   color.NRGBA{R: 0, G: 255, B: 0, A: 255},     // Foreground color for the unfocused selected entry
			Unselected:                 color.NRGBA{R: 254, G: 255, B: 255, A: 255}, // Foreground color for the unfocused unselected entry
			SelectedBackground:         color.NRGBA{R: 130, G: 130, B: 200, A: 255}, // Background color for the unfocused selected entry
			SelectingBackground:        color.NRGBA{R: 130, G: 130, B: 130, A: 255}, // Background color for the unfocused being selected entry
			SelectingFocusedBackground: color.NRGBA{R: 130, G: 140, B: 170, A: 255}, // Background color for the focused being selected entry
			SelectedFocusedBackground:  color.NRGBA{R: 130, G: 130, B: 170, A: 255}, // Background color for the focused selected entry
			FocusedBackground:          color.NRGBA{R: 170, G: 170, B: 180, A: 255}, // Background color for the focused unselected entry
			DisabledUnselected:         color.NRGBA{R: 100, G: 100, B: 100, A: 255}, // Foreground color for the disabled unselected entry
			DisabledSelected:           color.NRGBA{R: 100, G: 100, B: 100, A: 255}, // Foreground color for the disabled selected entry
			DisabledSelectedBackground: color.NRGBA{R: 100, G: 100, B: 100, A: 255}, // Background color for the disabled selected entry
		}),
		// This required function returns the string displayed in the list
		widget.ListOpts.EntryLabelFunc(func(e interface{}) string {
			return e.(string)
		}),
		// Padding for each entry
		widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(5)),
		// Text position for each entry
		widget.ListOpts.EntryTextPosition(widget.TextPositionStart, widget.TextPositionCenter),
		// This handler defines what function to run when a list item is selected.

	)

	return li

}

func addInfoListHandler(li *widget.List, em *common.EntityManager, info *InfoUI) {

	if info.removeHandlerFunc != nil {
		info.removeHandlerFunc()
	}

	info.removeHandlerFunc = li.EntrySelectedEvent.AddHandler(func(args interface{}) {

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry

		pixelPos := coords.PixelPosition{X: info.windowX, Y: info.windowY}
		logicalPos := coords.CoordManager.PixelToLogical(pixelPos)
		pos := common.Position{X: logicalPos.X, Y: logicalPos.Y} // Direct conversion, avoiding compatibility layer

		if a.Entry == LookAtCreatureOpt {

			ent := common.GetCreatureAtPosition(em, &pos)
			//cr := tracker.CreatureTracker.Get(&pos)

			// If it's nil, there's no creature at the position
			if ent != nil {
				info.CreatureInfoWindow(info.windowX, info.windowY)

				creature := common.GetComponentType[*monsters.Creature](ent, monsters.CreatureComponent)

				info.DisplayInfoTextArea.SetText(creature.DisplayString(ent))
			}

			// Examining creature
		} else if entry == LookAtTileOpt {
			// Examining tile
		}

	})

}
